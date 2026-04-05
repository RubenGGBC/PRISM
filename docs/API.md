# PRISM Platform - API Reference

## REST API Endpoints

### Files and Listing

#### GET `/api/files`

Returns list of all indexed files in the repository.

**Query Parameters:**
None

**Response:**
```json
{
  "files": [
    "src/auth.ts",
    "src/db.ts",
    "src/utils/helpers.ts",
    "src/components/Button.tsx"
  ]
}
```

**Example:**
```bash
curl http://localhost:8080/api/files
```

---

### Node Information

#### GET `/api/node`

Get detailed information about a specific node (function, class, etc.)

**Query Parameters:**
- `id` (required) - Node identifier (format: `"file.ts:functionName"`)

**Response:**
```json
{
  "id": "auth.ts:login",
  "name": "login",
  "type": "function",
  "file": "auth.ts",
  "signature": "export function login(user: User, password: string): Promise<void>",
  "body": "{ ... code content ... }",
  "line": 10,
  "end_line": 25,
  "metadata": {
    "deprecated": null,
    "hook": ["beforeAuth", "afterSession"],
    "todo": ["optimize performance"],
    "author": ["John Doe", "Jane Smith"]
  }
}
```

**Example:**
```bash
curl "http://localhost:8080/api/node?id=auth.ts:login"
```

---

### Search

#### GET `/api/search`

Semantic search across indexed code elements using LanceDB vector embeddings.

**Query Parameters:**
- `q` (required) - Search query (natural language or keywords)

**Response:**
```json
{
  "results": [
    {
      "id": "auth.ts:login",
      "name": "login",
      "type": "function",
      "file": "auth.ts",
      "similarity": 0.92,
      "signature": "export function login(...)"
    },
    {
      "id": "auth.ts:authenticate",
      "name": "authenticate",
      "type": "function",
      "file": "auth.ts",
      "similarity": 0.87,
      "signature": "export function authenticate(...)"
    }
  ],
  "query": "user authentication",
  "count": 2
}
```

**Example:**
```bash
curl "http://localhost:8080/api/search?q=user%20authentication"
```

---

#### GET `/api/search/metadata`

Search for code elements by metadata annotations.

**Query Parameters:**
- `type` (required) - Metadata type: `deprecated`, `hook`, `todo`, `author`
- `value` (optional) - Specific value to search for

**Response:**
```json
{
  "results": [
    {
      "id": "auth.ts:oldLogin",
      "name": "oldLogin",
      "type": "function",
      "file": "auth.ts",
      "metadata": {
        "deprecated": "use login instead"
      }
    }
  ],
  "type": "deprecated",
  "count": 1
}
```

**Example:**
```bash
curl "http://localhost:8080/api/search/metadata?type=deprecated"
curl "http://localhost:8080/api/search/metadata?type=author&value=John%20Doe"
```

---

### Graph Analysis

#### GET `/api/dependencies`

Get all dependencies (outgoing calls) for a specific node.

**Query Parameters:**
- `id` (required) - Node identifier

**Response:**
```json
{
  "id": "auth.ts:login",
  "dependencies": [
    {
      "id": "db.ts:findUser",
      "name": "findUser",
      "type": "function",
      "file": "db.ts"
    },
    {
      "id": "utils.ts:hashPassword",
      "name": "hashPassword",
      "type": "function",
      "file": "utils.ts"
    }
  ],
  "count": 2
}
```

**Example:**
```bash
curl "http://localhost:8080/api/dependencies?id=auth.ts:login"
```

---

#### GET `/api/dependents`

Get all dependents (incoming calls) for a specific node.

**Query Parameters:**
- `id` (required) - Node identifier

**Response:**
```json
{
  "id": "auth.ts:login",
  "dependents": [
    {
      "id": "api.ts:handleAuthRequest",
      "name": "handleAuthRequest",
      "type": "function",
      "file": "api.ts"
    }
  ],
  "count": 1
}
```

**Example:**
```bash
curl "http://localhost:8080/api/dependents?id=auth.ts:login"
```

---

#### GET `/api/impact`

Get blast radius - all functions affected if a node changes.

**Query Parameters:**
- `id` (required) - Node identifier

**Response:**
```json
{
  "id": "auth.ts:validateToken",
  "direct_dependents": 2,
  "transitive_dependents": 5,
  "impact": [
    {
      "id": "api.ts:authMiddleware",
      "depth": 1,
      "type": "direct"
    },
    {
      "id": "api.ts:handleRequest",
      "depth": 2,
      "type": "transitive"
    }
  ]
}
```

