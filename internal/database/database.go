package database

import (
	"database/sql"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

// Initialize creates and initializes the SQLite database
func Initialize(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, err
	}

	// Create tables
	if err := createTables(db); err != nil {
		return nil, err
	}

	return db, nil
}

func createTables(db *sql.DB) error {
	// Courses table
	coursesTable := `
	CREATE TABLE IF NOT EXISTS courses (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		description TEXT,
		version TEXT,
		package_path TEXT,
		manifest TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`

	// SCOs (Shareable Content Objects) table
	scosTable := `
	CREATE TABLE IF NOT EXISTS scos (
		id TEXT PRIMARY KEY,
		course_id TEXT NOT NULL,
		title TEXT NOT NULL,
		href TEXT,
		parameters TEXT,
		launch_url TEXT,
		FOREIGN KEY (course_id) REFERENCES courses(id) ON DELETE CASCADE
	)`

	// Registrations table
	registrationsTable := `
	CREATE TABLE IF NOT EXISTS registrations (
		id TEXT PRIMARY KEY,
		course_id TEXT NOT NULL,
		learner_id TEXT NOT NULL,
		learner_name TEXT NOT NULL,
		status TEXT DEFAULT 'not_attempted',
		score REAL,
		min_score REAL,
		max_score REAL,
		session_time TEXT DEFAULT '00:00:00',
		total_time TEXT DEFAULT '00:00:00',
		location TEXT,
		suspend_data TEXT,
		exit TEXT,
		entry TEXT DEFAULT 'ab-initio',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		completed_at DATETIME,
		FOREIGN KEY (course_id) REFERENCES courses(id) ON DELETE CASCADE,
		UNIQUE(course_id, learner_id)
	)`

	// CMI Data table for detailed tracking
	cmiDataTable := `
	CREATE TABLE IF NOT EXISTS cmi_data (
		id TEXT PRIMARY KEY,
		registration_id TEXT NOT NULL,
		element TEXT NOT NULL,
		value TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (registration_id) REFERENCES registrations(id) ON DELETE CASCADE
	)`

	// Interactions table for detailed interaction tracking
	interactionsTable := `
	CREATE TABLE IF NOT EXISTS interactions (
		id TEXT PRIMARY KEY,
		registration_id TEXT NOT NULL,
		interaction_id TEXT NOT NULL,
		type TEXT,
		objectives TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		correct_responses TEXT,
		weighting REAL,
		learner_response TEXT,
		result TEXT,
		latency TEXT,
		description TEXT,
		FOREIGN KEY (registration_id) REFERENCES registrations(id) ON DELETE CASCADE
	)`

	tables := []string{
		coursesTable,
		scosTable,
		registrationsTable,
		cmiDataTable,
		interactionsTable,
	}

	for _, table := range tables {
		if _, err := db.Exec(table); err != nil {
			return err
		}
	}

	// Create indexes for better performance
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_scos_course_id ON scos(course_id)`,
		`CREATE INDEX IF NOT EXISTS idx_registrations_course_id ON registrations(course_id)`,
		`CREATE INDEX IF NOT EXISTS idx_registrations_learner_id ON registrations(learner_id)`,
		`CREATE INDEX IF NOT EXISTS idx_cmi_data_registration_id ON cmi_data(registration_id)`,
		`CREATE INDEX IF NOT EXISTS idx_cmi_data_element ON cmi_data(element)`,
		`CREATE INDEX IF NOT EXISTS idx_interactions_registration_id ON interactions(registration_id)`,
	}

	for _, index := range indexes {
		if _, err := db.Exec(index); err != nil {
			return err
		}
	}

	// Create triggers to update updated_at timestamps
	updateTriggers := []string{
		`CREATE TRIGGER IF NOT EXISTS update_courses_timestamp 
		 AFTER UPDATE ON courses FOR EACH ROW
		 BEGIN
			 UPDATE courses SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		 END`,
		`CREATE TRIGGER IF NOT EXISTS update_registrations_timestamp 
		 AFTER UPDATE ON registrations FOR EACH ROW
		 BEGIN
			 UPDATE registrations SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		 END`,
	}

	for _, trigger := range updateTriggers {
		if _, err := db.Exec(trigger); err != nil {
			return err
		}
	}

	return nil
}
