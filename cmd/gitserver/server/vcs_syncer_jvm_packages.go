package server

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/internal/conf/reposource"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/jvmpackages/coursier"
	"github.com/sourcegraph/sourcegraph/internal/repos"
	"github.com/sourcegraph/sourcegraph/internal/vcs"
	"github.com/sourcegraph/sourcegraph/schema"
)

const (
	// DO NOT CHANGE. This timestamp needs to be stable so that JVM package
	// repos consistently produce the same git revhash.  Changing this
	// timestamp will cause links to JVM package repos to return 404s
	// because Sourcegraph URLs can optionally include the git commit sha.
	stableGitCommitDate = "Thu Apr 8 14:24:52 2021 +0200"

	jvmMajorVersion0 = 44
)

type JVMPackagesSyncer struct {
	Config  *schema.JVMPackagesConnection
	DBStore repos.JVMPackagesRepoStore
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

	dbDeps, err := s.DBStore.GetJVMDependencyRepos(context.TODO())
	if err != nil {
		return nil, err
	}

	for _, dep := range dbDeps {
		if module.MatchesDependencyString(dep.Identifier) {
			dependency, err := reposource.ParseMavenDependency(dep.Identifier + ":" + dep.Version)
			if err != nil {
				continue
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

	sourceCodePaths, err := coursier.FetchSources(ctx, s.Config, dependency)
	if err != nil {
		return err
	}

	if len(sourceCodePaths) == 0 {
		return errors.Errorf("no sources.jar for dependency %s", dependency)
	}

	sourceCodePath := sourceCodePaths[0]

	byteCodePaths, err := coursier.FetchByteCode(ctx, s.Config, dependency)
	if err != nil {
		return err
	}

	if len(byteCodePaths) == 0 {
		return errors.Errorf("no bytecode jar for dependency %s", dependency)
	}

	byteCodePath := byteCodePaths[0]

	cmd := exec.CommandContext(ctx, "git", "init")
	if _, err := runCommandInDirectory(ctx, cmd, tmpDirectory); err != nil {
		return err
	}

	err = s.commitJar(ctx, dependency, tmpDirectory, sourceCodePath, byteCodePath)
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
func (s *JVMPackagesSyncer) commitJar(ctx context.Context, dependency reposource.MavenDependency,
	workingDirectory, sourceCodeJarPath, byteCodeJarPath string) error {
	cmd := exec.CommandContext(ctx, "unzip", sourceCodeJarPath)
	if _, err := runCommandInDirectory(ctx, cmd, workingDirectory); err != nil {
		return err
	}

	file, err := os.Create(filepath.Join(workingDirectory, "lsif-java.json"))
	if err != nil {
		return err
	}
	defer file.Close()

	jvmVersion, err := inferJVMVersionFromByteCode(byteCodeJarPath)
	if err != nil {
		return err
	}

	jsonContents, err := json.Marshal(&lsifJavaJSON{
		Kind:         "maven",
		JVM:          jvmVersion,
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

// inferJVMVersionFromByteCode returns the JVM version that was used to compile
// the bytecode in the given jar file.
func inferJVMVersionFromByteCode(byteCodeJarPath string) (string, error) {
	majorVersionString, err := classFileMajorVersion(byteCodeJarPath)
	if err != nil {
		return "", err
	}
	majorVersion, err := strconv.Atoi(majorVersionString)
	if err != nil {
		return "", errors.Wrapf(err, "failed to convert string %s to int", majorVersion)
	}

	// Java 1.1 (aka "Java 1") has major version 45 and Java 8 has major
	// version 52. To go from the major version of Java version we subtract
	// 44.
	jvmVersion := majorVersion - jvmMajorVersion0

	// The motivation to round the JVM version to the nearst stable release
	// is so that we reduce the number of JDKs on sourcegraph.com. By having
	// fewer JDK versions, features like "find references" will return
	// aggregated results for non-LTS releases.
	roundedJvmVersion := roundJVMVersionToNearestStableVersion(jvmVersion)

	return strconv.Itoa(roundedJvmVersion), nil
}

// roundJVMVersionToNearestStableVersion returns the oldest stable JVM version
// that is compatible with the given version. Java uses a time-based release
// schedule since Java 11. A new major version is released every 6 month and
// every 6th release is an LTS release. This means that a new LTS release gets
// published every 3rd year.  See
// https://www.baeldung.com/java-time-based-releases for more details.  This
// method rounds up non-LTS versions to the nearest LTS release. For example, a
// library that's published for Java 10 should be indexed with Java 11.
func roundJVMVersionToNearestStableVersion(javaVersion int) int {
	if javaVersion <= 8 {
		return 8
	}
	if javaVersion <= 11 {
		return 11
	}
	// TODO: bump this up to 17 once Java 17 LTS has been released, see https://github.com/sourcegraph/lsif-java/issues/263
	if javaVersion <= 16 {
		return 16
	}
	// Version from the future, do not round up to the next stable release.
	return javaVersion
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

// classFileMajorVersion returns the "major" version of the first `*.class` file
// inside the given jar file. For example, a jar file for a Java 8 library has
// the major version 52.
func classFileMajorVersion(byteCodeJarPath string) (string, error) {
	file, err := os.OpenFile(byteCodeJarPath, os.O_RDONLY, 0644)
	if err != nil {
		return "", err
	}
	defer file.Close()

	stat, err := os.Stat(byteCodeJarPath)
	if err != nil {
		return "", err
	}
	zipReader, err := zip.NewReader(file, stat.Size())
	if err != nil {
		return "", errors.Wrapf(err, "failed to read jar file %s", byteCodeJarPath)
	}

	for _, zipEntry := range zipReader.File {
		if !strings.HasSuffix(zipEntry.Name, ".class") {
			continue
		}
		version, err := classFileEntryMajorVersion(byteCodeJarPath, zipEntry)
		if err != nil {
			return "", nil
		}
		if version == "" {
			continue // Not a classfile
		}
		return version, nil
	}

	return "", errors.Errorf("failed to infer JVM version for jar %s because it doesn't contain any classfiles", byteCodeJarPath)
}

func classFileEntryMajorVersion(byteCodeJarPath string, zipEntry *zip.File) (string, error) {
	classFileReader, err := zipEntry.Open()
	if err != nil {
		return "", err
	}

	magicBytes := make([]byte, 8)
	read, err := classFileReader.Read(magicBytes)
	defer classFileReader.Close()
	if err != nil {
		return "", err
	}
	if read != len(magicBytes) {
		return "", errors.Errorf("failed to read 8 bytes from file %s", byteCodeJarPath)
	}

	// The structure of `*.class` files is documented here
	// https://docs.oracle.com/javase/specs/jvms/se16/html/jvms-4.html#jvms-4.1 and also
	// https://en.wikipedia.org/wiki/Java_class_file#General_layout
	// - Bytes 0-4 must match 0xcafebabe.
	// - Bytes 4-5 represent the uint16 formatted "minor" version.
	// - Bytes 5-6 represent the uint16 formatted "major" version.
	// We're only interested in the major version.
	var cafebabe uint32
	var minor uint16
	var major uint16
	buf := bytes.NewReader(magicBytes)
	binary.Read(buf, binary.BigEndian, &cafebabe)
	if cafebabe != 0xcafebabe {
		return "", nil // Not a classfile
	}
	binary.Read(buf, binary.BigEndian, &minor)
	binary.Read(buf, binary.BigEndian, &major)
	return strconv.Itoa(int(major)), nil
}
