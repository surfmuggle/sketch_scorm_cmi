// SCORM CMI App JavaScript

// Global utilities
window.SCORMApp = {
    // Format date for display
    formatDate: function(dateString) {
        if (!dateString) return 'N/A';
        const date = new Date(dateString);
        return date.toLocaleDateString() + ' ' + date.toLocaleTimeString();
    },
    
    // Format duration (PT format to human readable)
    formatDuration: function(duration) {
        if (!duration || duration === '00:00:00') return 'No time recorded';
        return duration;
    },
    
    // Get status badge HTML
    getStatusBadge: function(status) {
        const badges = {
            'not_attempted': '<span class="badge badge-secondary">Not Started</span>',
            'incomplete': '<span class="badge badge-warning">In Progress</span>',
            'completed': '<span class="badge badge-success">Completed</span>',
            'passed': '<span class="badge badge-success">Passed</span>',
            'failed': '<span class="badge badge-danger">Failed</span>',
            'browsed': '<span class="badge badge-info">Browsed</span>'
        };
        return badges[status] || `<span class="badge badge-light">${status}</span>`;
    }
};

// HTMX event handlers
document.body.addEventListener('htmx:afterRequest', function(evt) {
    const xhr = evt.detail.xhr;
    const pathInfo = evt.detail.pathInfo;
    
    // Handle API responses
    if (pathInfo.requestPath.startsWith('/api/')) {
        if (xhr.status >= 400) {
            try {
                const error = JSON.parse(xhr.responseText);
                alert('Error: ' + error.error);
            } catch (e) {
                alert('An error occurred: ' + xhr.status);
            }
        }
    }
});

// Handle HTMX response transformations
document.body.addEventListener('htmx:beforeSwap', function(evt) {
    const response = evt.detail.xhr.responseText;
    const target = evt.detail.target;
    
    // Transform API responses to HTML
    if (evt.detail.pathInfo.requestPath === '/api/courses' && target.id === 'courses-content') {
        try {
            const courses = JSON.parse(response);
            evt.detail.serverResponse = transformCoursesToHTML(courses);
        } catch (e) {
            console.error('Error parsing courses response:', e);
        }
    }
    
    if (evt.detail.pathInfo.requestPath === '/api/registrations' && target.id === 'registrations-content') {
        try {
            const registrations = JSON.parse(response);
            evt.detail.serverResponse = transformRegistrationsToHTML(registrations);
        } catch (e) {
            console.error('Error parsing registrations response:', e);
        }
    }
    
    if (evt.detail.pathInfo.requestPath === '/api/reports/summary' && target.id === 'stats-content') {
        try {
            const summary = JSON.parse(response);
            evt.detail.serverResponse = transformSummaryToHTML(summary);
        } catch (e) {
            console.error('Error parsing summary response:', e);
        }
    }
});

// Transform functions
function transformCoursesToHTML(courses) {
    if (!courses.length) {
        return '<div class="text-center p-2">No courses found. Upload a SCORM package to get started.</div>';
    }
    
    let html = `
        <table class="content-table">
            <thead>
                <tr>
                    <th>Title</th>
                    <th>Description</th>
                    <th>Version</th>
                    <th>Created</th>
                    <th>Actions</th>
                </tr>
            </thead>
            <tbody>
    `;
    
    courses.forEach(course => {
        html += `
            <tr>
                <td><strong>${course.title}</strong></td>
                <td>${course.description || 'No description'}</td>
                <td>${course.version || 'N/A'}</td>
                <td>${SCORMApp.formatDate(course.created_at)}</td>
                <td>
                    <button onclick="showCourseDetails('${course.id}')" class="btn btn-primary btn-sm">View</button>
                    <button onclick="launchCourse('${course.id}')" class="btn btn-success btn-sm">Launch</button>
                    <button onclick="deleteCourse('${course.id}')" class="btn btn-danger btn-sm">Delete</button>
                </td>
            </tr>
        `;
    });
    
    html += '</tbody></table>';
    return html;
}

