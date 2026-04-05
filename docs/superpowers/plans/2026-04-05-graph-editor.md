# Graph Editor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an interactive tree editor where users can annotate indexed code nodes with comments, tags, and custom metadata, and have those annotations enhance context returned to Claude Code via MCP.

**Architecture:** Backend extends SQLite schema with annotation tables (comments, tags, node_metadata). React frontend renders collapsible tree with inline editing, autosave to backend. MCP tools (get_file_smart, search_context) enhanced to include user annotations.

**Tech Stack:** Go (database, REST API), React 18 (tree component, inline editing), SQLite 3 (schema migration), TailwindCSS (styling)

---

## File Structure

**Backend (Go):**
- `internal/models/models.go` - Modify: Add Annotations, Tags, Comments fields to GraphNode
- `db/migrations.go` - Create: SQLite migration functions for new tables
- `api/handlers.go` - Modify: Add UpdateNode, GetNodeFull handlers
- `api/annotations.go` - Create: Annotation CRUD logic
- `graph/annotations.go` - Create: Query methods for annotations
- `mcp/tools.go` - Modify: Enhance get_file_smart and search_context

**Frontend (React):**
- `frontend/src/components/GraphEditor/TreeNode.tsx` - Create: Collapsible tree node component
- `frontend/src/components/GraphEditor/EditNodePanel.tsx` - Create: Inline editing panel
- `frontend/src/components/GraphEditor/TagBadge.tsx` - Create: Tag display component
- `frontend/src/components/GraphEditor/CodeTreeView.tsx` - Create: Main tree view component
- `frontend/src/hooks/useAutosave.ts` - Create: Autosave hook
- `frontend/src/pages/GraphEditorPage.tsx` - Create: Main page layout
- `frontend/src/styles/tree.css` - Create: Tree styling

**Tests:**
- `api/handlers_test.go` - Modify: Add tests for update/get endpoints
- `graph/annotations_test.go` - Create: Tests for annotation queries
- `frontend/src/components/GraphEditor/__tests__/TreeNode.test.tsx` - Create: Component tests

---

### Task 1: Extend GraphNode Model with Annotation Fields

**Files:**
- Modify: `internal/models/models.go`
- Test: Verify fields compile

- [ ] **Step 1: Add annotation fields to GraphNode struct**

Open `internal/models/models.go` and add these fields to the GraphNode struct:

```go
type GraphNode struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	File      string            `json:"file"`
	Line      int               `json:"line"`
	EndLine   int               `json:"end_line"`
	Signature string            `json:"signature"`
	Docstring string            `json:"docstring"`
	Body      string            `json:"body"`
	// NEW ANNOTATION FIELDS
	Comments      string            `json:"comments,omitempty"`      // User comments
	Tags          []string          `json:"tags,omitempty"`          // User-added tags
	CustomMetadata map[string]string `json:"custom_metadata,omitempty"` // Key-value metadata
	UpdatedAt     string            `json:"updated_at,omitempty"`   // Last annotation update
}

// AnnotationUpdate is sent by frontend to update node annotations
type AnnotationUpdate struct {
	Comments       string            `json:"comments"`
	Tags           []string          `json:"tags"`
	CustomMetadata map[string]string `json:"custom_metadata"`
}
```

- [ ] **Step 2: Verify compilation**

```bash
cd C:\Users\rebel\GolandProjects\TokenCompressorUI
go build -o prism . 2>&1 | head -20
```

Expected: No errors, or only missing function errors (which other tasks will fix)

- [ ] **Step 3: Commit**

```bash
git add internal/models/models.go
git commit -m "feat: add annotation fields to GraphNode model"
```

---

### Task 2: Create SQLite Migration for Annotation Tables

**Files:**
- Create: `db/migrations.go`
- Modify: `db/init.go` - call migration on startup
- Test: Run `prism index` on test project

- [ ] **Step 1: Create migrations.go file**

Create `db/migrations.go`:

```go
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
```

- [ ] **Step 2: Call migrations in InitDB**

Open `db/init.go`, find the `InitDB` function, and add this after the nodes table creation:

```go
// Run migrations for new features
if err := RunMigrations(database); err != nil {
	fmt.Printf("Warning: Migration error (non-fatal): %v\n", err)
	// Don't exit, migrations may already exist
}
```

- [ ] **Step 3: Build and test**

```bash
cd C:\Users\rebel\GolandProjects\TokenCompressorUI
go build -o prism . 2>&1
cd C:\Users\rebel\G1-Practica
rm code_graph.db  # Start fresh
prism index
```

Expected: No errors, tables created in database

- [ ] **Step 4: Verify tables exist**

```bash
sqlite3 C:\Users\rebel\G1-Practica\code_graph.db ".tables"
```

Expected output includes: `node_comments node_metadata node_tags nodes edges ...`

- [ ] **Step 5: Commit**

```bash
git add db/migrations.go db/init.go
git commit -m "feat: add annotation tables (comments, tags, metadata)"
```

---

### Task 3: Create Annotation Query Methods in Graph Package

**Files:**
- Create: `graph/annotations.go`
- Test: Unit tests in `graph/annotations_test.go`

- [ ] **Step 1: Create graph/annotations.go**

