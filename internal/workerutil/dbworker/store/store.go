package store

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/derision-test/glock"
	"github.com/inconshreveable/log15"
	"github.com/keegancsmith/sqlf"
	"github.com/opentracing/opentracing-go/log"

	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
)

// Store is the persistence layer for the dbworker package that handles worker-side operations backed by a Postgres
// database. See Options for details on the required shape of the database tables (e.g. table column names/types).
type Store interface {
	basestore.ShareableStore

	// QueuedCount returns the number of records in the queued state matching the given conditions.
	QueuedCount(ctx context.Context, conditions []*sqlf.Query) (int, error)

	// Dequeue selects the first queued record matching the given conditions and updates the state to processing. If there
	// is such a record, it is returned. If there is no such unclaimed record, a nil record and and a nil cancel function
	// will be returned along with a false-valued flag. This method must not be called from within a transaction.
	//
	// A background goroutine that continuously updates the record's last modified time will be started. The returned cancel
	// function should be called once the record no longer needs to be locked from selection or reset by another process.
	// Most often, this will be when the handler moves the record into a terminal state.
	//
	// The supplied conditions may use the alias provided in `ViewName`, if one was supplied.
	Dequeue(ctx context.Context, workerHostname string, conditions []*sqlf.Query) (workerutil.Record, context.CancelFunc, bool, error)

	// Requeue updates the state of the record with the given identifier to queued and adds a processing delay before
	// the next dequeue of this record can be performed.
	Requeue(ctx context.Context, id int, after time.Time) error

	// AddExecutionLogEntry adds an executor log entry to the record.
	AddExecutionLogEntry(ctx context.Context, id int, entry workerutil.ExecutionLogEntry) error

	// MarkComplete attempts to update the state of the record to complete. If this record has already been moved from
	// the processing state to a terminal state, this method will have no effect. This method returns a boolean flag
	// indicating if the record was updated.
	MarkComplete(ctx context.Context, id int) (bool, error)

	// MarkErrored attempts to update the state of the record to errored. This method will only have an effect
	// if the current state of the record is processing or completed. A requeued record or a record already marked
	// with an error will not be updated. This method returns a boolean flag indicating if the record was updated.
	MarkErrored(ctx context.Context, id int, failureMessage string) (bool, error)

	// MarkFailed attempts to update the state of the record to failed. This method will only have an effect
	// if the current state of the record is processing or completed. A requeued record or a record already marked
	// with an error will not be updated. This method returns a boolean flag indicating if the record was updated.
	MarkFailed(ctx context.Context, id int, failureMessage string) (bool, error)

	// ResetStalled moves all processing records that have not received a heartbeat within `StalledMaxAge` back to the
	// queued state. In order to prevent input that continually crashes worker instances, records that have been reset
	// more than `MaxNumResets` times will be marked as errored. This method returns a list of record identifiers that
	// have been reset and a list of record identifiers that have been marked as errored.
	ResetStalled(ctx context.Context) (resetIDs, erroredIDs []int, err error)
}

type ExecutionLogEntry workerutil.ExecutionLogEntry

func (e *ExecutionLogEntry) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.Errorf("value is not []byte: %T", value)
	}

	return json.Unmarshal(b, &e)
}

func (e ExecutionLogEntry) Value() (driver.Value, error) {
	return json.Marshal(e)
}

func ExecutionLogEntries(raw []workerutil.ExecutionLogEntry) (entries []ExecutionLogEntry) {
	for _, entry := range raw {
		entries = append(entries, ExecutionLogEntry(entry))
	}

	return entries
}

type store struct {
	*basestore.Store
	options        Options
	columnReplacer *strings.Replacer
	operations     *operations
}

var _ Store = &store{}

