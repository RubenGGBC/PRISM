# Plan 2: Anotaciones estructuradas

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Añadir campos estructurados de anotación (why, status, entry_point, known_bug) a nodos, exponerlos en la API, en la UI y en las respuestas del MCP.

**Architecture:** Nueva migración en `db/migrations.go` crea tabla `node_annotations`. La API extiende el endpoint PATCH existente. El MCP incluye las anotaciones en todas las respuestas de búsqueda. La UI añade los nuevos campos al `EditNodePanel`.

**Tech Stack:** Go (SQLite), React/TypeScript

**Prerequisite:** Plan 1 completado.

---

## Archivos que se tocan

| Acción | Archivo |
|--------|---------|
| Modify | `db/migrations.go` |
| Modify | `internal/models/models.go` |
| Modify | `api/annotations.go` |
| Modify | `graph/builder.go` (o `graph/queries.go`) |
| Modify | `mcp/server.go` |
| Modify | `frontend/src/components/GraphEditor/EditNodePanel.tsx` |

---

### Task 1: Migración de base de datos

**Files:**
- Modify: `db/migrations.go`

- [ ] **Step 1: Añadir migración de node_annotations**

En `db/migrations.go`, añadir la llamada en `RunMigrations` y la función:
```go
func RunMigrations(db *sql.DB) error {
	if err := createCommentsTable(db); err != nil {
		return fmt.Errorf("failed to create comments table: %w", err)
	}
	if err := createTagsTable(db); err != nil {
		return fmt.Errorf("failed to create tags table: %w", err)
	}
	if err := createMetadataTable(db); err != nil {
		return fmt.Errorf("failed to create metadata table: %w", err)
	}
	if err := createAnnotationsTable(db); err != nil {
		return fmt.Errorf("failed to create annotations table: %w", err)
	}
	return nil
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
```

- [ ] **Step 2: Verificar que la migración funciona**

```bash
rm -f code_graph.db
go run . index -repo .
```

Expected: sin errores. La DB se crea correctamente.

```bash
sqlite3 code_graph.db ".tables"
```

Expected: `node_annotations` aparece en la lista.

- [ ] **Step 3: Commit**

```bash
git add db/migrations.go
git commit -m "feat: add node_annotations table migration"
```

---

### Task 2: Actualizar el modelo GraphNode

**Files:**
- Modify: `internal/models/models.go`

- [ ] **Step 1: Añadir campos a GraphNode y AnnotationUpdate**

En `internal/models/models.go`, añadir los nuevos campos a `GraphNode`:
```go
type GraphNode struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Type           string            `json:"type"`
	File           string            `json:"file"`
	Line           int               `json:"line"`
	EndLine        int               `json:"end_line"`
	Signature      string            `json:"signature"`
	Body           string            `json:"body"`
	Comments       string            `json:"comments,omitempty"`
	Tags           []string          `json:"tags,omitempty"`
	CustomMetadata map[string]string `json:"custom_metadata,omitempty"`
	UpdatedAt      string            `json:"updated_at,omitempty"`
	PageRank       float64           `json:"pagerank"`
	BlastRadius    int               `json:"blast_radius"`
	// Structured annotations
	Why        string `json:"why,omitempty"`
	Status     string `json:"status,omitempty"`
	EntryPoint bool   `json:"entry_point,omitempty"`
	KnownBug   string `json:"known_bug,omitempty"`
}
```

Añadir/actualizar `AnnotationUpdate` en el mismo archivo (si existe, extenderlo):
```go
type AnnotationUpdate struct {
	Comments       string            `json:"comments"`
	Tags           []string          `json:"tags"`
	CustomMetadata map[string]string `json:"custom_metadata"`
	Why            string            `json:"why"`
	Status         string            `json:"status"`
	EntryPoint     bool              `json:"entry_point"`
	KnownBug       string            `json:"known_bug"`
}
```

- [ ] **Step 2: Compilar**

```bash
go build ./...
```

Expected: sin errores.