function transformRegistrationsToHTML(registrations) {
    if (!registrations.length) {
        return '<div class="text-center p-2">No registrations found.</div>';
    }
    
    let html = `
        <table class="content-table">
            <thead>
                <tr>
                    <th>Learner</th>
                    <th>Course</th>
                    <th>Status</th>
                    <th>Score</th>
                    <th>Total Time</th>
                    <th>Last Updated</th>
                    <th>Actions</th>
                </tr>
            </thead>
            <tbody>
    `;
    
    registrations.forEach(item => {
        const reg = item.registration;
        const courseTitle = item.course_title;
        
        html += `
            <tr>
                <td>
                    <strong>${reg.learner_name}</strong><br>
                    <small>${reg.learner_id}</small>
                </td>
                <td>${courseTitle}</td>
                <td><span class="status-${reg.status}">${reg.status.replace('_', ' ')}</span></td>
                <td>${reg.score !== null ? reg.score : 'N/A'}</td>
                <td>${SCORMApp.formatDuration(reg.total_time)}</td>
                <td>${SCORMApp.formatDate(reg.updated_at)}</td>
                <td>
                    <button onclick="showRegistrationDetails('${reg.id}')" class="btn btn-primary btn-sm">View</button>
                </td>
            </tr>
        `;
    });
    
    html += '</tbody></table>';
    return html;
}

function transformSummaryToHTML(summary) {
    return `
        <div class="stat-card">
            <h3>${summary.total_courses}</h3>
            <p>Total Courses</p>
        </div>
        <div class="stat-card">
            <h3>${summary.total_registrations}</h3>
            <p>Total Registrations</p>
        </div>
        <div class="stat-card">
            <h3>${summary.completed_registrations}</h3>
            <p>Completed</p>
        </div>
        <div class="stat-card">
            <h3>${summary.completion_rate ? summary.completion_rate.toFixed(1) + '%' : '0%'}</h3>
            <p>Completion Rate</p>
        </div>
        ${summary.avg_score ? `
        <div class="stat-card">
            <h3>${summary.avg_score.toFixed(1)}</h3>
            <p>Average Score</p>
        </div>
        ` : ''}
        <div class="stat-card">
            <h3>${summary.active_registrations}</h3>
            <p>Active Learners</p>
        </div>
    `;
}

// Close modals when clicking outside
window.onclick = function(event) {
    const modals = document.querySelectorAll('.modal');
    modals.forEach(modal => {
        if (event.target === modal) {
            modal.style.display = 'none';
        }
    });
};

// Keyboard shortcuts
document.addEventListener('keydown', function(event) {
    // ESC to close modals
    if (event.key === 'Escape') {
        const visibleModals = document.querySelectorAll('.modal[style*="display: block"]');
        visibleModals.forEach(modal => {
            modal.style.display = 'none';
        });
    }
});

// Add CSS for button sizes
const additionalCSS = `
.btn-sm {
    padding: 0.25rem 0.5rem;
    font-size: 0.875rem;
    margin-right: 0.25rem;
}

.badge {
    display: inline-block;
    padding: 0.25rem 0.5rem;
    font-size: 0.75rem;
    font-weight: 700;
    border-radius: 0.25rem;
    text-transform: uppercase;
}

.badge-secondary { background-color: #6c757d; color: white; }
.badge-warning { background-color: #ffc107; color: #212529; }
.badge-success { background-color: #28a745; color: white; }
.badge-danger { background-color: #dc3545; color: white; }
.badge-info { background-color: #17a2b8; color: white; }
.badge-light { background-color: #f8f9fa; color: #212529; }
`;

// Inject additional CSS
const style = document.createElement('style');
style.textContent = additionalCSS;
document.head.appendChild(style);

console.log('SCORM CMI App JavaScript loaded');
