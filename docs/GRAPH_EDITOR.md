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

**User Annotations:**
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

Response:
```json
{
  "status": "ok",
  "nodeId": "function_id"
}
```

### GET /api/node/full
Get node with annotations

```bash
curl http://localhost:8080/api/node/full?id=function_id
```

Response:
```json
{
  "node": {
    "id": "function_id",
    "name": "functionName",
    "type": "function",
    "file": "path/to/file.py",
    "line": 10,
    "comments": "Main handler",
    "tags": ["critical"],
    "custom_metadata": {
      "owner": "john"
    }
  },
  "annotations": {
    "comments": "Main handler",
    "tags": ["critical"],
    "custom_metadata": {
      "owner": "john"
    }
  }
}
```

### GET /api/files
Get list of all files in the graph

```bash
curl http://localhost:8080/api/files
```

Response:
```json
{
  "files": [
    "path/to/file1.py",
    "path/to/file2.py",
    "path/to/file3.ts"
  ]
}
```

## Workflow Example

### Scenario: Reviewing and Annotating a Service

1. **Navigate** to your service in the tree
2. **Click** the node to load its details
3. **Add comment**: "Main request handler for user authentication"
4. **Add tags**: "critical", "reviewed"
5. **Add metadata**: 
   - owner: "alice"
   - last_reviewed: "2026-04-05"
   - status: "stable"
6. **Click Save** - changes are persisted to database

### Result in MCP

When Claude Code asks:
```
@get_file_smart auth/handler.py authenticate
```

Claude Code receives:
- Function signature and code
- Your comments about what it does
- Tags indicating it's critical and reviewed
- Metadata showing alice owns it and it's stable

This gives Claude Code rich context without needing to read the entire file!

## Troubleshooting

### Frontend not loading files
- Ensure HTTP API server is running on port 8080
- Check browser console for CORS errors
- Verify code_graph.db exists in the project directory

### Annotations not saving
- Check that the backend API is running
- Verify database has write permissions
- Look for error messages in the HTTP API logs

### MCP not showing annotations
- Ensure you've saved annotations in the Graph Editor
- Check that `prism serve` is running with the correct database path
- Verify Claude Code is configured with the correct MCP server

## Architecture

The Graph Editor consists of three main components:

1. **Backend (Go)**
   - SQLite annotation tables (node_comments, node_tags, node_metadata)
   - REST API endpoints for CRUD operations
   - MCP integration for Claude Code

2. **Frontend (React)**
   - Tree view component for navigation
   - Edit panel for inline annotation editing
   - Real-time API sync

3. **Integration**
   - MCP server enhances Claude Code with annotated context
   - WebSocket support for real-time collaboration (future)

## Future Enhancements

- Real-time collaboration (multiple users editing simultaneously)
- Annotation versioning and history
- Advanced filtering and search by tags/metadata
- Export annotations to various formats
- Integration with code review tools
