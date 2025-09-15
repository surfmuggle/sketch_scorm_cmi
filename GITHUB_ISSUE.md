# GitHub Issue: Course List Empty and Route Conflicts

## Issue Title
**Course list page shows empty results despite API returning data - Route conflicts causing display issues**

## Labels
`bug` `frontend` `routing` `high-priority`

## Description

The course management UI has several critical issues preventing proper course display and routing:

### 🐛 **Primary Issue: Empty Course List**
The `/courses` page shows an empty course list even when courses exist in the database. The API endpoint `/api/courses` returns data correctly, but the web UI template `web/templates/courses-list.html` fails to display the courses.

### 🔀 **Secondary Issue: Route Conflicts** 
Two route patterns are defined twice in `main.go`, potentially causing routing conflicts:

```go
// Web UI routes
mux.HandleFunc("/courses", h.HandleCoursesListPage)        // ✅ Specific match
mux.HandleFunc("/courses/", h.HandleCourseDetailPage)      // ❌ Conflicts with above
mux.HandleFunc("/courses/create", h.HandleCourseCreatePage) // ✅ More specific

// API routes  
mux.HandleFunc("/api/courses", h.HandleCourses)   // ✅ Specific match
mux.HandleFunc("/api/courses/", h.HandleCourse)   // ❌ Conflicts with above
```

### 📋 **Steps to Reproduce**
1. Start the application: `go run cmd/main.go`
2. Create a course via API: `curl -X POST -H "Content-Type: application/json" -d '{"title":"Test Course","description":"Test"}' http://localhost:8080/api/courses`
3. Verify API returns data: `curl http://localhost:8080/api/courses` ✅ Shows course
4. Visit web UI: `http://localhost:8080/courses` ❌ Shows "No courses found"
5. Check browser dev tools network tab - HTMX request may be failing

### 🔍 **Root Cause Analysis**

1. **HTMX Template Transformation**: The JavaScript function `transformCoursesToHTML()` in `/web/static/js/app.js` may not be properly triggered or the HTMX response transformation isn't working.

2. **Route Precedence**: Go's `http.ServeMux` matches the longest pattern first, but trailing slashes can cause unexpected behavior:
   - `/courses` vs `/courses/` creates ambiguous routing
   - Requests to `/courses/123` may not match `/courses/` properly

3. **Missing HTMX Configuration**: The template may not have proper HTMX attributes or target selectors.

### ✅ **Expected Behavior**
- `/courses` should display all courses in a responsive card layout
- `/courses/123#slug` should show individual course details  
- `/courses/create` should show course creation form
- API endpoints should work independently without conflicts

### 🛠️ **Proposed Solution**

1. **Fix Route Conflicts**:
   ```go
   // More specific patterns first
   mux.HandleFunc("/courses/create", h.HandleCourseCreatePage)
   mux.HandleFunc("/courses/", h.HandleCourseDetailPage)  // Handles /courses/{id}
   mux.HandleFunc("/courses", h.HandleCoursesListPage)    // Exact match only
   
   mux.HandleFunc("/api/courses/", h.HandleCourse)       // Handles /api/courses/{id}
   mux.HandleFunc("/api/courses", h.HandleCourses)        // Exact match only
   ```

2. **Debug HTMX Response**:
   - Add console logging to `transformCoursesToHTML()`
   - Verify HTMX request headers and response transformation
   - Check if `hx-target` and `hx-trigger` attributes are correct

3. **Update Documentation**: Add section to README.md explaining how course loading works in the web UI

### 🧪 **Testing Checklist**
- [ ] Course list displays existing courses
- [ ] Course creation form works
- [ ] Individual course pages load with proper URLs
- [ ] API endpoints remain functional
- [ ] No routing conflicts or 404 errors
- [ ] HTMX interactions work properly

### 📱 **Environment**
- **Go Version**: 1.21+
- **Dependencies**: Standard library + go-sqlite3, HTMX 1.9.10, Bulma CSS 0.9.4
- **Browser**: All modern browsers with JavaScript enabled

### 🏷️ **Priority**
**HIGH** - This breaks core functionality of the course management system.

### 💭 **Additional Context**
This issue affects the main user workflow for course management. Users cannot see or manage courses through the web interface, making the HTMX/Bulma UI non-functional despite the underlying API working correctly.

---

**Reporter**: @surfmuggle  
**Assignee**: TBD  
**Milestone**: Course Management UI v1.0
