package git

import (
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// AnalyzeTeamContext analiza quién sabe qué en el equipo
func (ga *GitAnalyzer) AnalyzeTeamContext(filePath, functionName string) (*TeamContext, error) {
	hist, err := ga.AnalyzeFunctionHistory(filePath, functionName)
	if err != nil {
		return nil, err
	}

	tc := &TeamContext{
		FunctionName: functionName,
		Authors:      make(map[string]AuthorInfo),
		Expertise:    make(map[string]string),
	}

	head, _ := ga.repo.Head()
	iter, _ := ga.repo.Log(&gogit.LogOptions{From: head.Hash()})

	authorData := make(map[string]*AuthorInfo)

	_ = iter.ForEach(func(commit *object.Commit) error {
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

	now := time.Now()

	for author, percent := range hist.TeamOwners {
		info := authorData[author]
		if info == nil {
			continue
		}

		info.Percent = float64(percent)
		info.DaysSinceActive = int(now.Sub(info.LastSeen).Hours() / 24)

		tc.Authors[author] = *info
		tc.Expertise[author] = ga.determineExpertise(info, float64(percent))
	}

	maxPercent := 0
	for author, percent := range hist.TeamOwners {
		if percent > maxPercent {
			maxPercent = percent
			tc.PrimaryOwner = author
		}
	}

	return tc, nil
}

// determineExpertise calcula nivel de expertise de un autor
func (ga *GitAnalyzer) determineExpertise(info *AuthorInfo, percent float64) string {
	threeMonthsAgo := time.Now().AddDate(0, -3, 0)
	sixMonthsAgo := time.Now().AddDate(0, -6, 0)

	if percent >= 50 && info.LastSeen.After(threeMonthsAgo) {
		return "expert"
	}

	if percent >= 20 {
		if info.LastSeen.After(sixMonthsAgo) {
			return "familiar"
		}
		return "minimal"
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

// WhoKnowsThis retorna quién es experto en esto
func (tc *TeamContext) WhoKnowsThis() []string {
	experts := []string{}
	for author, exp := range tc.Expertise {
		if exp == "expert" {
			experts = append(experts, author)
		}
	}
	return experts
}

// RiskOfChanging retorna riesgo si un autor abandona el proyecto
func (tc *TeamContext) RiskOfChanging(author string) string {
	exp := tc.Expertise[author]
	switch exp {
	case "expert":
		return "HIGH - Critical knowledge loss"
	case "familiar":
		return "MEDIUM - Some knowledge lost"
	default:
		return "LOW - Minimal impact"
	}
}
