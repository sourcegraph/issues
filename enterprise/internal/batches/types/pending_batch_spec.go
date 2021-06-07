package types

import "time"

type PendingBatchSpec struct {
	ID            int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
	CreatorUserID int32
	Spec          string

	// All of the following fields are used by workerutil.Worker.
	State          string
	FailureMessage string
	StartedAt      time.Time
	FinishedAt     time.Time
	ProcessAfter   time.Time
	NumResets      int64
	NumFailures    int64
}
