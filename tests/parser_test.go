package tests

import (
	"os"
	"testing"

	"github.com/ruffini/prism/parser"
)

func TestGoParser(t *testing.T) {
	f := writeTempFile(t, "*.go", []byte(`package main

import "fmt"

type Server struct {
	port int
}

func NewServer(port int) *Server {
	return &Server{port: port}
}

func (s *Server) Start() {
	fmt.Println("started")
}
`))
	result, err := parser.GetParser(f).ParseFile(f)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	if result.Language != "go" {
		t.Errorf("expected language=go, got %s", result.Language)
	}
	assertElementFound(t, result, "NewServer")
	assertElementFound(t, result, "Server.Start")
}

func TestRustParser(t *testing.T) {
	f := writeTempFile(t, "*.rs", []byte(`use std::io;

pub struct Config {
    pub port: u16,
}

pub fn new_config(port: u16) -> Config {
    Config { port }
}

impl Config {
    pub fn start(&self) {
        println!("started");
    }
}
`))
	result, err := parser.GetParser(f).ParseFile(f)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	if result.Language != "rust" {
		t.Errorf("expected language=rust, got %s", result.Language)
	}
	assertElementFound(t, result, "new_config")
	assertElementFound(t, result, "Config")
}

func TestPythonParser(t *testing.T) {
	f := writeTempFile(t, "*.py", []byte(`class Greeter:
    def greet(self, name: str) -> str:
        return f"hello {name}"

def standalone():
    pass
`))
	result, err := parser.GetParser(f).ParseFile(f)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	if result.Language != "python" {
		t.Errorf("expected language=python, got %s", result.Language)
	}
	assertElementFound(t, result, "Greeter")
	assertElementFound(t, result, "standalone")
}

func TestJavaParser(t *testing.T) {
	f := writeTempFile(t, "*.java", []byte(`public class Calculator {
    public int add(int a, int b) {
        return a + b;
    }
}
`))
	result, err := parser.GetParser(f).ParseFile(f)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	if result.Language != "java" {
		t.Errorf("expected language=java, got %s", result.Language)
	}
	assertElementFound(t, result, "Calculator")
	assertElementFound(t, result, "add")
}

func TestGetParserReturnsNilForUnknown(t *testing.T) {
	p := parser.GetParser("somefile.xyz")
	if p != nil {
		t.Error("expected nil parser for unknown extension")
	}
}

// helpers

func writeTempFile(t *testing.T, pattern string, content []byte) string {
	t.Helper()
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		t.Fatal(err)
	}
	f.Write(content)
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}

func assertElementFound(t *testing.T, result *parser.ParsedFile, name string) {
	t.Helper()
	for _, e := range result.Elements {
		if e.Name == name {
			return
		}
	}
	t.Errorf("expected to find element %q; got: %v", name, elementNames(result))
}

func elementNames(result *parser.ParsedFile) []string {
	names := make([]string, len(result.Elements))
	for i, e := range result.Elements {
		names[i] = e.Name
	}
	return names
}

func TestShouldSkipPath_Worktrees(t *testing.T) {
	cases := []struct {
		path   string
		expect bool
	}{
		{".claude/worktrees/agent-abc123/main.go", true},
		{".worktrees/feature-branch/src/foo.go", true},
		{"src/main.go", false},
		{"WhoIsThisPokemon/src/main.go", false},
	}
	for _, c := range cases {
		got := parser.ShouldSkipPath(c.path)
		if got != c.expect {
			t.Errorf("ShouldSkipPath(%q) = %v, want %v", c.path, got, c.expect)
		}
	}
}
