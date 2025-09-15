package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
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

// HandleIndex serves the main page
func (h *Handlers) HandleIndex(w http.ResponseWriter, r *http.Request) {
	if err := h.templates.ExecuteTemplate(w, "index.html", nil); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// HandleCoursesPage serves the courses management page
func (h *Handlers) HandleCoursesPage(w http.ResponseWriter, r *http.Request) {
	if err := h.templates.ExecuteTemplate(w, "courses.html", nil); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
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
