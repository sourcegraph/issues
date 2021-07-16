package reposource

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/Masterminds/semver"

	"github.com/sourcegraph/sourcegraph/internal/api"
)

type MavenModule struct {
	GroupID    string
	ArtifactID string
}

func (m *MavenModule) IsJdk() bool {
	return *m == jdkModule()
}

func (m *MavenModule) MatchesDependencyString(dependency string) bool {
	return strings.HasPrefix(dependency, fmt.Sprintf("%s:%s:", m.GroupID, m.ArtifactID))
}

func (m *MavenModule) SortText() string {
	return fmt.Sprintf("%s:%s", m.GroupID, m.ArtifactID)
}

func (m *MavenModule) LsifJavaKind() string {
	if m.IsJdk() {
		return "jdk"
	}
	return "maven"
}

func (m *MavenModule) RepoName() api.RepoName {
	if m.IsJdk() {
		return "jdk"
	}
	return api.RepoName(fmt.Sprintf("maven/%s/%s", m.GroupID, m.ArtifactID))
}

func (m *MavenModule) CloneURL() string {
	cloneURL := url.URL{Path: string(m.RepoName())}
	return cloneURL.String()
}

type MavenDependency struct {
	MavenModule
	Version         string
	SemanticVersion *semver.Version
}

// SortDependencies sorts the dependencies by the semantic version in descending
// order. The latest version of a dependency becomes the first element of the
// slice
func SortDependencies(dependencies []MavenDependency) {
	sort.Slice(dependencies, func(i, j int) bool {
		if dependencies[i].MavenModule == dependencies[j].MavenModule {
			return dependencies[i].SemanticVersion.GreaterThan(dependencies[j].SemanticVersion)
		}
		return dependencies[i].MavenModule.SortText() > dependencies[j].MavenModule.SortText()
	})
}

func (d *MavenDependency) IsJdk() bool {
	return d.MavenModule.IsJdk()
}

func (d *MavenDependency) CoursierSyntax() string {
	return fmt.Sprintf("%s:%s:%s", d.MavenModule.GroupID, d.MavenModule.ArtifactID, d.Version)
}

func (d *MavenDependency) GitTagFromVersion() string {
	return "v" + d.Version
}

func (m *MavenDependency) LsifJavaDependencies() []string {
	if m.IsJdk() {
		return []string{}
	}
	return []string{m.CoursierSyntax()}
}

// ParseMavenDependency parses a dependency string in the Coursier format (colon seperated group ID, artifact ID and version)
// into a MavenDependency.
func ParseMavenDependency(dependency string) (MavenDependency, error) {
	parts := strings.Split(dependency, ":")
	if len(parts) < 3 {
		return MavenDependency{}, fmt.Errorf("dependency %q must contain at least two colon ':' characters", dependency)
	}
	version := parts[2]

	// Ignore error from semantic version parsing because we only use the
	// semantic version for sorting dependencies, which falls back to
	// lexicographical ordering if the semantic version is missing. We can't
	// guarantee that every published Java package has a valid semantic
	// version according to the implementation of the Go-lang semver
	// package.
	semanticVersion, _ := semver.NewVersion(version)

	return MavenDependency{
		MavenModule: MavenModule{
			GroupID:    parts[0],
			ArtifactID: parts[1],
		},
		Version:         version,
		SemanticVersion: semanticVersion,
	}, nil
}

// ParseMavenModule returns a parsed JVM module from the provided URL path, without a leading `/`
func ParseMavenModule(urlPath string) (MavenModule, error) {
	if urlPath == "jdk" {
		return jdkModule(), nil
	}
	parts := strings.SplitN(strings.TrimPrefix(urlPath, "maven/"), "/", 2)
	if len(parts) != 2 {
		return MavenModule{}, fmt.Errorf("failed to parse a maven module from the path %s", urlPath)
	}

	return MavenModule{
		GroupID:    parts[0],
		ArtifactID: parts[1],
	}, nil
}

// jdkModule returns the module for the Java standard library (JDK). This module
// is technically not a "maven module" because the JDK is not published as a
// Maven library. The only difference that's relevant for Sourcegraph is that we
// use a different coursier command to download JDK sources compared to normal
// maven modules:
// - JDK sources: `coursier java-home --jvm VERSION`
// - Maven sources: `coursier fetch MAVEN_MODULE:VERSION --classifier=sources`
// Since the difference is so small, the code is easier to read/maintain if we
// model the JDK as a Maven module.
func jdkModule() MavenModule {
	return MavenModule{
		GroupID:    "jdk",
		ArtifactID: "jdk",
	}
}
