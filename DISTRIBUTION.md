# 📖 Distribution Guide for Your Friend

## What Your Friend Gets

A **single installer** (`prism-installer.exe`) that does EVERYTHING automatically:
- ✅ Installs PRISM
- ✅ Adds to system PATH
- ✅ Configures Claude Code / Copilot CLI automatically
- ✅ No technical knowledge required!

---

## 📦 How to Give PRISM to Your Friend

### Step 1: Build the Distribution Package

Run this on your machine:
```powershell
.\build.ps1
```

This creates: `dist\prism-release\` folder with:
- `prism.exe` - Main program
- `prism-installer.exe` - Installer wizard  
- `README.md` - Documentation
- `QUICKSTART.md` - Quick start guide

### Step 2: Share the Files

**Option A - USB/Shared Folder:**
1. Zip the `dist\prism-release\` folder
2. Give `prism-release.zip` to your friend

**Option B - GitHub Release:**
1. Create a release on GitHub
2. Upload the files from `dist\prism-release\`
3. Send your friend the release link

---

## 🚀 What Your Friend Does (3 Steps!)

### Step 1: Download & Extract
Extract `prism-release.zip` to any folder

### Step 2: Run Installer
1. Right-click `prism-installer.exe`
2. Select "Run as administrator"
3. Follow the wizard:

```
╔════════════════════════════════════════════════╗
║       PRISM Installer - v1.0.0                 ║
║   Code Intelligence Platform                   ║
╚════════════════════════════════════════════════╝

This will install PRISM on your system.

Installation path: C:\Program Files\PRISM

Continue? [Y/n]: Y

📁 Creating installation directory...
   ✓ Created

📦 Installing PRISM...
   ✓ PRISM installed

🔧 Adding to system PATH...
   ✓ Added to PATH

🔍 Detecting AI coding assistants...
   ✓ Claude Code detected
   Configure PRISM for Claude Code? [Y/n]: Y
   ✓ Claude Code configured

🧠 Checking for Ollama (optional)...
   ℹ Ollama not found (optional)
   Install Ollama for semantic search: https://ollama.ai

╔════════════════════════════════════════════════╗
║           ✅ Installation Complete!            ║
╚════════════════════════════════════════════════╝
```

### Step 3: Use PRISM!

**Open a NEW terminal** (important for PATH reload), then:

```bash
# Navigate to any project
cd C:\Users\Friend\my-awesome-project

# Start PRISM (everything automatic!)
prism start

# Output:
🔮 PRISM - Starting Unified Server
📦 First time setup - indexing repository...
   Repository: C:\Users\Friend\my-awesome-project
  📄 Indexed 10 files...
  ✓ Parsed 47 files
🧠 Generating embeddings in background...
✅ Repository indexed successfully
🌐 PRISM Server running at http://localhost:8080
   📊 Web UI: http://localhost:8080
   🔌 API: http://localhost:8080/api
📡 MCP server ready (stdio transport)
```

**Then open browser:** http://localhost:8080

---

## 🤖 Using with Claude Code / Copilot CLI

The installer **automatically configures** this! After installation:

1. Restart Claude Code / Copilot CLI
2. PRISM tools are now available!
3. Try asking Claude: *"Use PRISM to find the authentication flow in this codebase"*

### Manual Configuration (if needed)

If auto-config didn't work, create/edit `~/.claude/config.json`:

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

---

## 📁 How PRISM Works Per-Project

Each project gets its own index:

```
my-project/
  .prism/
    code_graph.db    ← Auto-created on first use

other-project/
  .prism/
    code_graph.db    ← Separate index
```

**No conflicts!** Each project is independent.

---

## ⚡ Commands Your Friend Should Know

```bash
# Start PRISM (recommended - does everything)
prism start

# Start PRISM on custom port
prism start -port 9000

# View help
prism help

# Manual index (if needed)
prism index

# Search code semantically
prism search "authentication logic"
```

---

## 🛠️ Troubleshooting

### "prism is not recognized..."
- Close and reopen terminal (PATH needs reload)
- Or log out and log back in

### Web UI doesn't load
- Check if another process uses port 8080
- Use custom port: `prism start -port 9000`

### No semantic search results
- Install Ollama: https://ollama.ai
- Then: `ollama pull nomic-embed-text`
- Restart PRISM

### Claude Code doesn't see PRISM
- Restart Claude Code after installation
- Check `~/.claude/config.json` exists
- Manually configure if needed (see above)

---

## 📊 What Gets Installed

```
C:\Program Files\PRISM\
  prism.exe           ← Main binary (all-in-one)

C:\Users\Friend\.claude\
  config.json         ← Auto-configured (if Claude Code detected)

System PATH
  C:\Program Files\PRISM\   ← Added automatically
```

**Total size:** ~10-15 MB (everything embedded!)

---

## 🎯 Summary for Non-Technical Users

**PRISM = Smart code search + AI assistant integration**

1. **Run installer** (needs admin once)
2. **Open terminal** in any project
3. **Type `prism start`**
4. **Done!** Web UI + AI tools work automatically

**No Node.js, no Go, no build tools needed!**

---

## 🆘 Support

If your friend has issues:
- Check `QUICKSTART.md` in the release folder
- GitHub Issues: [your-repo-url]/issues
- Or contact you directly!
