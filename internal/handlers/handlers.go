package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"scorm-cmi-app/internal/models"
	"strings"
	"time"
)

type Handlers struct {
	db        *sql.DB
	templates *template.Template
}

func New(db *sql.DB) *Handlers {
	// Parse templates
	tmpl := template.Must(template.ParseGlob("web/templates/*.html"))

	return &Handlers{
		db:        db,
		templates: tmpl,
	}
}

// generateID creates a simple UUID-like ID
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// createSlug creates a URL-friendly slug from title
func createSlug(title string) string {
	// Convert to lowercase and replace spaces/special chars with hyphens
	reg := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	slug := reg.ReplaceAllString(strings.ToLower(title), "-")
	// Remove leading/trailing hyphens
	return strings.Trim(slug, "-")
}

// extractCourseIDFromPath extracts course ID from URL path like /courses/123#slug
func extractCourseIDFromPath(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) >= 3 {
		// Handle both /courses/123 and /courses/123#slug
		courseIDPart := parts[2]
		if hashIndex := strings.Index(courseIDPart, "#"); hashIndex > 0 {
			return courseIDPart[:hashIndex]
		}
		return courseIDPart
	}
	return ""
}

// sendJSON sends a JSON response
func (h *Handlers) sendJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// sendError sends an error response
func (h *Handlers) sendError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// HandleIndex serves the main page with recent courses
func (h *Handlers) HandleIndex(w http.ResponseWriter, r *http.Request) {
	// Get 3 most recent courses
	courses, err := h.getRecentCourses(3)
	if err != nil {
		log.Printf("Error fetching recent courses: %v", err)
		// Continue with empty courses list
	}
	
	data := map[string]interface{}{
		"Title":   "Dashboard",
		"Courses": courses,
	}
	
	if err := h.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// HandleCoursesListPage serves the courses list page
func (h *Handlers) HandleCoursesListPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/courses" {
		http.NotFound(w, r)
		return
	}

	data := map[string]interface{}{
		"Title":    "Courses",
		"PageType": "courses-list",
	}

	if err := h.templates.ExecuteTemplate(w, "courses-list.html", data); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// HandleCourseCreatePage serves the course creation page