- [ ] **Step 3: Commit**

```bash
git add internal/models/models.go
git commit -m "feat: add structured annotation fields to GraphNode model"
```

---

### Task 3: Métodos de grafo para annotations

**Files:**
- Modify: `graph/builder.go`

- [ ] **Step 1: Añadir métodos para leer y escribir annotations**

Al final de `graph/builder.go`:
```go
// UpsertAnnotations guarda las anotaciones estructuradas de un nodo
func (g *CodeGraph) UpsertAnnotations(nodeID, why, status, knownBug string, entryPoint bool) error {
	query := `
	INSERT INTO node_annotations (node_id, why, status, entry_point, known_bug, updated_at)
	VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	ON CONFLICT(node_id) DO UPDATE SET
		why = excluded.why,
		status = excluded.status,
		entry_point = excluded.entry_point,
		known_bug = excluded.known_bug,
		updated_at = CURRENT_TIMESTAMP
	`
	_, err := g.DB.Exec(query, nodeID, why, status, entryPoint, knownBug)
	return err
}

// GetAnnotations devuelve las anotaciones estructuradas de un nodo
func (g *CodeGraph) GetAnnotations(nodeID string) (why, status, knownBug string, entryPoint bool, err error) {
	row := g.DB.QueryRow(
		`SELECT why, status, entry_point, known_bug FROM node_annotations WHERE node_id = ?`,
		nodeID,
	)
	var w, s, kb sql.NullString
	var ep sql.NullBool
	if err = row.Scan(&w, &s, &ep, &kb); err != nil {
		if err == sql.ErrNoRows {
			return "", "stable", "", false, nil
		}
		return
	}
	return w.String, s.String, kb.String, ep.Bool, nil
}
```

- [ ] **Step 2: Extender GetNodeWithAnnotations para incluir las nuevas anotaciones**

Buscar la función `GetNodeWithAnnotations` en `graph/` (puede estar en `queries.go` o `builder.go`) y añadir la lectura de `node_annotations`. El valor de retorno debe incluir `why`, `status`, `entry_point`, `known_bug` en el nodo.

Localizar el método:
```bash
grep -n "GetNodeWithAnnotations" graph/*.go
```

Añadir la lectura de annotations en esa función, después de cargar comments/tags/metadata:
```go
// Añadir al final de GetNodeWithAnnotations, antes del return:
why, status, knownBug, entryPoint, err := g.GetAnnotations(nodeID)
if err == nil {
    node.Why = why
    node.Status = status
    node.KnownBug = knownBug
    node.EntryPoint = entryPoint
}
```

- [ ] **Step 3: Compilar**

```bash
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add graph/builder.go graph/queries.go
git commit -m "feat: add UpsertAnnotations and GetAnnotations to graph"
```

---

### Task 4: Extender la API

**Files:**
- Modify: `api/annotations.go`

- [ ] **Step 1: Actualizar UpdateNodeRequest y HandleUpdateNode**

En `api/annotations.go`, reemplazar `UpdateNodeRequest` y actualizar `HandleUpdateNode`:
```go
type UpdateNodeRequest struct {
	Comments       string            `json:"comments"`
	Tags           []string          `json:"tags"`
	CustomMetadata map[string]string `json:"custom_metadata"`
	Why            string            `json:"why"`
	Status         string            `json:"status"`
	EntryPoint     bool              `json:"entry_point"`
	KnownBug       string            `json:"known_bug"`
}
```

En `HandleUpdateNode`, después del bloque de metadata, añadir:
```go
// Update structured annotations
if err := a.graph.UpsertAnnotations(nodeID, req.Why, req.Status, req.KnownBug, req.EntryPoint); err != nil {
    log.Printf("❌ Failed to update annotations: %v", err)
    http.Error(w, fmt.Sprintf("Failed to update annotations: %v", err), http.StatusInternalServerError)
    return
}
```

- [ ] **Step 2: Compilar y verificar**

```bash
go build ./...
```

