package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ruffini/prism/internal/git"
)

// GitStore maneja la persistencia de datos de análisis git
type GitStore struct {
	db *sql.DB
}

// NewGitStore crea un nuevo store para datos git
func NewGitStore(db *sql.DB) *GitStore {
	return &GitStore{db: db}
}

// InitGitTables crea las tablas si no existen
func (gs *GitStore) InitGitTables() error {
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
	CREATE INDEX IF NOT EXISTS idx_git_risk
		ON git_function_history(risk_assessment);
	CREATE INDEX IF NOT EXISTS idx_team_function
		ON git_team_context(function_id);
	CREATE INDEX IF NOT EXISTS idx_team_author
		ON git_team_context(author);
	`

	_, err := gs.db.Exec(schema)
	return err
}

// SaveFunctionHistory guarda o actualiza la historia de una función
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

// SaveTeamContext guarda el contexto de equipo de una función
func (gs *GitStore) SaveTeamContext(tc *git.TeamContext) error {
	var functionID int
	err := gs.db.QueryRow(
		`SELECT id FROM git_function_history WHERE function_name = ? LIMIT 1`,
		tc.FunctionName,
	).Scan(&functionID)

	if err != nil {
		return fmt.Errorf("function not found in git history: %w", err)
	}

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

// GetFunctionHistory obtiene la historia de una función desde la DB
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

	hist.Evolution = []git.EvolutionEvent{}
	hist.TeamOwners = make(map[string]int)
	_ = json.Unmarshal([]byte(evolutionJSON), &hist.Evolution)
	_ = json.Unmarshal([]byte(teamOwnersJSON), &hist.TeamOwners)

	return &hist, nil
}

// GetTeamContext obtiene el contexto del equipo desde la DB
func (gs *GitStore) GetTeamContext(functionName string) (*git.TeamContext, error) {
	var functionID int

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
		var expertise string

		err := rows.Scan(
			&author,
			&info.Commits,
			&info.Percent,
			&expertise,
			&info.FirstSeen,
			&info.LastSeen,
			&info.DaysSinceActive,
		)
		if err != nil {
			return nil, err
		}

		tc.Authors[author] = info
		tc.Expertise[author] = expertise

		if info.Percent > maxPercent {
			maxPercent = info.Percent
			tc.PrimaryOwner = author
		}
	}

	return tc, nil
}

// GetHighRiskFunctions retorna las funciones de alto riesgo
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
		_ = json.Unmarshal([]byte(evolutionJSON), &hist.Evolution)
		_ = json.Unmarshal([]byte(teamOwnersJSON), &hist.TeamOwners)

		results = append(results, hist)
	}

	return results, nil
}
