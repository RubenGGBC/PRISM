# Plan 4: Context Profiles + Token Dashboard + Export CLAUDE.md

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Añadir perfiles de contexto nombrados, un dashboard de tokens ahorrados y el comando `prism export` para generar CLAUDE.md desde el grafo.

**Architecture:** Nueva migración para tablas `profiles` y `profile_nodes`. API REST para CRUD de perfiles. Herramienta MCP `use_profile`. El WebSocket existente emite stats de tokens. El comando `export` en main.go genera CLAUDE.md. La UI tiene una sección de perfiles en la sidebar.

**Tech Stack:** Go, React/TypeScript

**Prerequisite:** Plan 2 completado.

---

## Archivos que se tocan

| Acción | Archivo |
|--------|---------|
| Modify | `db/migrations.go` |
| Modify | `api/handlers.go` |
| Modify | `mcp/server.go` |
| Modify | `api/websocket.go` |
| Modify | `main.go` |
| Create | `frontend/src/components/Profiles.tsx` |
| Modify | `frontend/src/components/GraphEditor/CodeTreeView.tsx` |

---

### Task 1: Migración — tablas de perfiles y sesiones MCP

**Files:**
- Modify: `db/migrations.go`

- [ ] **Step 1: Añadir migración de profiles y mcp_sessions**

En `db/migrations.go`, añadir en `RunMigrations`:
```go
if err := createProfilesTables(db); err != nil {
    return fmt.Errorf("failed to create profiles tables: %w", err)
}
if err := createMCPSessionsTable(db); err != nil {
    return fmt.Errorf("failed to create mcp_sessions table: %w", err)
}
```

Y las funciones:
```go
func createProfilesTables(db *sql.DB) error {
    _, err := db.Exec(`
    CREATE TABLE IF NOT EXISTS profiles (
        id          TEXT PRIMARY KEY,
        name        TEXT NOT NULL UNIQUE,
        description TEXT,
        created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
    );
    CREATE TABLE IF NOT EXISTS profile_nodes (
        profile_id TEXT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
        node_id    TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
        PRIMARY KEY (profile_id, node_id)
    );
    CREATE INDEX IF NOT EXISTS idx_profile_nodes_profile ON profile_nodes(profile_id);
    `)
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
```

- [ ] **Step 2: Verificar migración**

```bash
rm -f code_graph.db && go run . index -repo . && sqlite3 code_graph.db ".tables"
```

Expected: `profiles`, `profile_nodes`, `mcp_sessions` aparecen en la lista.

- [ ] **Step 3: Commit**

```bash
git add db/migrations.go
git commit -m "feat: add profiles and mcp_sessions tables"
```

---

### Task 2: API REST para perfiles

**Files:**
- Modify: `api/handlers.go`

- [ ] **Step 1: Añadir rutas en RegisterRoutes**

```go
mux.HandleFunc("/api/profiles", a.corsMiddleware(a.handleProfiles))
mux.HandleFunc("/api/profile/nodes", a.corsMiddleware(a.handleProfileNodes))
mux.HandleFunc("/api/stats", a.corsMiddleware(a.handleStats))
```

- [ ] **Step 2: Implementar handlers de perfiles**

