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
	CommitsCount    int `json:"commits_count"`
	BugFixes        int `json:"bug_fixes"`
	BreakingChanges int `json:"breaking_changes"`
	RefactorsCount  int `json:"refactors_count"`
	FeaturesCount   int `json:"features_count"`

	// Ownership
	TeamOwners map[string]int `json:"team_owners"` // author -> % ownership

	// Evolución temporal
	Evolution []EvolutionEvent `json:"evolution"`

	// Metadata
	IsActive       bool   `json:"is_active"`       // tocado en últimos 6 meses
	StabilityScore int    `json:"stability_score"` // 0-100
	RiskAssessment string `json:"risk_assessment"` // LOW, MEDIUM, HIGH
}

// EvolutionEvent representa un cambio importante en la historia
type EvolutionEvent struct {
	Date         time.Time `json:"date"`
	Author       string    `json:"author"`
	Type         string    `json:"type"` // "bugfix", "feature", "refactor", "breaking"
	Message      string    `json:"message"`
	CommitHash   string    `json:"commit_hash"`
	FilesChanged int       `json:"files_changed"`
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
	Commits         int       `json:"commits"`
	Percent         float64   `json:"percent"`
	FirstSeen       time.Time `json:"first_seen"`
	LastSeen        time.Time `json:"last_seen"`
	DaysSinceActive int       `json:"days_since_active"`
}

// RefactorSafetyAssessment evaluación de seguridad para refactoring
type RefactorSafetyAssessment struct {
	RiskLevel         string   `json:"risk_level"` // LOW, MEDIUM, HIGH
	RiskScore         int      `json:"risk_score"` // 0-100
	Warnings          []string `json:"warnings"`
	SafeRefactorPath  string   `json:"safe_refactor_path"`
	RecommendedOwners []string `json:"recommended_owners"`
}

// GitStats estadísticas globales
type GitStats struct {
	TotalFunctions        int
	TotalCommits          int
	UniqueAuthors         int
	AvgCommitsPerFunction float64
	MostActiveAuthor      string
}
