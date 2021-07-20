package batches

import (
	"context"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/sourcegraph/sourcegraph/enterprise/cmd/executor-queue/internal/config"
	btypes "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types"
	apiclient "github.com/sourcegraph/sourcegraph/enterprise/internal/executor"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtesting"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

func TestTransformRecord(t *testing.T) {
	accessToken := "thisissecret-dont-tell-anyone"
	database.Mocks.AccessTokens.Create = func(subjectUserID int32, scopes []string, note string, creatorID int32) (int64, string, error) {
		return 1234, accessToken, nil
	}
	t.Cleanup(func() { database.Mocks.AccessTokens.Create = nil })

	overwriteEnv := func(k, v string) {
		old := os.Getenv(k)
		os.Setenv(k, v)
		t.Cleanup(func() { os.Setenv(k, old) })
	}
	overwriteEnv("HOME", "/home/the-test-user")
	overwriteEnv("PATH", "/home/the-test-user/bin")

	testBatchSpec := `batchSpec: yeah`
	index := &btypes.BatchSpecExecution{
		ID:              42,
		UserID:          1,
		NamespaceUserID: 1,
		BatchSpec:       testBatchSpec,
	}
	config := &Config{
		Shared: &config.SharedConfig{
			FrontendURL:      "https://test.io",
			FrontendUsername: "test*",
			FrontendPassword: "hunter2",
		},
	}

	database.Mocks.Users.GetByID = func(ctx context.Context, id int32) (*types.User, error) {
		return &types.User{
			Username: "john_namespace",
		}, nil
	}
	t.Cleanup(func() {
		database.Mocks.Users.GetByID = nil
	})

	job, err := transformRecord(context.Background(), &dbtesting.MockDB{}, index, config)
	if err != nil {
		t.Fatalf("unexpected error transforming record: %s", err)
	}

	expected := apiclient.Job{
		ID:                  42,
		VirtualMachineFiles: map[string]string{"spec.yml": testBatchSpec},
		CliSteps: []apiclient.CliStep{
			{
				Commands: []string{
					"batch", "preview",
					"-f", "spec.yml",
					"-text-only",
					"-skip-errors",
					"-n", "john_namespace",
				},
				Dir: ".",
				Env: []string{
					"SRC_ENDPOINT=https://test%2A:hunter2@test.io",
					"SRC_ACCESS_TOKEN=" + accessToken,
					"HOME=/home/the-test-user",
					"PATH=/home/the-test-user/bin",
				},
			},
		},
		RedactedValues: map[string]string{
			"https://test%2A:hunter2@test.io": "https://USERNAME_REMOVED:PASSWORD_REMOVED@test.io",
			"test*":                           "USERNAME_REMOVED",
			"hunter2":                         "PASSWORD_REMOVED",
			accessToken:                       "SRC_ACCESS_TOKEN_REMOVED",
		},
	}
	if diff := cmp.Diff(expected, job); diff != "" {
		t.Errorf("unexpected job (-want +got):\n%s", diff)
	}
}
