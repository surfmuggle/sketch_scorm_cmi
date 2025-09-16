package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"scorm-cmi-app/internal/models"
	"strings"
	"time"
)

// SCORMStats represents statistics for SCORM files
type SCORMStats struct {
	TotalRegistrations      int `json:"total_registrations"`
	CompletedRegistrations  int `json:"completed_registrations"`
	InProgressRegistrations int `json:"in_progress_registrations"`
}

// HandleSCORMList displays the SCORM files list page (without HTMX)
func (h *Handlers) HandleSCORMList(w http.ResponseWriter, r *http.Request) {
	// Get all SCORM files (courses)
	courses, err := h.getAllCourses()
	if err != nil {
		log.Printf("Error fetching SCORM files: %v", err)
		courses = []models.Course{} // Empty slice on error
	}

	// Get statistics
	stats, err := h.getSCORMStats()
	if err != nil {
		log.Printf("Error fetching SCORM stats: %v", err)
		stats = SCORMStats{} // Empty stats on error
	}

	// Check for messages from redirects
	message := r.URL.Query().Get("message")
	messageType := r.URL.Query().Get("type")

	data := map[string]interface{}{
		"Title":       "SCORM Files",
		"SCORMFiles":  courses,
		"Stats":       stats,
		"Message":     message,
		"MessageType": messageType,
	}

	if err := h.templates.ExecuteTemplate(w, "scorm-list.html", data); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// HandleSCORMDetail displays a specific SCORM file detail page
func (h *Handlers) HandleSCORMDetail(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.NotFound(w, r)
		return
	}

	courseID := parts[2]
	if courseID == "" {
		http.NotFound(w, r)
		return
	}

	// Get course details
	course, err := h.getCourseByID(courseID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
		} else {
			log.Printf("Error fetching course: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Get registrations for this course (recent 5)
	registrations, err := h.getRecentRegistrations(courseID, 5)
	if err != nil {
		log.Printf("Error fetching registrations: %v", err)
		registrations = []models.Registration{} // Empty on error
	}

	// Get registration count and statistics
	registrationCount, err := h.getRegistrationCount(courseID)
	if err != nil {
		log.Printf("Error fetching registration count: %v", err)
		registrationCount = 0
	}

	completionRate, averageScore := h.getCourseStats(courseID)

	// Check for edit mode
	isEditing := r.URL.Query().Get("edit") == "true"

	// Check for messages
	message := r.URL.Query().Get("message")
	messageType := r.URL.Query().Get("type")

	data := map[string]interface{}{
		"Title":              fmt.Sprintf("%s - SCORM Details", course.Title),
		"Course":             course,
		"Registrations":      registrations,
		"RegistrationCount":  registrationCount,
		"TotalRegistrations": registrationCount, // For template compatibility
		"CompletionRate":     completionRate,
		"AverageScore":       averageScore,
		"IsEditing":          isEditing,
		"Message":            message,
		"MessageType":        messageType,
	}

	if err := h.templates.ExecuteTemplate(w, "scorm-detail.html", data); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// HandleSCORMUpload handles SCORM package upload
func (h *Handlers) HandleSCORMUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form
	err := r.ParseMultipartForm(50 << 20) // 50MB max
	if err != nil {
		http.Redirect(w, r, "/scorm?message=Failed to parse upload form&type=danger", http.StatusSeeOther)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	description := strings.TrimSpace(r.FormValue("description"))

	if title == "" {
		http.Redirect(w, r, "/scorm?message=Course title is required&type=danger", http.StatusSeeOther)
		return
	}

	// Get uploaded file
	file, header, err := r.FormFile("package")
	if err != nil {
		http.Redirect(w, r, "/scorm?message=Failed to get uploaded file&type=danger", http.StatusSeeOther)
		return
	}
	defer file.Close()

	// Create course record first
	course := models.Course{
		ID:          generateID(),
		Title:       title,
		Description: description,
		Version:     "1.0",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Process the SCORM package (this would normally include ZIP extraction and manifest parsing)
	packagePath := fmt.Sprintf("uploads/%s_%s", course.ID, header.Filename)
	course.PackagePath = &packagePath

	// Save course to database
	err = h.createCourseRecord(course)
	if err != nil {
		log.Printf("Error creating course record: %v", err)
		http.Redirect(w, r, "/scorm?message=Failed to create course record&type=danger", http.StatusSeeOther)
		return
	}

	// In a real implementation, you would:
	// 1. Save the uploaded file to the filesystem
	// 2. Extract the ZIP file
	// 3. Parse the imsmanifest.xml
	// 4. Create SCO records
	// 5. Validate the SCORM package structure

	// For now, we'll just simulate successful upload
	message := fmt.Sprintf("SCORM package '%s' uploaded successfully!", title)
	redirectURL := fmt.Sprintf("/scorm/%s?message=%s&type=success", course.ID, message)
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

// HandleSCORMUpdate handles SCORM file updates
func (h *Handlers) HandleSCORMUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.NotFound(w, r)
		return
	}

	courseID := parts[2]

	// Parse form
	err := r.ParseForm()
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/scorm/%s?message=Failed to parse form&type=danger", courseID), http.StatusSeeOther)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	description := strings.TrimSpace(r.FormValue("description"))
	version := strings.TrimSpace(r.FormValue("version"))

	if title == "" {
		http.Redirect(w, r, fmt.Sprintf("/scorm/%s?message=Course title is required&type=danger", courseID), http.StatusSeeOther)
		return
	}

	if version == "" {
		version = "1.0"
	}

	// Update course
	query := `
		UPDATE courses 
		SET title = ?, description = ?, version = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := h.db.Exec(query, title, description, version, time.Now(), courseID)
	if err != nil {
		log.Printf("Error updating course: %v", err)
		http.Redirect(w, r, fmt.Sprintf("/scorm/%s?message=Failed to update course&type=danger", courseID), http.StatusSeeOther)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.NotFound(w, r)
		return
	}

	message := "Course updated successfully!"
	http.Redirect(w, r, fmt.Sprintf("/scorm/%s?message=%s&type=success", courseID, message), http.StatusSeeOther)
}

// HandleSCORMDelete handles SCORM file deletion
func (h *Handlers) HandleSCORMDelete(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.NotFound(w, r)
		return
	}

	courseID := parts[2]

	// Delete course (cascade will handle related records)
	query := "DELETE FROM courses WHERE id = ?"
	result, err := h.db.Exec(query, courseID)
	if err != nil {
		log.Printf("Error deleting course: %v", err)
		http.Redirect(w, r, "/scorm?message=Failed to delete SCORM file&type=danger", http.StatusSeeOther)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.NotFound(w, r)
		return
	}

	// In a real implementation, you would also:
	// 1. Delete the uploaded package files from filesystem
	// 2. Clean up any extracted content

	message := "SCORM file deleted successfully!"
	http.Redirect(w, r, fmt.Sprintf("/scorm?message=%s&type=success", message), http.StatusSeeOther)
}

// HandleSCORMLaunch handles SCORM course launch
func (h *Handlers) HandleSCORMLaunch(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.NotFound(w, r)
		return
	}

	courseID := parts[2]

	// Get learner info from query parameters
	learnerID := r.URL.Query().Get("learner_id")
	learnerName := r.URL.Query().Get("learner_name")

	if learnerID == "" {
		learnerID = "default_learner"
	}
	if learnerName == "" {
		learnerName = "Default Learner"
	}

	// Check if registration exists
	var registrationID string
	existingQuery := "SELECT id FROM registrations WHERE course_id = ? AND learner_id = ?"
	err := h.db.QueryRow(existingQuery, courseID, learnerID).Scan(&registrationID)

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

		err = h.createRegistrationRecord(registration)
		if err != nil {
			log.Printf("Error creating registration: %v", err)
			http.Error(w, "Failed to create registration", http.StatusInternalServerError)
			return
		}
	} else if err != nil {
		log.Printf("Error checking existing registration: %v", err)
		http.Error(w, "Failed to check registration", http.StatusInternalServerError)
		return
	}

	// For now, we'll serve a simple SCORM content placeholder
	// In a real implementation, this would serve the actual SCORM content
	scormHTML := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>SCORM Content</title>
			<style>
				body { font-family: Arial, sans-serif; padding: 20px; }
				.scorm-container { max-width: 800px; margin: 0 auto; }
				.info-box { background: #f0f8ff; padding: 15px; border-radius: 5px; margin: 10px 0; }
				.button { background: #007cba; color: white; padding: 10px 20px; border: none; border-radius: 5px; cursor: pointer; }
				.button:hover { background: #005a87; }
			</style>
		</head>
		<body>
			<div class="scorm-container">
				<h1>SCORM Content Placeholder</h1>
				<div class="info-box">
					<p><strong>Registration ID:</strong> %s</p>
					<p><strong>Course ID:</strong> %s</p>
					<p><strong>Learner:</strong> %s (%s)</p>
				</div>
				<p>This is a placeholder for SCORM content. In a real implementation, this would load the actual SCORM package content.</p>
				<p>The SCORM API would be available at: <code>/api/registrations/%s/scorm-api</code></p>
				<br>
				<button class="button" onclick="completeCourse()">Mark as Completed</button>
				<button class="button" onclick="setScore()">Set Score</button>
			</div>
			
			<script>
				function completeCourse() {
					// In a real SCORM implementation, this would use the SCORM API
					alert('In a real SCORM package, this would set cmi.core.lesson_status to "completed"');
				}
				
				function setScore() {
					const score = prompt('Enter score (0-100):', '85');
					if (score) {
						alert('In a real SCORM package, this would set cmi.core.score.raw to "' + score + '"');
					}
				}
			</script>
		</body>
		</html>
	`, registrationID, courseID, learnerName, learnerID, registrationID)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(scormHTML))
}

// Helper functions

func (h *Handlers) getAllCourses() ([]models.Course, error) {
	query := `
		SELECT id, title, description, version, package_path, created_at, updated_at 
		FROM courses 
		ORDER BY created_at DESC
	`

	rows, err := h.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var courses []models.Course
	for rows.Next() {
		var course models.Course
		var packagePath, version sql.NullString
		err := rows.Scan(&course.ID, &course.Title, &course.Description,
			&version, &packagePath, &course.CreatedAt, &course.UpdatedAt)
		if err != nil {
			continue
		}

		if version.Valid {
			course.Version = version.String
		} else {
			course.Version = "1.0"
		}
		if packagePath.Valid {
			course.PackagePath = &packagePath.String
		}

		courses = append(courses, course)
	}

	return courses, nil
}

func (h *Handlers) getCourseByID(courseID string) (models.Course, error) {
	query := `
		SELECT id, title, description, version, package_path, manifest, created_at, updated_at 
		FROM courses WHERE id = ?
	`

	var course models.Course
	var packagePath, manifest, version sql.NullString
	err := h.db.QueryRow(query, courseID).Scan(&course.ID, &course.Title, &course.Description,
		&version, &packagePath, &manifest, &course.CreatedAt, &course.UpdatedAt)
	if err != nil {
		return course, err
	}

	if version.Valid {
		course.Version = version.String
	} else {
		course.Version = "1.0"
	}
	if packagePath.Valid {
		course.PackagePath = &packagePath.String
	}
	if manifest.Valid {
		course.Manifest = &manifest.String
	}

	return course, nil
}

func (h *Handlers) getSCORMStats() (SCORMStats, error) {
	var stats SCORMStats

	// Get total registrations
	err := h.db.QueryRow("SELECT COUNT(*) FROM registrations").Scan(&stats.TotalRegistrations)
	if err != nil {
		return stats, err
	}

	// Get completed registrations
	err = h.db.QueryRow("SELECT COUNT(*) FROM registrations WHERE status = 'completed'").Scan(&stats.CompletedRegistrations)
	if err != nil {
		return stats, err
	}

	// Get in-progress registrations
	err = h.db.QueryRow("SELECT COUNT(*) FROM registrations WHERE status = 'incomplete'").Scan(&stats.InProgressRegistrations)
	if err != nil {
		return stats, err
	}

	return stats, nil
}

func (h *Handlers) getRecentRegistrations(courseID string, limit int) ([]models.Registration, error) {
	query := `
		SELECT id, course_id, learner_id, learner_name, status, score, min_score, max_score,
		       session_time, total_time, location, suspend_data, exit, entry, created_at, updated_at, completed_at
		FROM registrations 
		WHERE course_id = ?
		ORDER BY updated_at DESC 
		LIMIT ?
	`

	rows, err := h.db.Query(query, courseID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var registrations []models.Registration
	for rows.Next() {
		var reg models.Registration
		var score, minScore, maxScore sql.NullFloat64
		var location, suspendData, exit sql.NullString
		var completedAt sql.NullTime

		err := rows.Scan(&reg.ID, &reg.CourseID, &reg.LearnerID, &reg.LearnerName,
			&reg.Status, &score, &minScore, &maxScore, &reg.SessionTime, &reg.TotalTime,
			&location, &suspendData, &exit, &reg.Entry, &reg.CreatedAt, &reg.UpdatedAt, &completedAt)
		if err != nil {
			continue
		}

		if score.Valid {
			reg.Score = &score.Float64
		}
		if minScore.Valid {
			reg.MinScore = &minScore.Float64
		}
		if maxScore.Valid {
			reg.MaxScore = &maxScore.Float64
		}
		if location.Valid {
			reg.Location = &location.String
		}
		if suspendData.Valid {
			reg.SuspendData = &suspendData.String
		}
		if exit.Valid {
			reg.Exit = &exit.String
		}
		if completedAt.Valid {
			reg.CompletedAt = &completedAt.Time
		}

		registrations = append(registrations, reg)
	}

	return registrations, nil
}

func (h *Handlers) getRegistrationCount(courseID string) (int, error) {
	var count int
	err := h.db.QueryRow("SELECT COUNT(*) FROM registrations WHERE course_id = ?", courseID).Scan(&count)
	return count, err
}

func (h *Handlers) getCourseStats(courseID string) (float64, *float64) {
	// Get completion rate
	var totalCount, completedCount int
	h.db.QueryRow("SELECT COUNT(*) FROM registrations WHERE course_id = ?", courseID).Scan(&totalCount)
	h.db.QueryRow("SELECT COUNT(*) FROM registrations WHERE course_id = ? AND status = 'completed'", courseID).Scan(&completedCount)

	var completionRate float64
	if totalCount > 0 {
		completionRate = (float64(completedCount) / float64(totalCount)) * 100
	}

	// Get average score
	var avgScore sql.NullFloat64
	h.db.QueryRow("SELECT AVG(score) FROM registrations WHERE course_id = ? AND score IS NOT NULL", courseID).Scan(&avgScore)

	var averageScore *float64
	if avgScore.Valid {
		averageScore = &avgScore.Float64
	}

	return completionRate, averageScore
}

func (h *Handlers) createCourseRecord(course models.Course) error {
	query := `
		INSERT INTO courses (id, title, description, version, package_path, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := h.db.Exec(query, course.ID, course.Title, course.Description,
		course.Version, course.PackagePath, course.CreatedAt, course.UpdatedAt)
	return err
}

func (h *Handlers) createRegistrationRecord(registration models.Registration) error {
	query := `
		INSERT INTO registrations (id, course_id, learner_id, learner_name, status, entry, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := h.db.Exec(query, registration.ID, registration.CourseID, registration.LearnerID,
		registration.LearnerName, registration.Status, registration.Entry, registration.CreatedAt, registration.UpdatedAt)
	return err
}
