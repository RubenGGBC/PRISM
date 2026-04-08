package git

import (
	"testing"
)

func TestClassifyCommit(t *testing.T) {
	analyzer := &GitAnalyzer{}

	tests := []struct {
		message  string
		expected string
	}{
		{"fix: auth bug", "bugfix"},
		{"bug: null pointer", "bugfix"},
		{"hotfix: crash on startup", "bugfix"},
		{"feat: add login", "feature"},
		{"feature: new dashboard", "feature"},
		{"breaking: change API", "breaking"},
		{"breaking change in auth", "breaking"},
		{"refactor: simplify logic", "refactor"},
		{"refactoring auth module", "refactor"},
		{"chore: update deps", "chore"},
		{"docs: update README", "chore"},
		{"update something unrelated", ""},
	}

	for _, tc := range tests {
		result := analyzer.classifyCommit(tc.message)
		if result != tc.expected {
			t.Errorf("classifyCommit(%q): expected %q, got %q", tc.message, tc.expected, result)
		}
	}
}

func TestCalculateStabilityScore(t *testing.T) {
	analyzer := &GitAnalyzer{}

	t.Run("stable active function", func(t *testing.T) {
		hist := &FunctionHistory{
			CommitsCount:    10,
			BugFixes:        1,
			BreakingChanges: 0,
			IsActive:        true,
		}
		score := analyzer.calculateStabilityScore(hist)
		if score < 70 || score > 100 {
			t.Errorf("expected score 70-100, got %d", score)
		}
	})

	t.Run("zero commits returns default", func(t *testing.T) {
		hist := &FunctionHistory{CommitsCount: 0}
		score := analyzer.calculateStabilityScore(hist)
		if score != 50 {
			t.Errorf("expected default score 50, got %d", score)
		}
	})

	t.Run("many bugs lowers score", func(t *testing.T) {
		stable := &FunctionHistory{CommitsCount: 10, BugFixes: 0, IsActive: true}
		unstable := &FunctionHistory{CommitsCount: 10, BugFixes: 8, IsActive: true}

		stableScore := analyzer.calculateStabilityScore(stable)
		unstableScore := analyzer.calculateStabilityScore(unstable)

		if stableScore <= unstableScore {
			t.Errorf("stable (%d) should score higher than unstable (%d)", stableScore, unstableScore)
		}
	})
}

func TestAssessRisk(t *testing.T) {
	analyzer := &GitAnalyzer{}

	tests := []struct {
		name     string
		commits  int
		bugs     int
		breaking int
		expected string
	}{
		{"low risk", 100, 5, 0, "LOW"},
		{"medium risk", 20, 5, 1, "MEDIUM"},
		{"high risk", 10, 5, 5, "HIGH"},
		{"zero commits", 0, 0, 0, "UNKNOWN"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			hist := &FunctionHistory{
				CommitsCount:    tc.commits,
				BugFixes:        tc.bugs,
				BreakingChanges: tc.breaking,
			}
			result := analyzer.assessRisk(hist)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestGetOwners(t *testing.T) {
	hist := &FunctionHistory{
		TeamOwners: map[string]int{
			"Alice": 60,
			"Bob":   30,
			"Carol": 10,
		},
	}

	owners := hist.GetOwners()
	if len(owners) != 3 {
		t.Errorf("expected 3 owners, got %d", len(owners))
	}
}
