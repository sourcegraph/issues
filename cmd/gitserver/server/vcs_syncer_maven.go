package server

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"

	"github.com/pkg/errors"

	"github.com/sourcegraph/sourcegraph/internal/conf/reposource"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/maven/coursier"
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
	dependency := reposource.DecomposeMavenPath(remoteURL.Path)
	sources, err := coursier.FetchSources(ctx, s.Config, dependency)
	if err != nil {
		return err
	}
	if len(sources) == 0 {
		return errors.Errorf("no sources.jar for dependency %s", dependency)
	}
	return nil
}

// CloneCommand returns the command to be executed for cloning from remote.
func (s MavenArtifactSyncer) CloneCommand(ctx context.Context, remoteURL *vcs.URL, tmpPath string) (cmd *exec.Cmd, err error) {
	dependency := reposource.DecomposeMavenPath(remoteURL.Path)

	paths, err := coursier.FetchSources(ctx, s.Config, dependency)
	if err != nil {
		return nil, err
	}

	if len(paths) == 0 {
		return nil, errors.Errorf("no sources.jar for dependency %s", dependency)
	}

	path := paths[0]

	initCmd := exec.CommandContext(ctx, "git", "init")
	initCmd.Dir = tmpPath
	if output, err := runWith(ctx, cmd, false, nil); err != nil {
		return nil, errors.Wrapf(err, "failed to init git repository with output %q", string(output))
	}

	return exec.CommandContext(ctx, "git", "--version"), s.commitJar(ctx, GitDir(tmpPath), dependency, path)
}

// Fetch does nothing for Maven packages because they are immutable and cannot be updated after publishing.
func (s MavenArtifactSyncer) Fetch(ctx context.Context, remoteURL *vcs.URL, dir GitDir) error {
	return nil
}

// RemoteShowCommand returns the command to be executed for showing remote.
func (s MavenArtifactSyncer) RemoteShowCommand(ctx context.Context, remoteURL *vcs.URL) (cmd *exec.Cmd, err error) {
	return exec.CommandContext(ctx, "git", "remote", "show", "./"), nil
}

func (s MavenArtifactSyncer) commitJar(ctx context.Context, dir GitDir, dependency, path string) error {
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
		Kind:         "maven",
		Jvm:          "8",
		Dependencies: []string{dependency},
	})
	if err != nil {
		return err
	}

	_, err = file.Write(jsonContents)
	if err != nil {
		return err
	}

	cmd = exec.CommandContext(ctx, "git", "add", ".")
	dir.Set(cmd)
	if output, err := runWith(ctx, cmd, false, nil); err != nil {
		return errors.Wrapf(err, "failed to git add with output %q", string(output))
	}

	cmd = exec.CommandContext(ctx, "git", "commit", "-m", dependency)
	dir.Set(cmd)
	if output, err := runWith(ctx, cmd, false, nil); err != nil {
		return errors.Wrapf(err, "failed to git commit with output %q", string(output))
	}

	return nil
}

type lsifJavaJson struct {
	Kind         string   `json:"kind"`
	Jvm          string   `json:"jvm"`
	Dependencies []string `json:"dependencies"`
}
