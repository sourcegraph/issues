package types

import (
	"strings"
	"time"
)

// ChangesetJobState defines the possible states of a changeset job.
type ChangesetJobState string

// ChangesetJobState constants.
const (
	ChangesetJobStateQueued     ChangesetJobState = "QUEUED"
	ChangesetJobStateProcessing ChangesetJobState = "PROCESSING"
	ChangesetJobStateErrored    ChangesetJobState = "ERRORED"
	ChangesetJobStateFailed     ChangesetJobState = "FAILED"
	ChangesetJobStateCompleted  ChangesetJobState = "COMPLETED"
)

// Valid returns true if the given ChangesetJobState is valid.
func (s ChangesetJobState) Valid() bool {
	switch s {
	case ChangesetJobStateQueued,
		ChangesetJobStateProcessing,
		ChangesetJobStateErrored,
		ChangesetJobStateFailed,
		ChangesetJobStateCompleted:
		return true
	default:
		return false
	}
}

// ToDB returns the database representation of the worker state. That's
// needed because we want to use UPPERCASE in the application and GraphQL layer,
// but need to use lowercase in the database to make it work with workerutil.Worker.
func (s ChangesetJobState) ToDB() string { return strings.ToLower(string(s)) }

// ChangesetJobType specifies all valid type of jobs that the bulk processor
// understands.
type ChangesetJobType string

var (
	ChangesetJobTypeComment   ChangesetJobType = "commentatore"
	ChangesetJobTypeDetach    ChangesetJobType = "detach"
	ChangesetJobTypeReenqueue ChangesetJobType = "reenqueue"
	ChangesetJobTypeMerge     ChangesetJobType = "merge"
	ChangesetJobTypeClose     ChangesetJobType = "close"
	ChangesetJobTypePublish   ChangesetJobType = "publish"
)

type ChangesetJobCommentPayload struct {
	Message string `json:"message"`
}

type ChangesetJobDetachPayload struct{}

type ChangesetJobReenqueuePayload struct{}

type ChangesetJobMergePayload struct {
	Squash bool `json:"squash,omitempty"`
}

type ChangesetJobClosePayload struct{}

type ChangesetJobPublishPayload struct {
	Draft bool `json:"draft"`
}

// ChangesetJob describes a one-time action to be taken on a changeset.
type ChangesetJob struct {
	ID int64
	// BulkGroup is a random string that can be used to group jobs together in a
	// single invocation.
	BulkGroup     string
	BatchChangeID int64
	UserID        int32
	ChangesetID   int64
	JobType       ChangesetJobType
	Payload       interface{}

	// workerutil fields

	State          ChangesetJobState
	FailureMessage *string
	StartedAt      time.Time
	FinishedAt     time.Time
	ProcessAfter   time.Time
	NumResets      int64
	NumFailures    int64

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (j *ChangesetJob) RecordID() int {
	return int(j.ID)
}