func (h *Handlers) HandleCourseCreatePage(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		data := map[string]interface{}{
			"Title":    "Create Course",
			"PageType": "course-create",
		}

		if err := h.templates.ExecuteTemplate(w, "course-create.html", data); err != nil {
			log.Printf("Error rendering template: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}

	case "POST":
		h.createCourseFromForm(w, r)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleCourseDetailPage serves individual course pages
func (h *Handlers) HandleCourseDetailPage(w http.ResponseWriter, r *http.Request) {
	// Skip if this is the courses list page or create page
	if r.URL.Path == "/courses" || r.URL.Path == "/courses/" {
		http.NotFound(w, r)
		return
	}
	if r.URL.Path == "/courses/create" {
		http.NotFound(w, r)
		return
	}

	courseID := extractCourseIDFromPath(r.URL.Path)
	if courseID == "" || courseID == "create" {
		http.NotFound(w, r)
		return
	}

	// Handle different HTTP methods
	switch r.Method {
	case "GET":
		h.showCourseDetail(w, r, courseID)
	case "POST":
		// Check if this is an update (PUT-like operation via POST)
		if r.FormValue("_method") == "PUT" {
			h.updateCourseFromForm(w, r, courseID)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	case "DELETE":
		h.deleteCourse(w, r, courseID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleRegistrationsPage serves the registrations management page
func (h *Handlers) HandleRegistrationsPage(w http.ResponseWriter, r *http.Request) {
	if err := h.templates.ExecuteTemplate(w, "registrations.html", nil); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// HandleCourses handles course-related API requests
func (h *Handlers) HandleCourses(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.getCourses(w, r)
	case "POST":
		h.createCourse(w, r)
	default:
		h.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleCourse handles individual course operations
func (h *Handlers) HandleCourse(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		h.sendError(w, "Invalid course ID", http.StatusBadRequest)
		return
	}

	courseID := pathParts[3]

	switch r.Method {
	case "GET":
		if len(pathParts) > 4 && pathParts[4] == "launch" {
			h.launchCourse(w, r, courseID)
		} else {
			h.getCourse(w, r, courseID)
		}
	case "DELETE":
		h.deleteCourse(w, r, courseID)
	default:
		h.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getCourses retrieves all courses
func (h *Handlers) getCourses(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT id, title, description, version, package_path, created_at, updated_at 
		FROM courses ORDER BY created_at DESC
	`

	rows, err := h.db.Query(query)
	if err != nil {
		log.Printf("Error querying courses: %v", err)
		h.sendError(w, "Failed to retrieve courses", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var courses []models.Course
	for rows.Next() {
		var course models.Course
		var packagePath, version sql.NullString
		err := rows.Scan(&course.ID, &course.Title, &course.Description,
			&version, &packagePath, &course.CreatedAt, &course.UpdatedAt)
		if err != nil {
			log.Printf("Error scanning course: %v", err)
			continue
		}

		// Handle nullable fields
		if version.Valid {
			course.Version = version.String
		}
		if packagePath.Valid {
			course.PackagePath = &packagePath.String
		}

		courses = append(courses, course)
	}

	// Ensure we always return a valid JSON array, even if empty
	if courses == nil {
		courses = []models.Course{}
	}
	h.sendJSON(w, courses)
}

// createCourse creates a new course
func (h *Handlers) createCourse(w http.ResponseWriter, r *http.Request) {
	var req models.CourseUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	course := models.Course{
		ID:          generateID(),
		Title:       req.Title,
		Description: req.Description,
		Version:     "1.0",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	query := `
		INSERT INTO courses (id, title, description, version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := h.db.Exec(query, course.ID, course.Title, course.Description,
		course.Version, course.CreatedAt, course.UpdatedAt)
	if err != nil {
		log.Printf("Error creating course: %v", err)
		h.sendError(w, "Failed to create course", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	h.sendJSON(w, course)
}

// getCourse retrieves a specific course
func (h *Handlers) getCourse(w http.ResponseWriter, r *http.Request, courseID string) {
	query := `
		SELECT id, title, description, version, package_path, manifest, created_at, updated_at 
		FROM courses WHERE id = ?
	`

	var course models.Course
	var packagePath, manifest, version sql.NullString
	err := h.db.QueryRow(query, courseID).Scan(&course.ID, &course.Title, &course.Description,
		&version, &packagePath, &manifest, &course.CreatedAt, &course.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			h.sendError(w, "Course not found", http.StatusNotFound)
		} else {
			log.Printf("Error querying course: %v", err)
			h.sendError(w, "Failed to retrieve course", http.StatusInternalServerError)
		}
		return
	}

	// Handle nullable fields
	if version.Valid {
		course.Version = version.String
	}
	if packagePath.Valid {
		course.PackagePath = &packagePath.String
	}
	if manifest.Valid {
		course.Manifest = &manifest.String
	}

	h.sendJSON(w, course)
}

// launchCourse creates a registration and returns launch information
func (h *Handlers) launchCourse(w http.ResponseWriter, r *http.Request, courseID string) {
	// Get learner info from query parameters or body
	learnerID := r.URL.Query().Get("learner_id")
	learnerName := r.URL.Query().Get("learner_name")

	if learnerID == "" {
		learnerID = "default_learner"
	}
	if learnerName == "" {
		learnerName = "Default Learner"
	}

	// Check if registration already exists
	var existingID string
	existingQuery := "SELECT id FROM registrations WHERE course_id = ? AND learner_id = ?"
	err := h.db.QueryRow(existingQuery, courseID, learnerID).Scan(&existingID)

	var registrationID string
	if err == sql.ErrNoRows {
		// Create new registration
		registrationID = generateID()
		registration := models.Registration{
			ID:          registrationID,
			CourseID:    courseID,
			LearnerID:   learnerID,
			LearnerName: learnerName,
			Status:      "not_attempted",
			Entry:       "ab-initio",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		insertQuery := `
			INSERT INTO registrations (id, course_id, learner_id, learner_name, status, entry, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`

		_, err = h.db.Exec(insertQuery, registration.ID, registration.CourseID, registration.LearnerID,
			registration.LearnerName, registration.Status, registration.Entry, registration.CreatedAt, registration.UpdatedAt)
		if err != nil {
			log.Printf("Error creating registration: %v", err)
			h.sendError(w, "Failed to create registration", http.StatusInternalServerError)
			return
		}
	} else if err != nil {
		log.Printf("Error checking existing registration: %v", err)
		h.sendError(w, "Failed to check registration", http.StatusInternalServerError)
		return
	} else {
		registrationID = existingID
	}

	response := models.LaunchResponse{
		RegistrationID: registrationID,
		LaunchURL:      fmt.Sprintf("/courses/%s/content", courseID),
		JSAPI:          fmt.Sprintf("/api/registrations/%s/scorm-api", registrationID),
	}

	h.sendJSON(w, response)
}

// deleteCourse deletes a course
func (h *Handlers) deleteCourse(w http.ResponseWriter, r *http.Request, courseID string) {
	query := "DELETE FROM courses WHERE id = ?"
	result, err := h.db.Exec(query, courseID)
	if err != nil {
		log.Printf("Error deleting course: %v", err)
		h.sendError(w, "Failed to delete course", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		h.sendError(w, "Course not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// createCourseFromForm handles course creation from web form
func (h *Handlers) createCourseFromForm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	description := strings.TrimSpace(r.FormValue("description"))
	version := strings.TrimSpace(r.FormValue("version"))

	if title == "" {
		w.Header().Set("HX-Retarget", "#form-errors")
		w.Header().Set("HX-Reswap", "innerHTML")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`<div class="notification is-danger">Title is required</div>`))
		return
	}

	if version == "" {
		version = "1.0"
	}

	course := models.Course{
		ID:          generateID(),
		Title:       title,
		Description: description,
		Version:     version,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	query := `
		INSERT INTO courses (id, title, description, version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := h.db.Exec(query, course.ID, course.Title, course.Description,
		course.Version, course.CreatedAt, course.UpdatedAt)
	if err != nil {
		log.Printf("Error creating course: %v", err)
		w.Header().Set("HX-Retarget", "#form-errors")
		w.Header().Set("HX-Reswap", "innerHTML")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`<div class="notification is-danger">Failed to create course</div>`))
		return
	}

	// Redirect to the new course detail page
	slug := createSlug(course.Title)
	w.Header().Set("HX-Redirect", fmt.Sprintf("/courses/%s#%s", course.ID, slug))
	w.WriteHeader(http.StatusCreated)
}

// showCourseDetail displays a course detail page
func (h *Handlers) showCourseDetail(w http.ResponseWriter, r *http.Request, courseID string) {
	query := `
		SELECT id, title, description, version, package_path, manifest, created_at, updated_at 
		FROM courses WHERE id = ?
	`

	var course models.Course
	var packagePath, manifest, version sql.NullString
	err := h.db.QueryRow(query, courseID).Scan(&course.ID, &course.Title, &course.Description,
		&version, &packagePath, &manifest, &course.CreatedAt, &course.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
		} else {
			log.Printf("Error querying course: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Handle nullable fields
	if version.Valid {
		course.Version = version.String
	}
	if packagePath.Valid {
		course.PackagePath = &packagePath.String
	}
	if manifest.Valid {
		course.Manifest = &manifest.String
	}

	// Get registration count for this course
	var registrationCount int
	h.db.QueryRow("SELECT COUNT(*) FROM registrations WHERE course_id = ?", courseID).Scan(&registrationCount)

	data := map[string]interface{}{
		"Title":             fmt.Sprintf("%s - Course Details", course.Title),
		"PageType":          "course-detail",
		"Course":            course,
		"Slug":              createSlug(course.Title),
		"RegistrationCount": registrationCount,
		"IsEditing":         r.URL.Query().Get("edit") == "true",
	}

	if err := h.templates.ExecuteTemplate(w, "course-detail.html", data); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// updateCourseFromForm handles course updates from web form
func (h *Handlers) updateCourseFromForm(w http.ResponseWriter, r *http.Request, courseID string) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	description := strings.TrimSpace(r.FormValue("description"))
	version := strings.TrimSpace(r.FormValue("version"))

	if title == "" {
		w.Header().Set("HX-Retarget", "#form-errors")
		w.Header().Set("HX-Reswap", "innerHTML")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`<div class="notification is-danger">Title is required</div>`))
		return
	}

	if version == "" {
		version = "1.0"
	}

	query := `
		UPDATE courses 
		SET title = ?, description = ?, version = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := h.db.Exec(query, title, description, version, time.Now(), courseID)
	if err != nil {
		log.Printf("Error updating course: %v", err)
		w.Header().Set("HX-Retarget", "#form-errors")
		w.Header().Set("HX-Reswap", "innerHTML")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`<div class="notification is-danger">Failed to update course</div>`))
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.NotFound(w, r)
		return
	}

	// Redirect to the updated course detail page
	slug := createSlug(title)
	w.Header().Set("HX-Redirect", fmt.Sprintf("/courses/%s#%s", courseID, slug))
	w.WriteHeader(http.StatusOK)
}

// getRecentCourses retrieves the most recent courses with slug generation
func (h *Handlers) getRecentCourses(limit int) ([]map[string]interface{}, error) {
	query := `
		SELECT id, title, description, version, created_at, updated_at 
		FROM courses 
		ORDER BY created_at DESC 
		LIMIT ?
	`
	
	rows, err := h.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var courses []map[string]interface{}
	for rows.Next() {
		var course models.Course
		var version sql.NullString
		
		err := rows.Scan(&course.ID, &course.Title, &course.Description, 
			&version, &course.CreatedAt, &course.UpdatedAt)
		if err != nil {
			log.Printf("Error scanning course: %v", err)
			continue
		}
		
		// Handle nullable version field
		if version.Valid {
			course.Version = version.String
		} else {
			course.Version = "1.0"
		}
		
		// Create course data with slug
		courseData := map[string]interface{}{
			"ID":          course.ID,
			"Title":       course.Title,
			"Description": course.Description,
			"Version":     course.Version,
			"CreatedAt":   course.CreatedAt,
			"UpdatedAt":   course.UpdatedAt,
			"Slug":        createSlug(course.Title),
		}
		
		courses = append(courses, courseData)
	}
	
	return courses, nil
}
