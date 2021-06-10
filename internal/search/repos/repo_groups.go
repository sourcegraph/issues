package repos

import (
	"context"
	"fmt"
	"regexp"

	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/schema"
)

// A repogroup value is either a exact repo path RepoPath, or a regular
// expression pattern RepoRegexpPattern.
type RepoGroupValue interface {
	value()
	String() string
}

type RepoPath string
type RepoRegexpPattern string

func (RepoPath) value() {}
func (r RepoPath) String() string {
	return string(r)
}

func (RepoRegexpPattern) value() {}
func (r RepoRegexpPattern) String() string {
	return string(r)
}

var MockResolveRepoGroups func() (map[string][]RepoGroupValue, error)

func ResolveRepoGroups(ctx context.Context, db dbutil.DB, settings *schema.Settings) (groups map[string][]RepoGroupValue, err error) {
	if MockResolveRepoGroups != nil {
		return MockResolveRepoGroups()
	}
	groups = map[string][]RepoGroupValue{}

	for name, values := range settings.SearchRepositoryGroups {
		repos := make([]RepoGroupValue, 0, len(values))

		for _, value := range values {
			switch path := value.(type) {
			case string:
				repos = append(repos, RepoPath(path))
			case map[string]interface{}:
				if stringRegex, ok := path["regex"].(string); ok {
					repos = append(repos, RepoRegexpPattern(stringRegex))
				} else {
					log15.Warn("ignoring repo group value because regex not specified", "regex-string", path["regex"])
				}
			default:
				log15.Warn("ignoring repo group value of unrecognized type", "value", value, "type", fmt.Sprintf("%T", value))
			}
		}
		groups[name] = repos
	}

	if mode, err := database.Users(db).CurrentUserAllowedExternalServices(ctx); err != nil {
		return groups, err
	} else if mode == conf.ExternalServiceModeDisabled {
		return groups, nil
	}

	a := actor.FromContext(ctx)
	repos, err := database.Repos(db).ListRepoNames(ctx, database.ReposListOptions{UserID: a.UID})
	if err != nil {
		log15.Warn("getting user added repos", "err", err)
		return groups, nil
	}

	if len(repos) == 0 {
		return groups, nil
	}

	values := make([]RepoGroupValue, 0, len(repos))
	for _, repo := range repos {
		values = append(values, RepoPath(repo.Name))
	}
	groups["my"] = values

	return groups, nil
}

// repoGroupValuesToRegexp does a lookup of all repo groups by name and converts
// their values to a list of regular expressions to search.
func repoGroupValuesToRegexp(groupNames []string, groups map[string][]RepoGroupValue) []string {
	var patterns []string
	for _, groupName := range groupNames {
		for _, value := range groups[groupName] {
			switch v := value.(type) {
			case RepoPath:
				patterns = append(patterns, "^"+regexp.QuoteMeta(v.String())+"$")
			case RepoRegexpPattern:
				patterns = append(patterns, v.String())
			default:
				panic("unreachable")
			}
		}
	}
	return patterns
}
