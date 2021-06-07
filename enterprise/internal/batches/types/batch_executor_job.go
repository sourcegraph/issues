package types

import (
	"time"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/executor"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
)

type BatchExecutorJob struct {
	ID            int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
	CreatorUserID int32
	Job           executor.Job

	// All of the following fields are used by workerutil.Worker.
	State          string
	FailureMessage string
	StartedAt      time.Time
	FinishedAt     time.Time
	ProcessAfter   time.Time
	NumResets      int64
	NumFailures    int64
	ExecutionLogs  []workerutil.ExecutionLogEntry
}
