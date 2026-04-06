# PRISM v2 — Design Spec

**Date:** 2026-04-07
**Status:** Approved

---

## Problema

PRISM nació para reducir el consumo de tokens de contexto en proyectos grandes. En vez de que Claude lea archivos enteros, PRISM indexa el codebase, genera embeddings semánticos, y sirve solo los fragmentos relevantes vía MCP (RAG quirúrgico). La UI permite enriquecer manualmente el grafo con conocimiento humano que ningún parser puede extraer.

**Limitaciones actuales:**
- Solo soporta Python y TypeScript
- El grafo en la UI muestra nodos aislados (sin edges de llamadas reales)
- El reindexado es manual (`prism index` + `prism serve`)
- Los embeddings no incluyen las anotaciones manuales del usuario
- No hay forma de agrupar contexto por área de trabajo
- El valor (tokens ahorrados) no es visible

**Diferenciación vs competidores (SoulForge, Aider, etc.):**
PRISM no es un agente sustituto de Claude Code — es la mejor capa RAG para código, enriquecida con conocimiento humano. Ningún competidor permite anotar nodos con contexto de negocio que luego alimenta el RAG.

---

## Sección 1: Infraestructura — Más lenguajes + RAG más sólido

### 1.1 Parser multi-lenguaje con tree-sitter

**Cambio:** Reemplazar `parser/python.go` y `parser/typescript.go` por un parser unificado basado en tree-sitter.

**Lenguajes objetivo (primera iteración):** Go, Rust, Java, C#, Ruby, PHP, Swift, Kotlin, C, C++. Cualquier lenguaje con gramática tree-sitter disponible (~40 total).

**Interfaz:** El parser unificado implementa la misma interfaz `Parser` que los parsers actuales — el resto del sistema no cambia.

**Detección de lenguaje:** `parser.DetectLanguage(path)` ya existe; se amplía con las nuevas extensiones.

### 1.2 File Watcher (reindexado automático)

**Cambio:** Añadir un goroutine en `prism serve` que observa el filesystem con `fsnotify`.

**Comportamiento:**
- Al detectar un archivo modificado/creado, lo reindexea incrementalmente (solo ese archivo)
- Al detectar un archivo eliminado, borra sus nodos del grafo
- Debounce de 500ms para evitar reindexados en cascada durante un guardado de editor
- No interrumpe el servidor MCP ni la API HTTP

**Config:** Flag `--watch` en `prism serve` (opcional, off por defecto).

### 1.3 Embeddings enriquecidos

**Cambio:** El texto que se embediza por nodo pasa de:
```
"function foo in bar.py"
```
a:
```
"function foo in bar.py
Signature: async foo(x, y) -> Result
Docstring: Processes payment for checkout
Annotation: Entry point del checkout. Bug conocido con montos >10k€.
Tags: critical, entry-point"
```

**Impacto:** Las anotaciones manuales del usuario mejoran la precisión del RAG. Cada vez que anotas un nodo, se regenera su embedding.

---

## Sección 2: UI como editor de conocimiento

### 2.1 Grafo con edges de llamadas reales

**Cambio:** El backend ya resuelve call edges (`graph/builder.go:ResolveCallEdges`). Hay que exponerlos en `/api/file/nodes` y en una nueva ruta `/api/graph/edges`.

**UI:** `GraphViz.tsx` distingue visualmente dos tipos de edge:
- **Gris claro** — relación archivo→nodo (contenido)
- **Azul** — llamada directa (A llama a B)

Los edges de llamada se renderizan con una flecha direccional.

### 2.2 Blast radius al seleccionar un nodo

**Cambio:** Al hacer click en un nodo, el grafo resalta:
- **Rojo** — nodos que dependen de este (se rompen si cambia)
- **Amarillo** — dependencias de segundo nivel

Usa el endpoint MCP `trace_impact` ya existente, expuesto también en `/api/node/impact?id=`.

### 2.3 Panel de anotaciones estructuradas

**Cambio:** `EditNodePanel.tsx` añade campos estructurados junto a comments/tags/metadata:

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `why` | texto libre | Contexto de negocio: por qué existe esta función |
| `status` | enum | `stable` / `legacy` / `deprecated` / `critical` |
| `entry_point` | boolean | Si Claude siempre debe conocer este nodo |
| `known_bug` | texto libre | Bug conocido que Claude debe tener en cuenta |

