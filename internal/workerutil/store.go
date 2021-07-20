package workerutil

import (
	"context"
	"time"
)

// Record is a generic interface for record conforming to the requirements of the store.
type Record interface {
	// RecordID returns the integer primary key of the record.
	RecordID() int
}

// Store is the persistence layer for the workerutil package that handles worker-side operations.
type Store interface {
	// QueuedCount returns the number of records in the queued state. Any extra arguments supplied will be used in
	// accordance with the concrete persistence layer (e.g. additional SQL conditions for a database layer).
	QueuedCount(ctx context.Context, extraArguments interface{}) (int, error)

	// Dequeue selects a record for processing. Any extra arguments supplied will be used in accordance with the
	// concrete persistence layer (e.g. additional SQL conditions for a database layer). This method returns a boolean
	// flag indicating the existence of a processable record along with a cancel function that should be called once
	// the record is finished being processed. This will release any resources used to lock the record from selection
	// by another concurrent worker process.
	Dequeue(ctx context.Context, workerHostname string, extraArguments interface{}) (Record, context.CancelFunc, bool, error)

	// AddExecutionLogEntry adds an executor log entry to the record.
	AddExecutionLogEntry(ctx context.Context, id int, entry ExecutionLogEntry) error

	// MarkComplete attempts to update the state of the record to complete. This method returns a boolean flag indicating
	// if the record was updated.
	MarkComplete(ctx context.Context, id int) (bool, error)

	// MarkErrored attempts to update the state of the record to errored. This method returns a boolean flag indicating
	// if the record was updated.
	MarkErrored(ctx context.Context, id int, failureMessage string) (bool, error)

	// MarkFailed attempts to update the state of the record to failed. This method returns a boolean flag indicating
	// if the record was updated.
	MarkFailed(ctx context.Context, id int, failureMessage string) (bool, error)
}

// ExecutionLogEntry represents a command run by the executor.
type ExecutionLogEntry struct {
	Key        string    `json:"key"`
	Command    []string  `json:"command"`
	StartTime  time.Time `json:"startTime"`
	ExitCode   int       `json:"exitCode"`
	Out        string    `json:"out"`
	DurationMs int       `json:"durationMs"`
}
