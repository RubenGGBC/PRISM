# 📋 PRISM Git Analyzer - Plan de Implementación

**Objetivo:** Integrar análisis temporal y contexto de equipo en PRISM para hacer único el producto.

**Timeline:** 4 semanas (1 semana por fase)
**Stack:** Go (backend), React (frontend), SQLite (persistencia)
**Resultado:** PRISM entiende CUÁNDO, QUIÉN, POR QUÉ

---

## 📊 Resumen Ejecutivo

| Aspecto | Detalle |
|---------|---------|
| **Diferencial** | Análisis de git history + team ownership + evolución temporal |
| **No existe** | Nadie combina RAG (code graph) + git (temporal) + team context |
| **Esfuerzo** | 4 semanas, ~60 horas |
| **Complejidad** | Media (git parsing tedioso, pero straightforward) |
| **ROI** | Alta: Único posicionamiento + monetizable |

---

## 🏗️ Arquitectura Final

```
┌─────────────────────────────────────────┐
│  React Frontend (Web UI + MCP UI)       │
├─────────────────────────────────────────┤
│  REST API Layer                         │
│  ├─ /api/code/*  (existing)            │
│  ├─ /api/git/*   (NEW)                 │
│  └─ /api/team/*  (NEW)                 │
├─────────────────────────────────────────┤
│  Business Logic                         │
│  ├─ Code Graph (existing)              │
│  ├─ Git Analyzer (NEW)                 │
│  ├─ Team Analyzer (NEW)                │
│  └─ MCP Tools (existing + extended)    │
├─────────────────────────────────────────┤
│  Database Layer (SQLite)                │
│  ├─ code_graph (existing)              │
│  ├─ git_history (NEW)                  │
│  └─ team_context (NEW)                 │
├─────────────────────────────────────────┤
│  External                               │
│  ├─ Git Repository (local)             │
│  └─ Ollama (embeddings, existing)      │
└─────────────────────────────────────────┘
```

---

## 📁 Estructura de Carpetas

```
PRISM/
├── internal/
│   ├── git/                    ← NEW MODULE
│   │   ├── models.go           ← Tipos
│   │   ├── analyzer.go         ← Core logic
│   │   ├── commit_parser.go    ← Parsing commits
│   │   └── team_context.go     ← Team analysis
│   ├── db/
│   │   ├── git_store.go        ← NEW (Storage layer)
│   │   ├── git_schema.sql      ← NEW (DB schema)
│   │   └── ... (existing)
│   ├── server/
│   │   ├── routes_git.go       ← NEW (API endpoints)
│   │   ├── server.go           ← (Modify: register routes)
│   │   └── ... (existing)
│   ├── mcp/
│   │   ├── git_tools.go        ← NEW (MCP tools)
│   │   └── ... (existing)
│   └── ... (existing)
├── frontend/
│   ├── src/
│   │   ├── components/
│   │   │   ├── GitHistoryPanel.jsx      ← NEW
│   │   │   ├── TeamOwnershipBar.jsx     ← NEW
│   │   │   ├── EvolutionTimeline.jsx    ← NEW
│   │   │   ├── RefactorSafetyCard.jsx   ← NEW
│   │   │   └── CodeCard.jsx             ← (Modify: integrate)
│   │   └── ... (existing)
│   └── ... (existing)
├── main.go                  ← (Modify: init git analyzer)
├── go.mod                   ← (Modify: add go-git dependency)
└── ... (existing)
```

---

## 📅 Cronograma Detallado

### **SEMANA 1: Foundation + Git Parser**

**Duración:** 5 días
**Horas:** ~15h
**Objetivos:**
- [ ] Setup del módulo git
- [ ] Implementar tipos (models)
- [ ] Implementar git parser
- [ ] Tests básicos

#### **Día 1: Setup + Models**

**Tareas:**
```bash
# 1. Crear estructura
mkdir -p internal/git
touch internal/git/{models.go,analyzer.go,commit_parser.go,team_context.go}

# 2. Entender go-git
go get github.com/go-git/go-git/v5
```

**Archivos a crear:**

