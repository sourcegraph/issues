package reposource

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sourcegraph/sourcegraph/internal/api"
)

func TestDecomposeMavenPath(t *testing.T) {
	obtained, _ := ParseMavenModule("maven/org.hamcrest/hamcrest-core")
	assert.Equal(t, obtained.GroupID, "org.hamcrest")
	assert.Equal(t, obtained.ArtifactID, "hamcrest-core")
	assert.Equal(t, api.RepoName("maven/org.hamcrest/hamcrest-core"), obtained.RepoName())
}

func ParseMavenDependencyOrPanic(t *testing.T, value string) MavenDependency {
	dependency, err := ParseMavenDependency(value)
	if err != nil {
		t.Fatalf("error=%s", err)
	}
	return dependency
}

func TestSortDependencies(t *testing.T) {
	dependencies := []MavenDependency{
		ParseMavenDependencyOrPanic(t, "a:c:1.2.0"),
		ParseMavenDependencyOrPanic(t, "a:a:1.2.0"),
		ParseMavenDependencyOrPanic(t, "a:b:1.2.0"),
		ParseMavenDependencyOrPanic(t, "a:b:1.11.0"),
		ParseMavenDependencyOrPanic(t, "a:b:1.2.0-M11"),
		ParseMavenDependencyOrPanic(t, "a:b:1.2.0-M1"),
		ParseMavenDependencyOrPanic(t, "a:b:1.1.0"),
	}
	expected := []MavenDependency{
		ParseMavenDependencyOrPanic(t, "a:c:1.2.0"),
		ParseMavenDependencyOrPanic(t, "a:b:1.11.0"),
		ParseMavenDependencyOrPanic(t, "a:b:1.2.0"),
		ParseMavenDependencyOrPanic(t, "a:b:1.2.0-M11"),
		ParseMavenDependencyOrPanic(t, "a:b:1.2.0-M1"),
		ParseMavenDependencyOrPanic(t, "a:b:1.1.0"),
		ParseMavenDependencyOrPanic(t, "a:a:1.2.0"),
	}
	SortDependencies(dependencies)
	assert.Equal(t, expected, dependencies)
}
