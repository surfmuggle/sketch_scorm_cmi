package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// HandleReports handles report generation requests
func (h *Handlers) HandleReports(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		h.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		h.sendError(w, "Invalid report endpoint", http.StatusBadRequest)
		return
	}

	reportType := pathParts[3]

	switch reportType {
	case "summary":
		h.getSummaryReport(w, r)
	case "course":
		if len(pathParts) < 5 {
			h.sendError(w, "Course ID required", http.StatusBadRequest)
			return
		}
		h.getCourseReport(w, r, pathParts[4])
	case "learner":
		if len(pathParts) < 5 {
			h.sendError(w, "Learner ID required", http.StatusBadRequest)
			return
		}
		h.getLearnerReport(w, r, pathParts[4])
	case "completion":
		h.getCompletionReport(w, r)
	default:
		h.sendError(w, "Invalid report type", http.StatusBadRequest)
	}
}

// getSummaryReport provides overall system statistics
func (h *Handlers) getSummaryReport(w http.ResponseWriter, r *http.Request) {
	type Summary struct {
		TotalCourses           int      `json:"total_courses"`
		TotalRegistrations     int      `json:"total_registrations"`
		CompletedRegistrations int      `json:"completed_registrations"`
		CompletionRate         float64  `json:"completion_rate"`
		AvgScore               *float64 `json:"avg_score,omitempty"`
		ActiveRegistrations    int      `json:"active_registrations"`
	}

	var summary Summary

	// Get total courses
	err := h.db.QueryRow("SELECT COUNT(*) FROM courses").Scan(&summary.TotalCourses)
	if err != nil {
		log.Printf("Error querying course count: %v", err)
		h.sendError(w, "Failed to generate summary", http.StatusInternalServerError)
		return
	}

	// Get total registrations
	err = h.db.QueryRow("SELECT COUNT(*) FROM registrations").Scan(&summary.TotalRegistrations)
	if err != nil {
		log.Printf("Error querying registration count: %v", err)
		h.sendError(w, "Failed to generate summary", http.StatusInternalServerError)
		return
	}

	// Get completed registrations
	err = h.db.QueryRow("SELECT COUNT(*) FROM registrations WHERE status IN ('completed', 'passed')").Scan(&summary.CompletedRegistrations)
	if err != nil {
		log.Printf("Error querying completed registrations: %v", err)
		h.sendError(w, "Failed to generate summary", http.StatusInternalServerError)
		return
	}

	// Calculate completion rate
	if summary.TotalRegistrations > 0 {
		summary.CompletionRate = float64(summary.CompletedRegistrations) / float64(summary.TotalRegistrations) * 100
	}

	// Get average score
	var avgScore sql.NullFloat64
	err = h.db.QueryRow("SELECT AVG(score) FROM registrations WHERE score IS NOT NULL").Scan(&avgScore)
	if err != nil {
		log.Printf("Error querying average score: %v", err)
		// Continue without average score
	} else if avgScore.Valid {
		summary.AvgScore = &avgScore.Float64
	}

	// Get active registrations (not completed, failed, or passed)
	err = h.db.QueryRow("SELECT COUNT(*) FROM registrations WHERE status NOT IN ('completed', 'passed', 'failed')").Scan(&summary.ActiveRegistrations)
	if err != nil {
		log.Printf("Error querying active registrations: %v", err)
		// Continue without active count
	}

	h.sendJSON(w, summary)
}