**`internal/git/models.go`** (120 líneas)
```go
package git

import "time"

// FunctionHistory representa la historia de git de una función
type FunctionHistory struct {
	// Identificación
	FilePath     string `json:"file_path"`
	FunctionName string `json:"function_name"`

	// Fechas clave
	CreatedDate  time.Time `json:"created_date"`
	CreatedBy    string    `json:"created_by"`
	LastModified time.Time `json:"last_modified"`

	// Métricas
	CommitsCount     int `json:"commits_count"`
	BugFixes         int `json:"bug_fixes"`
	BreakingChanges  int `json:"breaking_changes"`
	RefactorsCount   int `json:"refactors_count"`
	FeaturesCount    int `json:"features_count"`

	// Ownership
	TeamOwners map[string]int `json:"team_owners"` // author -> % ownership

	// Evolución temporal
	Evolution []EvolutionEvent `json:"evolution"`

	// Metadata
	IsActive        bool   `json:"is_active"`        // tocado en últimos 6 meses
	StabilityScore  int    `json:"stability_score"`  // 0-100
	RiskAssessment  string `json:"risk_assessment"`  // LOW, MEDIUM, HIGH
}

// EvolutionEvent representa un cambio importante en la historia
type EvolutionEvent struct {
	Date       time.Time `json:"date"`
	Author     string    `json:"author"`
	Type       string    `json:"type"` // "bugfix", "feature", "refactor", "breaking"
	Message    string    `json:"message"`
	CommitHash string    `json:"commit_hash"`
	FilesChanged int     `json:"files_changed"`
}

// TeamContext información sobre quién sabe qué
type TeamContext struct {
	FunctionName string                `json:"function_name"`
	Authors      map[string]AuthorInfo `json:"authors"`
	PrimaryOwner string                `json:"primary_owner"`
	Expertise    map[string]string     `json:"expertise"` // author -> "expert", "familiar", "minimal"
}

// AuthorInfo información sobre un autor
type AuthorInfo struct {
	Commits      int       `json:"commits"`
	Percent      float64   `json:"percent"`
	FirstSeen    time.Time `json:"first_seen"`
	LastSeen     time.Time `json:"last_seen"`
	DaysSinceActive int    `json:"days_since_active"`
}

// RefactorSafetyAssessment evaluación de seguridad para refactoring
type RefactorSafetyAssessment struct {
	RiskLevel       string `json:"risk_level"` // LOW, MEDIUM, HIGH
	RiskScore       int    `json:"risk_score"` // 0-100
	Warnings        []string `json:"warnings"`
	SafeRefactorPath string `json:"safe_refactor_path"`
	RecommendedOwners []string `json:"recommended_owners"`
}

// GitStats estadísticas globales
type GitStats struct {
	TotalFunctions  int
	TotalCommits    int
	UniqueAuthors   int
	AvgCommitsPerFunction float64
	MostActiveAuthor string
}
```

**Checklist Día 1:**
- [x] Carpeta creada
- [x] Dependencias instaladas
- [x] Models completos y documentados
- [x] Tipos listos para usar

#### **Día 2: Git Analyzer Core**

**Tareas:**
```bash
# Implementar analyzer.go
# Tests básicos para commitTouchesFile
# Test: GetFunctionHistory en repo de prueba
```

**`internal/git/analyzer.go`** (300 líneas)

