package coursier

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/schema"
)

func ListArtifactIDs(ctx context.Context, config *schema.MavenConnection, groupID string) ([]string, error) {
	return runCoursierCommand(ctx, config, "complete", groupID+":")
}

func ListVersions(ctx context.Context, config *schema.MavenConnection, groupID, artifactID string) ([]string, error) {
	return runCoursierCommand(ctx, config, "complete", groupID+":"+artifactID+":")
}

func FetchVersion(ctx context.Context, config *schema.MavenConnection, groupID, artifactID, version string) (string, error) {
	fetched, err := runCoursierCommand(
		ctx,
		config,
		"fetch", "--intransitive",
		strings.Join([]string{groupID, artifactID, version}, ":"),
		"--classifier", "sources",
	)

	if err != nil {
		return "", err
	}
	if len(fetched) != 1 {
		return "", errors.Errorf("unexpected number of paths returned from coursier fetch, want %v, got %v", 1, len(fetched))
	}

	return fetched[0], nil
}

func Exists(ctx context.Context, config *schema.MavenConnection, groupID, artifactID, version string) (bool, error) {
	versions, err := runCoursierCommand(
		ctx,
		config,
		"complete",
		strings.Join([]string{groupID, artifactID, version}, ":"),
	)
	return len(versions) > 0, err
}

func runCoursierCommand(ctx context.Context, config *schema.MavenConnection, args ...string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "coursier", args...)
	cmd.Env = append(cmd.Env, fmt.Sprintf("COURSIER_CREDENTIALS=%v", config.Credentials))
	cmd.Env = append(
		cmd.Env,
		fmt.Sprintf("COURSIER_REPOSITORIES=%v", strings.Join(config.Repositories, "|")),
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return nil, errors.Wrap(err, stderr.String())
	}

	return strings.Split(string(stdout.String()), "\n"), nil
}
