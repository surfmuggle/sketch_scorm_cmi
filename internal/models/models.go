package models

import (
	"time"
)

// Course represents a SCORM course/package
type Course struct {
	ID          string    `json:"id" db:"id"`
	Title       string    `json:"title" db:"title"`
	Description string    `json:"description" db:"description"`
	Version     string    `json:"version" db:"version"`
	PackagePath *string   `json:"package_path,omitempty" db:"package_path"`
	Manifest    *string   `json:"manifest,omitempty" db:"manifest"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// SCO represents a Shareable Content Object
type SCO struct {
	ID         string `json:"id" db:"id"`
	CourseID   string `json:"course_id" db:"course_id"`
	Title      string `json:"title" db:"title"`
	Href       string `json:"href" db:"href"`
	Parameters string `json:"parameters" db:"parameters"`
	LaunchURL  string `json:"launch_url" db:"launch_url"`
}

// Registration represents a learner's enrollment in a course
type Registration struct {
	ID          string     `json:"id" db:"id"`
	CourseID    string     `json:"course_id" db:"course_id"`
	LearnerID   string     `json:"learner_id" db:"learner_id"`
	LearnerName string     `json:"learner_name" db:"learner_name"`
	Status      string     `json:"status" db:"status"` // not_attempted, incomplete, completed, failed, passed, browsed
	Score       *float64   `json:"score,omitempty" db:"score"`
	MinScore    *float64   `json:"min_score,omitempty" db:"min_score"`
	MaxScore    *float64   `json:"max_score,omitempty" db:"max_score"`
	SessionTime string     `json:"session_time" db:"session_time"`
	TotalTime   string     `json:"total_time" db:"total_time"`
	Location    *string    `json:"location,omitempty" db:"location"` // bookmark
	SuspendData *string    `json:"suspend_data,omitempty" db:"suspend_data"`
	Exit        *string    `json:"exit,omitempty" db:"exit"`
	Entry       string     `json:"entry" db:"entry"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`
}

// CMIData represents detailed CMI data for tracking
type CMIData struct {
	ID             string    `json:"id" db:"id"`
	RegistrationID string    `json:"registration_id" db:"registration_id"`
	Element        string    `json:"element" db:"element"`
	Value          string    `json:"value" db:"value"`
	Timestamp      time.Time `json:"timestamp" db:"timestamp"`
}

// Interaction represents learner interactions for detailed tracking
type Interaction struct {
	ID               string    `json:"id" db:"id"`
	RegistrationID   string    `json:"registration_id" db:"registration_id"`
	InteractionID    string    `json:"interaction_id" db:"interaction_id"`
	Type             string    `json:"type" db:"type"`
	Objectives       string    `json:"objectives" db:"objectives"`
	Timestamp        time.Time `json:"timestamp" db:"timestamp"`
	CorrectResponses string    `json:"correct_responses" db:"correct_responses"`
	Weighting        *float64  `json:"weighting,omitempty" db:"weighting"`
	LearnerResponse  string    `json:"learner_response" db:"learner_response"`
	Result           string    `json:"result" db:"result"`
	Latency          string    `json:"latency" db:"latency"`
	Description      string    `json:"description" db:"description"`
}

// CourseUploadRequest represents a course package upload request
type CourseUploadRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// RegistrationRequest represents a registration creation request
type RegistrationRequest struct {
	CourseID    string `json:"course_id"`
	LearnerID   string `json:"learner_id"`
	LearnerName string `json:"learner_name"`
}

// CMIUpdateRequest represents a CMI data update request
type CMIUpdateRequest struct {
	Element string `json:"element"`
	Value   string `json:"value"`
}

// LaunchResponse represents a course launch response
type LaunchResponse struct {
	RegistrationID string `json:"registration_id"`
	LaunchURL      string `json:"launch_url"`
	JSAPI          string `json:"js_api"`
}