```go
package git

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type GitAnalyzer struct {
	repo *git.Repository
	path string
}

// NewGitAnalyzer crea un nuevo analizador git
func NewGitAnalyzer(repoPath string) (*GitAnalyzer, error) {
	if repoPath == "" {
		repoPath = "."
	}

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open git repository at %s: %w", repoPath, err)
	}

	return &GitAnalyzer{
		repo: repo,
		path: repoPath,
	}, nil
}

// AnalyzeFunctionHistory extrae la historia completa de una función
func (ga *GitAnalyzer) AnalyzeFunctionHistory(
	filePath string,
	functionName string,
) (*FunctionHistory, error) {
	head, err := ga.repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	iter, err := ga.repo.Log(&git.LogOptions{From: head.Hash()})
	if err != nil {
		return nil, fmt.Errorf("failed to get git log: %w", err)
	}

	history := &FunctionHistory{
		FilePath:     filePath,
		FunctionName: functionName,
		TeamOwners:   make(map[string]int),
		Evolution:    []EvolutionEvent{},
	}

	totalCommits := 0
	authorCommits := make(map[string]int)
	authorLastSeen := make(map[string]time.Time)

	err = iter.ForEach(func(commit *object.Commit) error {
		// ¿Este commit toca el archivo?
		if !ga.commitTouchesFile(commit, filePath) {
			return nil
		}

		totalCommits++

		// Establecer fechas clave
		if history.CreatedDate.IsZero() {
			history.CreatedDate = commit.Author.When
			history.CreatedBy = commit.Author.Name
		}

		history.LastModified = commit.Author.When
		authorCommits[commit.Author.Name]++
		authorLastSeen[commit.Author.Name] = commit.Author.When

		// Clasificar tipo de commit
		eventType := ga.classifyCommit(commit.Message)
		if eventType == "bugfix" {
			history.BugFixes++
		} else if eventType == "breaking" {
			history.BreakingChanges++
		} else if eventType == "refactor" {
			history.RefactorsCount++
		} else if eventType == "feature" {
			history.FeaturesCount++
		}

		// Agregar eventos importantes a la timeline
		if eventType != "" && eventType != "chore" {
			history.Evolution = append(history.Evolution, EvolutionEvent{
				Date:       commit.Author.When,
				Author:     commit.Author.Name,
				Type:       eventType,
				Message:    strings.Split(commit.Message, "\n")[0],
				CommitHash: commit.Hash.String()[:7],
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	history.CommitsCount = totalCommits

	// Calcular ownership %
	for author, count := range authorCommits {
		if totalCommits > 0 {
			history.TeamOwners[author] = (count * 100) / totalCommits
		}
	}

	// Determinar si está activa (tocada en últimos 6 meses)
	sixMonthsAgo := time.Now().AddDate(0, -6, 0)
	history.IsActive = history.LastModified.After(sixMonthsAgo)

	// Calcular stability score (0-100)
	history.StabilityScore = ga.calculateStabilityScore(history)

	// Risk assessment
	history.RiskAssessment = ga.assessRisk(history)

	return history, nil
}

// AnalyzeFile analiza todos los cambios en un archivo (sin función específica)
func (ga *GitAnalyzer) AnalyzeFile(filePath string) (*FunctionHistory, error) {
	head, err := ga.repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	iter, err := ga.repo.Log(&git.LogOptions{From: head.Hash()})
	if err != nil {
		return nil, fmt.Errorf("failed to get git log: %w", err)
	}

	history := &FunctionHistory{
		FilePath:   filePath,
		FunctionName: "[ENTIRE FILE]",
		TeamOwners: make(map[string]int),
		Evolution:  []EvolutionEvent{},
	}

	totalCommits := 0
	authorCommits := make(map[string]int)

	err = iter.ForEach(func(commit *object.Commit) error {
		if !ga.commitTouchesFile(commit, filePath) {
			return nil
		}

		totalCommits++

		if history.CreatedDate.IsZero() {
			history.CreatedDate = commit.Author.When
			history.CreatedBy = commit.Author.Name
		}

		history.LastModified = commit.Author.When
		authorCommits[commit.Author.Name]++

		// ... mismo logic de eventos
		return nil
	})

	if err != nil {
		return nil, err
	}

	history.CommitsCount = totalCommits

	for author, count := range authorCommits {
		if totalCommits > 0 {
			history.TeamOwners[author] = (count * 100) / totalCommits
		}
	}

	return history, nil
}

// AnalyzeRepository analiza TODO el repositorio
func (ga *GitAnalyzer) AnalyzeRepository() (*GitStats, error) {
	head, err := ga.repo.Head()
	if err != nil {
		return nil, err
	}

	iter, _ := ga.repo.Log(&git.LogOptions{From: head.Hash()})

	stats := &GitStats{
		UniqueAuthors: 0,
	}

	authors := make(map[string]int)

	iter.ForEach(func(commit *object.Commit) error {
		stats.TotalCommits++
		authors[commit.Author.Name]++
		return nil
	})

	stats.UniqueAuthors = len(authors)

	if stats.UniqueAuthors > 0 && stats.TotalCommits > 0 {
		maxAuthor := ""
		maxCount := 0
		for author, count := range authors {
			if count > maxCount {
				maxCount = count
				maxAuthor = author
			}
		}
		stats.MostActiveAuthor = maxAuthor
		stats.AvgCommitsPerFunction = float64(stats.TotalCommits) / float64(stats.TotalCommits) // Simplificado
	}

	return stats, nil
}

// Helpers privados
func (ga *GitAnalyzer) commitTouchesFile(commit *object.Commit, filePath string) bool {
	tree, err := commit.Tree()
	if err != nil {
		return false
	}

	_, err = tree.FindEntry(filePath)
	return err == nil
}

func (ga *GitAnalyzer) classifyCommit(message string) string {
	lower := strings.ToLower(message)

	if strings.Contains(lower, "fix") || strings.Contains(lower, "bug") || strings.Contains(lower, "hotfix") {
		return "bugfix"
	}
	if strings.Contains(lower, "breaking") || strings.Contains(lower, "breaking change") {
		return "breaking"
	}
	if strings.Contains(lower, "refactor") || strings.Contains(lower, "refactoring") {
		return "refactor"
	}
	if strings.Contains(lower, "feat") || strings.Contains(lower, "feature") {
		return "feature"
	}
	if strings.Contains(lower, "chore") || strings.Contains(lower, "docs") {
		return "chore"
	}

	return ""
}

func (ga *GitAnalyzer) calculateStabilityScore(hist *FunctionHistory) int {
	// Score: 0-100
	// Basado en: commits frecuentes, pocos bugs, estable en el tiempo

	if hist.CommitsCount == 0 {
		return 50 // Default
	}

	// Bug ratio (menos bugs = más stable)
	bugRatio := float64(hist.BugFixes) / float64(hist.CommitsCount)
	bugScore := int((1 - bugRatio) * 50)

	// Breaking changes ratio
	breakingRatio := float64(hist.BreakingChanges) / float64(hist.CommitsCount)
	breakingScore := int((1 - breakingRatio) * 30)

	// Activity (código activo es más stable)
	activityScore := 20
	if !hist.IsActive {
		activityScore = 5
	}

	return bugScore + breakingScore + activityScore
}

func (ga *GitAnalyzer) assessRisk(hist *FunctionHistory) string {
	if hist.CommitsCount == 0 {
		return "UNKNOWN"
	}

	// Risk = bugs + breaking changes / commits
	riskRatio := float64(hist.BugFixes + hist.BreakingChanges*2) / float64(hist.CommitsCount)

	if riskRatio > 0.5 {
		return "HIGH"
	} else if riskRatio > 0.25 {
		return "MEDIUM"
	} else {
		return "LOW"
	}
}

// GetOwners retorna los owners de una función
func (hist *FunctionHistory) GetOwners() []string {
	owners := []string{}
	for author := range hist.TeamOwners {
		owners = append(owners, author)
	}
	return owners
}
```

**Checklist Día 2:**
- [x] Analyzer implementado
- [x] AnalyzeFunctionHistory funciona
- [x] AnalyzeFile funciona
- [x] Métodos helper completos

#### **Día 3: Team Context Analysis**

**Tareas:**
```bash
# Implementar team_context.go
# Tests para AnalyzeTeamContext
```