```go
package graph

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// GetNodeAnnotations retrieves all annotations for a node
func (g *CodeGraph) GetNodeAnnotations(nodeID string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Get comment
	var comment sql.NullString
	err := g.db.QueryRow(`SELECT comment FROM node_comments WHERE node_id = ?`, nodeID).Scan(&comment)
	if err == nil && comment.Valid {
		result["comments"] = comment.String
	}

	// Get tags
	rows, err := g.db.Query(`SELECT tag FROM node_tags WHERE node_id = ? ORDER BY tag`, nodeID)
	if err == nil {
		defer rows.Close()
		var tags []string
		for rows.Next() {
			var tag string
			rows.Scan(&tag)
			tags = append(tags, tag)
		}
		if len(tags) > 0 {
			result["tags"] = tags
		}
	}

	// Get custom metadata
	rows, err = g.db.Query(`SELECT key, value FROM node_metadata WHERE node_id = ?`, nodeID)
	if err == nil {
		defer rows.Close()
		metadata := make(map[string]string)
		for rows.Next() {
			var key, value string
			rows.Scan(&key, &value)
			metadata[key] = value
		}
		if len(metadata) > 0 {
			result["custom_metadata"] = metadata
		}
	}

	return result, nil
}

// UpdateNodeComment updates or creates comment for a node
func (g *CodeGraph) UpdateNodeComment(nodeID, comment string) error {
	query := `
	INSERT INTO node_comments (node_id, comment, updated_at)
	VALUES (?, ?, CURRENT_TIMESTAMP)
	ON CONFLICT(node_id) DO UPDATE SET
		comment = excluded.comment,
		updated_at = CURRENT_TIMESTAMP
	`
	_, err := g.db.Exec(query, nodeID, comment)
	return err
}

// AddNodeTag adds a tag to a node
func (g *CodeGraph) AddNodeTag(nodeID, tag string) error {
	query := `
	INSERT OR IGNORE INTO node_tags (node_id, tag)
	VALUES (?, ?)
	`
	_, err := g.db.Exec(query, nodeID, strings.TrimSpace(tag))
	return err
}

// RemoveNodeTag removes a tag from a node
func (g *CodeGraph) RemoveNodeTag(nodeID, tag string) error {
	query := `DELETE FROM node_tags WHERE node_id = ? AND tag = ?`
	_, err := g.db.Exec(query, nodeID, strings.TrimSpace(tag))
	return err
}

// SetNodeMetadata sets custom key-value metadata on a node
func (g *CodeGraph) SetNodeMetadata(nodeID, key, value string) error {
	query := `
	INSERT INTO node_metadata (node_id, key, value, updated_at)
	VALUES (?, ?, ?, CURRENT_TIMESTAMP)
	ON CONFLICT(node_id, key) DO UPDATE SET
		value = excluded.value,
		updated_at = CURRENT_TIMESTAMP
	`
	_, err := g.db.Exec(query, nodeID, strings.TrimSpace(key), value)
	return err
}

// RemoveNodeMetadata removes a metadata key from a node
func (g *CodeGraph) RemoveNodeMetadata(nodeID, key string) error {
	query := `DELETE FROM node_metadata WHERE node_id = ? AND key = ?`
	_, err := g.db.Exec(query, nodeID, strings.TrimSpace(key))
	return err
}

// GetNodeWithAnnotations returns a node with all its annotations
func (g *CodeGraph) GetNodeWithAnnotations(nodeID string) (*models.GraphNode, map[string]interface{}, error) {
	node, err := g.GetNode(nodeID)
	if err != nil {
		return nil, nil, err
	}
	if node == nil {
		return nil, nil, fmt.Errorf("node not found")
	}

	annotations, err := g.GetNodeAnnotations(nodeID)
	if err != nil {
		return node, make(map[string]interface{}), nil // Return node even if annotations fail
	}

	// Populate node fields from annotations
	if comments, ok := annotations["comments"].(string); ok {
		node.Comments = comments
	}
	if tags, ok := annotations["tags"].([]string); ok {
		node.Tags = tags
	}
	if metadata, ok := annotations["custom_metadata"].(map[string]string); ok {
		node.CustomMetadata = metadata
	}
	node.UpdatedAt = time.Now().Format(time.RFC3339)

	return node, annotations, nil
}
```

Add this import at the top:

```go
package graph

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ruffini/prism/internal/models"
)
```

- [ ] **Step 2: Build to verify no syntax errors**

```bash
cd C:\Users\rebel\GolandProjects\TokenCompressorUI
go build -o prism . 2>&1
```

Expected: No errors

- [ ] **Step 3: Create graph/annotations_test.go**

```go
package graph

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/ruffini/prism/db"
)

func TestAnnotations(t *testing.T) {
	// Create in-memory test database
	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal("Failed to create test database:", err)
	}
	defer database.Close()

	// Initialize schema
	if err := db.InitDB(":memory:"); err != nil {
		t.Fatal("Failed to init DB:", err)
	}

	// Create a test graph
	graph := NewGraph(database)

	// Insert a test node
	nodeID := "test_func"
	graph.db.Exec(`
		INSERT INTO nodes (id, name, type, file, line, end_line)
		VALUES (?, 'test_func', 'function', 'test.py', 1, 10)
	`, nodeID)

	// Test UpdateNodeComment
	comment := "This is a test comment"
	if err := graph.UpdateNodeComment(nodeID, comment); err != nil {
		t.Errorf("UpdateNodeComment failed: %v", err)
	}

	// Test AddNodeTag
	if err := graph.AddNodeTag(nodeID, "important"); err != nil {
		t.Errorf("AddNodeTag failed: %v", err)
	}
	if err := graph.AddNodeTag(nodeID, "reviewed"); err != nil {
		t.Errorf("AddNodeTag failed: %v", err)
	}

	// Test SetNodeMetadata
	if err := graph.SetNodeMetadata(nodeID, "owner", "john"); err != nil {
		t.Errorf("SetNodeMetadata failed: %v", err)
	}

	// Test GetNodeAnnotations
	annotations, err := graph.GetNodeAnnotations(nodeID)
	if err != nil {
		t.Errorf("GetNodeAnnotations failed: %v", err)
	}

	if annotations["comments"] != comment {
		t.Errorf("Expected comment %q, got %q", comment, annotations["comments"])
	}

	if tags, ok := annotations["tags"].([]string); !ok || len(tags) != 2 {
		t.Errorf("Expected 2 tags, got %v", annotations["tags"])
	}

	if metadata, ok := annotations["custom_metadata"].(map[string]string); !ok || metadata["owner"] != "john" {
		t.Errorf("Expected metadata owner=john, got %v", annotations["custom_metadata"])
	}

	// Test RemoveNodeTag
	if err := graph.RemoveNodeTag(nodeID, "important"); err != nil {
		t.Errorf("RemoveNodeTag failed: %v", err)
	}

	annotations, _ = graph.GetNodeAnnotations(nodeID)
	tags := annotations["tags"].([]string)
	if len(tags) != 1 || tags[0] != "reviewed" {
		t.Errorf("Expected only 'reviewed' tag, got %v", tags)
	}

	t.Logf("✅ All annotation tests passed")
}
```