// Options configure the behavior of Store over a particular set of tables, columns, and expressions.
type Options struct {
	// Name denotes the name of the store used to distinguish log messages and emitted metrics. The
	// store constructor will fail if this field is not supplied.
	Name string

	// TableName is the name of the table containing work records.
	//
	// The target table (and the target view referenced by `ViewName`) must have the following columns
	// and types:
	//
	//   - id: integer primary key
	//   - state: text (may be updated to `queued`, `processing`, `errored`, or `failed`)
	//   - failure_message: text
	//   - started_at: timestamp with time zone
	//   - last_heartbeat_at: timestamp with time zone
	//   - finished_at: timestamp with time zone
	//   - process_after: timestamp with time zone
	//   - num_resets: integer not null
	//   - num_failures: integer not null
	//   - execution_logs: json[] (each entry has the form of `ExecutionLogEntry`)
	//
	// The names of these columns may be customized based on the table name by adding a replacement
	// pair in the AlternateColumnNames mapping.
	//
	// It's recommended to put an index or (or partial index) on the state column for more efficient
	// dequeue operations.
	TableName string

	// AlternateColumnNames is a map from expected column names to actual column names in the target
	// table. This allows existing tables to be more easily retrofitted into the expected record
	// shape.
	AlternateColumnNames map[string]string

	// ViewName is an optional name of a view on top of the table containing work records to query when
	// selecting a candidate. If this value is not supplied, `TableName` will be used. The value supplied
	// may also indicate a table alias, which can be referenced in `OrderByExpression`, `ColumnExpressions`,
	// and the conditions supplied to `Dequeue`.
	//
	// The target of this column must be a view on top of the configured table with the same column
	// requirements as the base table described above.
	//
	// Example use case:
	// The processor for LSIF uploads supplies `lsif_uploads_with_repository_name`, a view on top of the
	// `lsif_uploads` table that joins work records with the `repo` table and adds an additional repository
	// name column. This allows `Dequeue` to return a record with additional data so that a second query
	// is not necessary by the caller.
	ViewName string

	// Scan is the function used to convert a rows object into a record of the expected shape.
	Scan RecordScanFn

	// OrderByExpression is the SQL expression used to order candidate records when selecting the next
	// batch of work to perform. This expression may use the alias provided in `ViewName`, if one was
	// supplied.
	OrderByExpression *sqlf.Query

	// ColumnExpressions are the target columns provided to the query when selecting a job record. These
	// expressions may use the alias provided in `ViewName`, if one was supplied.
	ColumnExpressions []*sqlf.Query

	// HeartbeatInterval is the interval between heartbeat updates to a job's last_heartbeat_at field. This
	// field is periodically updated while being actively processed to signal to other workers that the
	// record is neither pending nor abandoned.
	HeartbeatInterval time.Duration

	// StalledMaxAge is the maximum allowed duration between heartbeat updates of a job's last_heartbeat_at
	// field. An unmodified row that is marked as processing likely indicates that the worker that dequeued
	// the record has died.
	StalledMaxAge time.Duration

	// MaxNumResets is the maximum number of times a record can be implicitly reset back to the queued
	// state (via `ResetStalled`). If a record's reset attempts counter reaches this threshold, it will
	// be moved into the errored state rather than queued on its next reset to prevent an infinite retry
	// cycle of the same input.
	MaxNumResets int

	// RetryAfter determines whether the store dequeues jobs that have errored more than RetryAfter ago.
	// Setting this value to zero will disable retries entirely.
	//
	// If RetryAfter is a non-zero duration, the store dequeues records where:
	//
	//   - the state is 'errored'
	//   - the failed attempts counter hasn't reached MaxNumRetries
	//   - the finished_at timestamp was more than RetryAfter ago
	RetryAfter time.Duration

	// MaxNumRetries is the maximum number of times a record can be retried after an explicit failure.
	// Setting this value to zero will disable retries entirely.
	MaxNumRetries int

	// clock is used to mock out the wall clock used for heartbeat updates.
	clock glock.Clock
}

// RecordScanFn is a function that interprets row values as a particular record. This function should
// return a false-valued flag if the given result set was empty. This function must close the rows
// value if the given error value is nil.
//
// See the `CloseRows` function in the store/base package for suggested implementation details.
type RecordScanFn func(rows *sql.Rows, err error) (workerutil.Record, bool, error)