- [ ] **Step 3: Test manual de la API**

Con el servidor corriendo (`go run . serve`):
```bash
curl -X PATCH "http://localhost:8080/api/node/update?id=main.go:main" \
  -H "Content-Type: application/json" \
  -d '{"why":"Entry point principal","status":"stable","entry_point":true}'
```

Expected: `{"status":"ok","nodeId":"main.go:main"}`

```bash
curl "http://localhost:8080/api/node/full?id=main.go:main"
```

Expected: respuesta JSON con campos `why`, `status`, `entry_point`.

- [ ] **Step 4: Commit**

```bash
git add api/annotations.go
git commit -m "feat: extend node update API with structured annotations"
```

---

### Task 5: Anotaciones en respuestas MCP

**Files:**
- Modify: `mcp/server.go`

- [ ] **Step 1: Encontrar donde se formatean los resultados de search_context**

```bash
grep -n "search_context\|formatNode\|Signature\|docstring" mcp/server.go | head -30
```

- [ ] **Step 2: Crear helper formatNodeWithAnnotations en mcp/server.go**

Añadir al final del archivo:
```go
// formatNodeWithAnnotations formatea un nodo incluyendo sus anotaciones humanas
func (m *MCPServer) formatNodeWithAnnotations(nodeID, name, nodeType, file string, line int, signature, docstring string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## %s (%s:%d)\n", name, file, line))
	sb.WriteString(fmt.Sprintf("Type: %s\n", nodeType))

	// Load structured annotations
	why, status, knownBug, entryPoint, err := m.graph.GetAnnotations(nodeID)
	if err == nil {
		if status != "" && status != "stable" {
			sb.WriteString(fmt.Sprintf("Status: %s\n", status))
		}
		if entryPoint {
			sb.WriteString("Entry point: YES\n")
		}
		if why != "" {
			sb.WriteString(fmt.Sprintf("Why: %s\n", why))
		}
		if knownBug != "" {
			sb.WriteString(fmt.Sprintf("Known bug: %s\n", knownBug))
		}
	}

	// Load user comments
	var comment string
	row := m.graph.DB.QueryRow(`SELECT comment FROM node_comments WHERE node_id = ?`, nodeID)
	row.Scan(&comment)
	if comment != "" {
		sb.WriteString(fmt.Sprintf("Notes: %s\n", comment))
	}

	if signature != "" {
		sb.WriteString(fmt.Sprintf("Signature: %s\n", signature))
	}
	if docstring != "" {
		sb.WriteString(fmt.Sprintf("Docs: %s\n", docstring))
	}
	return sb.String()
}
```

- [ ] **Step 3: Usar el helper en search_context**

En la función que maneja `search_context` (buscar con `grep -n "search_context" mcp/server.go`), reemplazar el formateo de cada resultado por una llamada a `formatNodeWithAnnotations`.

El código anterior probablemente hace algo como:
```go
result += fmt.Sprintf("### %d. %s ...", i+1, name, ...)
```

Reemplazarlo por:
```go
result += fmt.Sprintf("### %d. (similarity: %.2f%%)\n", i+1, r.Similarity*100)
result += m.formatNodeWithAnnotations(nodeID, name, nodeType, file, line, signature, docstring)
result += "\n"
```

- [ ] **Step 4: Compilar y verificar**

```bash
go build ./...
```

- [ ] **Step 5: Commit**

```bash
git add mcp/server.go
git commit -m "feat: include human annotations in MCP search_context responses"
```

---

### Task 6: UI — campos estructurados en EditNodePanel

**Files:**
- Modify: `frontend/src/components/GraphEditor/EditNodePanel.tsx`

- [ ] **Step 1: Añadir campos al estado del componente**

En `EditNodePanel.tsx`, añadir al estado:
```typescript
const [why, setWhy] = useState('');
const [status, setStatus] = useState('stable');
const [entryPoint, setEntryPoint] = useState(false);
const [knownBug, setKnownBug] = useState('');
```