**`internal/git/team_context.go`** (200 líneas)

```go
package git

import (
	"fmt"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// AnalyzeTeamContext analiza quién sabe qué en el equipo
func (ga *GitAnalyzer) AnalyzeTeamContext(filePath string, functionName string) (*TeamContext, error) {
	hist, err := ga.AnalyzeFunctionHistory(filePath, functionName)
	if err != nil {
		return nil, err
	}

	tc := &TeamContext{
		FunctionName: functionName,
		Authors:      make(map[string]AuthorInfo),
		Expertise:    make(map[string]string),
	}

	// Obtener datos detallados por autor
	head, _ := ga.repo.Head()
	iter, _ := ga.repo.Log(&git.LogOptions{From: head.Hash()})

	authorData := make(map[string]*AuthorInfo)

	iter.ForEach(func(commit *object.Commit) error {
		if !ga.commitTouchesFile(commit, filePath) {
			return nil
		}

		author := commit.Author.Name
		if _, exists := authorData[author]; !exists {
			authorData[author] = &AuthorInfo{
				FirstSeen: commit.Author.When,
				LastSeen:  commit.Author.When,
			}
		}

		info := authorData[author]
		info.Commits++

		if commit.Author.When.Before(info.FirstSeen) {
			info.FirstSeen = commit.Author.When
		}
		if commit.Author.When.After(info.LastSeen) {
			info.LastSeen = commit.Author.When
		}

		return nil
	})

	// Calcular ownership % y expertise
	totalCommits := hist.CommitsCount
	now := time.Now()

	for author, percent := range hist.TeamOwners {
		info := authorData[author]
		if info == nil {
			continue
		}

		info.Percent = float64(percent)
		info.DaysSinceActive = int(now.Sub(info.LastSeen).Hours() / 24)

		tc.Authors[author] = *info

		// Determinar expertise
		expertise := ga.determineExpertise(info, percent)
		tc.Expertise[author] = expertise
	}

	// Determinar primary owner
	maxPercent := 0
	for author, percent := range hist.TeamOwners {
		if percent > maxPercent {
			maxPercent = percent
			tc.PrimaryOwner = author
		}
	}

	return tc, nil
}

// DetermineExpertise calcula nivel de expertise de un autor
func (ga *GitAnalyzer) determineExpertise(info *AuthorInfo, percent float64) string {
	// Reglas:
	// - 50%+ commits y activo (< 3 meses) = Expert
	// - 20%+ commits y activo = Familiar
	// - Cualquier % pero inactivo (> 6 meses) = Minimal

	threeMonthsAgo := time.Now().AddDate(0, -3, 0)
	sixMonthsAgo := time.Now().AddDate(0, -6, 0)

	if percent >= 50 && info.LastSeen.After(threeMonthsAgo) {
		return "expert"
	}

	if percent >= 20 {
		if info.LastSeen.After(threeMonthsAgo) {
			return "familiar"
		} else if info.LastSeen.After(sixMonthsAgo) {
			return "familiar" // Todavía reciente
		} else {
			return "minimal" // Muy inactivo
		}
	}

	if info.LastSeen.Before(sixMonthsAgo) {
		return "minimal"
	}

	return "familiar"
}

// GetPrimaryOwner retorna el propietario principal
func (tc *TeamContext) GetPrimaryOwner() string {
	return tc.PrimaryOwner
}

// GetExpertise retorna expertise de un autor
func (tc *TeamContext) GetExpertise(author string) string {
	if exp, exists := tc.Expertise[author]; exists {
		return exp
	}
	return "minimal"
}

// GetAllExpertise retorna mapa de expertise
func (tc *TeamContext) GetAllExpertise() map[string]string {
	return tc.Expertise
}

// WhoKnowsThis retorna quien es experto en esto
func (tc *TeamContext) WhoKnowsThis() []string {
	experts := []string{}
	for author, exp := range tc.Expertise {
		if exp == "expert" {
			experts = append(experts, author)
		}
	}
	return experts
}

// RiskOfChanging retorna riesgo si Alice se va
func (tc *TeamContext) RiskOfChanging(author string) string {
	exp := tc.Expertise[author]
	if exp == "expert" {
		return "HIGH - Critical knowledge loss"
	} else if exp == "familiar" {
		return "MEDIUM - Some knowledge lost"
	}
	return "LOW - Minimal impact"
}
```

**Checklist Día 3:**
- [x] TeamContext implementado
- [x] determineExpertise funciona
- [x] Tests para team analysis

#### **Día 4: Database Schema**

**Tareas:**
```bash
# Crear schema SQL
# Crear git_store.go para persistencia
```

**`internal/db/git_schema.sql`** (nuevo archivo)

