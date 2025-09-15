package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"scorm-cmi-app/internal/models"
	"strings"
	"time"
)

// HandleRegistrations handles registration-related API requests
func (h *Handlers) HandleRegistrations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.getRegistrations(w, r)
	case "POST":
		h.createRegistration(w, r)
	default:
		h.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleRegistration handles individual registration operations
func (h *Handlers) HandleRegistration(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		h.sendError(w, "Invalid registration ID", http.StatusBadRequest)
		return
	}

	registrationID := pathParts[3]

	if len(pathParts) > 4 {
		switch pathParts[4] {
		case "commit":
			h.commitRegistration(w, r, registrationID)
		case "terminate":
			h.terminateRegistration(w, r, registrationID)
		case "scorm-api":
			h.handleSCORMAPI(w, r, registrationID)
		default:
			h.sendError(w, "Invalid endpoint", http.StatusBadRequest)
		}
		return
	}

	switch r.Method {
	case "GET":
		h.getRegistration(w, r, registrationID)
	case "PUT":
		h.updateRegistration(w, r, registrationID)
	case "DELETE":
		h.deleteRegistration(w, r, registrationID)
	default:
		h.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getRegistrations retrieves all registrations
func (h *Handlers) getRegistrations(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT r.id, r.course_id, r.learner_id, r.learner_name, r.status, 
		       r.score, r.min_score, r.max_score, r.session_time, r.total_time,
		       r.location, r.suspend_data, r.exit, r.entry, r.created_at, 
		       r.updated_at, r.completed_at, c.title as course_title
		FROM registrations r
		JOIN courses c ON r.course_id = c.id
		ORDER BY r.created_at DESC
	`

	// Filter by learner_id if provided
	learnerID := r.URL.Query().Get("learner_id")
	if learnerID != "" {
		query = `
			SELECT r.id, r.course_id, r.learner_id, r.learner_name, r.status, 
			       r.score, r.min_score, r.max_score, r.session_time, r.total_time,
			       r.location, r.suspend_data, r.exit, r.entry, r.created_at, 
			       r.updated_at, r.completed_at, c.title as course_title
			FROM registrations r
			JOIN courses c ON r.course_id = c.id
			WHERE r.learner_id = ?
			ORDER BY r.created_at DESC
		`
	}

	var rows *sql.Rows
	var err error

	if learnerID != "" {
		rows, err = h.db.Query(query, learnerID)
	} else {
		rows, err = h.db.Query(query)
	}

	if err != nil {
		log.Printf("Error querying registrations: %v", err)
		h.sendError(w, "Failed to retrieve registrations", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var registrations []map[string]interface{}
	for rows.Next() {
		var reg models.Registration
		var courseTitle string
		var location, suspendData, exit sql.NullString
		var completedAt sql.NullTime

		err := rows.Scan(&reg.ID, &reg.CourseID, &reg.LearnerID, &reg.LearnerName,
			&reg.Status, &reg.Score, &reg.MinScore, &reg.MaxScore, &reg.SessionTime,
			&reg.TotalTime, &location, &suspendData, &exit, &reg.Entry,
			&reg.CreatedAt, &reg.UpdatedAt, &completedAt, &courseTitle)
		if err != nil {
			log.Printf("Error scanning registration: %v", err)
			continue
		}

		// Handle nullable fields
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

		registrationData := map[string]interface{}{
			"registration": reg,
			"course_title": courseTitle,
		}

		registrations = append(registrations, registrationData)
	}

	// Ensure we always return a valid JSON array, even if empty
	if registrations == nil {
		registrations = []map[string]interface{}{}
	}
	h.sendJSON(w, registrations)
}

// createRegistration creates a new registration
func (h *Handlers) createRegistration(w http.ResponseWriter, r *http.Request) {
	var req models.RegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Check if course exists
	var courseExists bool
	err := h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM courses WHERE id = ?)", req.CourseID).Scan(&courseExists)
	if err != nil || !courseExists {
		h.sendError(w, "Course not found", http.StatusNotFound)
		return
	}

	registration := models.Registration{
		ID:          generateID(),
		CourseID:    req.CourseID,
		LearnerID:   req.LearnerID,
		LearnerName: req.LearnerName,
		Status:      "not_attempted",
		Entry:       "ab-initio",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	query := `
		INSERT INTO registrations (id, course_id, learner_id, learner_name, status, entry, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = h.db.Exec(query, registration.ID, registration.CourseID, registration.LearnerID,
		registration.LearnerName, registration.Status, registration.Entry, registration.CreatedAt, registration.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			h.sendError(w, "Registration already exists for this learner and course", http.StatusConflict)
		} else {
			log.Printf("Error creating registration: %v", err)
			h.sendError(w, "Failed to create registration", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
	h.sendJSON(w, registration)
}

// getRegistration retrieves a specific registration
func (h *Handlers) getRegistration(w http.ResponseWriter, r *http.Request, registrationID string) {
	query := `
		SELECT id, course_id, learner_id, learner_name, status, score, min_score, max_score,
		       session_time, total_time, location, suspend_data, exit, entry, 
		       created_at, updated_at, completed_at
		FROM registrations WHERE id = ?
	`

	var reg models.Registration
	var location, suspendData, exit sql.NullString
	var completedAt sql.NullTime

	err := h.db.QueryRow(query, registrationID).Scan(&reg.ID, &reg.CourseID, &reg.LearnerID,
		&reg.LearnerName, &reg.Status, &reg.Score, &reg.MinScore, &reg.MaxScore,
		&reg.SessionTime, &reg.TotalTime, &location, &suspendData, &exit,
		&reg.Entry, &reg.CreatedAt, &reg.UpdatedAt, &completedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			h.sendError(w, "Registration not found", http.StatusNotFound)
		} else {
			log.Printf("Error querying registration: %v", err)
			h.sendError(w, "Failed to retrieve registration", http.StatusInternalServerError)
		}
		return
	}

	// Handle nullable fields
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

	h.sendJSON(w, reg)
}

// updateRegistration updates a registration
func (h *Handlers) updateRegistration(w http.ResponseWriter, r *http.Request, registrationID string) {
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		h.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Build dynamic update query
	setClauses := []string{}
	args := []interface{}{}

	allowedFields := map[string]bool{
		"status": true, "score": true, "min_score": true, "max_score": true,
		"session_time": true, "total_time": true, "location": true,
		"suspend_data": true, "exit": true, "entry": true,
	}

	for field, value := range updates {
		if allowedFields[field] {
			setClauses = append(setClauses, field+" = ?")
			args = append(args, value)
		}
	}

	if len(setClauses) == 0 {
		h.sendError(w, "No valid fields to update", http.StatusBadRequest)
		return
	}

	setClauses = append(setClauses, "updated_at = ?")
	args = append(args, time.Now())
	args = append(args, registrationID)

	query := fmt.Sprintf("UPDATE registrations SET %s WHERE id = ?", strings.Join(setClauses, ", "))

	result, err := h.db.Exec(query, args...)
	if err != nil {
		log.Printf("Error updating registration: %v", err)
		h.sendError(w, "Failed to update registration", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		h.sendError(w, "Registration not found", http.StatusNotFound)
		return
	}

	// Return updated registration
	h.getRegistration(w, r, registrationID)
}

// deleteRegistration deletes a registration
func (h *Handlers) deleteRegistration(w http.ResponseWriter, r *http.Request, registrationID string) {
	query := "DELETE FROM registrations WHERE id = ?"
	result, err := h.db.Exec(query, registrationID)
	if err != nil {
		log.Printf("Error deleting registration: %v", err)
		h.sendError(w, "Failed to delete registration", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		h.sendError(w, "Registration not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// commitRegistration commits/saves the current state
func (h *Handlers) commitRegistration(w http.ResponseWriter, r *http.Request, registrationID string) {
	if r.Method != "POST" {
		h.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Update the updated_at timestamp to indicate last commit
	query := "UPDATE registrations SET updated_at = ? WHERE id = ?"
	_, err := h.db.Exec(query, time.Now(), registrationID)
	if err != nil {
		log.Printf("Error committing registration: %v", err)
		h.sendError(w, "Failed to commit registration", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	h.sendJSON(w, map[string]string{"status": "committed"})
}

// terminateRegistration terminates/finishes a session
func (h *Handlers) terminateRegistration(w http.ResponseWriter, r *http.Request, registrationID string) {
	if r.Method != "POST" {
		h.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Mark as completed if status indicates success
	var currentStatus string
	err := h.db.QueryRow("SELECT status FROM registrations WHERE id = ?", registrationID).Scan(&currentStatus)
	if err != nil {
		log.Printf("Error querying registration status: %v", err)
		h.sendError(w, "Registration not found", http.StatusNotFound)
		return
	}

	updates := map[string]interface{}{
		"updated_at": time.Now(),
		"exit":       "normal",
	}

	if currentStatus == "completed" || currentStatus == "passed" {
		updates["completed_at"] = time.Now()
	}

	// Build update query
	setClauses := []string{}
	args := []interface{}{}

	for field, value := range updates {
		setClauses = append(setClauses, field+" = ?")
		args = append(args, value)
	}
	args = append(args, registrationID)

	query := fmt.Sprintf("UPDATE registrations SET %s WHERE id = ?", strings.Join(setClauses, ", "))
	_, err = h.db.Exec(query, args...)
	if err != nil {
		log.Printf("Error terminating registration: %v", err)
		h.sendError(w, "Failed to terminate registration", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	h.sendJSON(w, map[string]string{"status": "terminated"})
}