// New creates a new store with the given database handle and options.
func New(handle *basestore.TransactableHandle, options Options) Store {
	return NewWithMetrics(handle, options, &observation.TestContext)
}

func NewWithMetrics(handle *basestore.TransactableHandle, options Options, observationContext *observation.Context) Store {
	return newStore(handle, options, observationContext)
}

func newStore(handle *basestore.TransactableHandle, options Options, observationContext *observation.Context) *store {
	if options.Name == "" {
		panic("no name supplied to github.com/sourcegraph/sourcegraph/internal/dbworker/store:newStore")
	}

	if options.ViewName == "" {
		options.ViewName = options.TableName
	}

	if options.clock == nil {
		options.clock = glock.NewRealClock()
	}

	alternateColumnNames := map[string]string{}
	for _, column := range columns {
		alternateColumnNames[column.name] = column.name
	}
	for k, v := range options.AlternateColumnNames {
		alternateColumnNames[k] = v
	}

	var replacements []string
	for k, v := range alternateColumnNames {
		replacements = append(replacements, fmt.Sprintf("{%s}", k), v)
	}

	return &store{
		Store:          basestore.NewWithHandle(handle),
		options:        options,
		columnReplacer: strings.NewReplacer(replacements...),
		operations:     newOperations(options.Name, observationContext),
	}
}

// columns contain the names of the columns expected to be defined by the target table.
var columns = []struct {
	name              string
	defaultExpression bool
}{
	{"id", true},
	{"state", true},
	{"failure_message", true},
	{"started_at", true},
	{"last_heartbeat_at", false},
	{"finished_at", true},
	{"process_after", true},
	{"num_resets", true},
	{"num_failures", true},
	{"execution_logs", true},
	{"worker_hostname", true},
}

// DefaultColumnExpressions returns a slice of expressions for the default column name we expect.
func DefaultColumnExpressions() []*sqlf.Query {
	expressions := make([]*sqlf.Query, 0, len(columns))
	for _, column := range columns {
		if column.defaultExpression {
			expressions = append(expressions, sqlf.Sprintf(column.name))
		}
	}

	return expressions
}

// QueuedCount returns the number of records in the queued state matching the given conditions.
func (s *store) QueuedCount(ctx context.Context, conditions []*sqlf.Query) (_ int, err error) {
	ctx, endObservation := s.operations.queuedCount.With(ctx, &err, observation.Args{})
	defer endObservation(1, observation.Args{})

	count, _, err := basestore.ScanFirstInt(s.Query(ctx, s.formatQuery(
		queuedCountQuery,
		quote(s.options.ViewName),
		s.options.MaxNumRetries,
		makeConditionSuffix(conditions),
	)))

	return count, err
}

const queuedCountQuery = `
-- source: internal/workerutil/store.go:QueuedCount
SELECT COUNT(*) FROM %s WHERE (
	{state} = 'queued' OR
	({state} = 'errored' AND {num_failures} < %s)
) %s
`

