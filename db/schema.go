package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// InitDB initializes SQLite database with schema
func InitDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrent performance
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set WAL mode: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Create schema
	if err := createSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	// Run migrations for new features
	if err := RunMigrations(db); err != nil {
		fmt.Printf("Warning: Migration error (non-fatal): %v\n", err)
		// Don't exit, migrations may already exist
	}

	return db, nil
}

func createSchema(db *sql.DB) error {
	schema := `
	-- Nodes table: stores all code elements (functions, classes, methods)
	CREATE TABLE IF NOT EXISTS nodes (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		file TEXT NOT NULL,
		language TEXT NOT NULL,
		line INTEGER NOT NULL,
		end_line INTEGER NOT NULL,
		signature TEXT,
		body TEXT,
		docstring TEXT,
		pagerank REAL DEFAULT 0.0,
		blast_radius INTEGER DEFAULT 0
	);

	-- Edges table: stores relationships between nodes
	CREATE TABLE IF NOT EXISTS edges (
		source TEXT NOT NULL,
		target TEXT NOT NULL,
		edge_type TEXT NOT NULL,
		PRIMARY KEY (source, target, edge_type)
	);

	-- Metadata table: stores additional function metadata
	CREATE TABLE IF NOT EXISTS metadata (
		function_id TEXT PRIMARY KEY,
		deprecated BOOLEAN DEFAULT FALSE,
		hooks TEXT,
		todos TEXT,
		authors TEXT,
		FOREIGN KEY (function_id) REFERENCES nodes(id) ON DELETE CASCADE
	);

	-- Indexes for fast queries
	CREATE INDEX IF NOT EXISTS idx_nodes_file ON nodes(file);
	CREATE INDEX IF NOT EXISTS idx_nodes_name ON nodes(name);
	CREATE INDEX IF NOT EXISTS idx_nodes_type ON nodes(type);
	CREATE INDEX IF NOT EXISTS idx_nodes_language ON nodes(language);
	CREATE INDEX IF NOT EXISTS idx_edges_source ON edges(source);
	CREATE INDEX IF NOT EXISTS idx_edges_target ON edges(target);
	CREATE INDEX IF NOT EXISTS idx_edges_type ON edges(edge_type);
	`

	_, err := db.Exec(schema)
	return err
}