- [ ] **Step 4: Run tests**

```bash
cd C:\Users\rebel\GolandProjects\TokenCompressorUI
go test ./graph -v -run TestAnnotations
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add graph/annotations.go graph/annotations_test.go
git commit -m "feat: add annotation query and update methods"
```

---

### Task 4: Create API Handlers for Node Updates

**Files:**
- Create: `api/annotations.go` (handler functions)
- Modify: `api/handlers.go` (register routes)

- [ ] **Step 1: Create api/annotations.go**

```go
package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/ruffini/prism/internal/models"
)

// UpdateNodeRequest is the request body for updating node annotations
type UpdateNodeRequest struct {
	Comments       string            `json:"comments"`
	Tags           []string          `json:"tags"`
	CustomMetadata map[string]string `json:"custom_metadata"`
}

// NodeFullResponse includes node data with annotations
type NodeFullResponse struct {
	Node        *models.GraphNode  `json:"node"`
	Annotations map[string]interface{} `json:"annotations"`
}

// handleUpdateNode updates node annotations (PATCH /api/node/:id)
func (a *APIServer) handleUpdateNode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	nodeID := r.URL.Query().Get("id")
	if nodeID == "" {
		http.Error(w, "Missing node id parameter", http.StatusBadRequest)
		return
	}

	var req UpdateNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	log.Printf("📝 Updating node annotations: %s", nodeID)

	// Update comment
	if req.Comments != "" {
		if err := a.graph.UpdateNodeComment(nodeID, req.Comments); err != nil {
			log.Printf("❌ Failed to update comment: %v", err)
			http.Error(w, fmt.Sprintf("Failed to update comment: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Update tags - first clear existing
	a.graph.db.Exec(`DELETE FROM node_tags WHERE node_id = ?`, nodeID)
	for _, tag := range req.Tags {
		if tag != "" {
			if err := a.graph.AddNodeTag(nodeID, tag); err != nil {
				log.Printf("❌ Failed to add tag: %v", err)
				http.Error(w, fmt.Sprintf("Failed to add tag: %v", err), http.StatusInternalServerError)
				return
			}
		}
	}

	// Update metadata - first clear existing
	a.graph.db.Exec(`DELETE FROM node_metadata WHERE node_id = ?`, nodeID)
	for key, value := range req.CustomMetadata {
		if key != "" {
			if err := a.graph.SetNodeMetadata(nodeID, key, value); err != nil {
				log.Printf("❌ Failed to set metadata: %v", err)
				http.Error(w, fmt.Sprintf("Failed to set metadata: %v", err), http.StatusInternalServerError)
				return
			}
		}
	}

	log.Printf("✅ Updated node: %s", nodeID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"nodeId": nodeID,
	})
}

// handleGetNodeFull returns node with annotations (GET /api/node/full?id=...)
func (a *APIServer) handleGetNodeFull(w http.ResponseWriter, r *http.Request) {
	nodeID := r.URL.Query().Get("id")
	if nodeID == "" {
		http.Error(w, "Missing node id parameter", http.StatusBadRequest)
		return
	}

	log.Printf("📄 Getting node with annotations: %s", nodeID)

	node, annotations, err := a.graph.GetNodeWithAnnotations(nodeID)
	if err != nil {
		log.Printf("❌ Error getting node: %v", err)
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}

	if node == nil {
		http.Error(w, "Node not found", http.StatusNotFound)
		return
	}

	log.Printf("✅ Found node with %d annotations", len(annotations))

	response := NodeFullResponse{
		Node:        node,
		Annotations: annotations,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
```

- [ ] **Step 2: Register routes in handlers.go**

Open `api/handlers.go`, find `RegisterRoutes` function, and add these routes:

```go
// In RegisterRoutes function, add:
mux.HandleFunc("/api/node/update", a.corsMiddleware(a.handleUpdateNode))
mux.HandleFunc("/api/node/full", a.corsMiddleware(a.handleGetNodeFull))
```

Also add the import at top of handlers.go if not present:

```go
import (
	"database/sql"
	// ... existing imports
)
```

Then add this field to APIServer struct:

```go
type APIServer struct {
	graph  *graph.CodeGraph
	vector *vector.VectorStore
	ws     *WSServer
	db     *sql.DB  // ADD THIS
}
```

Update `NewAPIServer`:

```go
func NewAPIServer(g *graph.CodeGraph, v *vector.VectorStore, db *sql.DB) *APIServer {
	return &APIServer{
		graph:  g,
		vector: v,
		ws:     NewWSServer(g),
		db:     db,
	}
}
```

- [ ] **Step 3: Update main.go to pass db to APIServer**

In `main.go`, find `startMCPServer` function and change:

```go
// OLD:
apiServer := api.NewAPIServer(codeGraph, vectorStore)

// NEW:
apiServer := api.NewAPIServer(codeGraph, vectorStore, database)
```

- [ ] **Step 4: Build and test**

```bash
cd C:\Users\rebel\GolandProjects\TokenCompressorUI
go build -o prism . 2>&1
```

Expected: No errors

- [ ] **Step 5: Manual API test**

```bash
# Start server in one terminal
cd C:\Users\rebel\G1-Practica
prism serve

# In another terminal, test update
curl -X PATCH http://localhost:8080/api/node/update?id=CombatService \
  -H "Content-Type: application/json" \
  -d '{
    "comments": "Main combat orchestrator",
    "tags": ["critical", "core"],
    "custom_metadata": {"owner": "john", "status": "stable"}
  }'

# Test get full
curl http://localhost:8080/api/node/full?id=CombatService
```