// Dequeue selects the first queued record matching the given conditions and updates the state to processing. If there
// is such a record, it is returned. If there is no such unclaimed record, a nil record and and a nil cancel function
// will be returned along with a false-valued flag. This method must not be called from within a transaction.
//
// A background goroutine that continuously updates the record's last modified time will be started. The returned cancel
// function should be called once the record no longer needs to be locked from selection or reset by another process.
// Most often, this will be when the handler moves the record into a terminal state.
//
// The supplied conditions may use the alias provided in `ViewName`, if one was supplied.
func (s *store) Dequeue(ctx context.Context, workerHostname string, conditions []*sqlf.Query) (_ workerutil.Record, _ context.CancelFunc, _ bool, err error) {
	ctx, traceLog, endObservation := s.operations.dequeue.WithAndLogger(ctx, &err, observation.Args{})
	defer endObservation(1, observation.Args{})

	if s.InTransaction() {
		return nil, nil, false, ErrDequeueTransaction
	}

	now := s.now()

	// Select and "lock" candidate record
	id, exists, err := basestore.ScanFirstInt(s.Query(ctx, s.formatQuery(
		selectCandidateQuery,
		quote(s.options.ViewName),
		now,
		int(s.options.RetryAfter/time.Second),
		now,
		int(s.options.RetryAfter/time.Second),
		s.options.MaxNumRetries,
		makeConditionSuffix(conditions),
		s.options.OrderByExpression,
		quote(s.options.TableName),
		now,
		now,
		workerHostname,
	)))
	if err != nil {
		return nil, nil, false, err
	}
	if !exists {
		return nil, nil, false, nil
	}
	traceLog(log.Int("id", id))

	// Scan the actual record after updating its state
	record, exists, err := s.options.Scan(s.Query(ctx, s.formatQuery(
		selectRecordQuery,
		sqlf.Join(s.options.ColumnExpressions, ", "),
		quote(s.options.ViewName),
		id,
	)))
	if err != nil {
		return nil, nil, false, err
	}
	if !exists {
		return nil, nil, false, nil
	}

	// Create a background routine that periodically writes the current time to the record.
	// This will keep a record claimed by an active worker for a small amount of time so that
	// it will not be processed by a second worker concurrently.

	heartbeatCtx, cancel := context.WithCancel(ctx)
	go func() {
		for {
			select {
			case <-heartbeatCtx.Done():
				return
			case <-s.options.clock.After(s.options.HeartbeatInterval):
			}

			if err = s.Exec(heartbeatCtx, s.formatQuery(updateCandidateQuery, quote(s.options.TableName), s.now(), record.RecordID())); err != nil {
				fmt.Printf("heartbeatCtx=%+v (%T), err=%+v (%T)\n", heartbeatCtx.Err(), heartbeatCtx.Err(), err, err)
				fmt.Printf("errors.Cause(err)=%+v (%T)\n", errors.Cause(err), errors.Cause(err))
				fmt.Printf("errors.Unwrap(err)=%+v (%T)\n", errors.Unwrap(err), errors.Unwrap(err))

				select {
				case <-heartbeatCtx.Done():
					return
				default:
					if err != heartbeatCtx.Err() {
						log15.Error("Failed to refresh last_heartbeat_at", "name", s.options.Name, "id", id, "error", err)
					}
				}
			}
		}
	}()

	return record, cancel, true, nil
}

const selectCandidateQuery = `
-- source: internal/workerutil/store.go:Dequeue
WITH candidate AS (
	SELECT {id} FROM %s
	WHERE
		(
			(
				{state} = 'queued' AND
				({process_after} IS NULL OR {process_after} <= %s)
			) OR (
				%s > 0 AND
				{state} = 'errored' AND
				%s - {finished_at} > (%s * '1 second'::interval) AND
				{num_failures} < %s
			)
		)
		%s
	ORDER BY %s
	FOR UPDATE SKIP LOCKED
	LIMIT 1
)
UPDATE %s
SET
	{state} = 'processing',
	{started_at} = %s,
	{last_heartbeat_at} = %s,
	{finished_at} = NULL,
	{failure_message} = NULL,
	{worker_hostname} = %s
WHERE {id} IN (SELECT {id} FROM candidate)
RETURNING {id}
`

const selectRecordQuery = `
-- source: internal/workerutil/store.go:Dequeue
SELECT %s FROM %s WHERE {id} = %s
`

const updateCandidateQuery = `
-- source: internal/workerutil/store.go:Dequeue
UPDATE %s
SET {last_heartbeat_at} = %s
WHERE {id} = %s AND {state} = 'processing'
`

// Requeue updates the state of the record with the given identifier to queued and adds a processing delay before
// the next dequeue of this record can be performed.
func (s *store) Requeue(ctx context.Context, id int, after time.Time) (err error) {
	ctx, endObservation := s.operations.requeue.With(ctx, &err, observation.Args{LogFields: []log.Field{
		log.Int("id", id),
		log.String("after", after.String()),
	}})
	defer endObservation(1, observation.Args{})

	return s.Exec(ctx, s.formatQuery(
		requeueQuery,
		quote(s.options.TableName),
		after,
		id,
	))
}

