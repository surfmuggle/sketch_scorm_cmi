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
    
    if (evt.detail.pathInfo.requestPath === '/api/reports/summary' && target.id === 'course-stats') {
        try {
            const summary = JSON.parse(response);
            evt.detail.serverResponse = transformSummaryToCourseStats(summary);
        } catch (e) {
            console.error('Error parsing summary response:', e);
        }
    }
    
    if (evt.detail.pathInfo.requestPath.startsWith('/api/reports/course/') && target.id === 'course-analytics') {
        try {
            const report = JSON.parse(response);
            evt.detail.serverResponse = transformCourseAnalytics(report);
        } catch (e) {
            console.error('Error parsing course analytics response:', e);
        }
    }
});

// Transform functions
function transformCoursesToHTML(courses) {
    if (!courses.length) {
        return `
            <div class="notification is-info">
                <div class="has-text-centered">
                    <i class="fas fa-info-circle fa-2x mb-2"></i>
                    <p><strong>No courses found</strong></p>
                    <p>Get started by creating a course or uploading a SCORM package.</p>
                    <div class="buttons is-centered mt-3">
                        <a href="/courses/create" class="button is-primary">
                            <i class="fas fa-plus mr-1"></i>
                            Create Course
                        </a>
                        <button onclick="showUploadModal()" class="button is-link">
                            <i class="fas fa-upload mr-1"></i>
                            Upload SCORM
                        </button>
                    </div>
                </div>
            </div>
        `;
    }
    
    let html = '<div class="columns is-multiline">';
    
    courses.forEach(course => {
        const slug = createSlug(course.title);
        const hasPackage = course.package_path ? true : false;
        
        html += `
            <div class="column is-6">
                <div class="card">
                    <div class="card-content">
                        <div class="media">
                            <div class="media-left">
                                <figure class="image is-48x48">
                                    <i class="fas fa-book fa-2x has-text-primary"></i>
                                </figure>
                            </div>
                            <div class="media-content">
                                <p class="title is-5">
                                    <a href="/courses/${course.id}#${slug}" class="has-text-dark">
                                        ${course.title}
                                    </a>
                                </p>
                                <p class="subtitle is-7">
                                    Version ${course.version || '1.0'} • 
                                    ${hasPackage ? 
                                        '<span class="has-text-success"><i class="fas fa-check-circle"></i> SCORM Package</span>' : 
                                        '<span class="has-text-warning"><i class="fas fa-exclamation-circle"></i> No Package</span>'
                                    }
                                </p>
                            </div>
                        </div>
                        
                        ${course.description ? `
                        <div class="content">
                            <p>${course.description.length > 100 ? course.description.substring(0, 100) + '...' : course.description}</p>
                        </div>
                        ` : ''}
                        
                        <div class="content">
                            <time class="has-text-grey is-size-7">
                                <i class="fas fa-calendar-alt mr-1"></i>
                                Created ${SCORMApp.formatDate(course.created_at)}
                            </time>
                        </div>
                    </div>
                    <footer class="card-footer">
                        <a href="/courses/${course.id}#${slug}" class="card-footer-item has-text-info">
                            <i class="fas fa-eye mr-1"></i>
                            View
                        </a>
                        <a href="/courses/${course.id}#${slug}?edit=true" class="card-footer-item has-text-warning">
                            <i class="fas fa-edit mr-1"></i>
                            Edit
                        </a>
                        <button onclick="launchCourse('${course.id}')" class="card-footer-item has-text-success" style="border: none; background: none;">
                            <i class="fas fa-play mr-1"></i>
                            Launch
                        </button>
                    </footer>
                </div>
            </div>
        `;
    });
    
    html += '</div>';
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

// Transform course statistics for dashboard
function transformSummaryToCourseStats(summary) {
    return `
        <div class="columns">
            <div class="column">
                <div class="box has-text-centered">
                    <p class="title is-4 has-text-primary">${summary.total_courses}</p>
                    <p class="subtitle is-6">Total Courses</p>
                </div>
            </div>
            <div class="column">
                <div class="box has-text-centered">
                    <p class="title is-4 has-text-info">${summary.total_registrations}</p>
                    <p class="subtitle is-6">Registrations</p>
                </div>
            </div>
            <div class="column">
                <div class="box has-text-centered">
                    <p class="title is-4 has-text-success">${summary.completed_registrations}</p>
                    <p class="subtitle is-6">Completed</p>
                </div>
            </div>
            <div class="column">
                <div class="box has-text-centered">
                    <p class="title is-4 has-text-warning">${summary.completion_rate ? summary.completion_rate.toFixed(1) + '%' : '0%'}</p>
                    <p class="subtitle is-6">Completion Rate</p>
                </div>
            </div>
        </div>
    `;
}

// Transform course analytics for course detail page
function transformCourseAnalytics(report) {
    let learnersTable = '';
    if (report.learners && report.learners.length > 0) {
        learnersTable = `
            <div class="table-container">
                <table class="table is-fullwidth is-striped">
                    <thead>
                        <tr>
                            <th>Learner</th>
                            <th>Status</th>
                            <th>Score</th>
                            <th>Time Spent</th>
                            <th>Last Activity</th>
                        </tr>
                    </thead>
                    <tbody>
        `;
        
        report.learners.forEach(learner => {
            const statusClass = {
                'not_attempted': 'is-light',
                'incomplete': 'is-warning',
                'completed': 'is-success',
                'passed': 'is-success',
                'failed': 'is-danger',
                'browsed': 'is-info'
            }[learner.status] || 'is-light';
            
            learnersTable += `
                <tr>
                    <td>
                        <strong>${learner.learner_name}</strong><br>
                        <small class="has-text-grey">${learner.learner_id}</small>
                    </td>
                    <td>
                        <span class="tag ${statusClass}">
                            ${learner.status.replace('_', ' ').toUpperCase()}
                        </span>
                    </td>
                    <td>${learner.score !== null ? learner.score : '-'}</td>
                    <td>${SCORMApp.formatDuration(learner.total_time)}</td>
                    <td>${SCORMApp.formatDate(learner.updated_at)}</td>
                </tr>
            `;
        });
        
        learnersTable += '</tbody></table></div>';
    } else {
        learnersTable = '<p class="has-text-grey has-text-centered p-4">No registrations yet</p>';
    }
    
    return `
        <div class="columns">
            <div class="column is-3">
                <div class="box has-text-centered">
                    <p class="title is-4 has-text-primary">${report.total_registrations}</p>
                    <p class="subtitle is-6">Total</p>
                </div>
            </div>
            <div class="column is-3">
                <div class="box has-text-centered">
                    <p class="title is-4 has-text-success">${report.completed_registrations}</p>
                    <p class="subtitle is-6">Completed</p>
                </div>
            </div>
            <div class="column is-3">
                <div class="box has-text-centered">
                    <p class="title is-4 has-text-warning">${report.in_progress_registrations}</p>
                    <p class="subtitle is-6">In Progress</p>
                </div>
            </div>
            <div class="column is-3">
                <div class="box has-text-centered">
                    <p class="title is-4 has-text-info">${report.completion_rate.toFixed(1)}%</p>
                    <p class="subtitle is-6">Completion Rate</p>
                </div>
            </div>
        </div>
        
        ${report.avg_score ? `
        <div class="notification is-primary is-light">
            <strong>Average Score:</strong> ${report.avg_score.toFixed(1)}
        </div>
        ` : ''}
        
        <h3 class="subtitle is-6 mt-4">Recent Learner Activity</h3>
        ${learnersTable}
    `;
}

// Helper function to create slugs (matching server-side implementation)
function createSlug(title) {
    return title.toLowerCase().replace(/[^a-zA-Z0-9]+/g, '-').replace(/^-|-$/g, '');
}