Expected: JSON response with node + annotations

- [ ] **Step 6: Commit**

```bash
git add api/annotations.go api/handlers.go main.go
git commit -m "feat: add API endpoints for node annotation updates"
```

---

### Task 5: Create Tree Node Component (React)

**Files:**
- Create: `frontend/src/components/GraphEditor/TreeNode.tsx`
- Create: `frontend/src/components/GraphEditor/TagBadge.tsx`

- [ ] **Step 1: Create TagBadge component**

Create `frontend/src/components/GraphEditor/TagBadge.tsx`:

```tsx
import React from 'react';

interface TagBadgeProps {
  tag: string;
  onRemove?: (tag: string) => void;
  editable?: boolean;
}

export const TagBadge: React.FC<TagBadgeProps> = ({ tag, onRemove, editable = false }) => {
  return (
    <div className="inline-flex items-center gap-1 px-2 py-1 bg-blue-100 text-blue-800 rounded-full text-sm">
      <span>{tag}</span>
      {editable && onRemove && (
        <button
          onClick={() => onRemove(tag)}
          className="ml-1 text-blue-600 hover:text-blue-800 font-bold"
        >
          ×
        </button>
      )}
    </div>
  );
};
```

- [ ] **Step 2: Create TreeNode component**

Create `frontend/src/components/GraphEditor/TreeNode.tsx`:

```tsx
import React, { useState } from 'react';
import { TagBadge } from './TagBadge';
import { ChevronRight, ChevronDown } from 'lucide-react';

interface TreeNodeData {
  id: string;
  name: string;
  type: string;
  file: string;
  line: number;
  comments?: string;
  tags?: string[];
  custom_metadata?: Record<string, string>;
  children?: TreeNodeData[];
}

interface TreeNodeProps {
  node: TreeNodeData;
  onSelectNode: (node: TreeNodeData) => void;
  onUpdateNode: (nodeId: string, data: any) => void;
  selectedNodeId?: string;
  isEditing?: boolean;
  editingNodeId?: string;
}

export const TreeNode: React.FC<TreeNodeProps> = ({
  node,
  onSelectNode,
  onUpdateNode,
  selectedNodeId,
  isEditing = false,
  editingNodeId,
}) => {
  const [isExpanded, setIsExpanded] = useState(false);
  const [showEditPanel, setShowEditPanel] = useState(false);
  const hasChildren = node.children && node.children.length > 0;
  const isSelected = selectedNodeId === node.id;
  const isCurrentlyEditing = editingNodeId === node.id;

  const toggleExpand = () => setIsExpanded(!isExpanded);

  const handleNodeClick = () => {
    onSelectNode(node);
    setShowEditPanel(true);
  };

  return (
    <div className="select-none">
      <div
        className={`flex items-center gap-2 px-2 py-1 rounded cursor-pointer transition-colors ${
          isSelected
            ? 'bg-blue-200 text-blue-900'
            : 'hover:bg-gray-100'
        }`}
        onClick={handleNodeClick}
      >
        {/* Expand/Collapse Button */}
        {hasChildren && (
          <button
            onClick={(e) => {
              e.stopPropagation();
              toggleExpand();
            }}
            className="p-0 w-5 h-5 flex items-center justify-center"
          >
            {isExpanded ? (
              <ChevronDown size={16} />
            ) : (
              <ChevronRight size={16} />
            )}
          </button>
        )}
        {!hasChildren && <div className="w-5" />}

        {/* Node Content */}
        <div className="flex-1">
          <div className="flex items-center gap-2">
            <span className="font-semibold">{node.name}</span>
            <span className="text-xs bg-gray-200 px-2 py-0.5 rounded">
              {node.type}
            </span>
            {node.tags && node.tags.length > 0 && (
              <div className="flex gap-1">
                {node.tags.slice(0, 2).map((tag) => (
                  <TagBadge key={tag} tag={tag} />
                ))}
                {node.tags.length > 2 && (
                  <span className="text-xs text-gray-500">
                    +{node.tags.length - 2}
                  </span>
                )}
              </div>
            )}
          </div>
          <div className="text-xs text-gray-500">
            {node.file}:{node.line}
          </div>
          {node.comments && (
            <div className="text-xs text-gray-600 mt-1 italic">
              "{node.comments.substring(0, 50)}..."
            </div>
          )}
        </div>
      </div>

      {/* Children */}
      {hasChildren && isExpanded && (
        <div className="ml-4 border-l border-gray-200">
          {node.children!.map((child) => (
            <TreeNode
              key={child.id}
              node={child}
              onSelectNode={onSelectNode}
              onUpdateNode={onUpdateNode}
              selectedNodeId={selectedNodeId}
              editingNodeId={editingNodeId}
            />
          ))}
        </div>
      )}
    </div>
  );
};
```

- [ ] **Step 3: Build and check for TypeScript errors**

```bash
cd C:\Users\rebel\GolandProjects\TokenCompressorUI\frontend
npm run build 2>&1 | grep -i "error" | head -20
```

Expected: No errors related to these files

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/GraphEditor/TreeNode.tsx \
        frontend/src/components/GraphEditor/TagBadge.tsx
git commit -m "feat: create tree node and tag badge components"
```

---

### Task 6: Create Edit Node Panel (React)

**Files:**
- Create: `frontend/src/components/GraphEditor/EditNodePanel.tsx`

- [ ] **Step 1: Create EditNodePanel component**

Create `frontend/src/components/GraphEditor/EditNodePanel.tsx`:

```tsx
import React, { useState, useEffect } from 'react';
import { TagBadge } from './TagBadge';
import { X } from 'lucide-react';

interface NodeData {
  id: string;
  name: string;
  type: string;
  file: string;
  line: number;
  signature?: string;
  comments?: string;
  tags?: string[];
  custom_metadata?: Record<string, string>;
}