const requeueQuery = `
-- source: internal/workerutil/store.go:Requeue
UPDATE %s
SET {state} = 'queued', {process_after} = %s
WHERE {id} = %s
`

// AddExecutionLogEntry adds an executor log entry to the record.
func (s *store) AddExecutionLogEntry(ctx context.Context, id int, entry workerutil.ExecutionLogEntry) (err error) {
	ctx, endObservation := s.operations.addExecutionLogEntry.With(ctx, &err, observation.Args{LogFields: []log.Field{
		log.Int("id", id),
	}})
	defer endObservation(1, observation.Args{})

	return s.Exec(ctx, s.formatQuery(
		addExecutionLogEntryQuery,
		quote(s.options.TableName),
		ExecutionLogEntry(entry),
		id,
	))
}

const addExecutionLogEntryQuery = `
-- source: internal/workerutil/store.go:AddExecutionLogEntry
UPDATE %s
SET {execution_logs} = {execution_logs} || %s::json
WHERE {id} = %s
`

// MarkComplete attempts to update the state of the record to complete. If this record has already been moved from
// the processing state to a terminal state, this method will have no effect. This method returns a boolean flag
// indicating if the record was updated.
func (s *store) MarkComplete(ctx context.Context, id int) (_ bool, err error) {
	ctx, endObservation := s.operations.markComplete.With(ctx, &err, observation.Args{LogFields: []log.Field{
		log.Int("id", id),
	}})
	defer endObservation(1, observation.Args{})

	_, ok, err := basestore.ScanFirstInt(s.Query(ctx, s.formatQuery(markCompleteQuery, quote(s.options.TableName), id)))
	return ok, err
}

const markCompleteQuery = `
-- source: internal/workerutil/store.go:MarkComplete
UPDATE %s
SET {state} = 'completed', {finished_at} = clock_timestamp()
WHERE {id} = %s AND {state} = 'processing'
RETURNING {id}
`

// MarkErrored attempts to update the state of the record to errored. This method will only have an effect
// if the current state of the record is processing or completed. A requeued record or a record already marked
// with an error will not be updated. This method returns a boolean flag indicating if the record was updated.
func (s *store) MarkErrored(ctx context.Context, id int, failureMessage string) (_ bool, err error) {
	ctx, endObservation := s.operations.markErrored.With(ctx, &err, observation.Args{LogFields: []log.Field{
		log.Int("id", id),
	}})
	defer endObservation(1, observation.Args{})

	q := s.formatQuery(markErroredQuery, quote(s.options.TableName), s.options.MaxNumRetries, failureMessage, id)
	_, ok, err := basestore.ScanFirstInt(s.Query(ctx, q))
	return ok, err
}

const markErroredQuery = `
-- source: internal/workerutil/store.go:MarkErrored
UPDATE %s
SET {state} = CASE WHEN {num_failures} + 1 = %d THEN 'failed' ELSE 'errored' END,
	{finished_at} = clock_timestamp(),
	{failure_message} = %s,
	{num_failures} = {num_failures} + 1
WHERE {id} = %s AND ({state} = 'processing' OR {state} = 'completed')
RETURNING {id}
`

// MarkFailed attempts to update the state of the record to failed. This method will only have an effect
// if the current state of the record is processing or completed. A requeued record or a record already marked
// with an error will not be updated. This method returns a boolean flag indicating if the record was updated.
func (s *store) MarkFailed(ctx context.Context, id int, failureMessage string) (_ bool, err error) {
	ctx, endObservation := s.operations.markFailed.With(ctx, &err, observation.Args{LogFields: []log.Field{
		log.Int("id", id),
	}})
	defer endObservation(1, observation.Args{})

	q := s.formatQuery(markFailedQuery, quote(s.options.TableName), failureMessage, id)
	_, ok, err := basestore.ScanFirstInt(s.Query(ctx, q))
	return ok, err
}

