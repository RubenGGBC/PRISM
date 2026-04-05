package graph

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/ruffini/prism/parser"
)

// CodeGraph manages the code dependency graph in SQLite
type CodeGraph struct {
	DB *sql.DB
}

// NewGraph creates a new graph instance
func NewGraph(db *sql.DB) *CodeGraph {
	return &CodeGraph{DB: db}
}

// AddNode inserts or updates a node in the graph
func (g *CodeGraph) AddNode(elem parser.CodeElement) error {
	query := `
	INSERT INTO nodes (id, name, type, file, language, line, end_line, signature, body, docstring, pagerank, blast_radius)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0.0, 0)
	ON CONFLICT(id) DO UPDATE SET
		name = excluded.name,
		type = excluded.type,
		file = excluded.file,
		language = excluded.language,
		line = excluded.line,
		end_line = excluded.end_line,
		signature = excluded.signature,
		body = excluded.body,
		docstring = excluded.docstring
	`

	_, err := g.DB.Exec(query,
		elem.ID,
		elem.Name,
		elem.Type,
		elem.File,
		elem.Language,
		elem.Line,
		elem.EndLine,
		elem.Signature,
		elem.Body,
		elem.DocString,
	)
	return err
}

// AddEdge inserts an edge between two nodes
func (g *CodeGraph) AddEdge(source, target, edgeType string) error {
	query := `
	INSERT INTO edges (source, target, edge_type)
	VALUES (?, ?, ?)
	ON CONFLICT(source, target, edge_type) DO NOTHING
	`
	_, err := g.DB.Exec(query, source, target, edgeType)
	return err
}

// BuildFromParsed builds the graph from parsed files
func (g *CodeGraph) BuildFromParsed(files map[string]*parser.ParsedFile) error {
	// Start a transaction for better performance
	tx, err := g.DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Prepare statements
	nodeStmt, err := tx.Prepare(`
		INSERT INTO nodes (id, name, type, file, language, line, end_line, signature, body, docstring, pagerank, blast_radius)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0.0, 0)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			type = excluded.type,
			file = excluded.file,
			language = excluded.language,
			line = excluded.line,
			end_line = excluded.end_line,
			signature = excluded.signature,
			body = excluded.body,
			docstring = excluded.docstring
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare node statement: %w", err)
	}
	defer nodeStmt.Close()

	edgeStmt, err := tx.Prepare(`
		INSERT INTO edges (source, target, edge_type)
		VALUES (?, ?, ?)
		ON CONFLICT(source, target, edge_type) DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare edge statement: %w", err)
	}
	defer edgeStmt.Close()

	// Insert all nodes and collect call edges
	for _, file := range files {
		for _, elem := range file.Elements {
			_, err := nodeStmt.Exec(
				elem.ID,
				elem.Name,
				elem.Type,
				elem.File,
				elem.Language,
				elem.Line,
				elem.EndLine,
				elem.Signature,
				elem.Body,
				elem.DocString,
			)
			if err != nil {
				return fmt.Errorf("failed to insert node %s: %w", elem.ID, err)
			}

			// Add call edges (unresolved - target is just the function name)
			for _, callsTo := range elem.CallsTo {
				_, err := edgeStmt.Exec(elem.ID, callsTo, "calls")
				if err != nil {
					return fmt.Errorf("failed to insert edge: %w", err)
				}
			}
		}

		// Add import edges
		for _, imp := range file.Imports {
			// Create a pseudo-node ID for the import
			for _, elem := range file.Elements {
				// Link elements in this file to their imports
				_, err := edgeStmt.Exec(elem.ID, imp.Module, "imports")
				if err != nil {
					return fmt.Errorf("failed to insert import edge: %w", err)
				}
			}
		}
	}

	// Store metadata
	for _, file := range files {
		for _, elem := range file.Elements {
			if elem.Metadata == nil {
				continue
			}

			// Extract metadata fields
			deprecated := false
			var hooks, todos, authors interface{}

			if d, ok := elem.Metadata["deprecated"].(bool); ok {
				deprecated = d
			}
			if h, ok := elem.Metadata["hooks"].([]string); ok {
				// Convert to comma-separated string
				hooks = strings.Join(h, ", ")
			}
			if t, ok := elem.Metadata["todos"].([]string); ok {
				todos = strings.Join(t, ", ")
			}
			if a, ok := elem.Metadata["authors"].([]string); ok {
				authors = strings.Join(a, ", ")
			}

			// Store in metadata table
			_, err := tx.Exec(`
				INSERT INTO metadata (function_id, deprecated, hooks, todos, authors)
				VALUES (?, ?, ?, ?, ?)
				ON CONFLICT(function_id) DO UPDATE SET
					deprecated = excluded.deprecated,
					hooks = excluded.hooks,
					todos = excluded.todos,
					authors = excluded.authors
			`,
				elem.ID,
				deprecated,
				hooks,
				todos,
				authors,
			)
			if err != nil {
				return fmt.Errorf("failed to store metadata for %s: %w", elem.ID, err)
			}
		}
	}

	return tx.Commit()
}

