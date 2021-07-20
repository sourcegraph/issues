package server

import (
	"archive/zip"
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/cockroachdb/errors"
	"github.com/stretchr/testify/assert"

	"github.com/sourcegraph/sourcegraph/internal/extsvc/jvmpackages/coursier"
	"github.com/sourcegraph/sourcegraph/internal/vcs"
	"github.com/sourcegraph/sourcegraph/schema"
)

const (
	exampleJar                = "sources.jar"
	exampleJar2               = "sources2.jar"
	exampleFilePath           = "Example.java"
	exampleFileContents       = "package example;\npublic class Example {}\n"
	exampleFileContents2      = "package example;\npublic class Example { public static final int x = 42; }\n"
	examplePackageVersion     = "1.0.0"
	examplePackageVersion2    = "2.0.0"
	examplePackageDependency  = "org.example:example:1.0.0"
	examplePackageDependency2 = "org.example:example:2.0.0"
)

func createPlaceholderSourcesJar(t *testing.T, dir, contents, jarName string) {
	t.Helper()
	sourcesPath, err := os.Create(path.Join(dir, jarName))
	assert.Nil(t, err)
	zipWriter := zip.NewWriter(sourcesPath)
	exampleWriter, err := zipWriter.Create(exampleFilePath)
	assert.Nil(t, err)
	_, err = exampleWriter.Write([]byte(contents))
	assert.Nil(t, err)
	assert.Nil(t, zipWriter.Close())
	assert.Nil(t, sourcesPath.Close())
}

func assertCommandOutput(t *testing.T, cmd *exec.Cmd, workingDir, expectedOut string) {
	t.Helper()
	cmd.Dir = workingDir
	showOut, err := cmd.Output()
	assert.Nil(t, errors.Wrapf(err, "cmd=%q", cmd))
	if string(showOut) != expectedOut {
		t.Fatalf("got %q, want %q", showOut, expectedOut)
	}
}

func coursierScript(t *testing.T, dir string) string {
	coursierPath, err := os.OpenFile(path.Join(dir, "coursier"), os.O_CREATE|os.O_RDWR, 07777)
	assert.Nil(t, err)
	defer coursierPath.Close()
	script := fmt.Sprintf(`#!/usr/bin/env bash
ARG="$3"
if [[ "$ARG" =~ "%s" ]]; then
  echo "%s"
elif [[ "$ARG" =~ "%s" ]]; then
  echo "%s"
else
  echo "Invalid argument $1"
  exit 1
fi
`,
		examplePackageVersion, path.Join(dir, exampleJar),
		examplePackageVersion2, path.Join(dir, exampleJar2))
	_, err = coursierPath.WriteString(script)
	assert.Nil(t, err)
	return coursierPath.Name()
}

func (s JVMPackagesSyncer) runCloneCommand(t *testing.T, bareGitDirectory string, dependencies []string) {
	url := vcs.URL{
		URL: url.URL{Path: "maven/org.example/example"},
	}
	s.Config.Maven.Dependencies = dependencies
	cmd, err := s.CloneCommand(context.Background(), &url, bareGitDirectory)
	assert.Nil(t, err)
	assert.Nil(t, cmd.Run())
}

func TestJVMCloneCommand(t *testing.T) {
	dir, err := os.MkdirTemp("", "")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	createPlaceholderSourcesJar(t, dir, exampleFileContents, exampleJar)
	createPlaceholderSourcesJar(t, dir, exampleFileContents2, exampleJar2)

	coursier.CoursierBinary = coursierScript(t, dir)

	s := JVMPackagesSyncer{Config: &schema.JVMPackagesConnection{
		Maven: &schema.Maven{Dependencies: []string{}},
	}}
	bareGitDirectory := path.Join(dir, "git")

	s.runCloneCommand(t, bareGitDirectory, []string{examplePackageDependency})
	assertCommandOutput(t,
		exec.Command("git", "tag", "--list"),
		bareGitDirectory,
		"v1.0.0\n",
	)
	assertCommandOutput(t,
		exec.Command("git", "show", fmt.Sprintf("v%s:%s", examplePackageVersion, exampleFilePath)),
		bareGitDirectory,
		exampleFileContents,
	)

	s.runCloneCommand(t, bareGitDirectory, []string{examplePackageDependency, examplePackageDependency2})
	assertCommandOutput(t,
		exec.Command("git", "tag", "--list"),
		bareGitDirectory,
		"v1.0.0\nv2.0.0\n", // verify that the v2.0.0 tag got added
	)
	assertCommandOutput(t,
		exec.Command("git", "show", fmt.Sprintf("v%s:%s", examplePackageVersion, exampleFilePath)),
		bareGitDirectory,
		exampleFileContents,
	)
	assertCommandOutput(t,
		exec.Command("git", "show", fmt.Sprintf("v%s:%s", examplePackageVersion2, exampleFilePath)),
		bareGitDirectory,
		exampleFileContents2,
	)

	s.runCloneCommand(t, bareGitDirectory, []string{examplePackageDependency})
	assertCommandOutput(t,
		exec.Command("git", "show", fmt.Sprintf("v%s:%s", examplePackageVersion, exampleFilePath)),
		bareGitDirectory,
		exampleFileContents,
	)
	assertCommandOutput(t,
		exec.Command("git", "tag", "--list"),
		bareGitDirectory,
		"v1.0.0\n", // verify that the v2.0.0 tag has been removed.
	)
}
