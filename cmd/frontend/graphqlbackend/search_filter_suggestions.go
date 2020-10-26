package graphqlbackend

import (
	"context"
	"regexp"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/internal/db"
)

// SearchFilterSuggestions provides search filter and default value suggestions.
func (r *schemaResolver) SearchFilterSuggestions(ctx context.Context) (*searchFilterSuggestions, error) {
	settings, err := decodedViewerFinalSettings(ctx)
	if err != nil {
		return nil, err
	}

	groupsByName, err := resolveRepoGroups(settings)
	if err != nil {
		return nil, err
	}
	repoGroups := make([]string, 0, len(groupsByName))
	for name := range groupsByName {
		repoGroups = append(repoGroups, name)
	}

	// List at most 10 repositories as default suggestions.
	repos, err := backend.Repos.List(ctx, db.ReposListOptions{
		LimitOffset: &db.LimitOffset{
			Limit: 10,
		},
	})
	if err != nil {
		return nil, err
	}
	repoNames := make([]string, len(repos))

	if getBoolPtr(settings.SearchGlobbing, false) {
		for i := range repos {
			repoNames[i] = string(repos[i].Name)
		}
	} else {
		for i := range repos {
			repoNames[i] = "^" + regexp.QuoteMeta(string(repos[i].Name)) + "$"
		}
	}

	return &searchFilterSuggestions{
		repogroups: repoGroups,
		repos:      repoNames,
	}, nil
}

// searchFilterSuggestions holds suggestions of search filters and their default values.
type searchFilterSuggestions struct {
	repogroups []string
	repos      []string
}

// Repogroup returns all repository groups defined in the settings.
func (s *searchFilterSuggestions) Repogroup() []string {
	return s.repogroups
}

// Repo returns a list of repositories as the default value for suggestion.
func (s *searchFilterSuggestions) Repo() []string {
	return s.repos
}
