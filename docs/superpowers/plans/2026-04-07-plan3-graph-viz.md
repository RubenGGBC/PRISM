# Plan 3: Visualización del grafo (call edges + blast radius)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Mostrar en el grafo los edges de llamadas reales entre funciones y resaltar el blast radius al seleccionar un nodo.

**Architecture:** Se añaden dos endpoints HTTP: `/api/graph/edges` para edges de llamadas y `/api/node/impact` para el blast radius. `GraphViz.tsx` distingue tipos de edge visualmente con flechas direccionales. `CodeTreeView.tsx` pasa el estado de blast radius al grafo al seleccionar un nodo.

**Tech Stack:** Go, React/TypeScript, D3

**Prerequisite:** Plan 1 completado.

---

## Archivos que se tocan

| Acción | Archivo |
|--------|---------|
| Modify | `api/handlers.go` |
| Modify | `frontend/src/components/GraphViz.tsx` |
| Modify | `frontend/src/components/GraphEditor/CodeTreeView.tsx` |

---

### Task 1: Endpoints de edges e impacto

**Files:**
- Modify: `api/handlers.go`

- [ ] **Step 1: Añadir rutas en RegisterRoutes**

En `api/handlers.go`, dentro de `RegisterRoutes`:
```go
mux.HandleFunc("/api/graph/edges", a.corsMiddleware(a.handleGetCallEdges))
mux.HandleFunc("/api/node/impact", a.corsMiddleware(a.handleGetNodeImpact))
```

- [ ] **Step 2: Implementar handleGetCallEdges**

Añadir al final de `api/handlers.go`:
```go
// handleGetCallEdges returns all resolved call edges (GET /api/graph/edges)
func (a *APIServer) handleGetCallEdges(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodOptions {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	rows, err := a.db.Query(`
		SELECT source, target, edge_type
		FROM edges
		WHERE edge_type = 'calls'
		  AND target IN (SELECT id FROM nodes)
		LIMIT 2000
	`)
	if err != nil {
		http.Error(w, fmt.Sprintf("Query error: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Edge struct {
		Source string `json:"source"`
		Target string `json:"target"`
		Type   string `json:"type"`
	}

	var edges []Edge
	for rows.Next() {
		var e Edge
		if err := rows.Scan(&e.Source, &e.Target, &e.Type); err != nil {
			continue
		}
		edges = append(edges, e)
	}
	if edges == nil {
		edges = []Edge{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"edges": edges})
}

// handleGetNodeImpact returns nodes affected if this node changes (GET /api/node/impact?id=...)
func (a *APIServer) handleGetNodeImpact(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodOptions {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	nodeID := r.URL.Query().Get("id")
	if nodeID == "" {
		http.Error(w, "Missing id parameter", http.StatusBadRequest)
		return
	}

	// Direct callers (level 1)
	level1, err := a.getDirectCallers(nodeID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Query error: %v", err), http.StatusInternalServerError)
		return
	}

	// Level 2 callers
	level2 := []string{}
	seen := map[string]bool{nodeID: true}
	for _, id := range level1 {
		seen[id] = true
	}
	for _, callerID := range level1 {
		callers, err := a.getDirectCallers(callerID)
		if err != nil {
			continue
		}
		for _, id := range callers {
			if !seen[id] {
				seen[id] = true
				level2 = append(level2, id)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"node_id": nodeID,
		"level1":  level1,
		"level2":  level2,
	})
}

func (a *APIServer) getDirectCallers(nodeID string) ([]string, error) {
	rows, err := a.db.Query(
		`SELECT source FROM edges WHERE target = ? AND edge_type = 'calls'`,
		nodeID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var callers []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			callers = append(callers, id)
		}
	}
	return callers, nil
}
```

- [ ] **Step 3: Compilar**

```bash
go build ./...
```

- [ ] **Step 4: Test manual de los endpoints**

Con el servidor corriendo:
```bash
curl "http://localhost:8080/api/graph/edges" | head -c 500
```
Expected: JSON con array de edges `{"edges":[{"source":"...","target":"...","type":"calls"}]}`

```bash
# Usa un ID de nodo que exista en tu DB
curl "http://localhost:8080/api/node/impact?id=main.go:main"
```
Expected: `{"node_id":"...","level1":[...],"level2":[...]}`

- [ ] **Step 5: Commit**

```bash
git add api/handlers.go
git commit -m "feat: add /api/graph/edges and /api/node/impact endpoints"
```

---

### Task 2: Actualizar GraphViz para edges direccionales

**Files:**
- Modify: `frontend/src/components/GraphViz.tsx`

- [ ] **Step 1: Actualizar la interfaz y props**

En `GraphViz.tsx`, actualizar la interfaz `GraphEdge` y `GraphVizProps`:
```typescript
interface GraphEdge {
  source: string
  target: string
  type?: 'contains' | 'calls'  // 'contains' = archivo→nodo, 'calls' = llamada real
}

interface GraphVizProps {
  nodes: GraphNode[]
  edges: GraphEdge[]
  highlightIds?: { level1: string[]; level2: string[] }  // blast radius
  onNodeClick?: (nodeId: string) => void
}
```

