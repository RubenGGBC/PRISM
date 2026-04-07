# 🔮 PRISM

**Program Representation & Intelligence Semantic Mapper**

An AI-powered code intelligence platform that understands your codebase through semantic search, dependency analysis, and intelligent code visualization. Integrates seamlessly with Claude Code, Copilot CLI, and other AI tools via MCP.

[![Release](https://img.shields.io/github/v/release/RubenGGBC/PRISM)](https://github.com/RubenGGBC/PRISM/releases)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## What PRISM Does

PRISM indexes your code repository and creates an intelligent semantic map. Instead of grepping for keywords, ask PRISM questions about your code in natural language: *"Show me the authentication flow"* or *"What breaks if I change this function?"* It understands meaning, not just text.

## ✨ Features

- 🔍 **Semantic Search** — Find code by what it does, not just keywords
- 📊 **Dependency Graph** — Visualize function calls, impact radius, and code relationships
- 🏷️ **Code Annotations** — Auto-extract `@deprecated`, `@hook`, `@todo`, `@author` metadata
- 🤖 **AI Integration** — Full MCP server support for Claude Code and Copilot CLI
- 🎨 **Interactive Web UI** — React frontend with Monaco editor and D3 visualizations (embedded!)
- ⚡ **Semantic Embeddings** — Powered by Ollama for local, private embeddings
- 🚀 **One-Command Setup** — `prism start` does everything automatically

---

## 📦 Installation

### For End Users (Easiest)

**Windows:**
1. Download `prism-installer-windows.exe` from [Releases](https://github.com/RubenGGBC/PRISM/releases)
2. Run as administrator
3. Follow the wizard
4. Done! Open terminal and run: `prism start`

**macOS:**
```bash
# Download and install
curl -L https://github.com/RubenGGBC/PRISM/releases/latest/download/prism-macos -o prism
chmod +x prism
sudo mv prism /usr/local/bin/

# Or with Homebrew (coming soon)
brew install prism
```

**Linux:**
```bash
curl -L https://github.com/RubenGGBC/PRISM/releases/latest/download/prism-linux.tar.gz | tar xz
chmod +x prism
sudo mv prism /usr/local/bin/
```

### For Developers (Build from Source)

**macOS/Linux:**
```bash
git clone https://github.com/RubenGGBC/PRISM.git
cd PRISM
chmod +x build.sh
./build.sh
```

**Windows:**
```powershell
git clone https://github.com/RubenGGBC/PRISM.git
cd PRISM
.\build.ps1
```

**See detailed guides:**
- [macOS Development Setup](MACOS_SETUP.md)
- [Distribution Guide](DISTRIBUTION.md)

---

## 🚀 Quick Start

### The Easy Way (Recommended)

```bash
# Navigate to your project
cd /path/to/your/project

# Start PRISM (auto-index, auto-embed, serve)
prism start

# That's it! 🎉
```

**What just happened:**
- ✅ Auto-indexed your code (first run only)
- ✅ Generated embeddings in background
- ✅ Started web UI at **http://localhost:8080**
- ✅ Started MCP server for AI assistants
- ✅ Created `.prism/` folder with database

**Access PRISM:**
- 🌐 **Web UI:** http://localhost:8080
- 🔌 **API:** http://localhost:8080/api
- 🤖 **MCP:** Auto-configured if using Claude Code/Copilot CLI

### Advanced Commands

```bash
# Index a specific directory
prism index -repo /path/to/code

# Generate embeddings manually
prism embed -db .prism/code_graph.db

# Search semantically
prism search "authentication flow"

# Start on custom port
prism start -port 9000

# MCP server only (no web UI)
prism start -mcp-only

# Parse a single file
prism parse src/main.go
```

---

## ⚙️ Configuration

PRISM works with **zero configuration** out of the box. All settings have sensible defaults.

### Optional: Custom Configuration

Create `config.json` in your project root:

```json
{
  "database": ".prism/code_graph.db",
  "ollama": "http://localhost:11434",
  "model": "nomic-embed-text",
  "port": 8080,
  "indexPath": ".",
  "supportedLanguages": ["go", "typescript", "python", "javascript"]
}
```

| Option | Default | Description |
|--------|---------|-------------|
| `database` | `.prism/code_graph.db` | SQLite database path |
| `ollama` | `http://localhost:11434` | Ollama server URL |
| `model` | `nomic-embed-text` | Embedding model name |
| `port` | `8080` | Web server port |
| `indexPath` | `.` | Directory to index |
| `supportedLanguages` | Multiple | Languages to parse |

---

## 🔌 Claude Code / Copilot CLI Integration

PRISM works seamlessly with AI coding assistants via MCP (Model Context Protocol).

### Auto-Configuration (Windows Installer)

If you used `prism-installer.exe`, it already configured Claude Code/Copilot CLI automatically! 🎉

### Manual Configuration

**Claude Code** (`~/.claude/config.json`):
```json
{
  "mcpServers": {
    "prism": {
      "command": "prism",
      "args": ["start", "--mcp-only", "--auto-index"]
    }
  }
}
```

**Copilot CLI** (`~/.copilot/config.json`):
```json
{
  "mcpServers": {
    "prism": {
      "command": "prism",
      "args": ["start", "--mcp-only", "--auto-index"]
    }
  }
}
```

### Available MCP Tools

Once configured, your AI assistant can use:

- **`search_context`** — Semantic search across your codebase
- **`get_file_smart`** — Get code with intelligent compression
- **`trace_impact`** — Analyze what breaks if you change a function
- **`list_functions`** — Browse all functions in a file
- **`index_docs`** — Index markdown documentation
- **`search_docs`** — Search documentation semantically

### Usage Example

Ask your AI assistant:
```
"Use PRISM to find the authentication flow in this codebase"
"What would break if I renamed this function?"
"Show me all deprecated functions"
```

---

## 📚 Available Commands

| Command | Purpose | Example |
|---------|---------|---------|
| `prism start` | **Start everything** (auto-index, embed, serve) | `prism start` |
| `prism start -port 9000` | Start on custom port | `prism start -port 9000` |
| `prism start -mcp-only` | MCP server only (for Claude Code) | `prism start -mcp-only` |
| `prism index` | Index current directory | `prism index` |
| `prism index -repo <path>` | Index specific directory | `prism index -repo ./my-code` |
| `prism embed` | Generate embeddings | `prism embed` |
| `prism search <query>` | Semantic code search | `prism search "auth flow"` |
| `prism parse <file>` | Parse single file | `prism parse main.go` |
| `prism export` | Export annotations to CLAUDE.md | `prism export` |
| `prism help` | Show help | `prism help` |

---

## 🛠️ Development

### Prerequisites

- **Go 1.21+**
- **Node.js 18+**
- **Ollama** (optional, for embeddings) — [Install Guide](https://ollama.ai)

### Build from Source

**macOS/Linux:**
```bash
git clone https://github.com/RubenGGBC/PRISM.git
cd PRISM
chmod +x build.sh
./build.sh
```

**Windows:**
```powershell
git clone https://github.com/RubenGGBC/PRISM.git
cd PRISM
.\build.ps1
```

### Running Tests

```bash
go test -v ./...
```

### Development Mode

**Backend (with hot reload):**
```bash
# Install air for hot reload
go install github.com/cosmtrek/air@latest

# Run with hot reload
air
```

**Frontend (with hot reload):**
```bash
cd frontend
npm install
npm run dev
# Opens at http://localhost:5173
```

**Full Stack Development:**
```bash
# Terminal 1: Backend
go run . start -port 8080

# Terminal 2: Frontend (optional, for UI development)
cd frontend
npm run dev
```

---

## 📖 Documentation

- **[macOS Setup Guide](MACOS_SETUP.md)** — Detailed macOS installation and development
- **[Distribution Guide](DISTRIBUTION.md)** — How to share PRISM with others
- **[Architecture Details](docs/architecture.md)** — How PRISM works internally
- **[MCP Setup Guide](docs/MCP_SETUP.md)** — Claude Code/Copilot CLI integration
- **[API Reference](docs/API.md)** — REST API documentation
- **[Testing Guide](docs/HOW_TO_TEST.md)** — How to run tests

---

## 📁 Project Structure

```
PRISM/
├── main.go                    # Entry point with unified 'start' command
├── internal/
│   └── server/               # Unified server (MCP + HTTP + Frontend)
│       ├── server.go         # Server implementation
│       ├── embedded.go       # Embedded frontend
│       └── frontend/         # Built frontend (auto-generated)
├── cmd/
│   └── installer/            # Windows installer wizard
├── api/                      # REST API endpoints
├── db/                       # SQLite database layer
├── graph/                    # Code graph builder
├── mcp/                      # MCP server implementation
├── parser/                   # Tree-sitter language parsers
├── vector/                   # Embeddings & semantic search
├── watcher/                  # File system watcher
├── frontend/                 # React UI
│   ├── src/                  # Source code
│   └── dist/                 # Build output
├── build.sh                  # macOS/Linux build script
├── build.ps1                 # Windows build script
└── .github/
    └── workflows/            # CI/CD for releases
```

---

## 📖 Architecture

```
Your Codebase
    ↓
[Tree-Sitter Parser]
    ↓
Functions, Classes, Dependencies
    ↓
[SQLite Graph Builder]
    ↓
Dependency Graph + Metadata
    ↓
[Ollama Embeddings]
    ↓
Semantic Vector Database
    ↓
[Unified Server: MCP + HTTP + React UI] ← AI Assistants / Browser
```

**Design Principles:**
- 🔒 **Local processing** — No cloud APIs, everything runs on your machine
- 🧠 **Semantic understanding** — Embeddings capture code meaning, not just keywords
- ⚡ **Performance** — SQLite for structure, vector DB for similarity
- 🔌 **Extensibility** — MCP interface works with any compatible AI tool

---

## 🎯 Use Cases

- **Code Review** — "Show me all error handling in the payment module"
- **Onboarding** — "Where is the authentication middleware?"
- **Refactoring** — "What breaks if I rename this function?"
- **Debugging** — "What calls this problematic function?"
- **Documentation** — "Which functions are deprecated?"
- **AI-Assisted Development** — Integrate with Claude Code/Copilot CLI

---

## 🚨 Troubleshooting

### "prism: command not found"

**Windows:**
- Close and reopen terminal after installation
- Or log out and log back in (PATH needs reload)

**macOS/Linux:**
```bash
# Check if prism is in PATH
which prism

# If not, reinstall or add manually
sudo mv prism /usr/local/bin/
```

### Port 8080 already in use

```bash
prism start -port 9000
```

### "No embeddings found"

```bash
# Make sure Ollama is running
ollama serve

# Pull the embedding model
ollama pull nomic-embed-text

# Restart PRISM
prism start
```

### Web UI not loading

```bash
# Rebuild everything
cd frontend
rm -rf dist node_modules
npm install
npm run build

# Rebuild PRISM
cd ..
./build.sh  # macOS/Linux
# or
.\build.ps1  # Windows
```

### Claude Code/Copilot CLI not detecting PRISM

1. Check config file exists: `~/.claude/config.json` or `~/.copilot/config.json`
2. Verify PRISM is in PATH: `which prism` (Mac/Linux) or `where prism` (Windows)
3. Restart your AI assistant
4. Check PRISM is running: `prism start -mcp-only`

### Database locked error

```bash
# Another PRISM process is running
# Find and stop it:
ps aux | grep prism  # macOS/Linux
tasklist | findstr prism  # Windows

# Kill the process or restart your computer
```

---

## 🤝 Contributing

Issues and PRs welcome! Please ensure tests pass:

```bash
go test -v ./...
```

### Reporting Bugs

Please include:
- OS and version (Windows/macOS/Linux)
- PRISM version (`prism help`)
- Steps to reproduce
- Error messages (if any)

---

## 📄 License

MIT License - See [LICENSE](LICENSE) file for details

---

## 🙏 Acknowledgments

- Tree-sitter for amazing code parsing
- Ollama for local embeddings
- MCP protocol for AI integration
- The open source community

---

## 📞 Support

- **Issues:** [GitHub Issues](https://github.com/RubenGGBC/PRISM/issues)
- **Discussions:** [GitHub Discussions](https://github.com/RubenGGBC/PRISM/discussions)
- **Documentation:** [docs/](./docs)

---

## 🗺️ Roadmap

- [ ] VS Code Extension
- [ ] Homebrew formula for macOS
- [ ] More language support (C++, Ruby, PHP)
- [ ] Cloud deployment option
- [ ] Team collaboration features
- [ ] Custom embedding models
- [ ] Plugin system

---

**Made with ❤️ for developers who love understanding code**
