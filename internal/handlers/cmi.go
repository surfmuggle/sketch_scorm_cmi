package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"scorm-cmi-app/internal/models"
	"strings"
	"time"
)

// HandleCMI handles CMI data operations
func (h *Handlers) HandleCMI(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		h.sendError(w, "Invalid CMI endpoint", http.StatusBadRequest)
		return
	}

	registrationID := pathParts[3]

	switch r.Method {
	case "GET":
		h.getCMIData(w, r, registrationID)
	case "POST":
		h.setCMIData(w, r, registrationID)
	default:
		h.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getCMIData retrieves CMI data for a registration
func (h *Handlers) getCMIData(w http.ResponseWriter, r *http.Request, registrationID string) {
	element := r.URL.Query().Get("element")

	if element != "" {
		// Get specific CMI element
		h.getSingleCMIElement(w, r, registrationID, element)
		return
	}

	// Get all CMI data
	query := `
		SELECT element, value, timestamp 
		FROM cmi_data 
		WHERE registration_id = ? 
		ORDER BY timestamp DESC
	`

	rows, err := h.db.Query(query, registrationID)
	if err != nil {
		log.Printf("Error querying CMI data: %v", err)
		h.sendError(w, "Failed to retrieve CMI data", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	cmiData := make(map[string]interface{})
	for rows.Next() {
		var element, value string
		var timestamp time.Time

		err := rows.Scan(&element, &value, &timestamp)
		if err != nil {
			log.Printf("Error scanning CMI data: %v", err)
			continue
		}

		cmiData[element] = map[string]interface{}{
			"value":     value,
			"timestamp": timestamp,
		}
	}

	// Also get registration data for core CMI elements
	regQuery := `
		SELECT status, score, min_score, max_score, session_time, total_time, 
		       location, suspend_data, exit, entry, learner_id, learner_name
		FROM registrations WHERE id = ?
	`

	var reg struct {
		Status      string
		Score       *float64
		MinScore    *float64
		MaxScore    *float64
		SessionTime string
		TotalTime   string
		Location    *string
		SuspendData *string
		Exit        *string
		Entry       string
		LearnerID   string
		LearnerName string
	}

	var location, suspendData, exit sql.NullString
	err = h.db.QueryRow(regQuery, registrationID).Scan(&reg.Status, &reg.Score, &reg.MinScore,
		&reg.MaxScore, &reg.SessionTime, &reg.TotalTime, &location, &suspendData,
		&exit, &reg.Entry, &reg.LearnerID, &reg.LearnerName)
	if err != nil {
		log.Printf("Error querying registration: %v", err)
		h.sendError(w, "Registration not found", http.StatusNotFound)
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

	// Add core CMI elements from registration
	cmiData["cmi.core.lesson_status"] = reg.Status
	if reg.Score != nil {
		cmiData["cmi.core.score.raw"] = *reg.Score
	}
	if reg.MinScore != nil {
		cmiData["cmi.core.score.min"] = *reg.MinScore
	}
	if reg.MaxScore != nil {
		cmiData["cmi.core.score.max"] = *reg.MaxScore
	}
	cmiData["cmi.core.session_time"] = reg.SessionTime
	cmiData["cmi.core.total_time"] = reg.TotalTime
	cmiData["cmi.core.student_id"] = reg.LearnerID
	cmiData["cmi.core.student_name"] = reg.LearnerName
	if reg.Exit != nil {
		cmiData["cmi.core.exit"] = *reg.Exit
	}
	cmiData["cmi.core.entry"] = reg.Entry
	if reg.SuspendData != nil {
		cmiData["cmi.suspend_data"] = *reg.SuspendData
	}
	if reg.Location != nil {
		cmiData["cmi.location"] = *reg.Location
	}

	h.sendJSON(w, cmiData)
}

// getSingleCMIElement retrieves a specific CMI element
func (h *Handlers) getSingleCMIElement(w http.ResponseWriter, r *http.Request, registrationID, element string) {
	// Map CMI elements to registration fields or cmi_data table
	coreElements := map[string]string{
		"cmi.core.lesson_status": "status",
		"cmi.core.score.raw":     "score",
		"cmi.core.score.min":     "min_score",
		"cmi.core.score.max":     "max_score",
		"cmi.core.session_time":  "session_time",
		"cmi.core.total_time":    "total_time",
		"cmi.core.student_id":    "learner_id",
		"cmi.core.student_name":  "learner_name",
		"cmi.core.exit":          "exit",
		"cmi.core.entry":         "entry",
		"cmi.suspend_data":       "suspend_data",
		"cmi.location":           "location",
	}

	if field, isCoreElement := coreElements[element]; isCoreElement {
		// Get from registrations table
		query := "SELECT " + field + " FROM registrations WHERE id = ?"
		var value interface{}
		err := h.db.QueryRow(query, registrationID).Scan(&value)
		if err != nil {
			log.Printf("Error querying core CMI element: %v", err)
			h.sendError(w, "Failed to retrieve CMI element", http.StatusInternalServerError)
			return
		}

		h.sendJSON(w, map[string]interface{}{
			"element": element,
			"value":   value,
		})
	} else {
		// Get from cmi_data table
		query := "SELECT value, timestamp FROM cmi_data WHERE registration_id = ? AND element = ? ORDER BY timestamp DESC LIMIT 1"
		var value string
		var timestamp time.Time

		err := h.db.QueryRow(query, registrationID, element).Scan(&value, &timestamp)
		if err != nil {
			log.Printf("Error querying CMI element: %v", err)
			h.sendError(w, "CMI element not found", http.StatusNotFound)
			return
		}

		h.sendJSON(w, map[string]interface{}{
			"element":   element,
			"value":     value,
			"timestamp": timestamp,
		})
	}
}

// setCMIData sets CMI data for a registration
func (h *Handlers) setCMIData(w http.ResponseWriter, r *http.Request, registrationID string) {
	var req models.CMIUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate registration exists
	var regExists bool
	err := h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM registrations WHERE id = ?)", registrationID).Scan(&regExists)
	if err != nil || !regExists {
		h.sendError(w, "Registration not found", http.StatusNotFound)
		return
	}

	// Map CMI elements to registration fields
	coreElements := map[string]string{
		"cmi.core.lesson_status": "status",
		"cmi.core.score.raw":     "score",
		"cmi.core.score.min":     "min_score",
		"cmi.core.score.max":     "max_score",
		"cmi.core.session_time":  "session_time",
		"cmi.core.total_time":    "total_time",
		"cmi.suspend_data":       "suspend_data",
		"cmi.location":           "location",
		"cmi.core.exit":          "exit",
	}

	if field, isCoreElement := coreElements[req.Element]; isCoreElement {
		// Update registration table
		query := "UPDATE registrations SET " + field + " = ?, updated_at = ? WHERE id = ?"
		_, err := h.db.Exec(query, req.Value, time.Now(), registrationID)
		if err != nil {
			log.Printf("Error updating core CMI element: %v", err)
			h.sendError(w, "Failed to update CMI element", http.StatusInternalServerError)
			return
		}
	} else {
		// Store in cmi_data table
		cmiData := models.CMIData{
			ID:             generateID(),
			RegistrationID: registrationID,
			Element:        req.Element,
			Value:          req.Value,
			Timestamp:      time.Now(),
		}

		query := "INSERT INTO cmi_data (id, registration_id, element, value, timestamp) VALUES (?, ?, ?, ?, ?)"
		_, err := h.db.Exec(query, cmiData.ID, cmiData.RegistrationID, cmiData.Element, cmiData.Value, cmiData.Timestamp)
		if err != nil {
			log.Printf("Error inserting CMI data: %v", err)
			h.sendError(w, "Failed to store CMI data", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	h.sendJSON(w, map[string]interface{}{
		"element": req.Element,
		"value":   req.Value,
		"status":  "success",
	})
}

// handleSCORMAPI provides SCORM API endpoints for content
func (h *Handlers) handleSCORMAPI(w http.ResponseWriter, r *http.Request, registrationID string) {
	// This endpoint provides the JavaScript API interface that SCORM content expects
	if r.Method == "GET" {
		// Return SCORM API JavaScript
		scormAPI := `
			// SCORM 2004 API Implementation
			var API_1484_11 = {
				Initialize: function(parameter) {
					console.log('SCORM API Initialize called');
					return "true";
				},
				
				Terminate: function(parameter) {
					console.log('SCORM API Terminate called');
					fetch('/api/registrations/` + registrationID + `/terminate', {
						method: 'POST',
						headers: {'Content-Type': 'application/json'}
					});
					return "true";
				},
				
				GetValue: function(element) {
					console.log('SCORM API GetValue called for:', element);
					// Synchronous call - in real implementation, you'd cache data
					return "";
				},
				
				SetValue: function(element, value) {
					console.log('SCORM API SetValue called:', element, value);
					fetch('/api/cmi/` + registrationID + `', {
						method: 'POST',
						headers: {'Content-Type': 'application/json'},
						body: JSON.stringify({element: element, value: value})
					});
					return "true";
				},
				
				Commit: function(parameter) {
					console.log('SCORM API Commit called');
					fetch('/api/registrations/` + registrationID + `/commit', {
						method: 'POST',
						headers: {'Content-Type': 'application/json'}
					});
					return "true";
				},
				
				GetLastError: function() {
					return "0";
				},
				
				GetErrorString: function(errorCode) {
					return "";
				},
				
				GetDiagnostic: function(errorCode) {
					return "";
				}
			};
			
			// Make API available globally
			window.API_1484_11 = API_1484_11;
		`

		w.Header().Set("Content-Type", "application/javascript")
		w.Write([]byte(scormAPI))
	} else {
		h.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
