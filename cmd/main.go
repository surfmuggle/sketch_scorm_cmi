package main

import (
	"log"
	"net/http"
	"os"
	"scorm-cmi-app/internal/database"
	"scorm-cmi-app/internal/handlers"
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

	// Web UI routes
	mux.HandleFunc("/", h.HandleIndex)
	mux.HandleFunc("/courses", h.HandleCoursesPage)
	mux.HandleFunc("/registrations", h.HandleRegistrationsPage)

	// API routes
	mux.HandleFunc("/api/courses", h.HandleCourses)
	mux.HandleFunc("/api/courses/", h.HandleCourse)
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
