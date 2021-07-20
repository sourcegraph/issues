package graphqlbackend

import (
	"context"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
)

type ExecutionLogEntryResolver interface {
	Key() string
	Command() []string
	StartTime() DateTime
	ExitCode() int32
	Out(ctx context.Context) (string, error)
	DurationMilliseconds() int32
}

func NewExecutionLogEntryResolver(db dbutil.DB, entry workerutil.ExecutionLogEntry) *executionLogEntryResolver {
	return &executionLogEntryResolver{
		db:    db,
		entry: entry,
	}
}

type executionLogEntryResolver struct {
	db    dbutil.DB
	entry workerutil.ExecutionLogEntry
}

var _ ExecutionLogEntryResolver = &executionLogEntryResolver{}

func (r *executionLogEntryResolver) Key() string       { return r.entry.Key }
func (r *executionLogEntryResolver) Command() []string { return r.entry.Command }
func (r *executionLogEntryResolver) ExitCode() int32   { return int32(r.entry.ExitCode) }

func (r *executionLogEntryResolver) StartTime() DateTime {
	return DateTime{Time: r.entry.StartTime}
}

func (r *executionLogEntryResolver) DurationMilliseconds() int32 {
	return int32(r.entry.DurationMs)
}

func (r *executionLogEntryResolver) Out(ctx context.Context) (string, error) {
	// 🚨 SECURITY: Only site admins can view executor log contents.
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx, r.db); err != nil {
		if err != backend.ErrMustBeSiteAdmin {
			return "", err
		}

		return "", nil
	}

	return r.entry.Out, nil
}
