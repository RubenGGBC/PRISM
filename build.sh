#!/bin/bash
# PRISM Build Script for macOS/Linux
# Run this after cloning the repository

set -e

echo "🔮 Building PRISM..."
echo ""

# Check dependencies
echo "📋 Checking dependencies..."
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Install from: https://go.dev/dl/"
    exit 1
fi
echo "  ✓ Go found: $(go version)"

if ! command -v npm &> /dev/null; then
    echo "❌ Node.js is not installed. Install from: https://nodejs.org/"
    exit 1
fi
echo "  ✓ Node.js found: $(node --version)"
echo ""

# Step 1: Build frontend
echo "📦 Building frontend..."
cd frontend
npm install
npm run build
cd ..
echo "  ✓ Frontend built"
echo ""

# Step 2: Copy frontend to embed location
echo "📂 Preparing embedded assets..."
mkdir -p internal/server/frontend/dist
cp -r frontend/dist/* internal/server/frontend/dist/
echo "  ✓ Assets ready"
echo ""

# Step 3: Download Go dependencies
echo "📥 Downloading Go dependencies..."
go mod download
echo "  ✓ Dependencies ready"
echo ""

# Step 4: Build PRISM binary
echo "🔨 Building PRISM..."
mkdir -p dist
go build -ldflags "-s -w" -o dist/prism .
echo "  ✓ PRISM built: dist/prism"
echo ""

# Summary
echo "╔════════════════════════════════════════════════╗"
echo "║         ✅ Build Complete!                     ║"
echo "╚════════════════════════════════════════════════╝"
echo ""
echo "🚀 Quick Start:"
echo "   ./dist/prism start"
echo ""
echo "Or install globally:"
echo "   sudo mv dist/prism /usr/local/bin/"
echo "   prism start"
echo ""