interface EditNodePanelProps {
  node: NodeData | null;
  onUpdate: (nodeId: string, data: any) => void;
  onClose: () => void;
  isSaving?: boolean;
}

export const EditNodePanel: React.FC<EditNodePanelProps> = ({
  node,
  onUpdate,
  onClose,
  isSaving = false,
}) => {
  const [comments, setComments] = useState('');
  const [tags, setTags] = useState<string[]>([]);
  const [newTag, setNewTag] = useState('');
  const [metadata, setMetadata] = useState<Record<string, string>>({});
  const [metaKey, setMetaKey] = useState('');
  const [metaValue, setMetaValue] = useState('');

  useEffect(() => {
    if (node) {
      setComments(node.comments || '');
      setTags(node.tags || []);
      setMetadata(node.custom_metadata || {});
      setNewTag('');
      setMetaKey('');
      setMetaValue('');
    }
  }, [node]);

  const handleSave = () => {
    if (!node) return;

    onUpdate(node.id, {
      comments,
      tags,
      custom_metadata: metadata,
    });
  };

  const handleAddTag = () => {
    if (newTag.trim() && !tags.includes(newTag.trim())) {
      setTags([...tags, newTag.trim()]);
      setNewTag('');
    }
  };

  const handleRemoveTag = (tagToRemove: string) => {
    setTags(tags.filter((t) => t !== tagToRemove));
  };

  const handleAddMetadata = () => {
    if (metaKey.trim() && metaValue.trim()) {
      setMetadata({
        ...metadata,
        [metaKey.trim()]: metaValue.trim(),
      });
      setMetaKey('');
      setMetaValue('');
    }
  };

  const handleRemoveMetadata = (key: string) => {
    const newMeta = { ...metadata };
    delete newMeta[key];
    setMetadata(newMeta);
  };

  if (!node) {
    return null;
  }

  return (
    <div className="w-full bg-white border-l border-gray-200 overflow-y-auto">
      {/* Header */}
      <div className="sticky top-0 bg-gray-50 border-b border-gray-200 p-4 flex justify-between items-center">
        <div>
          <h2 className="font-bold text-lg">{node.name}</h2>
          <p className="text-sm text-gray-600">{node.file}:{node.line}</p>
        </div>
        <button
          onClick={onClose}
          className="p-1 hover:bg-gray-200 rounded"
        >
          <X size={20} />
        </button>
      </div>

      {/* Content */}
      <div className="p-4 space-y-4">
        {/* Signature */}
        {node.signature && (
          <div>
            <h3 className="font-semibold text-sm mb-2">Signature</h3>
            <pre className="bg-gray-100 p-2 rounded text-xs overflow-x-auto">
              {node.signature}
            </pre>
          </div>
        )}

        {/* Comments */}
        <div>
          <h3 className="font-semibold text-sm mb-2">Comments</h3>
          <textarea
            value={comments}
            onChange={(e) => setComments(e.target.value)}
            placeholder="Add comments about this node..."
            className="w-full h-20 p-2 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>

        {/* Tags */}
        <div>
          <h3 className="font-semibold text-sm mb-2">Tags</h3>
          <div className="flex gap-2 mb-2">
            <input
              type="text"
              value={newTag}
              onChange={(e) => setNewTag(e.target.value)}
              onKeyPress={(e) => {
                if (e.key === 'Enter') {
                  handleAddTag();
                }
              }}
              placeholder="Add tag..."
              className="flex-1 px-2 py-1 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <button
              onClick={handleAddTag}
              className="px-3 py-1 bg-blue-500 text-white rounded text-sm hover:bg-blue-600"
            >
              Add
            </button>
          </div>
          <div className="flex flex-wrap gap-2">
            {tags.map((tag) => (
              <TagBadge
                key={tag}
                tag={tag}
                onRemove={handleRemoveTag}
                editable={true}
              />
            ))}
          </div>
        </div>

        {/* Custom Metadata */}
        <div>
          <h3 className="font-semibold text-sm mb-2">Custom Metadata</h3>
          <div className="space-y-2 mb-2">
            <div className="flex gap-2">
              <input
                type="text"
                value={metaKey}
                onChange={(e) => setMetaKey(e.target.value)}
                placeholder="Key..."
                className="flex-1 px-2 py-1 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
              <input
                type="text"
                value={metaValue}
                onChange={(e) => setMetaValue(e.target.value)}
                onKeyPress={(e) => {
                  if (e.key === 'Enter') {
                    handleAddMetadata();
                  }
                }}
                placeholder="Value..."
                className="flex-1 px-2 py-1 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
              <button
                onClick={handleAddMetadata}
                className="px-3 py-1 bg-blue-500 text-white rounded text-sm hover:bg-blue-600"
              >
                Add
              </button>
            </div>
          </div>
          <div className="space-y-1">
            {Object.entries(metadata).map(([key, value]) => (
              <div
                key={key}
                className="flex justify-between items-center bg-gray-50 p-2 rounded text-sm"
              >
                <div>
                  <span className="font-semibold">{key}:</span>
                  <span className="ml-2 text-gray-700">{value}</span>
                </div>
                <button
                  onClick={() => handleRemoveMetadata(key)}
                  className="text-red-600 hover:text-red-800 font-bold"
                >
                  ×
                </button>
              </div>
            ))}
          </div>
        </div>

        {/* Save Button */}
        <button
          onClick={handleSave}
          disabled={isSaving}
          className="w-full px-4 py-2 bg-green-500 text-white rounded hover:bg-green-600 disabled:bg-gray-400"
        >
          {isSaving ? 'Saving...' : 'Save Changes'}
        </button>
      </div>
    </div>
  );
};
```

- [ ] **Step 2: Add lucide-react dependency if needed**

Check if lucide-react is in package.json:

```bash
cd C:\Users\rebel\GolandProjects\TokenCompressorUI\frontend
grep lucide package.json
```

If not present, add it:

```bash
npm install lucide-react
```

- [ ] **Step 3: Build to check for errors**

```bash
cd C:\Users\rebel\GolandProjects\TokenCompressorUI\frontend
npm run build 2>&1 | grep -i "error" | head -20
```

Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/GraphEditor/EditNodePanel.tsx \
        frontend/package.json
git commit -m "feat: create edit node panel with inline annotation editing"
```

