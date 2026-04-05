# PRISM — Claude Usage Guide

## MCP Tools Available

| Tool | Use for | Cost |
|------|---------|------|
| `search_context` | Find code by meaning | ~200 tokens |
| `get_file_smart` | Get one function + callers/callees | ~300 tokens |
| `trace_impact` | What breaks if I change X | ~150 tokens |
| `list_functions` | Browse functions in a file | ~100 tokens |
| `index_docs` | Index project .md files for search | one-time |
| `search_docs` | Search documentation semantically | ~200 tokens |

## Workflow

**Any code question → search first, read never**

```
search_context("user authentication flow")     # find relevant code
get_file_smart("auth/login.ts", "loginUser")   # drill into specific symbol
trace_impact("auth/login.ts:loginUser")        # check blast radius before editing
```

**Any docs question → search_docs**

```
search_docs("how to configure the MCP server")
```
Run `index_docs` once (or after adding .md files) to index `./docs`, `./.github`, and root `*.md`.

## Token Budget

- Full file read: 5,000–30,000 tokens
- `search_context`: ~200–600 tokens
- `get_file_smart`: ~300–800 tokens
- **Savings: 90–95% per query**

## Rules

- Never `Read` a file to answer a code question — use `search_context` first
- Never read a whole file to find one function — use `get_file_smart`
- Always `trace_impact` before proposing changes to shared functions
- `search_docs` works with keyword fallback if Ollama is unavailable
- For large functions (>50 lines), `get_file_smart` truncates the body — search by subfunctionality instead: `search_context("end turn economy")` > `get_file_smart("turn_service.py", "end_turn")`