Añadir al final de `api/handlers.go`:
```go
// handleProfiles — GET lista perfiles, POST crea uno, DELETE elimina
func (a *APIServer) handleProfiles(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    switch r.Method {
    case http.MethodGet:
        rows, err := a.db.Query(`SELECT id, name, description, created_at FROM profiles ORDER BY name`)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        defer rows.Close()
        type Profile struct {
            ID          string `json:"id"`
            Name        string `json:"name"`
            Description string `json:"description"`
            CreatedAt   string `json:"created_at"`
        }
        var profiles []Profile
        for rows.Next() {
            var p Profile
            rows.Scan(&p.ID, &p.Name, &p.Description, &p.CreatedAt)
            profiles = append(profiles, p)
        }
        if profiles == nil {
            profiles = []Profile{}
        }
        json.NewEncoder(w).Encode(map[string]interface{}{"profiles": profiles})

    case http.MethodPost:
        var req struct {
            Name        string `json:"name"`
            Description string `json:"description"`
        }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
            http.Error(w, "invalid request", http.StatusBadRequest)
            return
        }
        id := fmt.Sprintf("profile_%d", time.Now().UnixNano())
        _, err := a.db.Exec(`INSERT INTO profiles (id, name, description) VALUES (?, ?, ?)`,
            id, req.Name, req.Description)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        json.NewEncoder(w).Encode(map[string]string{"id": id, "status": "ok"})

    case http.MethodDelete:
        profileID := r.URL.Query().Get("id")
        if profileID == "" {
            http.Error(w, "missing id", http.StatusBadRequest)
            return
        }
        a.db.Exec(`DELETE FROM profiles WHERE id = ?`, profileID)
        json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

    default:
        w.WriteHeader(http.StatusMethodNotAllowed)
    }
}

// handleProfileNodes — GET nodos de un perfil, POST añade nodo, DELETE quita nodo
func (a *APIServer) handleProfileNodes(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    profileID := r.URL.Query().Get("profile_id")
    if profileID == "" {
        http.Error(w, "missing profile_id", http.StatusBadRequest)
        return
    }

    switch r.Method {
    case http.MethodGet:
        rows, err := a.db.Query(`
            SELECT n.id, n.name, n.type, n.file, n.line
            FROM profile_nodes pn
            JOIN nodes n ON n.id = pn.node_id
            WHERE pn.profile_id = ?`, profileID)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        defer rows.Close()
        type Node struct {
            ID   string `json:"id"`
            Name string `json:"name"`
            Type string `json:"type"`
            File string `json:"file"`
            Line int    `json:"line"`
        }
        var nodes []Node
        for rows.Next() {
            var n Node
            rows.Scan(&n.ID, &n.Name, &n.Type, &n.File, &n.Line)
            nodes = append(nodes, n)
        }
        if nodes == nil {
            nodes = []Node{}
        }
        json.NewEncoder(w).Encode(map[string]interface{}{"nodes": nodes})

    case http.MethodPost:
        var req struct{ NodeID string `json:"node_id"` }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.NodeID == "" {
            http.Error(w, "invalid request", http.StatusBadRequest)
            return
        }
        a.db.Exec(`INSERT OR IGNORE INTO profile_nodes (profile_id, node_id) VALUES (?, ?)`,
            profileID, req.NodeID)
        json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

    case http.MethodDelete:
        nodeID := r.URL.Query().Get("node_id")
        if nodeID == "" {
            http.Error(w, "missing node_id", http.StatusBadRequest)
            return
        }
        a.db.Exec(`DELETE FROM profile_nodes WHERE profile_id = ? AND node_id = ?`, profileID, nodeID)
        json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

    default:
        w.WriteHeader(http.StatusMethodNotAllowed)
    }
}

// handleStats returns token savings stats for the current MCP session
func (a *APIServer) handleStats(w http.ResponseWriter, r *http.Request) {
    var served, saved int
    row := a.db.QueryRow(`
        SELECT COALESCE(SUM(tokens_served),0), COALESCE(SUM(tokens_saved),0)
        FROM mcp_sessions
        WHERE started_at > datetime('now', '-24 hours')
    `)
    row.Scan(&served, &saved)
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "tokens_served": served,
        "tokens_saved":  saved,
    })
}
```

Añadir el import de `time` si no está ya en el fichero.

- [ ] **Step 3: Compilar**

```bash
go build ./...
```

- [ ] **Step 4: Test manual**

```bash
# Crear un perfil
curl -X POST "http://localhost:8080/api/profiles" \
  -H "Content-Type: application/json" \
  -d '{"name":"auth","description":"Authentication flow"}'

# Listar perfiles
curl "http://localhost:8080/api/profiles"
```

Expected: JSON con el perfil creado.

- [ ] **Step 5: Commit**

```bash
git add api/handlers.go
git commit -m "feat: add profiles and stats API endpoints"
```

---

### Task 3: Herramienta MCP use_profile + token tracking

**Files:**
- Modify: `mcp/server.go`

- [ ] **Step 1: Añadir la herramienta use_profile**