---

### Task 7: Create Code Tree View Main Component

**Files:**
- Create: `frontend/src/components/GraphEditor/CodeTreeView.tsx`

- [ ] **Step 1: Create CodeTreeView component**

Create `frontend/src/components/GraphEditor/CodeTreeView.tsx`:

```tsx
import React, { useState, useEffect } from 'react';
import { TreeNode } from './TreeNode';
import { EditNodePanel } from './EditNodePanel';
import { Search } from 'lucide-react';

interface TreeNodeData {
  id: string;
  name: string;
  type: string;
  file: string;
  line: number;
  signature?: string;
  comments?: string;
  tags?: string[];
  custom_metadata?: Record<string, string>;
  children?: TreeNodeData[];
}

interface CodeTreeViewProps {
  apiBaseUrl?: string;
}

export const CodeTreeView: React.FC<CodeTreeViewProps> = ({
  apiBaseUrl = 'http://localhost:8080',
}) => {
  const [treeData, setTreeData] = useState<TreeNodeData[]>([]);
  const [selectedNode, setSelectedNode] = useState<TreeNodeData | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Load files on mount
  useEffect(() => {
    loadFiles();
  }, []);

  const loadFiles = async () => {
    try {
      setIsLoading(true);
      const response = await fetch(`${apiBaseUrl}/api/files`);
      if (!response.ok) throw new Error('Failed to load files');

      const data = await response.json();
      const files: TreeNodeData[] = (data.files || []).map((file: string) => ({
        id: file,
        name: file.split('\\').pop() || file,
        type: 'file',
        file,
        line: 0,
        children: [],
      }));

      setTreeData(files);
      setError(null);
    } catch (err) {
      setError(`Error loading files: ${err}`);
      console.error(err);
    } finally {
      setIsLoading(false);
    }
  };

  const handleSelectNode = async (node: TreeNodeData) => {
    try {
      const response = await fetch(
        `${apiBaseUrl}/api/node/full?id=${encodeURIComponent(node.id)}`
      );
      if (!response.ok) throw new Error('Failed to load node');

      const data = await response.json();
      setSelectedNode(data.node);
    } catch (err) {
      console.error('Error loading node:', err);
      setSelectedNode(node);
    }
  };

  const handleUpdateNode = async (nodeId: string, data: any) => {
    try {
      setIsSaving(true);
      const response = await fetch(`${apiBaseUrl}/api/node/update?id=${encodeURIComponent(nodeId)}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      });

      if (!response.ok) throw new Error('Failed to update node');

      // Reload selected node to show updated data
      if (selectedNode) {
        handleSelectNode(selectedNode);
      }

      setError(null);
    } catch (err) {
      setError(`Error updating node: ${err}`);
      console.error(err);
    } finally {
      setIsSaving(false);
    }
  };

  const filteredTree = treeData.filter((node) =>
    node.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    node.file.toLowerCase().includes(searchQuery.toLowerCase())
  );

  return (
    <div className="flex h-full bg-white">
      {/* Left Sidebar - Tree */}
      <div className="w-1/3 border-r border-gray-200 flex flex-col">
        {/* Search Bar */}
        <div className="p-4 border-b border-gray-200">
          <div className="relative">
            <Search size={16} className="absolute left-2 top-2.5 text-gray-400" />
            <input
              type="text"
              placeholder="Search files..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-8 pr-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
            />
          </div>
        </div>

        {/* Error Message */}
        {error && (
          <div className="p-2 m-2 bg-red-100 text-red-800 rounded text-sm">
            {error}
          </div>
        )}

        {/* Tree */}
        <div className="flex-1 overflow-y-auto p-2">
          {isLoading ? (
            <div className="text-gray-500 text-sm p-4">Loading...</div>
          ) : filteredTree.length === 0 ? (
            <div className="text-gray-500 text-sm p-4">No files found</div>
          ) : (
            filteredTree.map((node) => (
              <TreeNode
                key={node.id}
                node={node}
                onSelectNode={handleSelectNode}
                onUpdateNode={handleUpdateNode}
                selectedNodeId={selectedNode?.id}
              />
            ))
          )}
        </div>
      </div>

      {/* Right Sidebar - Edit Panel */}
      <div className="w-2/3">
        <EditNodePanel
          node={selectedNode}
          onUpdate={handleUpdateNode}
          onClose={() => setSelectedNode(null)}
          isSaving={isSaving}
        />
      </div>
    </div>
  );
};
```

- [ ] **Step 2: Build and check for errors**

```bash
cd C:\Users\rebel\GolandProjects\TokenCompressorUI\frontend
npm run build 2>&1 | grep -i "error" | head -20
```

Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/GraphEditor/CodeTreeView.tsx
git commit -m "feat: create main code tree view component"
```

---

### Task 8: Create Graph Editor Page and Add to App Layout

**Files:**
- Create: `frontend/src/pages/GraphEditorPage.tsx`
- Modify: `frontend/src/App.tsx` - add route for graph editor

- [ ] **Step 1: Create GraphEditorPage**

Create `frontend/src/pages/GraphEditorPage.tsx`:

```tsx
import React from 'react';
import { CodeTreeView } from '../components/GraphEditor/CodeTreeView';

export const GraphEditorPage: React.FC = () => {
  return (
    <div className="h-screen flex flex-col bg-white">
      {/* Header */}
      <div className="bg-gradient-to-r from-blue-600 to-blue-800 text-white p-4">
        <h1 className="text-2xl font-bold">Graph Editor</h1>
        <p className="text-blue-100 text-sm">Annotate and manage your code graph</p>
      </div>

      {/* Editor */}
      <div className="flex-1">
        <CodeTreeView apiBaseUrl="http://localhost:8080" />
      </div>
    </div>
  );
};
```