// ResolveCallEdges attempts to resolve function calls to their actual definitions
func (g *CodeGraph) ResolveCallEdges() error {
	// Get all unresolved call edges (target is just a function name, not a full ID)
	rows, err := g.DB.Query(`
		SELECT e.source, e.target 
		FROM edges e
		WHERE e.edge_type = 'calls' 
		AND NOT EXISTS (SELECT 1 FROM nodes n WHERE n.id = e.target)
	`)
	if err != nil {
		return fmt.Errorf("failed to query unresolved edges: %w", err)
	}
	defer rows.Close()

	// Build a map of function names to their full IDs
	nameToIDs := make(map[string][]string)
	nodeRows, err := g.DB.Query(`SELECT id, name FROM nodes WHERE type IN ('function', 'method')`)
	if err != nil {
		return fmt.Errorf("failed to query nodes: %w", err)
	}
	defer nodeRows.Close()

	for nodeRows.Next() {
		var id, name string
		if err := nodeRows.Scan(&id, &name); err != nil {
			return err
		}
		nameToIDs[name] = append(nameToIDs[name], id)
	}

	// Collect edges to update
	type edgeUpdate struct {
		source    string
		oldTarget string
		newTarget string
	}
	var updates []edgeUpdate

	for rows.Next() {
		var source, target string
		if err := rows.Scan(&source, &target); err != nil {
			return err
		}

		// Try to find a matching function
		if ids, ok := nameToIDs[target]; ok {
			// Prefer functions in the same file
			sourceFile := extractFile(source)
			bestMatch := ""
			for _, id := range ids {
				if extractFile(id) == sourceFile {
					bestMatch = id
					break
				}
			}
			if bestMatch == "" && len(ids) > 0 {
				bestMatch = ids[0]
			}
			if bestMatch != "" {
				updates = append(updates, edgeUpdate{source, target, bestMatch})
			}
		}
	}

	// Apply updates in a transaction
	if len(updates) > 0 {
		tx, err := g.DB.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		for _, u := range updates {
			// Delete old edge and insert new one
			_, err := tx.Exec(`DELETE FROM edges WHERE source = ? AND target = ? AND edge_type = 'calls'`,
				u.source, u.oldTarget)
			if err != nil {
				return err
			}
			_, err = tx.Exec(`INSERT INTO edges (source, target, edge_type) VALUES (?, ?, 'calls')
				ON CONFLICT DO NOTHING`, u.source, u.newTarget)
			if err != nil {
				return err
			}
		}

		return tx.Commit()
	}

	return nil
}

// extractFile extracts the file path from a node ID (format: "file.py:function_name")
func extractFile(id string) string {
	idx := strings.LastIndex(id, ":")
	if idx > 0 {
		return id[:idx]
	}
	return id
}

// Stats returns the count of nodes and edges
func (g *CodeGraph) Stats() (nodes int, edges int, err error) {
	err = g.DB.QueryRow(`SELECT COUNT(*) FROM nodes`).Scan(&nodes)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to count nodes: %w", err)
	}

	err = g.DB.QueryRow(`SELECT COUNT(*) FROM edges`).Scan(&edges)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to count edges: %w", err)
	}

	return nodes, edges, nil
}
