package codeintel

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/sourcegraph/sourcegraph/enterprise/cmd/executor-queue/internal/config"
	store "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/dbstore"
	apiclient "github.com/sourcegraph/sourcegraph/enterprise/internal/executor"
)

func TestTransformRecord(t *testing.T) {
	index := store.Index{
		ID:             42,
		Commit:         "deadbeef",
		RepositoryName: "linux",
		DockerSteps: []store.DockerStep{
			{
				Image:    "alpine",
				Commands: []string{"yarn", "install"},
				Root:     "web",
			},
		},
		Root:        "web",
		Indexer:     "lsif-node",
		IndexerArgs: []string{"-p", "."},
		Outfile:     "",
	}
	config := &Config{
		Shared: &config.SharedConfig{
			FrontendURL:      "https://test.io",
			FrontendUsername: "test*",
			FrontendPassword: "hunter2",
		},
	}

	job, err := transformRecord(index, config)
	if err != nil {
		t.Fatalf("unexpected error transforming record: %s", err)
	}

	expected := apiclient.Job{
		ID:     42,
		Commit: "deadbeef",
		Workspace: apiclient.Workspace{
			RepositoryName: "linux",
		},
		VirtualMachineFiles: nil,
		DockerSteps: []apiclient.DockerStep{
			{
				Image:    "alpine",
				Commands: []string{"yarn", "install"},
				Dir:      "web",
			},
			{
				Image:    "lsif-node",
				Commands: []string{"-p ."},
				Dir:      "web",
			},
		},
		CliSteps: []apiclient.CliStep{
			{
				Commands: []string{
					"lsif", "upload",
					"-no-progress",
					"-repo", "linux",
					"-commit", "deadbeef",
					"-root", "web",
					"-upload-route", "/.executors/lsif/upload",
					"-file", "dump.lsif",
					"-associated-index-id", "42",
				},
				Dir: "web",
				Env: []string{"SRC_ENDPOINT=https://test%2A:hunter2@test.io"},
			},
		},
		RedactedValues: map[string]string{
			"https://test%2A:hunter2@test.io": "https://USERNAME_REMOVED:PASSWORD_REMOVED@test.io",
			"test*":                           "USERNAME_REMOVED",
			"hunter2":                         "PASSWORD_REMOVED",
		},
	}
	if diff := cmp.Diff(expected, job); diff != "" {
		t.Errorf("unexpected job (-want +got):\n%s", diff)
	}
}

func TestTransformRecordWithoutIndexer(t *testing.T) {
	index := store.Index{
		ID:             42,
		Commit:         "deadbeef",
		RepositoryName: "linux",
		DockerSteps: []store.DockerStep{
			{
				Image:    "alpine",
				Commands: []string{"yarn", "install"},
				Root:     "web",
			},
			{
				Image:    "lsif-node",
				Commands: []string{"-p", "."},
				Root:     "web",
			},
		},
		Root:        "",
		Indexer:     "",
		IndexerArgs: nil,
		Outfile:     "other/path/lsif.dump",
	}
	config := &Config{
		Shared: &config.SharedConfig{
			FrontendURL:      "https://test.io",
			FrontendUsername: "test*",
			FrontendPassword: "hunter2",
		},
	}

	job, err := transformRecord(index, config)
	if err != nil {
		t.Fatalf("unexpected error transforming record: %s", err)
	}

	expected := apiclient.Job{
		ID:     42,
		Commit: "deadbeef",
		Workspace: apiclient.Workspace{
			RepositoryName: "linux",
		},
		VirtualMachineFiles: nil,
		DockerSteps: []apiclient.DockerStep{
			{
				Image:    "alpine",
				Commands: []string{"yarn", "install"},
				Dir:      "web",
			},
			{
				Image:    "lsif-node",
				Commands: []string{"-p", "."},
				Dir:      "web",
			},
		},
		CliSteps: []apiclient.CliStep{
			{
				Commands: []string{
					"lsif", "upload",
					"-no-progress",
					"-repo", "linux",
					"-commit", "deadbeef",
					"-root", ".",
					"-upload-route", "/.executors/lsif/upload",
					"-file", "other/path/lsif.dump",
					"-associated-index-id", "42",
				},
				Dir: "",
				Env: []string{"SRC_ENDPOINT=https://test%2A:hunter2@test.io"},
			},
		},
		RedactedValues: map[string]string{
			"https://test%2A:hunter2@test.io": "https://USERNAME_REMOVED:PASSWORD_REMOVED@test.io",
			"test*":                           "USERNAME_REMOVED",
			"hunter2":                         "PASSWORD_REMOVED",
		},
	}
	if diff := cmp.Diff(expected, job); diff != "" {
		t.Errorf("unexpected job (-want +got):\n%s", diff)
	}
}