- [ ] **Step 2: Cargar los nuevos campos desde el nodo**

En el `useEffect` que carga el nodo (donde se inicializan `comments`, `tags`, etc.), añadir:
```typescript
setWhy(node.why || '');
setStatus(node.status || 'stable');
setEntryPoint(node.entry_point || false);
setKnownBug(node.known_bug || '');
```

- [ ] **Step 3: Incluir los nuevos campos en handleSave**

En `handleSave`, actualizar el objeto enviado a `onUpdate`:
```typescript
onUpdate(node.id, {
  comments,
  tags,
  custom_metadata: metadata,
  why,
  status,
  entry_point: entryPoint,
  known_bug: knownBug,
});
```

- [ ] **Step 4: Añadir los campos al JSX**

Después del bloque de Comments y antes de Tags, añadir:
```tsx
{/* Why does this exist */}
<div className="space-y-2">
  <label className="flex items-center gap-2 text-xs font-semibold text-slate-300 uppercase tracking-wide">
    <span className="text-emerald-400">?</span>
    Por qué existe
  </label>
  <textarea
    value={why}
    onChange={(e) => setWhy(e.target.value)}
    placeholder="Contexto de negocio: por qué existe esta función..."
    className="w-full h-20 px-4 py-3 bg-slate-800/50 border border-slate-700 text-slate-100 placeholder-slate-500 rounded-lg focus:outline-none focus:ring-1 focus:ring-emerald-500 focus:border-emerald-500 text-sm transition resize-none"
  />
</div>

{/* Status */}
<div className="space-y-2">
  <label className="text-xs font-semibold text-slate-300 uppercase tracking-wide">Estado</label>
  <select
    value={status}
    onChange={(e) => setStatus(e.target.value)}
    className="w-full px-3 py-2 bg-slate-800/50 border border-slate-700 text-slate-100 rounded-lg focus:outline-none focus:ring-1 focus:ring-emerald-500 text-sm"
  >
    <option value="stable">Stable</option>
    <option value="legacy">Legacy</option>
    <option value="deprecated">Deprecated</option>
    <option value="critical">Critical</option>
  </select>
</div>

{/* Entry point toggle */}
<div className="flex items-center justify-between py-2">
  <label className="text-xs font-semibold text-slate-300 uppercase tracking-wide">Entry Point</label>
  <button
    onClick={() => setEntryPoint(!entryPoint)}
    className={`relative w-10 h-5 rounded-full transition ${entryPoint ? 'bg-emerald-500' : 'bg-slate-600'}`}
  >
    <span className={`absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full transition-transform ${entryPoint ? 'translate-x-5' : ''}`} />
  </button>
</div>

{/* Known bug */}
<div className="space-y-2">
  <label className="flex items-center gap-2 text-xs font-semibold text-slate-300 uppercase tracking-wide">
    <span className="text-red-400">⚠</span>
    Bug conocido
  </label>
  <textarea
    value={knownBug}
    onChange={(e) => setKnownBug(e.target.value)}
    placeholder="Describe bugs conocidos que Claude debe tener en cuenta..."
    className="w-full h-20 px-4 py-3 bg-slate-800/50 border border-slate-700/50 border-red-900/30 text-slate-100 placeholder-slate-500 rounded-lg focus:outline-none focus:ring-1 focus:ring-red-500 text-sm transition resize-none"
  />
</div>
```

- [ ] **Step 5: Actualizar la interfaz NodeData para incluir los nuevos campos**

Al inicio de `EditNodePanel.tsx`, en la interfaz `NodeData`:
```typescript
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
  why?: string;
  status?: string;
  entry_point?: boolean;
  known_bug?: string;
}
```

- [ ] **Step 6: Compilar el frontend**

```bash
cd frontend && npm run build
```

Expected: sin errores de TypeScript.

- [ ] **Step 7: Commit**

```bash
git add frontend/src/components/GraphEditor/EditNodePanel.tsx
git commit -m "feat: add structured annotation fields to EditNodePanel UI"
```
