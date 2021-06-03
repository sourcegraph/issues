package reposource

import (
	"net/url"
	"strings"
)

func DecomposeMavenPath(path string) string {
	return strings.TrimPrefix(path, "//maven/maven/")
}

func MavenRepoName(dependency string) string {
	return "maven/" + dependency
}

func MavenCloneURL(dependency string) string {
	cloneURL := url.URL{
		Host: "maven",
		Path: MavenRepoName(dependency),
	}
	return cloneURL.String()
}
