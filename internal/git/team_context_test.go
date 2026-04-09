package git

import (
	"testing"
	"time"
)

func TestDetermineExpertise(t *testing.T) {
	analyzer := &GitAnalyzer{}
	now := time.Now()

	tests := []struct {
		name     string
		info     *AuthorInfo
		percent  float64
		expected string
	}{
		{
			name:     "expert: high % and recently active",
			info:     &AuthorInfo{LastSeen: now.AddDate(0, -1, 0)},
			percent:  60,
			expected: "expert",
		},
		{
			name:     "familiar: medium % recently active",
			info:     &AuthorInfo{LastSeen: now.AddDate(0, -2, 0)},
			percent:  25,
			expected: "familiar",
		},
		{
			name:     "minimal: high % but very inactive",
			info:     &AuthorInfo{LastSeen: now.AddDate(0, -8, 0)},
			percent:  25,
			expected: "minimal",
		},
		{
			name:     "minimal: low % and inactive",
			info:     &AuthorInfo{LastSeen: now.AddDate(0, -9, 0)},
			percent:  5,
			expected: "minimal",
		},
		{
			name:     "familiar: low % but recently active",
			info:     &AuthorInfo{LastSeen: now.AddDate(0, -1, 0)},
			percent:  5,
			expected: "familiar",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := analyzer.determineExpertise(tc.info, tc.percent)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestTeamContextMethods(t *testing.T) {
	tc := &TeamContext{
		FunctionName: "authenticate",
		PrimaryOwner: "Alice",
		Expertise: map[string]string{
			"Alice": "expert",
			"Bob":   "familiar",
			"Carol": "minimal",
		},
	}

	t.Run("GetPrimaryOwner", func(t *testing.T) {
		if tc.GetPrimaryOwner() != "Alice" {
			t.Errorf("expected Alice, got %s", tc.GetPrimaryOwner())
		}
	})

	t.Run("GetExpertise known", func(t *testing.T) {
		if tc.GetExpertise("Bob") != "familiar" {
			t.Errorf("expected familiar for Bob")
		}
	})

	t.Run("GetExpertise unknown defaults to minimal", func(t *testing.T) {
		if tc.GetExpertise("Unknown") != "minimal" {
			t.Errorf("expected minimal for unknown author")
		}
	})

	t.Run("WhoKnowsThis returns only experts", func(t *testing.T) {
		experts := tc.WhoKnowsThis()
		if len(experts) != 1 || experts[0] != "Alice" {
			t.Errorf("expected [Alice], got %v", experts)
		}
	})

	t.Run("RiskOfChanging", func(t *testing.T) {
		if tc.RiskOfChanging("Alice") != "HIGH - Critical knowledge loss" {
			t.Error("expert leaving should be HIGH risk")
		}
		if tc.RiskOfChanging("Bob") != "MEDIUM - Some knowledge lost" {
			t.Error("familiar leaving should be MEDIUM risk")
		}
		if tc.RiskOfChanging("Carol") != "LOW - Minimal impact" {
			t.Error("minimal leaving should be LOW risk")
		}
	})
}