Estos campos se persisten en SQLite junto al nodo y se incluyen en las respuestas del MCP.

### 2.4 Context Profiles

**Nueva feature:** Sistema de perfiles de contexto nombrados.

**UI:** Nueva sección "Profiles" en la sidebar. El usuario:
1. Crea un perfil (ej. `auth`, `checkout`, `frontend`)
2. Selecciona nodos del grafo para incluir en el perfil
3. El perfil se guarda en la DB

**MCP:** Nueva herramienta `use_profile`:
```
use_profile("auth") → devuelve todos los nodos del perfil auth con sus anotaciones
```

**CLI:**
```bash
prism profile list
prism profile use auth
```

---

## Sección 3: Diferenciadores

### 3.1 Dashboard de tokens ahorrados

**Cambio:** El servidor MCP trackea por sesión:
- Nodos servidos y sus tamaños en tokens
- Tamaño real de los archivos completos correspondientes
- Ratio de compresión

**UI:** Barra superior en la UI con stats de la sesión MCP activa vía WebSocket (el WebSocket ya existe en `api/websocket.go`):
```
PRISM esta sesión: 2.3k tokens inyectados vs 87k en archivos completos — 97% ahorro
```

### 3.2 Export a CLAUDE.md

**Nuevo comando:** `prism export` genera un `CLAUDE.md` desde el grafo:

- Nodos con `entry_point: true` → sección "Arquitectura"
- Nodos con `status: deprecated/legacy` → sección "Código a evitar"
- Nodos con `status: critical` → sección "Áreas críticas"
- Nodos con `known_bug` → sección "Bugs conocidos"
- Context profiles → sección "Perfiles de contexto"

**UI:** Botón "Export CLAUDE.md" en la cabecera de la UI.

### 3.3 Anotaciones humanas en respuestas MCP

**Cambio:** `search_context`, `get_file_smart`, y `trace_impact` incluyen las anotaciones estructuradas en su respuesta:

```
## processPayment (checkout/payment.ts:45)
Status: critical
Why: Entry point del checkout. Coordinado con equipo de pagos.
Known bug: Falla con montos >10k€ por overflow en currency conversion.
Signature: async processPayment(amount, currency, userId): Promise<Result>
Tags: entry-point, critical
```

Claude recibe contexto humano + contexto de código en una sola llamada MCP.

---

## Arquitectura de datos

**Cambios en SQLite schema:**

```sql
-- Ampliar tabla nodes
ALTER TABLE nodes ADD COLUMN why TEXT;
ALTER TABLE nodes ADD COLUMN status TEXT DEFAULT 'stable';
ALTER TABLE nodes ADD COLUMN entry_point BOOLEAN DEFAULT FALSE;
ALTER TABLE nodes ADD COLUMN known_bug TEXT;

-- Nueva tabla profiles
CREATE TABLE profiles (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  description TEXT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE profile_nodes (
  profile_id TEXT REFERENCES profiles(id),
  node_id TEXT REFERENCES nodes(id),
  PRIMARY KEY (profile_id, node_id)
);

-- Nueva tabla mcp_sessions (para token tracking)
CREATE TABLE mcp_sessions (
  id TEXT PRIMARY KEY,
  started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  tokens_served INTEGER DEFAULT 0,
  tokens_saved INTEGER DEFAULT 0
);
```

---

## Orden de implementación sugerido

1. **Tree-sitter parser** — desbloquea el mayor número de usuarios
2. **Anotaciones estructuradas** — ampliar schema + API + UI
3. **Anotaciones en MCP** — incluirlas en respuestas existentes
4. **Blast radius en UI** — exponer `trace_impact` en la UI
5. **Edges de llamadas en grafo** — visualización real de dependencias
6. **File watcher** — reindexado automático
7. **Embeddings enriquecidos** — regenerar al anotar
8. **Context Profiles** — DB + MCP tool + UI
9. **Token dashboard** — WebSocket + UI
10. **Export CLAUDE.md** — CLI + botón UI
