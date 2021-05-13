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

func DecomposeMavenPath(path string) (groupID, artifactID, version string) {
	split := strings.Split(strings.TrimPrefix(path, "/"), "/")
	return split[0], split[1], split[2]
}