const markFailedQuery = `
-- source: internal/workerutil/store.go:MarkFailed
UPDATE %s
SET {state} = 'failed',
	{finished_at} = clock_timestamp(),
	{failure_message} = %s,
	{num_failures} = {num_failures} + 1
WHERE {id} = %s AND ({state} = 'processing' OR {state} = 'completed')
RETURNING {id}
`

// ResetStalled moves all processing records that have not received a heartbeat within `StalledMaxAge` back to the
// queued state. In order to prevent input that continually crashes worker instances, records that have been reset
// more than `MaxNumResets` times will be marked as errored. This method returns a list of record identifiers that
// have been reset and a list of record identifiers that have been marked as errored.
func (s *store) ResetStalled(ctx context.Context) (resetIDs, erroredIDs []int, err error) {
	ctx, traceLog, endObservation := s.operations.resetStalled.WithAndLogger(ctx, &err, observation.Args{})
	defer endObservation(1, observation.Args{})

	resetIDs, err = s.resetStalled(ctx, resetStalledQuery)
	if err != nil {
		return resetIDs, erroredIDs, err
	}
	traceLog(log.Int("numResetIDs", len(resetIDs)))

	erroredIDs, err = s.resetStalled(ctx, resetStalledMaxResetsQuery)
	if err != nil {
		return resetIDs, erroredIDs, err
	}
	traceLog(log.Int("numErroredIDs", len(erroredIDs)))

	return resetIDs, erroredIDs, nil
}

func (s *store) resetStalled(ctx context.Context, q string) ([]int, error) {
	return basestore.ScanInts(s.Query(
		ctx,
		s.formatQuery(
			q,
			quote(s.options.TableName),
			s.now(),
			int(s.options.StalledMaxAge/time.Second),
			s.options.MaxNumResets,
			quote(s.options.TableName),
		),
	))
}

const resetStalledQuery = `
-- source: internal/workerutil/store.go:ResetStalled
WITH stalled AS (
	SELECT {id} FROM %s
	WHERE
		{state} = 'processing' AND
		%s - {last_heartbeat_at} > (%s * '1 second'::interval) AND
		{num_resets} < %s
	FOR UPDATE SKIP LOCKED
)
UPDATE %s
SET
	{state} = 'queued',
	{started_at} = null,
	{num_resets} = {num_resets} + 1
WHERE {id} IN (SELECT {id} FROM stalled)
RETURNING {id}
`

const resetStalledMaxResetsQuery = `
-- source: internal/workerutil/store.go:ResetStalled
WITH stalled AS (
	SELECT {id} FROM %s
	WHERE
		{state} = 'processing' AND
		%s - {last_heartbeat_at} > (%s * '1 second'::interval) AND
		{num_resets} >= %s
	FOR UPDATE SKIP LOCKED
)
UPDATE %s
SET
	{state} = 'errored',
	{finished_at} = clock_timestamp(),
	{failure_message} = 'failed to process'
WHERE {id} IN (SELECT {id} FROM stalled)
RETURNING {id}
`

func (s *store) formatQuery(query string, args ...interface{}) *sqlf.Query {
	return sqlf.Sprintf(s.columnReplacer.Replace(query), args...)
}

func (s *store) now() time.Time {
	return s.options.clock.Now().UTC()
}

// quote wraps the given string in a *sqlf.Query so that it is not passed to the database
// as a parameter. It is necessary to quote things such as table names, columns, and other
// expressions that are not simple values.
func quote(s string) *sqlf.Query {
	return sqlf.Sprintf(s)
}

// makeConditionSuffix returns a *sqlf.Query containing "AND {c1 AND c2 AND ...}" when the
// given set of conditions is non-empty, and an empty string otherwise.
func makeConditionSuffix(conditions []*sqlf.Query) *sqlf.Query {
	if len(conditions) == 0 {
		return sqlf.Sprintf("")
	}

	var quotedConditions []*sqlf.Query
	for _, condition := range conditions {
		// Ensure everything is quoted in case the condition has an OR
		quotedConditions = append(quotedConditions, sqlf.Sprintf("(%s)", condition))
	}

	return sqlf.Sprintf("AND %s", sqlf.Join(quotedConditions, " AND "))
}
