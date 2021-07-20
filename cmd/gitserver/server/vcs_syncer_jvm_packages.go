package server

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/internal/conf/reposource"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/jvmpackages/coursier"
	"github.com/sourcegraph/sourcegraph/internal/vcs"
	"github.com/sourcegraph/sourcegraph/schema"
)

const (
	// DO NOT CHANGE. This timestamp needs to be stable so that JVM package
	// repos consistently produce the same git revhash.  Changing this
	// timestamp will cause links to JVM package repos to return 404s
	// because Sourcegraph URLs can optionally include the git commit sha.
	stableGitCommitDate = "Thu Apr 8 14:24:52 2021 +0200"
)

type JVMPackagesSyncer struct {
	Config *schema.JVMPackagesConnection
}

var _ VCSSyncer = &JVMPackagesSyncer{}

func (s *JVMPackagesSyncer) Type() string {
	return "jvm_packages"
}

// IsCloneable checks to see if the VCS remote URL is cloneable. Any non-nil
// error indicates there is a problem.
func (s *JVMPackagesSyncer) IsCloneable(ctx context.Context, remoteURL *vcs.URL) error {
	dependencies, err := s.packageDependencies(remoteURL.Path)
	if err != nil {
		return err
	}

	for _, dependency := range dependencies {
		sources, err := coursier.FetchSources(ctx, s.Config, dependency)
		if err != nil {
			return err
		}
		if len(sources) == 0 {
			return errors.Errorf("no sources.jar for dependency %s", dependency)
		}
	}
	return nil
}

// CloneCommand returns the command to be executed for cloning from remote.
// There is no external tool that performs all the step for creating a JVM
// package repository so the actual cloning happens inside this method and the
// returned command is a no-op. This means that the web UI can't display a
// helpful progress bar while cloning JVM package repositories, but that's an
// acceptable tradeoff we're willing to make.
func (s *JVMPackagesSyncer) CloneCommand(ctx context.Context, remoteURL *vcs.URL, bareGitDirectory string) (*exec.Cmd, error) {
	err := os.MkdirAll(bareGitDirectory, 0755)
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, "git", "--bare", "init")
	if _, err := runCommandInDirectory(ctx, cmd, bareGitDirectory); err != nil {
		return nil, err
	}

	// The Fetch method is responsible for cleaning up temporary directories.
	if err := s.Fetch(ctx, remoteURL, GitDir(bareGitDirectory)); err != nil {
		return nil, err
	}

	// no-op command to satisfy VCSSyncer interface, see docstring for more details.
	return exec.CommandContext(ctx, "git", "--version"), nil
}

// Fetch does nothing for Maven packages because they are immutable and cannot be updated after publishing.
func (s *JVMPackagesSyncer) Fetch(ctx context.Context, remoteURL *vcs.URL, dir GitDir) error {
	dependencies, err := s.packageDependencies(remoteURL.Path)
	if err != nil {
		return err
	}

	tags := make(map[string]bool)

	out, err := runCommandInDirectory(ctx, exec.CommandContext(ctx, "git", "tag"), string(dir))
	if err != nil {
		return err
	}

	for _, line := range strings.Split(out, "\n") {
		if len(line) == 0 {
			continue
		}
		tags[line] = true
	}

	for i, dependency := range dependencies {
		if tags[dependency.GitTagFromVersion()] {
			continue
		}
		// the gitPushDependencyTag method is reponsible for cleaning up temporary directories.
		if err := s.gitPushDependencyTag(ctx, string(dir), dependency, i == 0); err != nil {
			return errors.Wrapf(err, "error pushing dependency %q", dependency)
		}
	}

	dependencyTags := make(map[string]struct{})
	for _, dependency := range dependencies {
		dependencyTags[dependency.GitTagFromVersion()] = struct{}{}
	}

	for tag := range tags {
		if _, isDependencyTag := dependencyTags[tag]; !isDependencyTag {
			cmd := exec.CommandContext(ctx, "git", "tag", "-d", tag)
			if _, err := runCommandInDirectory(ctx, cmd, string(dir)); err != nil {
				log15.Error("Failed to delete git tag", "error", err, "tag", tag)
				continue
			}
		}
	}

	return nil
}

// RemoteShowCommand returns the command to be executed for showing remote.
func (s *JVMPackagesSyncer) RemoteShowCommand(ctx context.Context, remoteURL *vcs.URL) (cmd *exec.Cmd, err error) {
	return exec.CommandContext(ctx, "git", "remote", "show", "./"), nil
}

