# 🍎 PRISM Development Setup - macOS

## Prerequisites

Install these if you don't have them:

### 1. Homebrew (optional but recommended)
```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

### 2. Go 1.21+
```bash
# Option 1: Homebrew
brew install go

# Option 2: Download from
# https://go.dev/dl/
```

### 3. Node.js 18+
```bash
# Option 1: Homebrew
brew install node

# Option 2: Download from
# https://nodejs.org/
```

### 4. Ollama (optional - for semantic search)
```bash
# Download from: https://ollama.ai
# Or via Homebrew:
brew install ollama
ollama serve  # Start in background
ollama pull nomic-embed-text
```

---

## 🚀 Quick Start (After Cloning)

### Option 1: Automated Build (Recommended)
```bash
# Clone the repo
git clone https://github.com/RubenGGBC/PRISM.git
cd PRISM

# Make build script executable
chmod +x build.sh

# Run build script
./build.sh

# Run PRISM
./dist/prism start
```

### Option 2: Manual Build
```bash
# 1. Build frontend
cd frontend
npm install
npm run build
cd ..

# 2. Prepare embedded assets
mkdir -p internal/server/frontend/dist
cp -r frontend/dist/* internal/server/frontend/dist/

# 3. Download Go dependencies
go mod download

# 4. Build PRISM
go build -o dist/prism .

# 5. Run
./dist/prism start
```

---

## 📦 Installation (Make it Global)

After building, install globally:

```bash
sudo mv dist/prism /usr/local/bin/
```

Now you can run from anywhere:
```bash
cd ~/my-project
prism start
```

---

## 🔧 Development Workflow

### Run in Development Mode

**Backend (Go):**
```bash
# Auto-rebuild on changes (requires: go install github.com/cosmtrek/air@latest)
air

# Or manual:
go run . start
```

**Frontend (React):**
```bash
cd frontend
npm run dev
# Opens at http://localhost:5173
```

**Full Stack:**
```bash
# Terminal 1: Backend
go run . start -port 8080

# Terminal 2: Frontend dev server (optional, for hot reload)
cd frontend
npm run dev
```

---

## 🧪 Testing

```bash
# Run all tests
go test -v ./...

# Run specific package tests
go test -v ./parser
go test -v ./graph
go test -v ./vector
```

---

## 🐛 Troubleshooting

### "go: module not found"
```bash
go mod tidy
go mod download
```

### "npm: command not found"
```bash
brew install node
```

### "Frontend not loading"
```bash
# Rebuild frontend
cd frontend
rm -rf dist node_modules
npm install
npm run build
cd ..

# Re-copy assets
rm -rf internal/server/frontend/dist
mkdir -p internal/server/frontend/dist
cp -r frontend/dist/* internal/server/frontend/dist/

# Rebuild
go build -o dist/prism .
```

### Port 8080 already in use
```bash
# Use different port
./dist/prism start -port 9000
```

---

## 📁 Project Structure

```
PRISM/
├── main.go                 # Entry point
├── internal/
│   └── server/            # Unified server
│       ├── server.go      # HTTP + MCP server
│       ├── embedded.go    # Embedded frontend
│       └── frontend/      # Build artifacts (generated)
│           └── dist/
├── api/                   # REST API
├── db/                    # SQLite database
├── graph/                 # Code graph
├── mcp/                   # MCP server
├── parser/                # Tree-sitter parsers
├── vector/                # Embeddings & search
├── watcher/               # File watcher
└── frontend/              # React UI
    ├── src/
    ├── dist/              # Build output
    └── package.json
```

---

## 🎯 Common Commands

```bash
# Start PRISM (auto-index, auto-embed, serve)
prism start

# Index a specific directory
prism index -repo /path/to/code

# Generate embeddings
prism embed -db code_graph.db

# Search code
prism search "authentication flow"

# Start MCP server only (for Claude Code)
prism start -mcp-only

# Parse a single file
prism parse src/main.go
```

---

## 🔄 Updating Dependencies

### Frontend:
```bash
cd frontend
npm update
npm run build
```

### Backend:
```bash
go get -u ./...
go mod tidy
```

---

## 📝 Notes for macOS

- **No installer needed** - Just build and copy to `/usr/local/bin/`
- **Permissions**: May need `chmod +x` on scripts
- **M1/M2 Macs**: Everything works natively (ARM64)
- **Ollama**: Best installed via official installer (not always in Homebrew)

---

## 🆘 Get Help

- Issues: https://github.com/RubenGGBC/PRISM/issues
- Discussions: https://github.com/RubenGGBC/PRISM/discussions
- Docs: https://github.com/RubenGGBC/PRISM/tree/main/docs
