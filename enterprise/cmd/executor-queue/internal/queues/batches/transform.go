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
	// TODO: createAccessToken is a bit of technical debt until we figure out a
	// better solution. The problem is that src-cli needs to make requests to
	// the Sourcegraph instance *on behalf of the user*.
	//
	// Ideally we'd have something like one-time tokens that
	// * we could hand to src-cli
	// * are not visible to the user in the Sourcegraph web UI
	// * valid only for the duration of the batch spec execution
	// * and cleaned up after batch spec is executed
	//
	// Until then we create a fresh access token every time.
	//
	// GetOrCreate doesn't work because once an access token has been created
	// in the database Sourcegraph can't access the plain-text token anymore.
	// Only a hash for verification is kept in the database.
	token, err := createAccessToken(ctx, db, exec.UserID)
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

	var namespaceName string
	if exec.NamespaceUserID != 0 {
		user, err := database.Users(db).GetByID(ctx, exec.NamespaceUserID)
		if err != nil {
			return apiclient.Job{}, err
		}
		namespaceName = user.Username
	} else {
		org, err := database.Orgs(db).GetByID(ctx, exec.NamespaceOrgID)
		if err != nil {
			return apiclient.Job{}, err
		}
		namespaceName = org.Name
	}

	return apiclient.Job{
		ID:                  int(exec.ID),
		VirtualMachineFiles: map[string]string{"spec.yml": exec.BatchSpec},
		DockerSteps: []apiclient.DockerStep{ },
		CliSteps: []apiclient.CliStep{
			{
				Commands: []string{
					"batch",
					"preview",
					"-f", "spec.yml",
					"-text-only",
					"-skip-errors",
					"-n", namespaceName,
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

			// 🚨 SECURITY: Redact the access token used for src-cli to talk to
			// Sourcegraph instance.
			token: "SRC_ACCESS_TOKEN_REMOVED",
		},
	}, nil
}

const (
	accessTokenNote  = "batch-spec-execution"
	accessTokenScope = "user:all"
)

func createAccessToken(ctx context.Context, db dbutil.DB, userID int32) (string, error) {
	_, token, err := database.AccessTokens(db).Create(ctx, userID, []string{accessTokenScope}, accessTokenNote, userID)
	if err != nil {
		return "", err
	}
	return token, err
}

func makeURL(base, username, password string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}

	u.User = url.UserPassword(username, password)
	return u.String(), nil
}