// packageDependencies returns the list of JVM dependencies that belong to the given URL path.
// The returned package dependencies are sorted by semantic versioning.
// A URL maps to a single JVM package, which may contain multiple versions (one git tag per version).
func (s *JVMPackagesSyncer) packageDependencies(repoUrlPath string) (dependencies []reposource.MavenDependency, err error) {
	module, err := reposource.ParseMavenModule(repoUrlPath)
	if err != nil {
		return nil, err
	}
	for _, dependency := range s.Config.Maven.Dependencies {
		if module.MatchesDependencyString(dependency) {
			dependency, err := reposource.ParseMavenDependency(dependency)
			if err != nil {
				return nil, err
			}
			dependencies = append(dependencies, dependency)
		}
	}
	if len(dependencies) == 0 {
		return nil, errors.Errorf("no tracked dependencies for URL path %s", repoUrlPath)
	}
	reposource.SortDependencies(dependencies)
	return dependencies, nil
}

// gitPushDependencyTag pushes a git tag to the given bareGitDirectory path. The
// tag points to a commit that adds all sources of given dependency. When
// isMainBranch is true, the main branch of the bare git directory will also be
// updated to point to the same commit as the git tag.
func (s *JVMPackagesSyncer) gitPushDependencyTag(ctx context.Context, bareGitDirectory string, dependency reposource.MavenDependency, isLatestVersion bool) error {
	tmpDirectory, err := ioutil.TempDir("", "maven")
	if err != nil {
		return err
	}
	// Always clean up created temporary directories.
	defer os.RemoveAll(tmpDirectory)

	paths, err := coursier.FetchSources(ctx, s.Config, dependency)
	if err != nil {
		return err
	}

	if len(paths) == 0 {
		return errors.Errorf("no sources.jar for dependency %s", dependency)
	}

	path := paths[0]

	cmd := exec.CommandContext(ctx, "git", "init")
	if _, err := runCommandInDirectory(ctx, cmd, tmpDirectory); err != nil {
		return err
	}

	err = s.commitJar(ctx, dependency, tmpDirectory, path)
	if err != nil {
		return err
	}

	cmd = exec.CommandContext(ctx, "git", "remote", "add", "origin", bareGitDirectory)
	if _, err := runCommandInDirectory(ctx, cmd, tmpDirectory); err != nil {
		return err
	}

	cmd = exec.CommandContext(ctx, "git", "push", "--force", "origin", "--tags")
	if _, err := runCommandInDirectory(ctx, cmd, tmpDirectory); err != nil {
		return err
	}

	if isLatestVersion {
		defaultBranch, err := runCommandInDirectory(ctx, exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD"), tmpDirectory)
		if err != nil {
			return err
		}
		cmd = exec.CommandContext(ctx, "git", "push", "--force", "origin", strings.TrimSpace(defaultBranch)+":latest", dependency.GitTagFromVersion())
		if _, err := runCommandInDirectory(ctx, cmd, tmpDirectory); err != nil {
			return err
		}
	}

	return nil
}

// commitJar creates a git commit in the given working directory that adds all the file contents of the given jar file.
// A `*.jar` file works the same way as a `*.zip` file, it can even be uncompressed with the `unzip` command-line tool.
func (s *JVMPackagesSyncer) commitJar(ctx context.Context, dependency reposource.MavenDependency, workingDirectory, jarPath string) error {
	cmd := exec.CommandContext(ctx, "unzip", jarPath)
	if _, err := runCommandInDirectory(ctx, cmd, workingDirectory); err != nil {
		return err
	}

	file, err := os.Create(filepath.Join(workingDirectory, "lsif-java.json"))
	if err != nil {
		return err
	}
	defer file.Close()

	jsonContents, err := json.Marshal(&lsifJavaJSON{
		Kind:         "maven",
		JVM:          "8",
		Dependencies: []string{dependency.CoursierSyntax()},
	})
	if err != nil {
		return err
	}

	_, err = file.Write(jsonContents)
	if err != nil {
		return err
	}

	cmd = exec.CommandContext(ctx, "git", "add", ".")
	if _, err := runCommandInDirectory(ctx, cmd, workingDirectory); err != nil {
		return err
	}

	cmd = exec.CommandContext(ctx, "git", "commit", "-m", dependency.CoursierSyntax(), "--date", stableGitCommitDate)
	if _, err := runCommandInDirectory(ctx, cmd, workingDirectory); err != nil {
		return err
	}

	cmd = exec.CommandContext(ctx, "git", "tag", "-m", dependency.CoursierSyntax(), dependency.GitTagFromVersion())
	if _, err := runCommandInDirectory(ctx, cmd, workingDirectory); err != nil {
		return err
	}

	return nil
}

func runCommandInDirectory(ctx context.Context, cmd *exec.Cmd, workingDirectory string) (string, error) {
	cmd.Dir = workingDirectory
	output, err := runWith(ctx, cmd, false, nil)
	if err != nil {
		return "", errors.Wrapf(err, "command %s failed with output %s", cmd.Args, string(output))
	}
	return string(output), nil
}

type lsifJavaJSON struct {
	Kind         string   `json:"kind"`
	JVM          string   `json:"jvm"`
	Dependencies []string `json:"dependencies"`
}