En la sección donde se registran las herramientas MCP (buscar con `grep -n "AddTool\|search_context" mcp/server.go | head -10`), añadir:

```go
s.AddTool(mcp.NewTool("use_profile",
    mcp.WithDescription("Load a named context profile — returns all nodes in the profile with their annotations. Use before starting work on a specific area."),
    mcp.WithString("name", mcp.Required(), mcp.Description("Profile name (e.g. 'auth', 'checkout', 'frontend')")),
), m.handleUseProfile)
```

- [ ] **Step 2: Implementar handleUseProfile**

Añadir al final de `mcp/server.go`:
```go
func (m *MCPServer) handleUseProfile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    name, _ := req.Params.Arguments["name"].(string)
    if name == "" {
        return mcp.NewToolResultError("profile name is required"), nil
    }

    // Find profile by name
    var profileID string
    err := m.graph.DB.QueryRow(`SELECT id FROM profiles WHERE name = ?`, name).Scan(&profileID)
    if err != nil {
        return mcp.NewToolResultError(fmt.Sprintf("profile '%s' not found", name)), nil
    }

    // Get nodes in profile
    rows, err := m.graph.DB.Query(`
        SELECT n.id, n.name, n.type, n.file, n.line, n.signature, n.docstring
        FROM profile_nodes pn
        JOIN nodes n ON n.id = pn.node_id
        WHERE pn.profile_id = ?`, profileID)
    if err != nil {
        return mcp.NewToolResultError(fmt.Sprintf("query error: %v", err)), nil
    }
    defer rows.Close()

    var result strings.Builder
    result.WriteString(fmt.Sprintf("# Context Profile: %s\n\n", name))

    count := 0
    totalTokens := 0
    for rows.Next() {
        var id, nName, nType, file, sig, doc string
        var line int
        var sigNull, docNull sql.NullString
        rows.Scan(&id, &nName, &nType, &file, &line, &sigNull, &docNull)
        sig = sigNull.String
        doc = docNull.String

        entry := m.formatNodeWithAnnotations(id, nName, nType, file, line, sig, doc)
        result.WriteString(entry)
        result.WriteString("\n")
        count++

        if m.tokenizer != nil {
            totalTokens += len(m.tokenizer.Encode(entry, nil, nil))
        }
    }

    if count == 0 {
        return mcp.NewToolResultError(fmt.Sprintf("profile '%s' has no nodes. Add nodes via the UI.", name)), nil
    }

    m.logger.Printf("📦 Profile '%s': %d nodes, ~%d tokens", name, count, totalTokens)
    return mcp.NewToolResultText(result.String()), nil
}
```

- [ ] **Step 2: Añadir tracking de tokens en search_context**

En la función `handleSearchContext` (o equivalente), después de construir el resultado, añadir:
```go
// Track tokens served vs tokens that would have been needed reading full files
if m.tokenizer != nil {
    tokensServed := len(m.tokenizer.Encode(resultText, nil, nil))
    // Estimate full file tokens: average 500 tokens per file read
    tokensSaved := len(results) * 500
    m.graph.DB.Exec(`
        INSERT INTO mcp_sessions (id, tokens_served, tokens_saved)
        VALUES (?, ?, ?)
        ON CONFLICT(id) DO UPDATE SET
            tokens_served = tokens_served + excluded.tokens_served,
            tokens_saved = tokens_saved + excluded.tokens_saved`,
        "current_session", tokensServed, tokensSaved-tokensServed)
}
```

- [ ] **Step 3: Compilar**

```bash
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add mcp/server.go
git commit -m "feat: add use_profile MCP tool and token tracking"
```

---

### Task 4: Export CLAUDE.md

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Añadir case "export" en main.go**

En el switch de `main()`:
```go
case "export":
    exportCmd := flag.NewFlagSet("export", flag.ExitOnError)
    dbPath := exportCmd.String("db", "code_graph.db", "Database path")
    output := exportCmd.String("output", "CLAUDE.md", "Output file path")
    exportCmd.Parse(os.Args[2:])
    exportClaudeMD(*dbPath, *output)
```

- [ ] **Step 2: Implementar exportClaudeMD**

