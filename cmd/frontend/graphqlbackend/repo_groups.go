package graphqlbackend

import (
	"context"

	"github.com/sourcegraph/sourcegraph/internal/api"
)

type repoGroup struct {
	name         string
	repositories []api.RepoName
}

func (g repoGroup) Name() string { return g.name }

func (g repoGroup) Repositories() []string { return repoNamesToStrings(g.repositories) }

func (r *schemaResolver) RepoGroups(ctx context.Context) ([]*repoGroup, error) {
	settings, err := decodedViewerFinalSettings(ctx)
	if err != nil {
		return nil, err
	}

	groupsByName, err := resolveRepoGroups(settings)
	if err != nil {
		return nil, err
	}

	groups := make([]*repoGroup, 0, len(groupsByName))
	for name, values := range groupsByName {
		var repoPaths []api.RepoName
		for _, value := range values {
			switch v := value.(type) {
			case RepoPath:
				repoPaths = append(repoPaths, api.RepoName(v.String()))
			case RepoRegexpPattern:
				// TODO(@sourcegraph/search): decide how to handle
				// regexp patterns associated with repogroups.
				// Currently they are skipped. They either need to
				// resolve to a set of api.RepoNames or return the
				// pattern as a string.
				continue
			default:
				panic("unreachable")

			}
		}
		groups = append(groups, &repoGroup{
			name:         name,
			repositories: repoPaths,
		})
	}
	return groups, nil
}