// getCourseReport provides detailed report for a specific course
func (h *Handlers) getCourseReport(w http.ResponseWriter, r *http.Request, courseID string) {
	type CourseReport struct {
		Course         interface{}   `json:"course"`
		Registrations  int           `json:"total_registrations"`
		Completed      int           `json:"completed_registrations"`
		InProgress     int           `json:"in_progress_registrations"`
		NotStarted     int           `json:"not_started_registrations"`
		Failed         int           `json:"failed_registrations"`
		CompletionRate float64       `json:"completion_rate"`
		AvgScore       *float64      `json:"avg_score,omitempty"`
		Learners       []interface{} `json:"learners"`
	}

	// Get course details
	courseQuery := "SELECT id, title, description, version, created_at, updated_at FROM courses WHERE id = ?"
	var course struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Version     string `json:"version"`
		CreatedAt   string `json:"created_at"`
		UpdatedAt   string `json:"updated_at"`
	}

	err := h.db.QueryRow(courseQuery, courseID).Scan(&course.ID, &course.Title, &course.Description,
		&course.Version, &course.CreatedAt, &course.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			h.sendError(w, "Course not found", http.StatusNotFound)
		} else {
			log.Printf("Error querying course: %v", err)
			h.sendError(w, "Failed to retrieve course", http.StatusInternalServerError)
		}
		return
	}

	var report CourseReport
	report.Course = course

	// Get registration statistics
	statsQuery := `
		SELECT 
			COUNT(*) as total,
			SUM(CASE WHEN status IN ('completed', 'passed') THEN 1 ELSE 0 END) as completed,
			SUM(CASE WHEN status = 'incomplete' THEN 1 ELSE 0 END) as in_progress,
			SUM(CASE WHEN status = 'not_attempted' THEN 1 ELSE 0 END) as not_started,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed,
			AVG(score) as avg_score
		FROM registrations WHERE course_id = ?
	`

	var avgScore sql.NullFloat64
	err = h.db.QueryRow(statsQuery, courseID).Scan(&report.Registrations, &report.Completed,
		&report.InProgress, &report.NotStarted, &report.Failed, &avgScore)
	if err != nil {
		log.Printf("Error querying registration stats: %v", err)
		h.sendError(w, "Failed to generate course report", http.StatusInternalServerError)
		return
	}

	if avgScore.Valid {
		report.AvgScore = &avgScore.Float64
	}

	// Calculate completion rate
	if report.Registrations > 0 {
		report.CompletionRate = float64(report.Completed) / float64(report.Registrations) * 100
	}

	// Get learner details
	learnersQuery := `
		SELECT learner_id, learner_name, status, score, total_time, updated_at, completed_at
		FROM registrations 
		WHERE course_id = ? 
		ORDER BY updated_at DESC
	`

	rows, err := h.db.Query(learnersQuery, courseID)
	if err != nil {
		log.Printf("Error querying learners: %v", err)
		// Continue without learner details
	} else {
		defer rows.Close()

		for rows.Next() {
			var learner struct {
				LearnerID   string   `json:"learner_id"`
				LearnerName string   `json:"learner_name"`
				Status      string   `json:"status"`
				Score       *float64 `json:"score,omitempty"`
				TotalTime   string   `json:"total_time"`
				UpdatedAt   string   `json:"updated_at"`
				CompletedAt *string  `json:"completed_at,omitempty"`
			}

			err := rows.Scan(&learner.LearnerID, &learner.LearnerName, &learner.Status,
				&learner.Score, &learner.TotalTime, &learner.UpdatedAt, &learner.CompletedAt)
			if err != nil {
				log.Printf("Error scanning learner: %v", err)
				continue
			}

			report.Learners = append(report.Learners, learner)
		}
	}

	h.sendJSON(w, report)
}