Añadir la función en `main.go`:
```go
func exportClaudeMD(dbPath, outputPath string) {
    database, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        fmt.Printf("❌ Failed to open database: %v\n", err)
        os.Exit(1)
    }
    defer database.Close()

    var sb strings.Builder
    sb.WriteString("# CLAUDE.md — Generado por PRISM\n\n")
    sb.WriteString("> Este archivo fue generado automáticamente desde las anotaciones del grafo de código.\n\n")

    // Entry points → Architecture
    rows, _ := database.Query(`
        SELECT n.name, n.file, n.line, na.why
        FROM nodes n
        JOIN node_annotations na ON na.node_id = n.id
        WHERE na.entry_point = TRUE
        ORDER BY n.file, n.line
    `)
    if rows != nil {
        defer rows.Close()
        sb.WriteString("## Arquitectura — Entry Points\n\n")
        for rows.Next() {
            var name, file, why string
            var line int
            rows.Scan(&name, &file, &line, &why)
            sb.WriteString(fmt.Sprintf("- **%s** (`%s:%d`)", name, file, line))
            if why != "" {
                sb.WriteString(fmt.Sprintf(" — %s", why))
            }
            sb.WriteString("\n")
        }
        sb.WriteString("\n")
    }

    // Critical nodes
    rows2, _ := database.Query(`
        SELECT n.name, n.file, n.line, na.why, na.known_bug
        FROM nodes n
        JOIN node_annotations na ON na.node_id = n.id
        WHERE na.status = 'critical'
        ORDER BY n.file
    `)
    if rows2 != nil {
        defer rows2.Close()
        sb.WriteString("## Áreas Críticas\n\n")
        for rows2.Next() {
            var name, file, why, knownBug string
            var line int
            rows2.Scan(&name, &file, &line, &why, &knownBug)
            sb.WriteString(fmt.Sprintf("- **%s** (`%s:%d`)", name, file, line))
            if why != "" {
                sb.WriteString(fmt.Sprintf(": %s", why))
            }
            if knownBug != "" {
                sb.WriteString(fmt.Sprintf(" ⚠️ Bug: %s", knownBug))
            }
            sb.WriteString("\n")
        }
        sb.WriteString("\n")
    }

    // Deprecated/Legacy nodes
    rows3, _ := database.Query(`
        SELECT n.name, n.file, na.status, na.why
        FROM nodes n
        JOIN node_annotations na ON na.node_id = n.id
        WHERE na.status IN ('deprecated', 'legacy')
        ORDER BY na.status, n.file
    `)
    if rows3 != nil {
        defer rows3.Close()
        sb.WriteString("## Código a Evitar\n\n")
        for rows3.Next() {
            var name, file, status, why string
            rows3.Scan(&name, &file, &status, &why)
            sb.WriteString(fmt.Sprintf("- **%s** (`%s`) [%s]", name, file, status))
            if why != "" {
                sb.WriteString(fmt.Sprintf(" — %s", why))
            }
            sb.WriteString("\n")
        }
        sb.WriteString("\n")
    }

    // Known bugs
    rows4, _ := database.Query(`
        SELECT n.name, n.file, n.line, na.known_bug
        FROM nodes n
        JOIN node_annotations na ON na.node_id = n.id
        WHERE na.known_bug IS NOT NULL AND na.known_bug != ''
        ORDER BY n.file
    `)
    if rows4 != nil {
        defer rows4.Close()
        sb.WriteString("## Bugs Conocidos\n\n")
        for rows4.Next() {
            var name, file, knownBug string
            var line int
            rows4.Scan(&name, &file, &line, &knownBug)
            sb.WriteString(fmt.Sprintf("- **%s** (`%s:%d`): %s\n", name, file, line, knownBug))
        }
        sb.WriteString("\n")
    }

    // Context profiles
    rows5, _ := database.Query(`SELECT name, description FROM profiles ORDER BY name`)
    if rows5 != nil {
        defer rows5.Close()
        sb.WriteString("## Perfiles de Contexto PRISM\n\n")
        sb.WriteString("Usa `prism use <nombre>` para cargar el contexto de un área:\n\n")
        for rows5.Next() {
            var name, desc string
            rows5.Scan(&name, &desc)
            sb.WriteString(fmt.Sprintf("- `%s`", name))
            if desc != "" {
                sb.WriteString(fmt.Sprintf(" — %s", desc))
            }
            sb.WriteString("\n")
        }
        sb.WriteString("\n")
    }

    if err := os.WriteFile(outputPath, []byte(sb.String()), 0644); err != nil {
        fmt.Printf("❌ Failed to write %s: %v\n", outputPath, err)
        os.Exit(1)
    }
    fmt.Printf("✅ Exported to %s\n", outputPath)
}
```

