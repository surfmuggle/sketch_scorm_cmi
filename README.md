
# Disclaimer 
This is a work in progress unfinished and 99.99% [vibe coded](https://www.urbandictionary.com/define.php?term=Vibe+Coding) app. 

Quote:
> In other words, it's where random non-technical monkeys with little to no programming language, dish out sloppy games and other software through blindly copy-pasting AI generated code from LLMs like ChatGPT, Claude, and Cursor to make fast and easy money.

Do not use it or expect anything from it. 
You have been warned. 

# SCORM CMI Application
A comprehensive Go-based web application for managing SCORM 2004 courses and Computer Managed Instruction (CMI) data. This application provides a REST API and HTMX-powered web interface for uploading SCORM packages, managing learner registrations, tracking progress, and generating learning analytics reports.

## Features - Vaporware

> The bot lied here - none of this is working see [Vaporware](https://www.urbandictionary.com/define.php?term=Vaporware) 

### 📚 Course Management 🚫
- Upload and validate SCORM 2004 packages (ZIP files) [⏳ open #4](https://github.com/surfmuggle/sketch_scorm_cmi/issues/4)
- Parse and store SCORM manifests 🔴
- Extract Shareable Content Objects (SCOs) 🔴
- Course metadata management 🔴

### 👥 Learner Registration 🚫
- Create learner enrollments in courses
- Launch courses with automatic registration
- Track multiple registrations per learner
- Manage registration states and lifecycle

### 📊 CMI Data Tracking 🚫
Supports all major SCORM 2004 CMI data elements:
- **Core elements**: lesson_status, score (raw/min/max), session_time, total_time
- **Navigation**: location (bookmarking), entry/exit conditions
- **State management**: suspend_data for session persistence
- **Learner info**: learner_id, learner_name
- **Custom interactions**: detailed interaction tracking

### 📈 Learning Analytics 🚫
- System-wide summary statistics
- Per-course completion reports
- Individual learner progress tracking
- Completion rate analysis
- Score analytics and averages

### 🔧 Technical Features 🚫
- **REST API**: Comprehensive OpenAPI 3.0 specification
- **Web Interface**: Modern HTMX-powered frontend
- **Database**: SQLite with proper schema and indexing
- **SCORM API**: JavaScript API for content integration
- **Package Processing**: ZIP extraction and manifest validation

## Architecture

```
├── cmd/
│   └── main.go                 # Application entry point
├── internal/
│   ├── database/
│   │   └── database.go         # SQLite setup and schema
│   ├── handlers/
│   │   ├── handlers.go         # Core HTTP handlers
│   │   ├── registrations.go    # Registration management
│   │   ├── cmi.go             # CMI data handling
│   │   ├── upload.go          # SCORM package upload
│   │   └── reports.go         # Analytics and reporting
│   └── models/
│       └── models.go          # Data structures
├── web/
│   ├── templates/             # HTML templates
│   └── static/               # CSS, JavaScript assets
├── api/
│   └── openapi.yaml          # OpenAPI specification
└── uploads/                  # SCORM package storage
```

## Quick Start

### Prerequisites
- Go 1.21 or later
- No additional dependencies (uses only standard library + go-sqlite3)

### Installation

```bash
# Clone and build
go mod tidy
go build -o scorm-cmi-app cmd/main.go

# Run the application
./scorm-cmi-app
```

The application will start on port 8080 by default.

### Environment Variables
- `PORT`: Server port (default: 8080)

## Usage

### Web Interface
Open your browser to `http://localhost:8080` to access the web interface:

- **Dashboard**: System overview with statistics
- **Courses**: Upload and manage SCORM packages at `/courses`
- **Registrations**: View and manage learner enrollments

### API Endpoints

#### Course Management
```bash
# List all courses
GET /api/courses

# Create a course
POST /api/courses
{
  "title": "Course Title",
  "description": "Course Description"
}

# Get course details
GET /api/courses/{courseId}

# Delete a course
DELETE /api/courses/{courseId}

# Launch a course (creates registration)
GET /api/courses/{courseId}/launch?learner_id=123&learner_name=John+Doe
```

#### Package Upload
```bash
# Upload SCORM package
POST /api/packages/upload
Content-Type: multipart/form-data

Fields:
- title: Course title
- description: Course description  
- package: SCORM ZIP file
```

#### Registration Management
```bash
# List registrations
GET /api/registrations[?learner_id=123]

# Create registration
POST /api/registrations
{
  "course_id": "course123",
  "learner_id": "learner123", 
  "learner_name": "John Doe"
}

# Get registration details
GET /api/registrations/{registrationId}

# Update registration
PUT /api/registrations/{registrationId}
{
  "status": "completed",
  "score": 85.5
}

# Commit registration state
POST /api/registrations/{registrationId}/commit

# Terminate session
POST /api/registrations/{registrationId}/terminate
```

#### CMI Data
```bash
# Get all CMI data for registration
GET /api/cmi/{registrationId}

# Get specific CMI element
GET /api/cmi/{registrationId}?element=cmi.core.lesson_status

# Set CMI data
POST /api/cmi/{registrationId}
{
  "element": "cmi.core.lesson_status",
  "value": "completed"
}
```

#### Reports
```bash
# System summary
GET /api/reports/summary

# Course report
GET /api/reports/course/{courseId}

# Learner report  
GET /api/reports/learner/{learnerId}

# Completion report
GET /api/reports/completion[?limit=50]
```

#### SCORM API
```bash
# Get SCORM JavaScript API for content
GET /api/registrations/{registrationId}/scorm-api
```

## SCORM 2004 Support

### Supported CMI Elements
- `cmi.core.lesson_status` - completion status
- `cmi.core.score.raw/min/max` - scoring data
- `cmi.core.session_time` - current session time
- `cmi.core.total_time` - cumulative time
- `cmi.core.student_id/student_name` - learner identification
- `cmi.core.entry/exit` - entry/exit conditions
- `cmi.location` - bookmarking support
- `cmi.suspend_data` - session state persistence
- Custom CMI elements via extensible storage

### SCORM Package Requirements
- Valid SCORM 2004 ZIP package
- Must contain `imsmanifest.xml`
- Manifest must have valid structure with resources
- Content files referenced in manifest

## Database Schema

The application uses SQLite with the following main tables:

- **courses**: SCORM course metadata and manifests
- **scos**: Shareable Content Objects extracted from packages
- **registrations**: Learner enrollments with CMI core data
- **cmi_data**: Extended CMI element storage
- **interactions**: Detailed interaction tracking

## API Documentation

Complete OpenAPI 3.0 specification is available at `/api/openapi.yaml`.

Key features of the API:
- RESTful design with proper HTTP methods
- JSON request/response format
- Comprehensive error handling
- Pagination support where applicable
- Proper HTTP status codes

## Development

### Project Structure
- Clean architecture with separated concerns
- Standard Go project layout
- HTMX for dynamic frontend interactions
- SQLite for simple, embedded storage
- No external runtime dependencies

### Adding Features
1. Define models in `internal/models/`
2. Add database schema in `internal/database/`
3. Implement handlers in `internal/handlers/`
4. Update OpenAPI spec in `api/openapi.yaml`
5. Add frontend components in `web/`

## Testing

```bash
# Test API endpoints
curl -X POST -H "Content-Type: application/json" \
     -d '{"title":"Test Course"}' \
     http://localhost:8080/api/courses

# Test SCORM package upload
curl -X POST -F "title=Test Course" \
     -F "package=@/path/to/scorm.zip" \
     http://localhost:8080/api/packages/upload
```

## License

MIT License - see LICENSE file for details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## Support

For issues and questions:
- Check the OpenAPI specification for API details
- Review the source code for implementation details
- Create GitHub issues for bugs or feature requests

## Course Management UI Architecture

The course management system uses **HTMX** for dynamic interactions and **Bulma CSS** for styling:

### Data Loading Flow

1. **Course List** (`/courses`):
   - Page loads with empty container showing loading spinner
   - HTMX automatically triggers `GET /api/courses` on page load via `hx-get` attribute
   - JavaScript `htmx:beforeSwap` event intercepts JSON response
   - `transformCoursesToHTML(courses)` converts JSON to Bulma card HTML
   - Courses display in responsive grid layout with action buttons

2. **Course Creation** (`/courses/create`):
   - Form submission uses HTMX `POST /courses/create`
   - Server validates and creates course, returns `HX-Redirect` header
   - HTMX automatically redirects to new course detail page

3. **Course Details** (`/courses/{id}#slug`):
   - URL uses course title as slug for SEO-friendly URLs
   - Edit mode activated with `?edit=true` query parameter
   - Analytics loaded dynamically via `GET /api/reports/course/{id}`

4. **Course Updates**:
   - Form uses method override: `<input type="hidden" name="_method" value="PUT">`
   - HTMX sends `POST /courses/{id}` with `_method=PUT`
   - Server processes as PUT request and redirects

### Key Functions

**Frontend (JavaScript)**:
- `transformCoursesToHTML(courses)` - Converts JSON course array to Bulma card HTML
- `createSlug(title)` - Creates URL-friendly slugs from course titles  
- `launchCourse(courseId)` - Prompts for learner details and launches course
- `showUploadModal()` / `hideUploadModal()` - SCORM package upload interface

**Backend (Go)**:
- `HandleCoursesListPage()` - Serves course list template
- `HandleCourseCreatePage()` - Handles GET (form) and POST (submission)
- `HandleCourseDetailPage()` - Shows course details with edit mode support
- `createCourseFromForm()` - Processes form data and creates course
- `updateCourseFromForm()` - Updates existing course from form data

### Routing Architecture

```
Web UI Routes (HTML responses):
/courses/create     → Course creation form
/courses/{id}       → Course detail/edit pages  
/courses            → Course list page

API Routes (JSON responses):
/api/courses/{id}   → Individual course CRUD
/api/courses        → Course collection operations
```

### Troubleshooting Course Loading

If courses don't appear on `/courses`:

1. **Check Browser Console**: Look for JavaScript errors in transformation
2. **Verify API**: `curl http://localhost:8080/api/courses` should return JSON
3. **Check HTMX**: Network tab should show `GET /api/courses` request
4. **Debug Transform**: Console should log "Transforming courses: N courses found"

The system uses event-driven architecture where HTMX triggers API calls and JavaScript transforms the responses for seamless user experience.
