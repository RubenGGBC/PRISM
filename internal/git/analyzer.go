package git

import (
	"fmt"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type GitAnalyzer struct {
	repo *gogit.Repository
	path string
}

// NewGitAnalyzer crea un nuevo analizador git
func NewGitAnalyzer(repoPath string) (*GitAnalyzer, error) {
	if repoPath == "" {
		repoPath = "."
	}

	repo, err := gogit.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open git repository at %s: %w", repoPath, err)
	}

	return &GitAnalyzer{
		repo: repo,
		path: repoPath,
	}, nil
}

// AnalyzeFunctionHistory extrae la historia completa de una función
func (ga *GitAnalyzer) AnalyzeFunctionHistory(filePath, functionName string) (*FunctionHistory, error) {
	head, err := ga.repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	iter, err := ga.repo.Log(&gogit.LogOptions{From: head.Hash()})
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

		eventType := ga.classifyCommit(commit.Message)
		switch eventType {
		case "bugfix":
			history.BugFixes++
		case "breaking":
			history.BreakingChanges++
		case "refactor":
			history.RefactorsCount++
		case "feature":
			history.FeaturesCount++
		}

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

	for author, count := range authorCommits {
		if totalCommits > 0 {
			history.TeamOwners[author] = (count * 100) / totalCommits
		}
	}

	sixMonthsAgo := time.Now().AddDate(0, -6, 0)
	history.IsActive = history.LastModified.After(sixMonthsAgo)
	history.StabilityScore = ga.calculateStabilityScore(history)
	history.RiskAssessment = ga.assessRisk(history)

	return history, nil
}

// AnalyzeFile analiza todos los cambios en un archivo (sin función específica)
func (ga *GitAnalyzer) AnalyzeFile(filePath string) (*FunctionHistory, error) {
	head, err := ga.repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	iter, err := ga.repo.Log(&gogit.LogOptions{From: head.Hash()})
	if err != nil {
		return nil, fmt.Errorf("failed to get git log: %w", err)
	}

	history := &FunctionHistory{
		FilePath:     filePath,
		FunctionName: "[ENTIRE FILE]",
		TeamOwners:   make(map[string]int),
		Evolution:    []EvolutionEvent{},
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

	sixMonthsAgo := time.Now().AddDate(0, -6, 0)
	history.IsActive = history.LastModified.After(sixMonthsAgo)
	history.StabilityScore = ga.calculateStabilityScore(history)
	history.RiskAssessment = ga.assessRisk(history)

	return history, nil
}

// AnalyzeRepository analiza estadísticas globales del repositorio
func (ga *GitAnalyzer) AnalyzeRepository() (*GitStats, error) {
	head, err := ga.repo.Head()
	if err != nil {
		return nil, err
	}

	iter, err := ga.repo.Log(&gogit.LogOptions{From: head.Hash()})
	if err != nil {
		return nil, err
	}

	stats := &GitStats{}
	authors := make(map[string]int)

	_ = iter.ForEach(func(commit *object.Commit) error {
		stats.TotalCommits++
		authors[commit.Author.Name]++
		return nil
	})

	stats.UniqueAuthors = len(authors)

	maxCount := 0
	for author, count := range authors {
		if count > maxCount {
			maxCount = count
			stats.MostActiveAuthor = author
		}
	}

	if stats.TotalCommits > 0 {
		stats.AvgCommitsPerFunction = float64(stats.TotalCommits)
	}

	return stats, nil
}

// GetOwners retorna los owners de una función
func (hist *FunctionHistory) GetOwners() []string {
	owners := make([]string, 0, len(hist.TeamOwners))
	for author := range hist.TeamOwners {
		owners = append(owners, author)
	}
	return owners
}

// helpers privados

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

	switch {
	case strings.Contains(lower, "fix") || strings.Contains(lower, "bug") || strings.Contains(lower, "hotfix"):
		return "bugfix"
	case strings.Contains(lower, "breaking change") || strings.Contains(lower, "breaking"):
		return "breaking"
	case strings.Contains(lower, "refactor") || strings.Contains(lower, "refactoring"):
		return "refactor"
	case strings.Contains(lower, "feat") || strings.Contains(lower, "feature"):
		return "feature"
	case strings.Contains(lower, "chore") || strings.Contains(lower, "docs"):
		return "chore"
	default:
		return ""
	}
}

func (ga *GitAnalyzer) calculateStabilityScore(hist *FunctionHistory) int {
	if hist.CommitsCount == 0 {
		return 50
	}

	bugRatio := float64(hist.BugFixes) / float64(hist.CommitsCount)
	bugScore := int((1 - bugRatio) * 50)

	breakingRatio := float64(hist.BreakingChanges) / float64(hist.CommitsCount)
	breakingScore := int((1 - breakingRatio) * 30)

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

	riskRatio := float64(hist.BugFixes+hist.BreakingChanges*2) / float64(hist.CommitsCount)

	switch {
	case riskRatio > 0.5:
		return "HIGH"
	case riskRatio > 0.25:
		return "MEDIUM"
	default:
		return "LOW"
	}
}
