package handlers

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"scorm-cmi-app/internal/models"
	"strings"
	"time"
)

// SCORMManifest represents the basic structure of a SCORM manifest
type SCORMManifest struct {
	XMLName       xml.Name      `xml:"manifest"`
	Identifier    string        `xml:"identifier,attr"`
	Version       string        `xml:"version,attr"`
	Metadata      Metadata      `xml:"metadata"`
	Organizations Organizations `xml:"organizations"`
	Resources     Resources     `xml:"resources"`
}

type Metadata struct {
	Schema        string `xml:"schema"`
	SchemaVersion string `xml:"schemaversion"`
}

type Organizations struct {
	Default      string         `xml:"default,attr"`
	Organization []Organization `xml:"organization"`
}

type Organization struct {
	Identifier string `xml:"identifier,attr"`
	Title      string `xml:"title"`
	Items      []Item `xml:"item"`
}

type Item struct {
	Identifier    string `xml:"identifier,attr"`
	IdentifierRef string `xml:"identifierref,attr"`
	Title         string `xml:"title"`
}

type Resources struct {
	Resource []Resource `xml:"resource"`
}

type Resource struct {
	Identifier string `xml:"identifier,attr"`
	Type       string `xml:"type,attr"`
	Href       string `xml:"href,attr"`
	Files      []File `xml:"file"`
}

type File struct {
	Href string `xml:"href,attr"`
}

