# PRISM Build Script
# Compiles PRISM and installer for distribution

Write-Host "🔮 Building PRISM Distribution..." -ForegroundColor Cyan
Write-Host ""

# Step 1: Build frontend
Write-Host "📦 Building frontend..." -ForegroundColor Yellow
Push-Location frontend
npm run build
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Frontend build failed" -ForegroundColor Red
    Pop-Location
    exit 1
}
Pop-Location
Write-Host "✓ Frontend built" -ForegroundColor Green
Write-Host ""

# Step 2: Copy frontend to embed location
Write-Host "📂 Preparing embedded assets..." -ForegroundColor Yellow
New-Item -ItemType Directory -Force -Path "internal\server\frontend\dist" | Out-Null
Copy-Item -Recurse -Force "frontend\dist\*" "internal\server\frontend\dist\"
Write-Host "✓ Assets ready" -ForegroundColor Green
Write-Host ""

# Step 3: Build PRISM main binary
Write-Host "🔨 Building PRISM..." -ForegroundColor Yellow
go build -ldflags "-s -w" -o "dist\prism.exe" .
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ PRISM build failed" -ForegroundColor Red
    exit 1
}
Write-Host "✓ PRISM built: dist\prism.exe" -ForegroundColor Green
Write-Host ""

# Step 4: Build installer
Write-Host "🔨 Building installer..." -ForegroundColor Yellow
New-Item -ItemType Directory -Force -Path "dist" | Out-Null
go build -ldflags "-s -w -H windowsgui" -o "dist\prism-installer.exe" .\cmd\installer
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Installer build failed" -ForegroundColor Red
    exit 1
}
Write-Host "✓ Installer built: dist\prism-installer.exe" -ForegroundColor Green
Write-Host ""

# Step 5: Create release package
Write-Host "📦 Creating release package..." -ForegroundColor Yellow
$releaseDir = "dist\prism-release"
New-Item -ItemType Directory -Force -Path $releaseDir | Out-Null

# Copy files
Copy-Item "dist\prism.exe" "$releaseDir\prism.exe"
Copy-Item "dist\prism-installer.exe" "$releaseDir\prism-installer.exe"
Copy-Item "README.md" "$releaseDir\README.md"

# Create quick start guide
$quickStart = @"
# PRISM Quick Start

## Installation

### Option 1: Installer (Recommended)
1. Double-click `prism-installer.exe`
2. Follow the wizard
3. Restart your terminal
4. Done!

### Option 2: Manual
1. Copy `prism.exe` to a folder (e.g., C:\Tools\PRISM\)
2. Add that folder to your PATH
3. Restart your terminal

## Usage

### Quick Start
``````
cd your-project
prism start
``````
Then open http://localhost:8080

### For Claude Code / Copilot CLI
The installer will configure this automatically.

Manual configuration (.claude/config.json or .copilot/config.json):
``````json
{
  "mcpServers": {
    "prism": {
      "command": "prism",
      "args": ["start", "--mcp-only", "--auto-index"]
    }
  }
}
``````

### Optional: Install Ollama
For semantic search capabilities:
https://ollama.ai

## Support
- Documentation: https://github.com/ruffini/prism
- Issues: https://github.com/ruffini/prism/issues
"@

Set-Content -Path "$releaseDir\QUICKSTART.md" -Value $quickStart

Write-Host "✓ Release package created: $releaseDir" -ForegroundColor Green
Write-Host ""

# Summary
Write-Host "╔════════════════════════════════════════════════╗" -ForegroundColor Green
Write-Host "║         ✅ Build Complete!                     ║" -ForegroundColor Green
Write-Host "╚════════════════════════════════════════════════╝" -ForegroundColor Green
Write-Host ""
Write-Host "Distribution files:" -ForegroundColor Cyan
Write-Host "  • $releaseDir\prism.exe" -ForegroundColor White
Write-Host "  • $releaseDir\prism-installer.exe" -ForegroundColor White
Write-Host "  • $releaseDir\README.md" -ForegroundColor White
Write-Host "  • $releaseDir\QUICKSTART.md" -ForegroundColor White
Write-Host ""
Write-Host "To distribute:" -ForegroundColor Yellow
Write-Host "  1. Zip the $releaseDir folder" -ForegroundColor White
Write-Host "  2. Upload to GitHub Releases" -ForegroundColor White
Write-Host "  3. Users download and run prism-installer.exe" -ForegroundColor White
Write-Host ""
