package codeintel

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	store "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/dbstore"
	apiclient "github.com/sourcegraph/sourcegraph/enterprise/internal/executor"
)

const defaultOutfile = "dump.lsif"
const uploadRoute = "/.executors/lsif/upload"

func transformRecord(index store.Index, config *Config) (apiclient.Job, error) {
	dockerSteps := make([]apiclient.DockerStep, 0, len(index.DockerSteps)+2)
	for _, dockerStep := range index.DockerSteps {
		dockerSteps = append(dockerSteps, apiclient.DockerStep{
			Image:    dockerStep.Image,
			Commands: dockerStep.Commands,
			Dir:      dockerStep.Root,
			Env:      nil,
		})
	}

	if index.Indexer != "" {
		dockerSteps = append(dockerSteps, apiclient.DockerStep{
			Image:    index.Indexer,
			Commands: append(index.LocalSteps, strings.Join(index.IndexerArgs, " ")),
			Dir:      index.Root,
			Env:      nil,
		})
	}

	srcEndpoint, err := makeURL(config.Shared.FrontendURL, config.Shared.FrontendUsername, config.Shared.FrontendPassword)
	if err != nil {
		return apiclient.Job{}, err
	}

	redactedSrcEndpoint, err := makeURL(config.Shared.FrontendURL, "USERNAME_REMOVED", "PASSWORD_REMOVED")
	if err != nil {
		return apiclient.Job{}, err
	}

	root := index.Root
	if root == "" {
		root = "."
	}

	outfile := index.Outfile
	if outfile == "" {
		outfile = defaultOutfile
	}

	return apiclient.Job{
		ID:             index.ID,
		Commit:         index.Commit,
		RepositoryName: index.RepositoryName,
		DockerSteps:    dockerSteps,
		CliSteps: []apiclient.CliStep{
			{
				Commands: []string{
					"lsif", "upload",
					"-no-progress",
					"-repo", index.RepositoryName,
					"-commit", index.Commit,
					"-root", root,
					"-upload-route", uploadRoute,
					"-file", outfile,
					"-associated-index-id", strconv.Itoa(index.ID),
				},
				Dir: index.Root,
				Env: []string{
					fmt.Sprintf("SRC_ENDPOINT=%s", srcEndpoint),
				},
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