// HandlePackageUpload handles SCORM package upload
func (h *Handlers) HandlePackageUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		h.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form
	err := r.ParseMultipartForm(32 << 20) // 32 MB limit
	if err != nil {
		h.sendError(w, "Failed to parse multipart form", http.StatusBadRequest)
		return
	}

	// Get uploaded file
	file, fileHeader, err := r.FormFile("package")
	if err != nil {
		h.sendError(w, "No package file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Get metadata from form
	title := r.FormValue("title")
	description := r.FormValue("description")

	if title == "" {
		title = strings.TrimSuffix(fileHeader.Filename, filepath.Ext(fileHeader.Filename))
	}

	// Validate file is a ZIP
	if !strings.HasSuffix(strings.ToLower(fileHeader.Filename), ".zip") {
		h.sendError(w, "Package must be a ZIP file", http.StatusBadRequest)
		return
	}

	// Create course ID and directories
	courseID := generateID()
	packageDir := filepath.Join("uploads", courseID)
	if err := os.MkdirAll(packageDir, 0755); err != nil {
		log.Printf("Error creating package directory: %v", err)
		h.sendError(w, "Failed to create package directory", http.StatusInternalServerError)
		return
	}

	// Save uploaded file
	packagePath := filepath.Join(packageDir, "package.zip")
	outFile, err := os.Create(packagePath)
	if err != nil {
		log.Printf("Error creating package file: %v", err)
		h.sendError(w, "Failed to save package file", http.StatusInternalServerError)
		return
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, file)
	if err != nil {
		log.Printf("Error copying package file: %v", err)
		h.sendError(w, "Failed to save package file", http.StatusInternalServerError)
		return
	}

	// Extract and validate package
	extractDir := filepath.Join(packageDir, "content")
	manifest, err := h.extractAndValidatePackage(packagePath, extractDir)
	if err != nil {
		log.Printf("Error extracting/validating package: %v", err)
		h.sendError(w, fmt.Sprintf("Package validation failed: %v", err), http.StatusBadRequest)
		return
	}

	// Store course in database
	manifestXML, _ := xml.Marshal(manifest)
	manifestString := string(manifestXML)
	course := models.Course{
		ID:          courseID,
		Title:       title,
		Description: description,
		Version:     manifest.Version,
		PackagePath: &packagePath,
		Manifest:    &manifestString,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	query := `
		INSERT INTO courses (id, title, description, version, package_path, manifest, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = h.db.Exec(query, course.ID, course.Title, course.Description,
		course.Version, course.PackagePath, course.Manifest, course.CreatedAt, course.UpdatedAt)
	if err != nil {
		log.Printf("Error storing course in database: %v", err)
		h.sendError(w, "Failed to store course", http.StatusInternalServerError)
		return
	}

	// Store SCOs
	err = h.storeSCOs(courseID, manifest)
	if err != nil {
		log.Printf("Error storing SCOs: %v", err)
		// Continue anyway, SCO extraction is not critical for basic functionality
	}

	w.WriteHeader(http.StatusCreated)
	h.sendJSON(w, map[string]interface{}{
		"course":  course,
		"message": "Package uploaded and processed successfully",
	})
}

// extractAndValidatePackage extracts a SCORM package and validates its manifest
func (h *Handlers) extractAndValidatePackage(packagePath, extractDir string) (*SCORMManifest, error) {
	// Open ZIP file
	reader, err := zip.OpenReader(packagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open package: %w", err)
	}
	defer reader.Close()

	// Create extraction directory
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create extraction directory: %w", err)
	}

	var manifestFile *zip.File

	// Extract files
	for _, file := range reader.File {
		path := filepath.Join(extractDir, file.Name)

		// Ensure the file path is safe
		if !strings.HasPrefix(path, filepath.Clean(extractDir)+string(os.PathSeparator)) {
			return nil, fmt.Errorf("invalid file path: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			os.MkdirAll(path, file.FileInfo().Mode())
			continue
		}

		// Create directory for file
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}

		// Extract file
		fileReader, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open file in archive: %w", err)
		}
		defer fileReader.Close()

		outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.FileInfo().Mode())
		if err != nil {
			return nil, fmt.Errorf("failed to create extracted file: %w", err)
		}
		defer outFile.Close()

		_, err = io.Copy(outFile, fileReader)
		if err != nil {
			return nil, fmt.Errorf("failed to extract file: %w", err)
		}

		// Check if this is the manifest file
		if strings.ToLower(file.Name) == "imsmanifest.xml" {
			manifestFile = file
		}
	}

	if manifestFile == nil {
		return nil, fmt.Errorf("imsmanifest.xml not found in package")
	}

	// Parse manifest
	manifestReader, err := manifestFile.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open manifest: %w", err)
	}
	defer manifestReader.Close()

	manifestData, err := io.ReadAll(manifestReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest SCORMManifest
	err = xml.Unmarshal(manifestData, &manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to parse manifest XML: %w", err)
	}

	// Basic validation
	if manifest.Identifier == "" {
		return nil, fmt.Errorf("manifest missing identifier")
	}

	if len(manifest.Resources.Resource) == 0 {
		return nil, fmt.Errorf("manifest contains no resources")
	}

	return &manifest, nil
}

// storeSCOs extracts and stores SCO information from manifest
func (h *Handlers) storeSCOs(courseID string, manifest *SCORMManifest) error {
	// Create a map of resource identifiers to resources
	resourceMap := make(map[string]Resource)
	for _, resource := range manifest.Resources.Resource {
		resourceMap[resource.Identifier] = resource
	}

	// Process organizations to create SCOs
	for _, org := range manifest.Organizations.Organization {
		for _, item := range org.Items {
			if item.IdentifierRef != "" {
				resource, exists := resourceMap[item.IdentifierRef]
				if !exists {
					continue
				}

				sco := models.SCO{
					ID:        generateID(),
					CourseID:  courseID,
					Title:     item.Title,
					Href:      resource.Href,
					LaunchURL: fmt.Sprintf("/courses/%s/content/%s", courseID, resource.Href),
				}

				query := "INSERT INTO scos (id, course_id, title, href, launch_url) VALUES (?, ?, ?, ?, ?)"
				_, err := h.db.Exec(query, sco.ID, sco.CourseID, sco.Title, sco.Href, sco.LaunchURL)
				if err != nil {
					log.Printf("Error storing SCO: %v", err)
					// Continue with other SCOs
				}
			}
		}
	}

	return nil
}
