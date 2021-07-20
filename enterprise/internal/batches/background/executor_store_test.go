package background

import (
	"context"
	"testing"
	"time"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/store"
	ct "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/testing"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
	dbworkerstore "github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker/store"

	btypes "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types"
)

func TestLoadAndExtractBatchSpecRandID(t *testing.T) {
	db := dbtest.NewDB(t, "")
	user := ct.CreateTestUser(t, db, true)

	s := store.New(db, nil)
	workStore := dbworkerstore.NewWithMetrics(s.Handle(), executorWorkerStoreOptions, &observation.TestContext)

	t.Run("success", func(t *testing.T) {
		specExec := &btypes.BatchSpecExecution{
			State:           btypes.BatchSpecExecutionStateProcessing,
			BatchSpec:       `name: testing`,
			UserID:          user.ID,
			NamespaceUserID: user.ID,
		}

		if err := s.CreateBatchSpecExecution(context.Background(), specExec); err != nil {
			t.Fatal(err)
		}

		entries := []workerutil.ExecutionLogEntry{
			{
				Key:        "setup.firecracker.start",
				Command:    []string{"ignite", "run"},
				StartTime:  time.Now().Add(-5 * time.Second),
				Out:        `stdout: cool`,
				DurationMs: 200,
			},
			{
				Key:       "step.src.0",
				Command:   []string{"src", "batch", "preview", "-f", "spec.yml", "-text-only"},
				StartTime: time.Now().Add(-5 * time.Second),
				Out: `stdout: {"operation":"PARSING_BATCH_SPEC","timestamp":"2021-07-06T09:38:51.481Z","status":"STARTED"}
stdout: {"operation":"PARSING_BATCH_SPEC","timestamp":"2021-07-06T09:38:51.481Z","status":"SUCCESS"}
stdout: {"operation":"CREATING_BATCH_SPEC","timestamp":"2021-07-06T09:38:51.528Z","status":"STARTED"}
stdout: {"operation":"CREATING_BATCH_SPEC","timestamp":"2021-07-06T09:38:51.535Z","status":"SUCCESS","message":"http://USERNAME_REMOVED:PASSWORD_REMOVED@localhost:3080/users/mrnugget/batch-changes/apply/QmF0Y2hTcGVjOiJBZFBMTDU5SXJmWCI="}
`,
				DurationMs: 200,
			},
		}

		for _, e := range entries {
			err := workStore.AddExecutionLogEntry(context.Background(), int(specExec.ID), e)
			if err != nil {
				t.Fatal(err)
			}
		}

		want := "AdPLL59IrfX"
		have, err := loadAndExtractBatchSpecRandID(context.Background(), s, specExec.ID)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		if have != want {
			t.Fatalf("wrong rand id extracted. want=%s, have=%s", want, have)
		}
	})

	t.Run("without log entry", func(t *testing.T) {
		specExec := &btypes.BatchSpecExecution{
			State:           btypes.BatchSpecExecutionStateProcessing,
			BatchSpec:       `name: testing`,
			UserID:          user.ID,
			NamespaceUserID: user.ID,
		}

		if err := s.CreateBatchSpecExecution(context.Background(), specExec); err != nil {
			t.Fatal(err)
		}

		_, err := loadAndExtractBatchSpecRandID(context.Background(), s, specExec.ID)
		if err == nil {
			t.Fatalf("expected error but got none")
		}

		if err.Error() != "no execution logs" {
			t.Fatalf("wrong error: %q", err.Error())
		}
	})
}