- [ ] **Step 2: Reemplazar el componente completo con soporte de flechas y colores**

Reemplazar el contenido de `GraphViz.tsx` (el componente completo):
```typescript
import React, { useEffect, useRef } from 'react'
import * as d3 from 'd3'

interface GraphNode {
  id: string
  name: string
  type: string
}

interface GraphEdge {
  source: string
  target: string
  type?: 'contains' | 'calls'
}

interface GraphVizProps {
  nodes: GraphNode[]
  edges: GraphEdge[]
  highlightIds?: { level1: string[]; level2: string[] }
  onNodeClick?: (nodeId: string) => void
}

export function GraphViz({ nodes, edges, highlightIds, onNodeClick }: GraphVizProps) {
  const svgRef = useRef<SVGSVGElement>(null)

  useEffect(() => {
    if (!svgRef.current || nodes.length === 0) return

    const width = svgRef.current.clientWidth || 800
    const height = svgRef.current.clientHeight || 600

    d3.select(svgRef.current).selectAll('*').remove()

    const svg = d3.select(svgRef.current)
      .attr('width', width)
      .attr('height', height)

    // Arrow markers for call edges
    svg.append('defs').selectAll('marker')
      .data(['calls', 'contains'])
      .enter().append('marker')
      .attr('id', d => `arrow-${d}`)
      .attr('viewBox', '0 -5 10 10')
      .attr('refX', 18)
      .attr('refY', 0)
      .attr('markerWidth', 6)
      .attr('markerHeight', 6)
      .attr('orient', 'auto')
      .append('path')
      .attr('d', 'M0,-5L10,0L0,5')
      .attr('fill', d => d === 'calls' ? '#60a5fa' : '#444')

    const simulation = d3.forceSimulation(nodes as any)
      .force('link', d3.forceLink(edges as any).id((d: any) => d.id).distance(120))
      .force('charge', d3.forceManyBody().strength(-400))
      .force('center', d3.forceCenter(width / 2, height / 2))
      .force('collision', d3.forceCollide(20))

    const link = svg.selectAll('line')
      .data(edges)
      .enter().append('line')
      .attr('stroke', (d: any) => d.type === 'calls' ? '#3b82f6' : '#374151')
      .attr('stroke-width', (d: any) => d.type === 'calls' ? 1.5 : 1)
      .attr('stroke-opacity', (d: any) => d.type === 'calls' ? 0.7 : 0.4)
      .attr('marker-end', (d: any) => d.type === 'calls' ? 'url(#arrow-calls)' : null)

    const getNodeColor = (d: any) => {
      if (highlightIds?.level1?.includes(d.id)) return '#ef4444'
      if (highlightIds?.level2?.includes(d.id)) return '#f59e0b'
      if (d.type === 'function' || d.type === 'method') return '#60a5fa'
      if (d.type === 'class' || d.type === 'struct' || d.type === 'interface') return '#34d399'
      if (d.type === 'file') return '#8b5cf6'
      return '#fbbf24'
    }

    const node = svg.selectAll('circle')
      .data(nodes)
      .enter().append('circle')
      .attr('r', (d: any) => d.type === 'file' ? 10 : 7)
      .attr('fill', getNodeColor)
      .attr('stroke', (d: any) => highlightIds?.level1?.includes(d.id) ? '#dc2626' : 'none')
      .attr('stroke-width', 2)
      .call(d3.drag()
        .on('start', (event: any, d: any) => {
          if (!event.active) simulation.alphaTarget(0.3).restart()
          d.fx = d.x; d.fy = d.y
        })
        .on('drag', (event: any, d: any) => { d.fx = event.x; d.fy = event.y })
        .on('end', (event: any, d: any) => {
          if (!event.active) simulation.alphaTarget(0)
          d.fx = null; d.fy = null
        }) as any)
      .on('click', (_: any, d: any) => onNodeClick?.(d.id))

    const labels = svg.selectAll('text')
      .data(nodes)
      .enter().append('text')
      .attr('text-anchor', 'middle')
      .attr('fill', '#cbd5e1')
      .attr('font-size', '10px')
      .attr('pointer-events', 'none')
      .text((d: any) => d.name.substring(0, 12))

    simulation.on('tick', () => {
      link
        .attr('x1', (d: any) => d.source.x)
        .attr('y1', (d: any) => d.source.y)
        .attr('x2', (d: any) => d.target.x)
        .attr('y2', (d: any) => d.target.y)
      node.attr('cx', (d: any) => d.x).attr('cy', (d: any) => d.y)
      labels.attr('x', (d: any) => d.x).attr('y', (d: any) => d.y - 12)
    })

  }, [nodes, edges, highlightIds, onNodeClick])

  return (
    <div className="bg-gray-900 h-full rounded-lg overflow-hidden relative">
      <svg ref={svgRef} className="w-full h-full" />
      {highlightIds && (highlightIds.level1.length > 0 || highlightIds.level2.length > 0) && (
        <div className="absolute bottom-4 left-4 flex gap-3 text-xs">
          <span className="flex items-center gap-1">
            <span className="w-3 h-3 rounded-full bg-red-500 inline-block" />
            Callers directos ({highlightIds.level1.length})
          </span>
          <span className="flex items-center gap-1">
            <span className="w-3 h-3 rounded-full bg-amber-500 inline-block" />
            Nivel 2 ({highlightIds.level2.length})
          </span>
        </div>
      )}
    </div>
  )
}
```

