package reposource

import (
	"strings"

	"github.com/sourcegraph/sourcegraph/internal/api"
)

func MavenRepoName(repositoryPathPattern, artifact string) api.RepoName {
	if repositoryPathPattern == "" {
		repositoryPathPattern = "{artifact}"
	}

	return api.RepoName(strings.NewReplacer(
		"{artifact}", artifact,
	).Replace(repositoryPathPattern))
}

func DecomposeMavenPath(path string) string {
	split := strings.Split(strings.TrimPrefix(path, "/"), "/")
	return strings.Join([]string{split[0], split[1], split[2]}, ":")
}
