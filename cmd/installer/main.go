package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/windows/registry"
)

const (
	defaultInstallPath = "C:\\Program Files\\PRISM"
	productName        = "PRISM"
	version            = "1.0.0"
)

func main() {
	fmt.Println("╔════════════════════════════════════════════════╗")
	fmt.Println("║       PRISM Installer - v" + version + "              ║")
	fmt.Println("║   Code Intelligence Platform               ║")
	fmt.Println("╚════════════════════════════════════════════════╝")
	fmt.Println()

	// Check if running as administrator
	if !isAdmin() {
		fmt.Println("❌ This installer requires administrator privileges.")
		fmt.Println("   Please right-click and select 'Run as administrator'")
		fmt.Println()
		fmt.Print("Press Enter to exit...")
		fmt.Scanln()
		os.Exit(1)
	}

	// Welcome
	fmt.Println("This will install PRISM on your system.")
	fmt.Println()
	fmt.Printf("Installation path: %s\n", defaultInstallPath)
	fmt.Println()
	fmt.Print("Continue? [Y/n]: ")
	
	var response string
	fmt.Scanln(&response)
	if response != "" && response != "Y" && response != "y" {
		fmt.Println("Installation cancelled.")
		os.Exit(0)
	}

	// Step 1: Create installation directory
	fmt.Println()
	fmt.Println("📁 Creating installation directory...")
	if err := os.MkdirAll(defaultInstallPath, 0755); err != nil {
		fatal("Failed to create installation directory", err)
	}
	fmt.Println("   ✓ Created")

	// Step 2: Copy prism.exe to installation directory
	fmt.Println()
	fmt.Println("📦 Installing PRISM...")
	exePath, err := os.Executable()
	if err != nil {
		fatal("Failed to get installer path", err)
	}
	
	// Look for prism.exe in the same directory as installer
	installerDir := filepath.Dir(exePath)
	prismSource := filepath.Join(installerDir, "prism.exe")
	prismDest := filepath.Join(defaultInstallPath, "prism.exe")
	
	if err := copyFile(prismSource, prismDest); err != nil {
		fatal("Failed to copy prism.exe", err)
	}
	fmt.Println("   ✓ PRISM installed")

	// Step 3: Add to PATH
	fmt.Println()
	fmt.Println("🔧 Adding to system PATH...")
	if err := addToPath(defaultInstallPath); err != nil {
		fmt.Printf("   ⚠ Warning: Failed to add to PATH: %v\n", err)
		fmt.Printf("   You may need to add '%s' to PATH manually\n", defaultInstallPath)
	} else {
		fmt.Println("   ✓ Added to PATH")
	}

	// Step 4: Detect Claude Code / Copilot CLI
	fmt.Println()
	fmt.Println("🔍 Detecting AI coding assistants...")
	
	claudeConfigPath := detectClaudeCode()
	copilotConfigPath := detectCopilotCLI()
	
	if claudeConfigPath != "" {
		fmt.Println("   ✓ Claude Code detected")
		fmt.Print("   Configure PRISM for Claude Code? [Y/n]: ")
		var resp string
		fmt.Scanln(&resp)
		if resp == "" || resp == "Y" || resp == "y" {
			if err := configureClaudeCode(claudeConfigPath); err != nil {
				fmt.Printf("   ⚠ Failed to configure: %v\n", err)
			} else {
				fmt.Println("   ✓ Claude Code configured")
			}
		}
	}
	
	if copilotConfigPath != "" {
		fmt.Println("   ✓ Copilot CLI detected")
		fmt.Print("   Configure PRISM for Copilot CLI? [Y/n]: ")
		var resp string
		fmt.Scanln(&resp)
		if resp == "" || resp == "Y" || resp == "y" {
			if err := configureCopilotCLI(copilotConfigPath); err != nil {
				fmt.Printf("   ⚠ Failed to configure: %v\n", err)
			} else {
				fmt.Println("   ✓ Copilot CLI configured")
			}
		}
	}
	
	if claudeConfigPath == "" && copilotConfigPath == "" {
		fmt.Println("   ℹ No AI coding assistants detected")
		fmt.Println("   You can configure PRISM manually later")
	}

	// Step 5: Check for Ollama
	fmt.Println()
	fmt.Println("🧠 Checking for Ollama (optional)...")
	if isOllamaInstalled() {
		fmt.Println("   ✓ Ollama detected")
	} else {
		fmt.Println("   ℹ Ollama not found (optional)")
		fmt.Println("   Install Ollama for semantic search: https://ollama.ai")
	}

	// Installation complete
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════╗")
	fmt.Println("║           ✅ Installation Complete!            ║")
	fmt.Println("╚════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("Quick Start:")
	fmt.Println("  1. Open a NEW terminal (to reload PATH)")
	fmt.Println("  2. Navigate to your project: cd C:\\path\\to\\project")
	fmt.Println("  3. Run: prism start")
	fmt.Println("  4. Open http://localhost:8080")
	fmt.Println()
	fmt.Println("For Claude Code/Copilot users:")
	fmt.Println("  PRISM is now available as an MCP server!")
	fmt.Println("  Restart your AI assistant to use PRISM tools.")
	fmt.Println()
	fmt.Print("Press Enter to exit...")
	fmt.Scanln()
}

func isAdmin() bool {
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	return err == nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func addToPath(newPath string) error {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, 
		`SYSTEM\CurrentControlSet\Control\Session Manager\Environment`, 
		registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	currentPath, _, err := k.GetStringValue("Path")
	if err != nil {
		return err
	}

	// Check if already in PATH
	paths := strings.Split(currentPath, ";")
	for _, p := range paths {
		if strings.EqualFold(strings.TrimSpace(p), newPath) {
			return nil // Already in PATH
		}
	}

	// Add to PATH
	newPathValue := currentPath
	if !strings.HasSuffix(currentPath, ";") {
		newPathValue += ";"
	}
	newPathValue += newPath

	return k.SetStringValue("Path", newPathValue)
}

func detectClaudeCode() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	// Check common Claude Code config paths
	possiblePaths := []string{
		filepath.Join(homeDir, ".claude", "config.json"),
		filepath.Join(homeDir, ".config", "claude", "config.json"),
	}

	for _, p := range possiblePaths {
		if _, err := os.Stat(filepath.Dir(p)); err == nil {
			return p
		}
	}

	return ""
}

func detectCopilotCLI() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	// Check common Copilot CLI config paths
	possiblePaths := []string{
		filepath.Join(homeDir, ".copilot", "config.json"),
		filepath.Join(homeDir, ".config", "copilot", "config.json"),
	}

	for _, p := range possiblePaths {
		if _, err := os.Stat(filepath.Dir(p)); err == nil {
			return p
		}
	}

	return ""
}

func configureClaudeCode(configPath string) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	// Create or update config
	config := `{
  "mcpServers": {
    "prism": {
      "command": "prism",
      "args": ["start", "--mcp-only", "--auto-index"]
    }
  }
}`

	return os.WriteFile(configPath, []byte(config), 0644)
}

func configureCopilotCLI(configPath string) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	// Create or update config (similar to Claude Code)
	config := `{
  "mcpServers": {
    "prism": {
      "command": "prism",
      "args": ["start", "--mcp-only", "--auto-index"]
    }
  }
}`

	return os.WriteFile(configPath, []byte(config), 0644)
}

func isOllamaInstalled() bool {
	cmd := exec.Command("ollama", "list")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run() == nil
}

func fatal(msg string, err error) {
	fmt.Printf("❌ %s: %v\n", msg, err)
	fmt.Println()
	fmt.Print("Press Enter to exit...")
	fmt.Scanln()
	os.Exit(1)
}
