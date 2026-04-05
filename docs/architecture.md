# Architecture

## Overview

PRISM Platform is a graph-based code analysis system that provides:
1. **Static analysis** via Tree-sitter AST parsing
2. **Dependency tracking** via SQLite graph database
3. **Semantic search** via Ollama embeddings
4. **IDE integration** via MCP (Model Context Protocol)

## Components

### Parser (parser/)

Tree-sitter extracts AST elements from source files:
- Functions, methods, classes
- Parameters, return types
- Call expressions (who calls whom)
- Docstrings and comments

**Supported Languages:**
- TypeScript/JavaScript (.ts, .tsx, .js, .jsx)
- Python (.py)

**Key Types:**
- `CodeElement` - A function, method, or class
- `ParsedFile` - All elements from a single file
- `ImportInfo` - Import/export relationships

### Graph (graph/)

SQLite stores the code graph with two tables:
- `nodes` - Functions, classes, methods
- `edges` - Calls, imports relationships

**Features:**
- Fast querying with SQL
- PageRank for importance ranking (planned)
- Blast radius for impact analysis (planned)

**Key Functions:**
- `BuildFromParsed()` - Populates graph from parsed files
- `ResolveCallEdges()` - Links function calls to definitions
- `GetCallers()/GetCallees()` - Navigate call graph

### Vector (vector/)

Ollama generates embeddings for semantic search:
- Uses `nomic-embed-text` model (768 dimensions)
- Embeddings stored in SQLite
- Cosine similarity for search

**Key Functions:**
- `Embed(text)` - Generate embedding via Ollama
- `Store(nodeID, embedding)` - Save to database
- `Search(embedding, k)` - Find k nearest neighbors

### MCP Server (mcp/)

Model Context Protocol integration for Claude Code:
- Runs as stdio server
- Provides tools for code navigation
- Returns compressed, relevant context

**Available Tools:**
- `search_context` - Semantic search
- `get_file_smart` - Intelligent file retrieval
- `trace_impact` - Change impact analysis

## Data Flow

```
Source Files → Parser → Graph Builder → SQLite Database
                                            ↓
                           Embeddings ← Ollama ← Text Representations
                                            ↓
                           MCP Server → Claude Code
```

## Database Schema

### nodes
| Column | Type | Description |
|--------|------|-------------|
| id | TEXT | Unique ID (file:name) |
| name | TEXT | Function/class name |
| type | TEXT | function/method/class |
| file | TEXT | File path |
| line | INT | Start line |
| end_line | INT | End line |
| signature | TEXT | Function signature |
| body | TEXT | First 500 chars |
| docstring | TEXT | Comments/docs |
| pagerank | REAL | Importance score |
| blast_radius | INT | Impact count |

### edges
| Column | Type | Description |
|--------|------|-------------|
| source | TEXT | Caller node ID |
| target | TEXT | Callee node ID |
| edge_type | TEXT | calls/imports |

### embeddings
| Column | Type | Description |
|--------|------|-------------|
| node_id | TEXT | Node ID |
| embedding | BLOB | 768-dim vector |

## Future Plans

- Go parser support
- Class hierarchy tracking
- Incremental re-indexing
- REST API + WebSocket
- React frontend with Monaco editor
- Graph visualization with D3
