package reposource

import (
	"net/url"
	"strings"
)

func DecomposeMavenPath(path string) string {
	return strings.TrimPrefix(path, "//maven/maven/")
}

func MavenRepoName(dependency string) string {
	return "maven/" + strings.Replace(dependency, ":", "/", -1)
}

func MavenCloneURL(dependency string) string {
	cloneURL := url.URL{
		Host: "maven",
		Path: MavenRepoName(dependency),
	}
	return cloneURL.String()
}