Añadir al import `"strings"` si no está.

- [ ] **Step 3: Añadir "export" al printUsage**

```go
  export               Generate CLAUDE.md from graph annotations

Options for 'export':
  -db <path>          Database path (default: code_graph.db)
  -output <path>      Output file (default: CLAUDE.md)
```

- [ ] **Step 4: Compilar y probar**

```bash
go build ./...
go run . export -db code_graph.db -output CLAUDE_test.md
cat CLAUDE_test.md
```

Expected: archivo Markdown con secciones de arquitectura, críticos, legacy, bugs, perfiles.

- [ ] **Step 5: Commit**

```bash
git add main.go
git commit -m "feat: add prism export command to generate CLAUDE.md from graph"
```

---

### Task 5: UI — sección de Perfiles + Token Dashboard

**Files:**
- Create: `frontend/src/components/Profiles.tsx`
- Modify: `frontend/src/components/GraphEditor/CodeTreeView.tsx`

- [ ] **Step 1: Crear el componente Profiles**

Crear `frontend/src/components/Profiles.tsx`:
```tsx
import React, { useState, useEffect } from 'react';
import { Plus, Trash2, Tag } from 'lucide-react';

interface Profile {
  id: string;
  name: string;
  description: string;
}

interface ProfilesProps {
  apiBaseUrl: string;
  selectedNodeId?: string;
}

export const Profiles: React.FC<ProfilesProps> = ({ apiBaseUrl, selectedNodeId }) => {
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [newName, setNewName] = useState('');
  const [newDesc, setNewDesc] = useState('');
  const [creating, setCreating] = useState(false);

  useEffect(() => {
    loadProfiles();
  }, []);

  const loadProfiles = async () => {
    try {
      const res = await fetch(`${apiBaseUrl}/api/profiles`);
      const data = await res.json();
      setProfiles(data.profiles || []);
    } catch {}
  };

  const createProfile = async () => {
    if (!newName.trim()) return;
    await fetch(`${apiBaseUrl}/api/profiles`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: newName.trim(), description: newDesc.trim() }),
    });
    setNewName('');
    setNewDesc('');
    setCreating(false);
    loadProfiles();
  };

  const deleteProfile = async (id: string) => {
    await fetch(`${apiBaseUrl}/api/profiles?id=${encodeURIComponent(id)}`, { method: 'DELETE' });
    loadProfiles();
  };

  const addNodeToProfile = async (profileId: string) => {
    if (!selectedNodeId) return;
    await fetch(`${apiBaseUrl}/api/profile/nodes?profile_id=${encodeURIComponent(profileId)}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ node_id: selectedNodeId }),
    });
  };

  return (
    <div className="px-3 py-3 border-t border-slate-700/50">
      <div className="flex items-center justify-between mb-2">
        <span className="text-xs font-semibold text-slate-400 uppercase tracking-wide">Perfiles</span>
        <button
          onClick={() => setCreating(!creating)}
          className="p-1 rounded hover:bg-slate-800 text-slate-500 hover:text-emerald-400 transition"
        >
          <Plus size={14} />
        </button>
      </div>

      {creating && (
        <div className="space-y-1.5 mb-2">
          <input
            type="text"
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            placeholder="Nombre (ej. auth)"
            className="w-full px-2 py-1.5 bg-slate-900 border border-slate-700 text-slate-100 placeholder-slate-500 rounded text-xs focus:outline-none focus:ring-1 focus:ring-emerald-500"
          />
          <input
            type="text"
            value={newDesc}
            onChange={(e) => setNewDesc(e.target.value)}
            placeholder="Descripción (opcional)"
            className="w-full px-2 py-1.5 bg-slate-900 border border-slate-700 text-slate-100 placeholder-slate-500 rounded text-xs focus:outline-none focus:ring-1 focus:ring-emerald-500"
          />
          <button
            onClick={createProfile}
            className="w-full py-1.5 bg-emerald-600 hover:bg-emerald-700 text-white rounded text-xs font-medium transition"
          >
            Crear perfil
          </button>
        </div>
      )}

      <div className="space-y-1">
        {profiles.map((p) => (
          <div key={p.id} className="flex items-center gap-1.5 group">
            <span className="flex-1 text-xs text-slate-300 truncate">{p.name}</span>
            {selectedNodeId && (
              <button
                onClick={() => addNodeToProfile(p.id)}
                title="Añadir nodo seleccionado"
                className="p-0.5 rounded opacity-0 group-hover:opacity-100 hover:bg-slate-700 text-slate-500 hover:text-emerald-400 transition"
              >
                <Tag size={12} />
              </button>
            )}
            <button
              onClick={() => deleteProfile(p.id)}
              className="p-0.5 rounded opacity-0 group-hover:opacity-100 hover:bg-slate-700 text-slate-500 hover:text-red-400 transition"
            >
              <Trash2 size={12} />
            </button>
          </div>
        ))}
        {profiles.length === 0 && (
          <p className="text-xs text-slate-600">Sin perfiles aún</p>
        )}
      </div>
    </div>
  );
};
```

- [ ] **Step 2: Crear TokenDashboard en CodeTreeView**

En `CodeTreeView.tsx`, añadir estado para stats:
```typescript
const [tokenStats, setTokenStats] = useState<{ tokens_served: number; tokens_saved: number } | null>(null);
```

Añadir función para cargar stats:
```typescript
const loadStats = async () => {
  try {
    const res = await fetch(`${apiBaseUrl}/api/stats`);
    const data = await res.json();
    setTokenStats(data);
  } catch {}
};
```

Llamar `loadStats()` cada 30 segundos con un intervalo:
```typescript
useEffect(() => {
  loadStats();
  const interval = setInterval(loadStats, 30000);
  return () => clearInterval(interval);
}, []);
```

- [ ] **Step 3: Añadir el componente Profiles y el token banner al JSX**

En el JSX de `CodeTreeView`, en la sidebar izquierda, antes del footer de stats:
```tsx
<Profiles
  apiBaseUrl={apiBaseUrl}
  selectedNodeId={selectedNode?.id}