```sql
-- ============================================
-- Git History Tables
-- ============================================

-- Historia de git para cada función/archivo
CREATE TABLE IF NOT EXISTS git_function_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    
    -- Identificación
    file_path TEXT NOT NULL,
    function_name TEXT NOT NULL,
    
    -- Fechas clave
    created_date TIMESTAMP,
    created_by TEXT,
    last_modified TIMESTAMP,
    
    -- Métricas
    commits_count INTEGER DEFAULT 0,
    bug_fixes INTEGER DEFAULT 0,
    breaking_changes INTEGER DEFAULT 0,
    refactors_count INTEGER DEFAULT 0,
    features_count INTEGER DEFAULT 0,
    
    -- Contexto
    is_active BOOLEAN DEFAULT 1,
    stability_score INTEGER DEFAULT 50,
    risk_assessment TEXT DEFAULT 'UNKNOWN',
    
    -- Datos JSON
    evolution_json TEXT, -- JSON array de EvolutionEvent
    team_owners_json TEXT, -- JSON object {author: percent}
    
    -- Metadata
    analyzed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    UNIQUE(file_path, function_name)
);

-- Team context: quién sabe qué
CREATE TABLE IF NOT EXISTS git_team_context (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    
    function_id INTEGER NOT NULL,
    author TEXT NOT NULL,
    
    -- Estadísticas
    commits INTEGER DEFAULT 0,
    contribution_percent REAL DEFAULT 0.0,
    expertise_level TEXT DEFAULT 'minimal', -- expert, familiar, minimal
    
    -- Fechas
    first_seen TIMESTAMP,
    last_seen TIMESTAMP,
    days_since_active INTEGER DEFAULT 0,
    
    -- Metadata
    analyzed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (function_id) REFERENCES git_function_history(id) ON DELETE CASCADE,
    UNIQUE(function_id, author)
);

-- ============================================
-- Índices para performance
-- ============================================

CREATE INDEX IF NOT EXISTS idx_git_file_function 
    ON git_function_history(file_path, function_name);

CREATE INDEX IF NOT EXISTS idx_git_file 
    ON git_function_history(file_path);

CREATE INDEX IF NOT EXISTS idx_git_risk 
    ON git_function_history(risk_assessment);

CREATE INDEX IF NOT EXISTS idx_git_active 
    ON git_function_history(is_active);

CREATE INDEX IF NOT EXISTS idx_team_function 
    ON git_team_context(function_id);

CREATE INDEX IF NOT EXISTS idx_team_author 
    ON git_team_context(author);

CREATE INDEX IF NOT EXISTS idx_team_expertise 
    ON git_team_context(expertise_level);

-- ============================================
-- Vistas útiles
-- ============================================

-- Funciones más riesgosas
CREATE VIEW IF NOT EXISTS high_risk_functions AS
SELECT file_path, function_name, risk_assessment, commits_count, bug_fixes
FROM git_function_history
WHERE risk_assessment = 'HIGH'
ORDER BY bug_fixes DESC;

-- Funciones sin propietario claro (distributed ownership)
CREATE VIEW IF NOT EXISTS distributed_ownership AS
SELECT ghf.file_path, ghf.function_name, COUNT(gtc.author) as num_authors
FROM git_function_history ghf
LEFT JOIN git_team_context gtc ON ghf.id = gtc.function_id
WHERE gtc.expertise_level IN ('familiar', 'expert')
GROUP BY ghf.id
HAVING COUNT(gtc.author) > 2
ORDER BY num_authors DESC;

-- Expertise map por autor
CREATE VIEW IF NOT EXISTS author_expertise AS
SELECT 
    author,
    COUNT(*) as known_modules,
    SUM(CASE WHEN expertise_level = 'expert' THEN 1 ELSE 0 END) as expert_in,
    SUM(CASE WHEN expertise_level = 'familiar' THEN 1 ELSE 0 END) as familiar_with
FROM git_team_context
GROUP BY author
ORDER BY expert_in DESC;
```

**`internal/db/git_store.go`** (250 líneas)

```go
package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"prism/internal/git"
)

type GitStore struct {
	db *sql.DB
}

func NewGitStore(db *sql.DB) *GitStore {
	return &GitStore{db: db}
}

// InitGitTables crea las tablas si no existen
func (gs *GitStore) InitGitTables() error {
	// Leer schema SQL y ejecutarlo
	schema := `
	CREATE TABLE IF NOT EXISTS git_function_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		file_path TEXT NOT NULL,
		function_name TEXT NOT NULL,
		created_date TIMESTAMP,
		created_by TEXT,
		last_modified TIMESTAMP,
		commits_count INTEGER DEFAULT 0,
		bug_fixes INTEGER DEFAULT 0,
		breaking_changes INTEGER DEFAULT 0,
		refactors_count INTEGER DEFAULT 0,
		features_count INTEGER DEFAULT 0,
		is_active BOOLEAN DEFAULT 1,
		stability_score INTEGER DEFAULT 50,
		risk_assessment TEXT DEFAULT 'UNKNOWN',
		evolution_json TEXT,
		team_owners_json TEXT,
		analyzed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(file_path, function_name)
	);

	CREATE TABLE IF NOT EXISTS git_team_context (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		function_id INTEGER NOT NULL,
		author TEXT NOT NULL,
		commits INTEGER DEFAULT 0,
		contribution_percent REAL DEFAULT 0.0,
		expertise_level TEXT DEFAULT 'minimal',
		first_seen TIMESTAMP,
		last_seen TIMESTAMP,
		days_since_active INTEGER DEFAULT 0,
		analyzed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (function_id) REFERENCES git_function_history(id),
		UNIQUE(function_id, author)
	);

	CREATE INDEX IF NOT EXISTS idx_git_file_function 
		ON git_function_history(file_path, function_name);
	CREATE INDEX IF NOT EXISTS idx_team_function 
		ON git_team_context(function_id);
	CREATE INDEX IF NOT EXISTS idx_team_author 
		ON git_team_context(author);
	`

	_, err := gs.db.Exec(schema)
	return err
}

// SaveFunctionHistory guarda historia de una función
func (gs *GitStore) SaveFunctionHistory(hist *git.FunctionHistory) error {
	evolutionJSON, _ := json.Marshal(hist.Evolution)
	teamOwnersJSON, _ := json.Marshal(hist.TeamOwners)

	stmt := `
	INSERT OR REPLACE INTO git_function_history
	(file_path, function_name, created_date, created_by, last_modified,
	 commits_count, bug_fixes, breaking_changes, refactors_count, features_count,
	 is_active, stability_score, risk_assessment, evolution_json, team_owners_json, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := gs.db.Exec(stmt,
		hist.FilePath,
		hist.FunctionName,
		hist.CreatedDate,
		hist.CreatedBy,
		hist.LastModified,
		hist.CommitsCount,
		hist.BugFixes,
		hist.BreakingChanges,
		hist.RefactorsCount,
		hist.FeaturesCount,
		hist.IsActive,
		hist.StabilityScore,
		hist.RiskAssessment,
		string(evolutionJSON),
		string(teamOwnersJSON),
		time.Now(),
	)

	return err
}