- [ ] **Step 3: Compilar frontend**

```bash
cd frontend && npm run build
```

Expected: sin errores TypeScript.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/GraphViz.tsx
git commit -m "feat: add directional call edges and blast radius highlighting to GraphViz"
```

---

### Task 3: Integrar call edges y blast radius en CodeTreeView

**Files:**
- Modify: `frontend/src/components/GraphEditor/CodeTreeView.tsx`

- [ ] **Step 1: Añadir estado para call edges y blast radius**

Al principio de `CodeTreeView`, añadir:
```typescript
const [callEdges, setCallEdges] = useState<{ source: string; target: string; type: 'calls' }[]>([]);
const [blastRadius, setBlastRadius] = useState<{ level1: string[]; level2: string[] } | undefined>();
```

- [ ] **Step 2: Cargar call edges al iniciar**

Crear la función y llamarla en el useEffect inicial:
```typescript
const loadCallEdges = async () => {
  try {
    const response = await fetch(`${apiBaseUrl}/api/graph/edges`);
    if (!response.ok) return;
    const data = await response.json();
    setCallEdges((data.edges || []).map((e: any) => ({ ...e, type: 'calls' as const })));
  } catch {
    // silently ignore
  }
};

useEffect(() => {
  loadFiles();
  loadCallEdges();
}, []);
```

- [ ] **Step 3: Combinar los edges en graphEdges**

Reemplazar el useMemo de `graphEdges`:
```typescript
const graphEdges = useMemo(() => {
  const containsEdges = treeData.flatMap((file) =>
    (file.children || []).map((child) => ({
      source: file.id,
      target: child.id,
      type: 'contains' as const,
    }))
  );
  return [...containsEdges, ...callEdges];
}, [treeData, callEdges]);
```

- [ ] **Step 4: Cargar blast radius al seleccionar un nodo**

Actualizar `handleSelectNode` para cargar el impacto:
```typescript
const handleSelectNode = async (node: TreeNodeData) => {
  try {
    const [nodeRes, impactRes] = await Promise.all([
      fetch(`${apiBaseUrl}/api/node/full?id=${encodeURIComponent(node.id)}`),
      fetch(`${apiBaseUrl}/api/node/impact?id=${encodeURIComponent(node.id)}`),
    ]);

    if (nodeRes.ok) {
      const data = await nodeRes.json();
      setSelectedNode(data.node);
    } else {
      setSelectedNode(node);
    }

    if (impactRes.ok) {
      const impact = await impactRes.json();
      if (impact.level1?.length > 0 || impact.level2?.length > 0) {
        setBlastRadius({ level1: impact.level1 || [], level2: impact.level2 || [] });
      } else {
        setBlastRadius(undefined);
      }
    }
  } catch (err) {
    setSelectedNode(node);
    setBlastRadius(undefined);
  }
};
```

- [ ] **Step 5: Pasar highlightIds al GraphViz**

En el JSX donde se renderiza `<GraphViz>`:
```tsx
<GraphViz
  nodes={graphNodes}
  edges={graphEdges}
  highlightIds={blastRadius}
  onNodeClick={(nodeId) => {
    const flat = treeData.flatMap((f) => [f, ...(f.children || [])]);
    const found = flat.find((n) => n.id === nodeId);
    if (found) handleSelectNode(found);
  }}
/>
```

- [ ] **Step 6: Limpiar blast radius al deseleccionar**

En el botón de close de `EditNodePanel` (el `onClose`), añadir limpieza de blast radius:
```typescript
// Actualizar donde se pasa onClose al EditNodePanel:
onClose={() => {
  setSelectedNode(null);
  setBlastRadius(undefined);
}}
```

- [ ] **Step 7: Compilar frontend**

```bash
cd frontend && npm run build
```

Expected: sin errores TypeScript.

- [ ] **Step 8: Verificar visualmente**

```bash
# Terminal 1: servidor
go run . serve

# Terminal 2: frontend dev
cd frontend && npm run dev
```

Abrir http://localhost:5173, click en un nodo → el grafo debe resaltar en rojo sus callers directos.

- [ ] **Step 9: Commit**

```bash
git add frontend/src/components/GraphEditor/CodeTreeView.tsx
git commit -m "feat: integrate call edges and blast radius into CodeTreeView"
```
