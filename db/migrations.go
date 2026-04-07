package db

import (
	"database/sql"
	"fmt"
)

// RunMigrations applies pending schema changes
func RunMigrations(db *sql.DB) error {
	// Create node_comments table
	if err := createCommentsTable(db); err != nil {
		return fmt.Errorf("failed to create comments table: %w", err)
	}

	// Create node_tags table
	if err := createTagsTable(db); err != nil {
		return fmt.Errorf("failed to create tags table: %w", err)
	}

	// Create node_metadata table
	if err := createMetadataTable(db); err != nil {
		return fmt.Errorf("failed to create metadata table: %w", err)
	}

	// Create node_annotations table
	if err := createAnnotationsTable(db); err != nil {
		return fmt.Errorf("failed to create annotations table: %w", err)
	}
	if err := createProfilesTables(db); err != nil {
		return fmt.Errorf("failed to create profiles tables: %w", err)
	}
	if err := createMCPSessionsTable(db); err != nil {
		return fmt.Errorf("failed to create mcp_sessions table: %w", err)
	}

	return nil
}

func createProfilesTables(db *sql.DB) error {
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS profiles (
		id          TEXT PRIMARY KEY,
		name        TEXT NOT NULL UNIQUE,
		description TEXT,
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS profile_nodes (
		profile_id TEXT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
		node_id    TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
		PRIMARY KEY (profile_id, node_id)
	)`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_profile_nodes_profile ON profile_nodes(profile_id)`)
	return err
}

func createMCPSessionsTable(db *sql.DB) error {
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS mcp_sessions (
		id           TEXT PRIMARY KEY,
		started_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
		tokens_served INTEGER DEFAULT 0,
		tokens_saved  INTEGER DEFAULT 0
	)`)
	return err
}

func createCommentsTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS node_comments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		node_id TEXT NOT NULL UNIQUE,
		comment TEXT,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE CASCADE
	)`
	_, err := db.Exec(query)
	return err
}

func createTagsTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS node_tags (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		node_id TEXT NOT NULL,
		tag TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE CASCADE,
		UNIQUE(node_id, tag)
	)`
	_, err := db.Exec(query)
	return err
}

func createAnnotationsTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS node_annotations (
		node_id     TEXT PRIMARY KEY,
		why         TEXT,
		status      TEXT DEFAULT 'stable',
		entry_point BOOLEAN DEFAULT FALSE,
		known_bug   TEXT,
		updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE CASCADE
	)`
	_, err := db.Exec(query)
	return err
}

func createMetadataTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS node_metadata (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		node_id TEXT NOT NULL,
		key TEXT NOT NULL,
		value TEXT,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE CASCADE,
		UNIQUE(node_id, key)
	)`
	_, err := db.Exec(query)
	return err
}
