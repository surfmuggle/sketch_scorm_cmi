package main

import (
	"log"
	"net/http"
	"os"
	"scorm-cmi-app/internal/database"
	"scorm-cmi-app/internal/handlers"
	"strings"
)

func main() {
	// Initialize database
	db, err := database.Initialize("./scorm.db")
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Create handlers with database dependency
	h := handlers.New(db)

	// Setup routes
	mux := http.NewServeMux()

	// Serve static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	// Web UI routes (more specific patterns first)
	mux.HandleFunc("/", h.HandleIndex)
	mux.HandleFunc("/courses/create", h.HandleCourseCreatePage)
	mux.HandleFunc("/courses/", h.HandleCourseDetailPage)
	mux.HandleFunc("/courses", h.HandleCoursesListPage)
	mux.HandleFunc("/registrations", h.HandleRegistrationsPage)

	// New SCORM routes (without HTMX)
	mux.HandleFunc("/scorm", h.HandleSCORMList)
	mux.HandleFunc("/scorm/upload", h.HandleSCORMUpload)
	mux.HandleFunc("/scorm/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/scorm/" {
			h.HandleSCORMList(w, r)
			return
		}

		// Handle specific SCORM file actions
		if len(path) > 7 { // "/scorm/" = 7 chars
			if strings.HasSuffix(path, "/delete") {
				h.HandleSCORMDelete(w, r)
			} else if strings.HasSuffix(path, "/update") {
				h.HandleSCORMUpdate(w, r)
			} else if strings.HasSuffix(path, "/launch") {
				h.HandleSCORMLaunch(w, r)
			} else {
				h.HandleSCORMDetail(w, r)
			}
		}
	})

	// API routes (more specific patterns first)
	mux.HandleFunc("/api/courses/", h.HandleCourse)
	mux.HandleFunc("/api/courses", h.HandleCourses)
	mux.HandleFunc("/api/packages/upload", h.HandlePackageUpload)
	mux.HandleFunc("/api/registrations", h.HandleRegistrations)
	mux.HandleFunc("/api/registrations/", h.HandleRegistration)
	mux.HandleFunc("/api/cmi/", h.HandleCMI)
	mux.HandleFunc("/api/reports/", h.HandleReports)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting SCORM CMI server on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