**Example:**
```bash
curl "http://localhost:8080/api/impact?id=auth.ts:validateToken"
```

---

## WebSocket API

### Connection

**Endpoint:** `ws://localhost:8080/ws`

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');
```

---

### Message Types

#### Get File Data

Request file contents and elements:

```json
{
  "action": "get_file",
  "file": "src/auth.ts"
}
```

Response:

```json
{
  "type": "file_data",
  "file": "src/auth.ts",
  "data": [
    {
      "id": "auth.ts:login",
      "name": "login",
      "type": "function",
      "signature": "...",
      "line": 10
    }
  ]
}
```

---

#### Semantic Search

Send search request:

```json
{
  "action": "search",
  "query": "authentication logic"
}
```

Response:

```json
{
  "type": "search_results",
  "query": "authentication logic",
  "results": [
    {
      "id": "auth.ts:login",
      "name": "login",
      "similarity": 0.92
    }
  ]
}
```

---

#### Get Dependencies

Request dependency graph:

```json
{
  "action": "get_dependencies",
  "node_id": "auth.ts:login"
}
```

Response:

```json
{
  "type": "dependencies",
  "node_id": "auth.ts:login",
  "dependencies": [
    {
      "id": "db.ts:findUser",
      "type": "function"
    }
  ]
}
```

---

## MCP Server Tools

The platform also provides a Model Context Protocol (MCP) server for integration with Claude Code.

### search_context

Semantic search across the codebase.

**Input:**
```json
{
  "query": "how to authenticate users"
}
```

**Output:**
```json
{
  "results": [
    {
      "id": "auth.ts:login",
      "name": "login",
      "type": "function",
      "file": "auth.ts",
      "score": 0.92
    }
  ]
}
```

---

### get_file_smart

Get function or class with callers and callees.

**Input:**
```json
{
  "file": "auth.ts",
  "symbol": "login"
}
```

**Output:**
```json
{
  "id": "auth.ts:login",
  "name": "login",
  "type": "function",
  "signature": "export function login(...)",
  "body": "...",
  "callers": ["api.ts:handleAuthRequest"],
  "callees": ["db.ts:findUser", "utils.ts:hashPassword"]
}
```

---

### list_functions

List all functions in a file or matching a pattern.

**Input:**
```json
{
  "file": "auth.ts"
}
```

**Output:**
```json
{
  "functions": [
    {
      "id": "auth.ts:login",
      "name": "login",
      "type": "function",
      "line": 10
    },
    {
      "id": "auth.ts:logout",
      "name": "logout",
      "type": "function",
      "line": 45
    }
  ]
}
```

---

### trace_impact

Show blast radius - what functions would be affected by changing a node.

**Input:**
```json
{
  "function_id": "auth.ts:validateToken"
}
```

**Output:**
```json
{
  "function_id": "auth.ts:validateToken",
  "direct_dependents": 2,
  "transitive_dependents": 5,
  "affected_functions": [
    {
      "id": "api.ts:authMiddleware",
      "depth": 1
    }
  ]
}
```

---

## Error Responses

All endpoints return error responses in the following format:

```json
{
  "error": "Description of what went wrong",
  "code": "ERROR_CODE"
}
```

Common error codes:
- `INVALID_QUERY` - Query parameter is missing or invalid
- `NOT_FOUND` - Requested resource not found
- `PARSE_ERROR` - Error parsing the request
- `DATABASE_ERROR` - Database query failed
- `INTERNAL_ERROR` - Unexpected server error

**HTTP Status Codes:**
- `200 OK` - Successful request
- `400 Bad Request` - Invalid query parameters
- `404 Not Found` - Resource not found
- `500 Internal Server Error` - Server error

---

## Rate Limiting

No rate limiting is currently implemented for local deployments. For production deployments, implement rate limiting as needed.

---

## Examples

### Search for Authentication Functions

```bash
curl "http://localhost:8080/api/search?q=user%20authentication"
```

### Get Dependencies for a Function

```bash
curl "http://localhost:8080/api/dependencies?id=auth.ts:login"
```

### Find All Deprecated Functions

```bash
curl "http://localhost:8080/api/search/metadata?type=deprecated"
```

### Analyze Impact of Changes

```bash
curl "http://localhost:8080/api/impact?id=auth.ts:validateToken"
```

### WebSocket Search Example

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onopen = () => {
  ws.send(JSON.stringify({
    action: 'search',
    query: 'authentication'
  }));
};

ws.onmessage = (event) => {
  const response = JSON.parse(event.data);
  console.log('Search results:', response.results);
};
```
