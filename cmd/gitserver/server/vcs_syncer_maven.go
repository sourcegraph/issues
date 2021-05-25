package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"

	"github.com/sourcegraph/sourcegraph/internal/conf/reposource"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/maven/coursier"
	"github.com/sourcegraph/sourcegraph/internal/lazyregexp"
	"github.com/sourcegraph/sourcegraph/internal/vcs"
	"github.com/sourcegraph/sourcegraph/schema"
)

type MavenArtifactSyncer struct {
	Config *schema.MavenConnection
}

var _ VCSSyncer = &MavenArtifactSyncer{}

func (s MavenArtifactSyncer) Type() string {
	return "maven"
}

// IsCloneable checks to see if the VCS remote URL is cloneable. Any non-nil
// error indicates there is a problem.
func (s MavenArtifactSyncer) IsCloneable(ctx context.Context, remoteURL *vcs.URL) error {
	groupID, artifactID, version := reposource.DecomposeMavenPath(remoteURL.Path)
	exists, err := coursier.Exists(ctx, s.Config, groupID, artifactID, version)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	return errors.New(fmt.Sprintf("Maven repo %v not found", remoteURL))
}

// CloneCommand returns the command to be executed for cloning from remote.
func (s MavenArtifactSyncer) CloneCommand(ctx context.Context, remoteURL *vcs.URL, tmpPath string) (cmd *exec.Cmd, err error) {
	groupID, artifactID, version := reposource.DecomposeMavenPath(remoteURL.Path)

	path, err := coursier.FetchVersion(ctx, s.Config, groupID, artifactID, version)
	if err != nil {
		return nil, err
	}

	initCmd := exec.CommandContext(ctx, "git", "init")
	initCmd.Dir = tmpPath
	if output, err := runWith(ctx, cmd, false, nil); err != nil {
		return nil, errors.Wrapf(err, "failed to init git repository with output %q", string(output))
	}

	return exec.CommandContext(ctx, "git", "--version"), s.commitJar(ctx, GitDir(tmpPath), groupID, artifactID, path, version)
}

var versionPattern = lazyregexp.New(`refs/heads/(.+)$`)

// Fetch tries to fetch updates from the remote to given directory.
func (s MavenArtifactSyncer) Fetch(ctx context.Context, remoteURL *vcs.URL, dir GitDir) error {
	return nil
}

// RemoteShowCommand returns the command to be executed for showing remote.
func (s MavenArtifactSyncer) RemoteShowCommand(ctx context.Context, remoteURL *vcs.URL) (cmd *exec.Cmd, err error) {
	return exec.CommandContext(ctx, "git", "remote", "show", "./"), nil
}

func (s MavenArtifactSyncer) commitJar(ctx context.Context, dir GitDir, groupID, artifactID, path, version string) error {
	cmd := exec.CommandContext(ctx, "unzip", path, "-d", "./")
	dir.Set(cmd)
	if output, err := runWith(ctx, cmd, false, nil); err != nil {
		return errors.Wrapf(err, "failed to unzip with output %q", string(output))
	}

	file, err := os.Create(dir.Path("lsif-java.json"))
	if err != nil {
		return err
	}
	defer file.Close()

	jsonContents, err := json.Marshal(&lsifJavaJson{
		kind:         "maven",
		jvm:          "8",
		dependencies: []string{strings.Join([]string{groupID, artifactID, version}, ":")},
	})
	if err != nil {
		return err
	}

	_, err = file.Write(jsonContents)
	if err != nil {
		return err
	}

	cmd = exec.CommandContext(ctx, "git", "add", "*")
	dir.Set(cmd)
	if output, err := runWith(ctx, cmd, false, nil); err != nil {
		return errors.Wrapf(err, "failed to git add with output %q", string(output))
	}

	cmd = exec.CommandContext(ctx, "git", "commit", "-m", version)
	dir.Set(cmd)
	if output, err := runWith(ctx, cmd, false, nil); err != nil {
		return errors.Wrapf(err, "failed to git commit with output %q", string(output))
	}

	return nil
}

type lsifJavaJson struct {
	kind         string
	jvm          string
	dependencies []string
}