func TestExtractBatchSpecRandID(t *testing.T) {
	tests := []struct {
		name       string
		entries    []workerutil.ExecutionLogEntry
		wantRandID string
		wantErr    error
	}{
		{
			name: "success",
			entries: []workerutil.ExecutionLogEntry{
				{Key: "setup.firecracker.start"},
				// Reduced log output because we don't care about _all_ lines
				{
					Key: "step.src.0",
					Out: `stdout: {"operation":"PARSING_BATCH_SPEC","timestamp":"2021-07-06T09:38:51.481Z","status":"STARTED"}
stdout: {"operation":"PARSING_BATCH_SPEC","timestamp":"2021-07-06T09:38:51.481Z","status":"SUCCESS"}
stdout: {"operation":"CREATING_BATCH_SPEC","timestamp":"2021-07-06T09:38:51.528Z","status":"STARTED"}
stdout: {"operation":"CREATING_BATCH_SPEC","timestamp":"2021-07-06T09:38:51.535Z","status":"SUCCESS","message":"http://USERNAME_REMOVED:PASSWORD_REMOVED@localhost:3080/users/mrnugget/batch-changes/apply/QmF0Y2hTcGVjOiJBZFBMTDU5SXJmWCI="}
`,
				},
			},
			// Run `echo "QmF0Y2hTcGVjOiJBZFBMTDU5SXJmWCI=" |base64 -d` to get this
			wantRandID: "AdPLL59IrfX",
		},
		{

			name:    "no step.src.0 log entry",
			entries: []workerutil.ExecutionLogEntry{},
			wantErr: ErrNoBatchSpecRandID,
		},

		{

			name: "no url in the output",
			entries: []workerutil.ExecutionLogEntry{
				{
					Key: "step.src.0",
					Out: `stdout: {"operation":"PARSING_BATCH_SPEC","timestamp":"2021-07-06T09:38:51.481Z","status":"STARTED"}
stdout: {"operation":"PARSING_BATCH_SPEC","timestamp":"2021-07-06T09:38:51.481Z","status":"SUCCESS"}
`,
				},
			},
			wantErr: ErrNoBatchSpecRandID,
		},
		{
			name: "invalid url in the output",
			entries: []workerutil.ExecutionLogEntry{
				{
					Key: "step.src.0",
					Out: `stdout: {"operation":"PARSING_BATCH_SPEC","timestamp":"2021-07-06T09:38:51.481Z","status":"STARTED"}
stdout: {"operation":"PARSING_BATCH_SPEC","timestamp":"2021-07-06T09:38:51.481Z","status":"SUCCESS"}
stdout: {"operation":"CREATING_BATCH_SPEC","timestamp":"2021-07-06T09:38:51.528Z","status":"STARTED"}
stdout: {"operation":"CREATING_BATCH_SPEC","timestamp":"2021-07-06T09:38:51.535Z","status":"SUCCESS","message":"http://horse.txt"}
`,
				},
			},
			wantErr: ErrNoBatchSpecRandID,
		},

		{
			name: "additional text in log output",
			entries: []workerutil.ExecutionLogEntry{
				{
					Key: "step.src.0",
					Out: `stdout: {"operation":"PARSING_BATCH_SPEC","timestamp":"2021-07-06T09:38:51.481Z","status":"STARTED"}
stderr: HORSE
stdout: {"operation":"PARSING_BATCH_SPEC","timestamp":"2021-07-06T09:38:51.481Z","status":"SUCCESS"}
stderr: HORSE
stdout: {"operation":"CREATING_BATCH_SPEC","timestamp":"2021-07-06T09:38:51.528Z","status":"STARTED"}
stderr: HORSE
stdout: {"operation":"CREATING_BATCH_SPEC","timestamp":"2021-07-06T09:38:51.535Z","status":"SUCCESS","message":"http://USERNAME_REMOVED:PASSWORD_REMOVED@localhost:3080/users/mrnugget/batch-changes/apply/QmF0Y2hTcGVjOiJBZFBMTDU5SXJmWCI="}
`,
				},
			},
			wantRandID: "AdPLL59IrfX",
		},

		{
			name: "invalid json",
			entries: []workerutil.ExecutionLogEntry{
				{
					Key: "step.src.0",
					Out: `stdout: {"operation":"PARSING_BATCH_SPEC","timestamp":"2021-07-06T09:38:51.481Z","status":"STARTED"}
stdout: {HOOOORSE}
stdout: {HORSE}
stdout: {HORSE}
`,
				},
			},
			wantErr: ErrNoBatchSpecRandID,
		},

		{
			name: "non-json output inbetween valid json",
			entries: []workerutil.ExecutionLogEntry{
				{
					Key: "step.src.0",
					Out: `stdout: {"operation":"PARSING_BATCH_SPEC","timestamp":"2021-07-12T12:25:33.965Z","status":"STARTED"}
stdout: No changeset specs created
stdout: {"operation":"CREATING_BATCH_SPEC","timestamp":"2021-07-12T12:26:01.165Z","status":"SUCCESS","message":"https://example.com/users/erik/batch-changes/apply/QmF0Y2hTcGVjOiI5cFZPcHJyTUhNQiI="}`,
				},
			},
			wantRandID: "9pVOprrMHMB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			have, err := extractBatchSpecRandID(tt.entries)
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Fatalf("wrong error. want=%s, got=%s", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if have != tt.wantRandID {
				t.Fatalf("wrong batch spec rand id extracted. want=%q, have=%q", tt.wantRandID, have)
			}
		})
	}

}