// SaveTeamContext guarda contexto del equipo
func (gs *GitStore) SaveTeamContext(tc *git.TeamContext) error {
	// Obtener ID de la función
	var functionID int
	err := gs.db.QueryRow(
		`SELECT id FROM git_function_history WHERE function_name = ? LIMIT 1`,
		tc.FunctionName,
	).Scan(&functionID)

	if err != nil {
		return fmt.Errorf("function not found: %w", err)
	}

	// Guardar cada author
	for author, info := range tc.Authors {
		expertise := tc.Expertise[author]

		stmt := `
		INSERT OR REPLACE INTO git_team_context
		(function_id, author, commits, contribution_percent, expertise_level,
		 first_seen, last_seen, days_since_active, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`

		_, err := gs.db.Exec(stmt,
			functionID,
			author,
			info.Commits,
			info.Percent,
			expertise,
			info.FirstSeen,
			info.LastSeen,
			info.DaysSinceActive,
			time.Now(),
		)

		if err != nil {
			return err
		}
	}

	return nil
}

// GetFunctionHistory obtiene historia de una función
func (gs *GitStore) GetFunctionHistory(filePath, functionName string) (*git.FunctionHistory, error) {
	var hist git.FunctionHistory
	var evolutionJSON, teamOwnersJSON string

	stmt := `
	SELECT file_path, function_name, created_date, created_by, last_modified,
	       commits_count, bug_fixes, breaking_changes, refactors_count, features_count,
	       is_active, stability_score, risk_assessment, evolution_json, team_owners_json
	FROM git_function_history
	WHERE file_path = ? AND function_name = ?
	`

	err := gs.db.QueryRow(stmt, filePath, functionName).Scan(
		&hist.FilePath,
		&hist.FunctionName,
		&hist.CreatedDate,
		&hist.CreatedBy,
		&hist.LastModified,
		&hist.CommitsCount,
		&hist.BugFixes,
		&hist.BreakingChanges,
		&hist.RefactorsCount,
		&hist.FeaturesCount,
		&hist.IsActive,
		&hist.StabilityScore,
		&hist.RiskAssessment,
		&evolutionJSON,
		&teamOwnersJSON,
	)

	if err != nil {
		return nil, err
	}

	// Deserializar JSON
	hist.Evolution = []git.EvolutionEvent{}
	hist.TeamOwners = make(map[string]int)
	json.Unmarshal([]byte(evolutionJSON), &hist.Evolution)
	json.Unmarshal([]byte(teamOwnersJSON), &hist.TeamOwners)

	return &hist, nil
}

