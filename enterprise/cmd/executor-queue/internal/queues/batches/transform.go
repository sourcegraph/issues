package batches

import (
	"context"
	"fmt"
	"net/url"
	"os"

	btypes "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types"
	apiclient "github.com/sourcegraph/sourcegraph/enterprise/internal/executor"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
)

// transformRecord transforms a *btypes.BatchSpecExecution into an apiclient.Job.
func transformRecord(ctx context.Context, db dbutil.DB, exec *btypes.BatchSpecExecution, config *Config) (apiclient.Job, error) {
	_, token, err := database.AccessTokens(db).Create(ctx, exec.UserID, []string{"user:all"}, "batchspecexecution", exec.UserID)
	if err != nil {
		return apiclient.Job{}, err
	}

	srcEndpoint, err := makeURL(config.Shared.FrontendURL, config.Shared.FrontendUsername, config.Shared.FrontendPassword)
	if err != nil {
		return apiclient.Job{}, err
	}

	redactedSrcEndpoint, err := makeURL(config.Shared.FrontendURL, "USERNAME_REMOVED", "PASSWORD_REMOVED")
	if err != nil {
		return apiclient.Job{}, err
	}

	cliEnv := []string{
		fmt.Sprintf("SRC_ENDPOINT=%s", srcEndpoint),
		fmt.Sprintf("SRC_ACCESS_TOKEN=%s", token),

		// TODO: This is wrong here, it should be set on the executor machine
		fmt.Sprintf("HOME=%s", os.Getenv("HOME")),
		fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
	}

	return apiclient.Job{
		ID:                  int(exec.ID),
		VirtualMachineFiles: map[string]string{"spec.yml": exec.BatchSpec},
		CliSteps: []apiclient.CliStep{
			{
				Commands: []string{
					"batch",
					"preview",
					"-f", "spec.yml",
					"-text-only",
				},
				Dir: ".",
				Env: cliEnv,
			},
		},
		RedactedValues: map[string]string{
			// 🚨 SECURITY: Catch leak of upload endpoint. This is necessary in addition
			// to the below in case the username or password contains illegal URL characters,
			// which are then urlencoded and are not replaceable via byte comparison.
			srcEndpoint: redactedSrcEndpoint,

			// 🚨 SECURITY: Catch uses of fragments pulled from URL to construct another target
			// (in src-cli). We only pass the constructed URL to src-cli, which we trust not to
			// ship the values to a third party, but not to trust to ensure the values are absent
			// from the command's stdout or stderr streams.
			config.Shared.FrontendUsername: "USERNAME_REMOVED",
			config.Shared.FrontendPassword: "PASSWORD_REMOVED",
		},
	}, nil
}

func makeURL(base, username, password string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}

	u.User = url.UserPassword(username, password)
	return u.String(), nil
}