- [ ] **Step 2: Update App.tsx to include GraphEditorPage**

Open `frontend/src/App.tsx` and update it:

```tsx
import React from 'react';
import { GraphEditorPage } from './pages/GraphEditorPage';
import './styles/globals.css';

function App() {
  return (
    <div className="h-screen">
      <GraphEditorPage />
    </div>
  );
}

export default App;
```

- [ ] **Step 3: Build and test locally**

```bash
cd C:\Users\rebel\GolandProjects\TokenCompressorUI\frontend
npm run build 2>&1 | tail -20
```

Expected: Build succeeds

- [ ] **Step 4: Commit**

```bash
git add frontend/src/pages/GraphEditorPage.tsx frontend/src/App.tsx
git commit -m "feat: add graph editor page and integrate into app"
```

---

### Task 9: Enhance MCP Tools to Include Annotations

**Files:**
- Modify: `mcp/tools.go` - update get_file_smart and search_context handlers

- [ ] **Step 1: Enhance get_file_smart to include annotations**

Open `mcp/tools.go`, find `handleGetFileSmart` function, and after getting the node, add annotation retrieval:

```go
// After getting the node, add this (before formatting result):
// Get annotations
annotations, _ := m.graph.GetNodeAnnotations(node.ID)
if len(annotations) > 0 {
	node.Comments, _ = annotations["comments"].(string)
	if tags, ok := annotations["tags"].([]string); ok {
		node.Tags = tags
	}
	if metadata, ok := annotations["custom_metadata"].(map[string]string); ok {
		node.CustomMetadata = metadata
	}
}
```

Then update the result formatting to include annotations in the markdown output.

- [ ] **Step 2: Update result formatting**

In the same function, update the markdown generation to include:

```go
// After metadata section, add annotations section:
if len(node.Comments) > 0 || len(node.Tags) > 0 || len(node.CustomMetadata) > 0 {
	result += "**Annotations:**\n"
	if node.Comments != "" {
		result += fmt.Sprintf("- **Comment:** %s\n", node.Comments)
	}
	if len(node.Tags) > 0 {
		result += fmt.Sprintf("- **Tags:** %v\n", node.Tags)
	}
	if len(node.CustomMetadata) > 0 {
		result += "- **Metadata:**\n"
		for key, value := range node.CustomMetadata {
			result += fmt.Sprintf("  - %s: %s\n", key, value)
		}
	}
	result += "\n"
}
```

- [ ] **Step 3: Enhance search_context similarly**

In `handleSearchContext`, when returning results, also include annotations for each result node.

- [ ] **Step 4: Build and test**

```bash
cd C:\Users\rebel\GolandProjects\TokenCompressorUI
go build -o prism . 2>&1
```

Expected: No errors

- [ ] **Step 5: Test MCP returns annotations**

Start server:

```bash
cd C:\Users\rebel\G1-Practica
prism serve &
sleep 2
```

In Claude Code terminal:

```
@get_file_smart backend/app/services/combat_service.py CombatService
```

Expected: Response includes "Annotations:" section if any exist

- [ ] **Step 6: Commit**

```bash
git add mcp/tools.go
git commit -m "feat: enhance MCP tools to include node annotations"
```

---

### Task 10: Create Integration Tests

**Files:**
- Create: `tests/graph_editor_test.go`

- [ ] **Step 1: Create integration test**

Create `tests/graph_editor_test.go`:

```go
package tests

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/ruffini/prism/api"
	"github.com/ruffini/prism/db"
	"github.com/ruffini/prism/graph"
)

func TestGraphEditorIntegration(t *testing.T) {
	// Create test database
	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer database.Close()

	// Initialize schema
	if err := db.InitDB(":memory:"); err != nil {
		t.Fatalf("Failed to init schema: %v", err)
	}

	// Create graph
	codeGraph := graph.NewGraph(database)

	// Insert test node
	nodeID := "test_combat_service"
	database.Exec(`
		INSERT INTO nodes (id, name, type, file, line, end_line, signature)
		VALUES (?, 'CombatService', 'class', 'combat.py', 10, 50, 'class CombatService:')
	`, nodeID)

	// Create API server
	apiServer := api.NewAPIServer(codeGraph, nil, database)

	t.Run("Update node annotations", func(t *testing.T) {
		updateData := api.AnnotationUpdate{
			Comments: "Main combat handler",
			Tags:     []string{"critical", "core"},
			CustomMetadata: map[string]string{
				"owner":  "john",
				"status": "stable",
			},
		}

		body, _ := json.Marshal(updateData)
		req := httptest.NewRequest("PATCH", "/api/node/update?id="+nodeID, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		apiServer.handleUpdateNode(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Verify in database
		annotations, _ := codeGraph.GetNodeAnnotations(nodeID)
		if comment, ok := annotations["comments"].(string); !ok || comment != "Main combat handler" {
			t.Errorf("Comment not saved correctly: %v", annotations["comments"])
		}

		if tags, ok := annotations["tags"].([]string); !ok || len(tags) != 2 {
			t.Errorf("Tags not saved correctly: %v", annotations["tags"])
		}

		t.Logf("✅ Annotation update test passed")
	})

	t.Run("Get node with annotations", func(t *testing.T) {
		// First update node with annotations
		codeGraph.UpdateNodeComment(nodeID, "Test comment")
		codeGraph.AddNodeTag(nodeID, "important")
		codeGraph.SetNodeMetadata(nodeID, "reviewer", "alice")

		// Get node
		req := httptest.NewRequest("GET", "/api/node/full?id="+nodeID, nil)
		w := httptest.NewRecorder()

		apiServer.handleGetNodeFull(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
			return
		}

		var response api.NodeFullResponse
		json.NewDecoder(w.Body).Decode(&response)

		if response.Node.Comments != "Test comment" {
			t.Errorf("Expected comment 'Test comment', got '%s'", response.Node.Comments)
		}

		if len(response.Node.Tags) != 1 || response.Node.Tags[0] != "important" {
			t.Errorf("Expected tag 'important', got %v", response.Node.Tags)
		}

		if response.Node.CustomMetadata["reviewer"] != "alice" {
			t.Errorf("Expected metadata reviewer=alice, got %v", response.Node.CustomMetadata)
		}

		t.Logf("✅ Get node with annotations test passed")
	})
}
```