// getLearnerReport provides detailed report for a specific learner
func (h *Handlers) getLearnerReport(w http.ResponseWriter, r *http.Request, learnerID string) {
	type LearnerReport struct {
		LearnerID     string        `json:"learner_id"`
		LearnerName   string        `json:"learner_name"`
		Registrations []interface{} `json:"registrations"`
		TotalCourses  int           `json:"total_courses"`
		Completed     int           `json:"completed_courses"`
		InProgress    int           `json:"in_progress_courses"`
		AvgScore      *float64      `json:"avg_score,omitempty"`
	}

	// Get learner registrations
	query := `
		SELECT r.id, r.course_id, c.title as course_title, r.learner_name, r.status, 
		       r.score, r.total_time, r.created_at, r.updated_at, r.completed_at
		FROM registrations r
		JOIN courses c ON r.course_id = c.id
		WHERE r.learner_id = ?
		ORDER BY r.updated_at DESC
	`

	rows, err := h.db.Query(query, learnerID)
	if err != nil {
		log.Printf("Error querying learner registrations: %v", err)
		h.sendError(w, "Failed to generate learner report", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var report LearnerReport
	var totalScore float64
	var scoreCount int

	for rows.Next() {
		var reg struct {
			ID          string   `json:"id"`
			CourseID    string   `json:"course_id"`
			CourseTitle string   `json:"course_title"`
			LearnerName string   `json:"learner_name"`
			Status      string   `json:"status"`
			Score       *float64 `json:"score,omitempty"`
			TotalTime   string   `json:"total_time"`
			CreatedAt   string   `json:"created_at"`
			UpdatedAt   string   `json:"updated_at"`
			CompletedAt *string  `json:"completed_at,omitempty"`
		}

		err := rows.Scan(&reg.ID, &reg.CourseID, &reg.CourseTitle, &reg.LearnerName,
			&reg.Status, &reg.Score, &reg.TotalTime, &reg.CreatedAt, &reg.UpdatedAt, &reg.CompletedAt)
		if err != nil {
			log.Printf("Error scanning registration: %v", err)
			continue
		}

		// Set learner info from first registration
		if report.LearnerID == "" {
			report.LearnerID = learnerID
			report.LearnerName = reg.LearnerName
		}

		report.Registrations = append(report.Registrations, reg)
		report.TotalCourses++

		switch reg.Status {
		case "completed", "passed":
			report.Completed++
		case "incomplete":
			report.InProgress++
		}

		if reg.Score != nil {
			totalScore += *reg.Score
			scoreCount++
		}
	}

	if report.TotalCourses == 0 {
		h.sendError(w, "Learner not found", http.StatusNotFound)
		return
	}

	// Calculate average score
	if scoreCount > 0 {
		avgScore := totalScore / float64(scoreCount)
		report.AvgScore = &avgScore
	}

	h.sendJSON(w, report)
}

// getCompletionReport provides completion statistics with optional filtering
func (h *Handlers) getCompletionReport(w http.ResponseWriter, r *http.Request) {
	type CompletionData struct {
		CourseID    string  `json:"course_id"`
		CourseTitle string  `json:"course_title"`
		Total       int     `json:"total_registrations"`
		Completed   int     `json:"completed_registrations"`
		Rate        float64 `json:"completion_rate"`
	}

	// Parse query parameters
	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	query := `
		SELECT 
			c.id as course_id,
			c.title as course_title,
			COUNT(r.id) as total_registrations,
			SUM(CASE WHEN r.status IN ('completed', 'passed') THEN 1 ELSE 0 END) as completed_registrations
		FROM courses c
		LEFT JOIN registrations r ON c.id = r.course_id
		GROUP BY c.id, c.title
		ORDER BY total_registrations DESC
		LIMIT ?
	`

	rows, err := h.db.Query(query, limit)
	if err != nil {
		log.Printf("Error querying completion data: %v", err)
		h.sendError(w, "Failed to generate completion report", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var completionData []CompletionData
	for rows.Next() {
		var data CompletionData
		err := rows.Scan(&data.CourseID, &data.CourseTitle, &data.Total, &data.Completed)
		if err != nil {
			log.Printf("Error scanning completion data: %v", err)
			continue
		}

		// Calculate completion rate
		if data.Total > 0 {
			data.Rate = float64(data.Completed) / float64(data.Total) * 100
		}

		completionData = append(completionData, data)
	}

	h.sendJSON(w, completionData)
}
