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

func FetchSources(ctx context.Context, config *schema.MavenConnection, dependency string) ([]string, error) {
	return runCoursierCommand(
		ctx,
		config,
		"fetch", "--intransitive",
		dependency,
		"--classifier", "sources",
	)
}

func Exists(ctx context.Context, config *schema.MavenConnection, dependency string) (bool, error) {
	versions, err := runCoursierCommand(
		ctx,
		config,
		"complete",
		dependency,
	)
	return len(versions) > 0, err
}

func runCoursierCommand(ctx context.Context, config *schema.MavenConnection, args ...string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "coursier", args...)
	if config.Credentials != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("COURSIER_CREDENTIALS=%v", config.Credentials))
	}
	if len(config.Repositories) > 0 {
		cmd.Env = append(
			cmd.Env,
			fmt.Sprintf("COURSIER_REPOSITORIES=%v", strings.Join(config.Repositories, "|")),
		)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return nil, errors.Wrap(err, stderr.String())
	}

	return strings.Split(strings.Trim(stdout.String(), " \n"), "\n"), nil
}