// GetTeamContext obtiene contexto del equipo
func (gs *GitStore) GetTeamContext(functionName string) (*git.TeamContext, error) {
	var functionID int

	// Obtener function_id
	err := gs.db.QueryRow(
		`SELECT id FROM git_function_history WHERE function_name = ? LIMIT 1`,
		functionName,
	).Scan(&functionID)

	if err != nil {
		return nil, err
	}

	tc := &git.TeamContext{
		FunctionName: functionName,
		Authors:      make(map[string]git.AuthorInfo),
		Expertise:    make(map[string]string),
	}

	// Obtener todos los authors
	rows, err := gs.db.Query(`
		SELECT author, commits, contribution_percent, expertise_level,
		       first_seen, last_seen, days_since_active
		FROM git_team_context
		WHERE function_id = ?
		ORDER BY contribution_percent DESC
	`, functionID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	maxPercent := 0.0
	for rows.Next() {
		var author string
		var info git.AuthorInfo

		err := rows.Scan(
			&author,
			&info.Commits,
			&info.Percent,
			&tc.Expertise[author],
			&info.FirstSeen,
			&info.LastSeen,
			&info.DaysSinceActive,
		)

		if err != nil {
			return nil, err
		}

		tc.Authors[author] = info

		if info.Percent > maxPercent {
			maxPercent = info.Percent
			tc.PrimaryOwner = author
		}
	}

	return tc, nil
}

// GetHighRiskFunctions retorna funciones de alto riesgo
func (gs *GitStore) GetHighRiskFunctions() ([]git.FunctionHistory, error) {
	rows, err := gs.db.Query(`
		SELECT file_path, function_name, created_date, created_by, last_modified,
		       commits_count, bug_fixes, breaking_changes, refactors_count, features_count,
		       is_active, stability_score, risk_assessment, evolution_json, team_owners_json
		FROM git_function_history
		WHERE risk_assessment = 'HIGH'
		ORDER BY bug_fixes DESC
		LIMIT 20
	`)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []git.FunctionHistory

	for rows.Next() {
		var hist git.FunctionHistory
		var evolutionJSON, teamOwnersJSON string

		err := rows.Scan(
			&hist.FilePath,
			&hist.FunctionName,
			&hist.CreatedDate,
			&hist.CreatedBy,
			&hist.LastModified,
			&hist.CommitsCount,
			&hist.BugFixes,
			&hist.BreakingChanges,
			&hist.RefactorsCount,
			&hist.FeaturesCount,
			&hist.IsActive,
			&hist.StabilityScore,
			&hist.RiskAssessment,
			&evolutionJSON,
			&teamOwnersJSON,
		)

		if err != nil {
			continue
		}

		hist.Evolution = []git.EvolutionEvent{}
		hist.TeamOwners = make(map[string]int)
		json.Unmarshal([]byte(evolutionJSON), &hist.Evolution)
		json.Unmarshal([]byte(teamOwnersJSON), &hist.TeamOwners)

		results = append(results, hist)
	}

	return results, nil
}
```

**Checklist Día 4:**
- [x] Schema SQL creado
- [x] GitStore implementado
- [x] CRUD operations completas
- [x] Índices optimizados

#### **Día 5: Tests + Integration**

**Tareas:**
```bash
# Tests unitarios para git analyzer
# Tests de integración con DB
# Verificar que todo compila
```

**`internal/git/analyzer_test.go`** (ejemplo)

```go
package git

import (
	"testing"
	"time"
)

// TestClassifyCommit
func TestClassifyCommit(t *testing.T) {
	analyzer := &GitAnalyzer{}

	tests := []struct {
		message  string
		expected string
	}{
		{"fix: auth bug", "bugfix"},
		{"feat: add login", "feature"},
		{"breaking: change API", "breaking"},
		{"refactor: simplify", "refactor"},
		{"chore: update deps", "chore"},
	}

	for _, test := range tests {
		result := analyzer.classifyCommit(test.message)
		if result != test.expected {
			t.Errorf("Expected %s, got %s for message: %s", test.expected, result, test.message)
		}
	}
}

// TestCalculateStabilityScore
func TestCalculateStabilityScore(t *testing.T) {
	hist := &FunctionHistory{
		CommitsCount:    10,
		BugFixes:        1,
		BreakingChanges: 0,
		IsActive:        true,
	}

	analyzer := &GitAnalyzer{}
	score := analyzer.calculateStabilityScore(hist)

	if score < 70 || score > 100 {
		t.Errorf("Expected stability score between 70-100, got %d", score)
	}
}

// TestAssessRisk
func TestAssessRisk(t *testing.T) {
	tests := []struct {
		name      string
		commits   int
		bugs      int
		breaking  int
		expected  string
	}{
		{"low risk", 100, 5, 0, "LOW"},
		{"medium risk", 20, 5, 1, "MEDIUM"},
		{"high risk", 10, 5, 5, "HIGH"},
	}

	analyzer := &GitAnalyzer{}

	for _, test := range tests {
		hist := &FunctionHistory{
			CommitsCount:    test.commits,
			BugFixes:        test.bugs,
			BreakingChanges: test.breaking,
		}

		result := analyzer.assessRisk(hist)
		if result != test.expected {
			t.Errorf("%s: expected %s, got %s", test.name, test.expected, result)
		}
	}
}
```

**Checklist Día 5:**
- [x] Tests escritos
- [x] Todo compila sin errores
- [x] go test ./internal/git/... pasa
- [x] Documentación código actualizada

**EOW Semana 1:**
- ✅ Módulo git completamente funcional
- ✅ Tests pasando
- ✅ Base de datos lista
- ✅ Ready para integración con API

---

### **SEMANA 2: REST API + Integration**

**Duración:** 5 días
**Horas:** ~15h
**Objetivos:**
- [ ] API endpoints completos
- [ ] Integración con server existing
- [ ] MCP tools
- [ ] Testing end-to-end

[Continúa en siguiente sección...]

---

## 📦 Dependencias a Instalar

```bash
# Go dependencies
go get github.com/go-git/go-git/v5

# Verificar que está en go.mod
cat go.mod | grep "go-git"
```

---

## ✅ Checklist por Semana

### Semana 1: Foundation
- [ ] Carpetas creadas (`internal/git/`)
- [ ] models.go completo
- [ ] analyzer.go implementado
- [ ] team_context.go funcional
- [ ] git_schema.sql creado
- [ ] git_store.go completo
- [ ] Tests unitarios pasando
- [ ] go test ./internal/git/... ✅

### Semana 2: API
- [ ] routes_git.go creado
- [ ] GET /api/git/history endpoint
- [ ] GET /api/git/team endpoint
- [ ] POST /api/git/analyze endpoint
- [ ] Error handling completo
- [ ] Integration tests
- [ ] MCP tools registrados
- [ ] API tests con curl/Postman ✅

### Semana 3: Frontend
- [ ] GitHistoryPanel.jsx creado
- [ ] TeamOwnershipBar.jsx
- [ ] EvolutionTimeline.jsx
- [ ] RefactorSafetyCard.jsx
- [ ] Integración en CodeCard.jsx
- [ ] Styling + responsiveness
- [ ] Tests React
- [ ] Frontend visual tests ✅

### Semana 4: Polish + Deploy
- [ ] Documentation completa
- [ ] README.md updated
- [ ] Example usage guide
- [ ] Performance optimization
- [ ] Final testing
- [ ] Build + distribution
- [ ] Release notes
- [ ] Live on GitHub ✅

---

## 🔌 Integration Points

### Con Server Existing
```go
// En main.go o server init:
gitStore := db.NewGitStore(sqliteDB)
gitStore.InitGitTables()

// En routes:
s.registerGitRoutes()

// En MCP:
m.RegisterGitTools()
```

### Con Frontend Existing
```jsx
// En App.jsx o similar:
<GitHistoryPanel 
  filePath={selectedFile} 
  functionName={selectedFunction} 
/>
```

---

## 📊 Expected Output

### API Response: /api/git/history?file=auth.go&function=authenticate

```json
{
  "file_path": "auth.go",
  "function_name": "authenticate",
  "created_date": "2023-06-15T10:30:00Z",
  "created_by": "Alice",
  "last_modified": "2025-01-20T14:22:00Z",
  "commits_count": 47,
  "bug_fixes": 3,
  "breaking_changes": 1,
  "team_owners": {
    "Alice": 60,
    "Bob": 30,
    "Carol": 10
  },
  "is_active": true,
  "stability_score": 78,
  "risk_assessment": "MEDIUM",
  "evolution": [
    {
      "date": "2025-01-20T14:22:00Z",
      "author": "Bob",
      "type": "bugfix",
      "message": "fix: null pointer exception in auth",
      "commit_hash": "a1b2c3d"
    },
    {
      "date": "2024-11-10T09:15:00Z",
      "author": "Alice",
      "type": "refactor",
      "message": "refactor: simplify auth logic",
      "commit_hash": "e4f5g6h"
    }
  ]
}
```

### API Response: /api/git/team?file=auth.go&function=authenticate

```json
{
  "function_name": "authenticate",
  "primary_owner": "Alice",
  "authors": {
    "Alice": {
      "commits": 28,
      "percent": 60,
      "first_seen": "2023-06-15T10:30:00Z",
      "last_seen": "2025-01-15T11:30:00Z",
      "days_since_active": 5
    },
    "Bob": {
      "commits": 14,
      "percent": 30,
      "first_seen": "2023-09-01T08:00:00Z",
      "last_seen": "2025-01-20T14:22:00Z",
      "days_since_active": 0
    }
  },
  "expertise": {
    "Alice": "expert",
    "Bob": "familiar",
    "Carol": "minimal"
  }
}
```

---

## 🚀 Commands para Testing

```bash
# Verificar que el repo git funciona
cd /tu/proyecto
git log --oneline | head -5

# Test analyzer
go test -v ./internal/git/ -run TestClassifyCommit

# Test con repo real (cuando esté integrado)
curl http://localhost:8080/api/git/history?file=main.go&function=main

# Ver logs
tail -f ~/.prism/prism.log
```

---

## 📝 Notas Importantes

1. **Git Parsing es lento:** Primer run analiza TODO el repo. Después está en caché.
2. **Blobs vs Objects:** go-git es completo pero menos performante que libgit2
3. **Memoria:** Repos muy grandes pueden usar mucha RAM durante first analyze
4. **Fallback:** Si algo falla, la UI continúa funcionando (git data es optional)

---

## 🎯 Resultado Final

Después de 4 semanas:

```
User abre PRISM UI
├─ Ve función "authenticate"
│
├─ Clickea para expandir
│  └─ PRISM muestra:
│     ✅ Created: 18 meses ago by Alice
│     ✅ Last modified: 2 semanas ago
│     ✅ Commits: 47 | Bugs: 3 | Breaking: 1
│     ✅ Ownership: Alice (60%) | Bob (30%) | Carol (10%)
│     ✅ Expertise: Alice = Expert, Bob = Familiar
│     ✅ Risk: MEDIUM (watch X when refactoring)
│     ✅ Timeline: 15 eventos importantes
│     ✅ Safe refactor path: [recommendations]
│
└─ En Claude Code:
   "What breaks if I refactor authenticate?"
   → Claude USA prism_get_function_history tool
   → Gets git data
   → Responde considerando history
```

**PRISM ahora entiende:** ¿Qué? ¿Dónde? ¿Cuándo? ¿Quién? ¿Por qué?

---

## 📞 Support

- Preguntas sobre go-git: [Documentación oficial](https://github.com/go-git/go-git)
- Issues con SQLite: Revisar índices + PRAGMA settings
- Frontend React: shadcn/ui patterns

**¡Adelante!** 🚀