/>
```

Y en la parte superior del centro (encima de `<GraphViz>`):
```tsx
{tokenStats && tokenStats.tokens_saved > 0 && (
  <div className="bg-slate-800/50 border-b border-slate-700/50 px-4 py-2 text-xs text-slate-400 flex items-center gap-3">
    <span className="text-emerald-400 font-semibold">PRISM</span>
    <span>{tokenStats.tokens_served.toLocaleString()} tokens servidos</span>
    <span className="text-slate-600">vs</span>
    <span>{(tokenStats.tokens_served + tokenStats.tokens_saved).toLocaleString()} sin PRISM</span>
    <span className="ml-auto text-emerald-400 font-medium">
      {Math.round((tokenStats.tokens_saved / (tokenStats.tokens_served + tokenStats.tokens_saved)) * 100)}% ahorro
    </span>
  </div>
)}
```

- [ ] **Step 4: Añadir import de Profiles en CodeTreeView**

Al inicio de `CodeTreeView.tsx`:
```typescript
import { Profiles } from '../Profiles';
```

- [ ] **Step 5: Compilar frontend**

```bash
cd frontend && npm run build
```

Expected: sin errores TypeScript.

- [ ] **Step 6: Verificar visualmente**

```bash
go run . serve
cd frontend && npm run dev
```

Abrir http://localhost:5173 — debe aparecer la sección de Perfiles en la sidebar y el banner de tokens (solo si hay sesiones MCP activas).

- [ ] **Step 7: Commit final**

```bash
git add frontend/src/components/Profiles.tsx frontend/src/components/GraphEditor/CodeTreeView.tsx
git commit -m "feat: add profiles UI and token savings dashboard"
```