- [ ] **Step 2: Run tests**

```bash
cd C:\Users\rebel\GolandProjects\TokenCompressorUI
go test ./tests -v -run TestGraphEditorIntegration
```

Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add tests/graph_editor_test.go
git commit -m "test: add graph editor integration tests"
```

---

### Task 11: Create Documentation for Graph Editor

**Files:**
- Create: `docs/GRAPH_EDITOR.md`

- [ ] **Step 1: Create documentation**

Create `docs/GRAPH_EDITOR.md`:

```markdown
# Graph Editor

The Graph Editor is an interactive interface for annotating and managing your code graph.

## Features

- **Tree View**: Browse indexed code organized by files and types
- **Inline Editing**: Add comments, tags, and custom metadata to any code element
- **Autosave**: Changes automatically sync to the backend
- **Search**: Quickly find files or code elements
- **Visual Tags**: See tags as colored badges on nodes
- **Metadata**: Add custom key-value pairs for domain-specific information

## Usage

### Start the Editor

1. Index your code:
```bash
prism index -repo /path/to/project
prism embed
```

2. Start the server:
```bash
prism serve
```

3. Open the frontend:
```bash
cd frontend
npm run dev
```

4. Navigate to `http://localhost:5173`

### Adding Annotations

1. Click on any node in the tree
2. The right panel shows the node details
3. Edit comments, add tags, or metadata
4. Click "Save Changes" (or autosave will kick in)

### Comments

- Add detailed notes about what a function does
- Supports multi-line text
- Visible in MCP tools (returned with context to Claude Code)

### Tags

- Useful for categorizing code
- Examples: "deprecated", "critical", "todo", "refactor"
- Multiple tags per node supported
- Displayed as colored badges

### Custom Metadata

- Key-value pairs for domain-specific information
- Examples: owner, status, complexity, last_review_date
- Searchable and filterable

## Integration with MCP

When Claude Code asks for context about a node, annotations are included:

```
@get_file_smart file.py FunctionName

Returns:
## FunctionName (function)

**File:** file.py (lines 10-50)

**Signature:**
def FunctionName():

**Annotations:**
- **Comment:** Main authentication handler
- **Tags:** [critical, reviewed]
- **Metadata:**
  - owner: john
  - status: stable
```

This enhances Claude Code's understanding without needing full file reads.

## Best Practices

1. **Comments**: Use for domain knowledge not in code (why, not what)
2. **Tags**: Keep consistent (use lowercase, use hyphens for multi-word)
3. **Metadata**: Use for tracking (owner, last_review, complexity)
4. **Regular Updates**: Annotate as you refactor or review code

## API Endpoints

### PATCH /api/node/update
Update node annotations

```bash
curl -X PATCH http://localhost:8080/api/node/update?id=function_id \
  -H "Content-Type: application/json" \
  -d '{
    "comments": "Main handler",
    "tags": ["critical"],
    "custom_metadata": {"owner": "john"}
  }'
```

### GET /api/node/full
Get node with annotations

```bash
curl http://localhost:8080/api/node/full?id=function_id
```

Returns node with `annotations` object containing comments, tags, and metadata.
```

- [ ] **Step 2: Commit**

```bash
git add docs/GRAPH_EDITOR.md
git commit -m "docs: add graph editor user guide"
```

---

### Task 12: Final Integration and Testing

**Files:**
- Test all components together
- Frontend, Backend, MCP

- [ ] **Step 1: Full system build**

```bash
cd C:\Users\rebel\GolandProjects\TokenCompressorUI
go build -o prism . 2>&1 && echo "✅ Backend built"
cd frontend
npm run build 2>&1 && echo "✅ Frontend built"
```

Expected: Both succeed

- [ ] **Step 2: Start the system**

Terminal 1:

```bash
cd C:\Users\rebel\G1-Practica
prism serve
```

Terminal 2:

```bash
cd C:\Users\rebel\GolandProjects\TokenCompressorUI\frontend
npm run dev
```

Expected: Both start without errors

- [ ] **Step 3: Test frontend UI**

Open `http://localhost:5173` - should see:
- File tree on left
- Edit panel on right
- Search bar working
- Click nodes and edit annotations

- [ ] **Step 4: Test API directly**

```bash
# Get files
curl http://localhost:8080/api/files

# Update a node
curl -X PATCH http://localhost:8080/api/node/update?id=CombatService \
  -H "Content-Type: application/json" \
  -d '{
    "comments": "Strategic combat system",
    "tags": ["important"],
    "custom_metadata": {"owner": "team"}
  }'

# Get with annotations
curl http://localhost:8080/api/node/full?id=CombatService
```

Expected: All return valid JSON

- [ ] **Step 5: Test MCP integration**

In Claude Code terminal:

```
@get_file_smart backend/app/services/tactical_combat_service.py TacticalCombatService
```

Expected: Response includes annotations if any exist

- [ ] **Step 6: Final commit**

```bash
git add -A
git commit -m "feat: complete graph editor system with frontend, API, and MCP integration"
```

---

## Execution Summary

**Recommended approach:** Use superpowers:subagent-driven-development to execute tasks 1-12 sequentially with spec compliance and code quality reviews between each task.

**Total effort:** ~6-8 hours across 12 focused tasks
**Test coverage:** Unit tests (Task 10), integration tests (Task 10), manual UI tests (Task 12)
**Commit frequency:** After every 1-2 steps
