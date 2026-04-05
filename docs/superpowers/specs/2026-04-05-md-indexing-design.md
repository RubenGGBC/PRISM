---
title: Markdown Indexing & Embedding via MCP
date: 2026-04-05
status: approved
---

# Markdown Indexing & Embedding via MCP

## Overview

Add two new MCP tools to the PRISM Platform that allow indexing and semantically searching project markdown documentation. These tools are independent from the existing code-graph tools and use dedicated storage.

## Goals

- Index `.md` files from predefined project locations into a SQLite-backed vector store
- Expose a `search_docs` MCP tool for semantic search over documentation
- Expose an `index_docs` MCP tool to trigger (re)indexing from the MCP client
- Keep documentation storage fully decoupled from the code graph

## Schema

Two new tables added via migration in `db/migrations.go`:

```sql
CREATE TABLE IF NOT EXISTS doc_chunks (
    id          TEXT PRIMARY KEY,   -- format: "relative/path.md#chunk_N"
    file        TEXT NOT NULL,      -- relative path from project root
    chunk_index INTEGER NOT NULL,
    line_start  INTEGER NOT NULL,   -- approximate line number of chunk start
    content     TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS doc_embeddings (
    chunk_id  TEXT PRIMARY KEY,
    embedding BLOB NOT NULL,
    FOREIGN KEY (chunk_id) REFERENCES doc_chunks(id) ON DELETE CASCADE
);
```

## Chunking Strategy

Implemented in a new package `docs` (`docs/chunker.go`):

- Split text into words
- Chunk size: **200 words**
- Overlap: **30 words** from the end of the previous chunk prepended to the next chunk
- Track `line_start` by counting newlines up to the chunk's word offset
- Chunk ID format: `"<relative_file_path>#chunk_<index>"`

## Auto-Discovery Paths

When no explicit path is given, `index_docs` scans these locations relative to the working directory:

1. `./docs/**/*.md`
2. `./.github/**/*.md`
3. `./*.md` (root-level markdown files only)

Duplicate files are deduplicated by resolved absolute path.

## MCP Tools

### `index_docs`

**Description:** Index project markdown files into the vector store for semantic search.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| path | string | no | Directory to scan for `.md` files. Defaults to `./docs`, `./.github`, and root `*.md` |

**Behavior:**
1. Discover `.md` files
2. For each file, read content and split into chunks (200 words, 30-word overlap)
3. Clear existing chunks for re-indexed files (`DELETE FROM doc_chunks WHERE file = ?`)
4. Insert chunks into `doc_chunks`
5. Generate embeddings via Ollama (same embedder already configured in MCPServer)
6. Store embeddings in `doc_embeddings`
7. Return stats: files indexed, chunks created, chunks skipped (already embedded), errors

**Error handling:**
- If Ollama is unavailable, return a clear error message listing which files failed
- If no `.md` files are found, return an informational message (not an error)

### `search_docs`

**Description:** Semantic search over indexed markdown documentation.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| query | string | yes | Natural language search query |
| limit | number | no | Max results to return (default: 5) |

**Behavior:**
1. Check `doc_embeddings` count; if 0, return "No docs indexed. Run index_docs first."
2. Embed the query via Ollama
3. Brute-force cosine similarity against all `doc_embeddings`
4. Join with `doc_chunks` to get file, line_start, content
5. Return top-k results formatted as markdown

**Result format per match:**
```
### 1. docs/README.md (line ~42, similarity: 87%)
relative/path.md

> chunk text here...
```

## Files Changed / Created

| File | Action |
|------|--------|
| `docs/chunker.go` | New — `ChunkText(content, file string) []DocChunk` |
| `db/migrations.go` | Add `createDocChunksTable` and `createDocEmbeddingsTable` calls |
| `vector/doc_store.go` | New — `DocVectorStore` wrapping doc-specific embed/search on SQLite |
| `mcp/server.go` | Add `handleIndexDocs`, `handleSearchDocs` handlers; register tools |
| `mcp/tools.go` | Add `index_docs` and `search_docs` to `GetToolDefinitions()` |

## Data Flow

```
index_docs call
    → discover .md files
    → docs.ChunkText(content, file)
    → DELETE existing chunks for file
    → INSERT doc_chunks rows
    → embedder.EmbedBatch(chunk texts)
    → doc_store.StoreBatch(chunk_ids, embeddings)
    → return stats

search_docs call
    → embedder.Embed(query)
    → doc_store.Search(queryEmbedding, limit)
    → JOIN doc_chunks for content + metadata
    → return formatted results
```

## Out of Scope

- Incremental indexing (only changed files): not in this iteration; full re-index per file on each call
- Non-markdown formats (HTML, PDF, RST)
- Chunking by heading (section-aware)
- A `list_docs` tool
