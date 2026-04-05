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

	return nil
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
