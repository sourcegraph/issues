package store

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/derision-test/glock"
	"github.com/google/go-cmp/cmp"
	"github.com/keegancsmith/sqlf"

	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
)

func TestStoreQueuedCount(t *testing.T) {
	db := setupStoreTest(t)

	if _, err := db.ExecContext(context.Background(), `
		INSERT INTO workerutil_test (id, state, uploaded_at)
		VALUES
			(1, 'queued', NOW() - '1 minute'::interval),
			(2, 'queued', NOW() - '2 minute'::interval),
			(3, 'state2', NOW() - '3 minute'::interval),
			(4, 'queued', NOW() - '4 minute'::interval),
			(5, 'state2', NOW() - '5 minute'::interval)
	`); err != nil {
		t.Fatalf("unexpected error inserting records: %s", err)
	}

	count, err := testStore(db, defaultTestStoreOptions(nil)).QueuedCount(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error getting queued count: %s", err)
	}
	if count != 3 {
		t.Errorf("unexpected count. want=%d have=%d", 3, count)
	}
}

func TestStoreQueuedCountFailed(t *testing.T) {
	db := setupStoreTest(t)

	if _, err := db.ExecContext(context.Background(), `
		INSERT INTO workerutil_test (id, state, uploaded_at, num_failures)
		VALUES
			(1, 'queued', NOW() - '1 minute'::interval, 0),
			(2, 'errored', NOW() - '2 minute'::interval, 2),
			(3, 'state2', NOW() - '3 minute'::interval, 0),
			(4, 'errored', NOW() - '4 minute'::interval, 3),
			(5, 'state2', NOW() - '5 minute'::interval, 0),
			(6, 'failed', NOW() - '6 minute'::interval, 1)
	`); err != nil {
		t.Fatalf("unexpected error inserting records: %s", err)
	}

	count, err := testStore(db, defaultTestStoreOptions(nil)).QueuedCount(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error getting queued count: %s", err)
	}
	if count != 2 {
		t.Errorf("unexpected count. want=%d have=%d", 2, count)
	}
}

func TestStoreQueuedCountConditions(t *testing.T) {
	db := setupStoreTest(t)

	if _, err := db.ExecContext(context.Background(), `
		INSERT INTO workerutil_test (id, state, uploaded_at)
		VALUES
			(1, 'queued', NOW() - '1 minute'::interval),
			(2, 'queued', NOW() - '2 minute'::interval),
			(3, 'state2', NOW() - '3 minute'::interval),
			(4, 'queued', NOW() - '4 minute'::interval),
			(5, 'state2', NOW() - '5 minute'::interval)
	`); err != nil {
		t.Fatalf("unexpected error inserting records: %s", err)
	}

	conditions := []*sqlf.Query{sqlf.Sprintf("w.id < 4")}
	count, err := testStore(db, defaultTestStoreOptions(nil)).QueuedCount(context.Background(), conditions)
	if err != nil {
		t.Fatalf("unexpected error getting queued count: %s", err)
	}
	if count != 2 {
		t.Errorf("unexpected count. want=%d have=%d", 2, count)
	}
}

func TestStoreDequeueKeepsHeartbeat(t *testing.T) {
	db := setupStoreTest(t)

	if _, err := db.ExecContext(context.Background(), `
		INSERT INTO workerutil_test (id, state, uploaded_at)
		VALUES
			(1, 'queued', NOW() - '1 minute'::interval),
			(2, 'queued', NOW() - '2 minute'::interval),
			(3, 'state2', NOW() - '3 minute'::interval),
			(4, 'queued', NOW() - '4 minute'::interval),
			(5, 'state2', NOW() - '5 minute'::interval)
	`); err != nil {
		t.Fatalf("unexpected error inserting records: %s", err)
	}

	now := time.Unix(1587396557, 0).UTC()
	clock := glock.NewMockClock()
	clock.SetCurrent(now)

	record, cancel, ok, err := testStore(db, defaultTestStoreOptions(clock)).Dequeue(context.Background(), "test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !ok {
		t.Fatalf("expected a dequeueable record")
	}
	defer cancel()

	if val := record.(TestRecord).ID; val != 4 {
		t.Errorf("unexpected id. want=%d have=%d", 4, val)
	}
	if val := record.(TestRecord).State; val != "processing" {
		t.Errorf("unexpected state. want=%s have=%s", "processing", val)
	}

	getTime := func() time.Time {
		time, ok, err := basestore.ScanFirstTime(db.QueryContext(context.Background(), "SELECT last_heartbeat_at FROM workerutil_test WHERE id = 4"))
		if err != nil {
			t.Fatalf("unexpected error scanning last updated at: %s", err)
		}
		if !ok {
			t.Fatalf("expected record to exist")
		}

		return time
	}

	timeout := time.Second * 5

	for i := 0; i < 10; i++ {
		expectedTime := now.Add(time.Second * time.Duration(i+1))

		// Trigger db write
		clock.BlockingAdvance(time.Second)

		// Check the last heartbeat timestamp on the target record. We don't know
		// when the db write ends and since we use multiple connections we need to
		// poll for a short period.
		if !assertEventually(timeout, func() bool { return getTime().Equal(expectedTime) }) {
			t.Fatalf("unexpected last_heartbeat_at after %s. want=%s", timeout, expectedTime)
		}
	}
}

// assertEventually calls f in a loop until it returns true, or until the given timeout
// duration elapses. This function returns true if f returns true within the timeout and
// false otherwise.
func assertEventually(timeout time.Duration, f func() bool) bool {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(timeout))
	defer cancel()

	for {
		if f() {
			return true
		}

		select {
		case <-ctx.Done():
			return false
		default:
		}
	}
}

func TestStoreDequeueState(t *testing.T) {
	db := setupStoreTest(t)

	if _, err := db.ExecContext(context.Background(), `
		INSERT INTO workerutil_test (id, state, uploaded_at)
		VALUES
			(1, 'queued', NOW() - '1 minute'::interval),
			(2, 'queued', NOW() - '2 minute'::interval),
			(3, 'state2', NOW() - '3 minute'::interval),
			(4, 'queued', NOW() - '4 minute'::interval),
			(5, 'state2', NOW() - '5 minute'::interval)
	`); err != nil {
		t.Fatalf("unexpected error inserting records: %s", err)
	}

	record, cancel, ok, err := testStore(db, defaultTestStoreOptions(nil)).Dequeue(context.Background(), "test", nil)
	assertDequeueRecordResult(t, 4, record, cancel, ok, err)
}

func TestStoreDequeueOrder(t *testing.T) {
	db := setupStoreTest(t)

	if _, err := db.ExecContext(context.Background(), `
		INSERT INTO workerutil_test (id, state, uploaded_at)
		VALUES
			(1, 'queued', NOW() - '2 minute'::interval),
			(2, 'queued', NOW() - '5 minute'::interval),
			(3, 'queued', NOW() - '3 minute'::interval),
			(4, 'queued', NOW() - '1 minute'::interval),
			(5, 'queued', NOW() - '4 minute'::interval)
	`); err != nil {
		t.Fatalf("unexpected error inserting records: %s", err)
	}

	record, cancel, ok, err := testStore(db, defaultTestStoreOptions(nil)).Dequeue(context.Background(), "test", nil)
	assertDequeueRecordResult(t, 2, record, cancel, ok, err)
}

func TestStoreDequeueConditions(t *testing.T) {
	db := setupStoreTest(t)

	if _, err := db.ExecContext(context.Background(), `
		INSERT INTO workerutil_test (id, state, uploaded_at)
		VALUES
			(1, 'queued', NOW() - '1 minute'::interval),
			(2, 'queued', NOW() - '2 minute'::interval),
			(3, 'queued', NOW() - '3 minute'::interval),
			(4, 'queued', NOW() - '4 minute'::interval),
			(5, 'queued', NOW() - '5 minute'::interval)
	`); err != nil {
		t.Fatalf("unexpected error inserting records: %s", err)
	}

	conditions := []*sqlf.Query{sqlf.Sprintf("w.id < 4")}
	record, cancel, ok, err := testStore(db, defaultTestStoreOptions(nil)).Dequeue(context.Background(), "test", conditions)
	assertDequeueRecordResult(t, 3, record, cancel, ok, err)
}

func TestStoreDequeueDelay(t *testing.T) {
	db := setupStoreTest(t)

	if _, err := db.ExecContext(context.Background(), `
		INSERT INTO workerutil_test (id, state, uploaded_at, process_after)
		VALUES
			(1, 'queued', NOW() - '1 minute'::interval, NULL),
			(2, 'queued', NOW() - '2 minute'::interval, NULL),
			(3, 'queued', NOW() - '3 minute'::interval, NOW() + '2 minute'::interval),
			(4, 'queued', NOW() - '4 minute'::interval, NULL),
			(5, 'queued', NOW() - '5 minute'::interval, NOW() + '1 minute'::interval)
	`); err != nil {
		t.Fatalf("unexpected error inserting records: %s", err)
	}

	record, cancel, ok, err := testStore(db, defaultTestStoreOptions(nil)).Dequeue(context.Background(), "test", nil)
	assertDequeueRecordResult(t, 4, record, cancel, ok, err)
}

func TestStoreDequeueView(t *testing.T) {
	db := setupStoreTest(t)

	if _, err := db.ExecContext(context.Background(), `
		INSERT INTO workerutil_test (id, state, uploaded_at)
		VALUES
			(1, 'queued', NOW() - '1 minute'::interval),
			(2, 'queued', NOW() - '2 minute'::interval),
			(3, 'queued', NOW() - '3 minute'::interval),
			(4, 'queued', NOW() - '4 minute'::interval),
			(5, 'queued', NOW() - '5 minute'::interval)
	`); err != nil {
		t.Fatalf("unexpected error inserting records: %s", err)
	}

	options := defaultTestStoreOptions(nil)
	options.ViewName = "workerutil_test_view v"
	options.Scan = testScanFirstRecordView
	options.OrderByExpression = sqlf.Sprintf("v.uploaded_at")
	options.ColumnExpressions = []*sqlf.Query{
		sqlf.Sprintf("v.id"),
		sqlf.Sprintf("v.state"),
		sqlf.Sprintf("v.new_field"),
	}

	conditions := []*sqlf.Query{sqlf.Sprintf("v.new_field < 15")}
	record, cancel, ok, err := testStore(db, options).Dequeue(context.Background(), "test", conditions)
	assertDequeueRecordViewResult(t, 2, 14, record, cancel, ok, err)
}

func TestStoreDequeueConcurrent(t *testing.T) {
	db := setupStoreTest(t)

	if _, err := db.ExecContext(context.Background(), `
		INSERT INTO workerutil_test (id, state, uploaded_at)
		VALUES
			(1, 'queued', NOW() - '2 minute'::interval),
			(2, 'queued', NOW() - '1 minute'::interval)
	`); err != nil {
		t.Fatalf("unexpected error inserting records: %s", err)
	}

	store := testStore(db, defaultTestStoreOptions(nil))

	// Worker A
	record1, cancel1, ok, err := store.Dequeue(context.Background(), "test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !ok {
		t.Fatalf("expected a dequeueable record")
	}
	defer func() { cancel1() }()

	// Worker B
	record2, cancel2, ok, err := store.Dequeue(context.Background(), "test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !ok {
		t.Fatalf("expected a second dequeueable record")
	}
	defer func() { cancel2() }()

	if val := record1.(TestRecord).ID; val != 1 {
		t.Errorf("unexpected id. want=%d have=%d", 1, val)
	}
	if val := record2.(TestRecord).ID; val != 2 {
		t.Errorf("unexpected id. want=%d have=%d", 2, val)
	}

	// Worker C
	_, _, ok, err = store.Dequeue(context.Background(), "test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if ok {
		t.Fatalf("did not expect a third dequeueable record")
	}
}

func TestStoreDequeueRetryAfter(t *testing.T) {
	db := setupStoreTest(t)

	if _, err := db.ExecContext(context.Background(), `
		INSERT INTO workerutil_test (id, state, finished_at, failure_message, num_failures, uploaded_at)
		VALUES
			(1, 'errored', NOW() - '6 minute'::interval, 'error', 3, NOW() - '2 minutes'::interval),
			(2, 'errored', NOW() - '4 minute'::interval, 'error', 0, NOW() - '3 minutes'::interval),
			(3, 'errored', NOW() - '6 minute'::interval, 'error', 5, NOW() - '4 minutes'::interval),
			(4, 'queued',                          NULL,    NULL, 0, NOW() - '1 minutes'::interval)
	`); err != nil {
		t.Fatalf("unexpected error inserting records: %s", err)
	}

	options := defaultTestStoreOptions(nil)
	options.Scan = testScanFirstRecordRetry
	options.MaxNumRetries = 5
	options.RetryAfter = 5 * time.Minute
	options.ColumnExpressions = []*sqlf.Query{
		sqlf.Sprintf("w.id"),
		sqlf.Sprintf("w.state"),
		sqlf.Sprintf("w.num_resets"),
	}
	store := testStore(db, options)

	// Dequeue errored record
	record1, cancel, ok, err := store.Dequeue(context.Background(), "test", nil)
	assertDequeueRecordRetryResult(t, 1, record1, cancel, ok, err)

	// Dequeue non-errored record
	record2, cancel, ok, err := store.Dequeue(context.Background(), "test", nil)
	assertDequeueRecordRetryResult(t, 4, record2, cancel, ok, err)

	// Does not dequeue old or max retried errored
	if _, _, ok, _ := store.Dequeue(context.Background(), "test", nil); ok {
		t.Fatalf("did not expect a third dequeueable record")
	}
}

func TestStoreDequeueRetryAfterDisabled(t *testing.T) {
	db := setupStoreTest(t)

	if _, err := db.ExecContext(context.Background(), `
		INSERT INTO workerutil_test (id, state, finished_at, failure_message, num_failures, uploaded_at)
		VALUES
			(1, 'errored', NOW() - '6 minute'::interval, 'error', 3, NOW() - '2 minutes'::interval),
			(2, 'errored', NOW() - '4 minute'::interval, 'error', 0, NOW() - '3 minutes'::interval),
			(3, 'errored', NOW() - '6 minute'::interval, 'error', 5, NOW() - '4 minutes'::interval),
			(4, 'queued',                          NULL,    NULL, 0, NOW() - '1 minutes'::interval)
	`); err != nil {
		t.Fatalf("unexpected error inserting records: %s", err)
	}

	options := defaultTestStoreOptions(nil)
	options.Scan = testScanFirstRecordRetry
	options.MaxNumRetries = 5
	options.RetryAfter = 0
	options.ColumnExpressions = []*sqlf.Query{
		sqlf.Sprintf("w.id"),
		sqlf.Sprintf("w.state"),
		sqlf.Sprintf("w.num_resets"),
	}

	store := testStore(db, options)

	// Dequeue non-errored record only
	record2, cancel, ok, err := store.Dequeue(context.Background(), "test", nil)
	assertDequeueRecordRetryResult(t, 4, record2, cancel, ok, err)

	// Does not dequeue errored
	if _, _, ok, _ := store.Dequeue(context.Background(), "test", nil); ok {
		t.Fatalf("did not expect a second dequeueable record")
	}
}

func TestStoreRequeue(t *testing.T) {
	db := setupStoreTest(t)

	if _, err := db.ExecContext(context.Background(), `
		INSERT INTO workerutil_test (id, state)
		VALUES
			(1, 'processing')
	`); err != nil {
		t.Fatalf("unexpected error inserting records: %s", err)
	}

	after := testNow().Add(time.Hour)

	if err := testStore(db, defaultTestStoreOptions(nil)).Requeue(context.Background(), 1, after); err != nil {
		t.Fatalf("unexpected error requeueing record: %s", err)
	}

	rows, err := db.QueryContext(context.Background(), `SELECT state, process_after FROM workerutil_test WHERE id = 1`)
	if err != nil {
		t.Fatalf("unexpected error querying record: %s", err)
	}
	defer func() { _ = basestore.CloseRows(rows, nil) }()

	if !rows.Next() {
		t.Fatal("expected record to exist")
	}

	var state string
	var processAfter *time.Time

	if err := rows.Scan(&state, &processAfter); err != nil {
		t.Fatalf("unexpected error scanning record: %s", err)
	}
	if state != "queued" {
		t.Errorf("unexpected state. want=%q have=%q", "queued", state)
	}
	if processAfter == nil || !processAfter.Equal(after) {
		t.Errorf("unexpected process after. want=%s have=%s", after, processAfter)
	}
}

func TestStoreAddExecutionLogEntry(t *testing.T) {
	db := setupStoreTest(t)

	if _, err := db.ExecContext(context.Background(), `
		INSERT INTO workerutil_test (id, state)
		VALUES
			(1, 'processing')
	`); err != nil {
		t.Fatalf("unexpected error inserting records: %s", err)
	}

	numEntries := 5

	for i := 0; i < numEntries; i++ {
		command := []string{"ls", "-a", fmt.Sprintf("%d", i+1)}
		payload := fmt.Sprintf("<load payload %d>", i+1)

		entry := workerutil.ExecutionLogEntry{
			Command: command,
			Out:     payload,
		}
		if err := testStore(db, defaultTestStoreOptions(nil)).AddExecutionLogEntry(context.Background(), 1, entry); err != nil {
			t.Fatalf("unexpected error adding executor log entry: %s", err)
		}
	}

	contents, err := basestore.ScanStrings(db.QueryContext(context.Background(), `SELECT unnest(execution_logs)::text FROM workerutil_test WHERE id = 1`))
	if err != nil {
		t.Fatalf("unexpected error scanning record: %s", err)
	}
	if len(contents) != numEntries {
		t.Fatalf("unexpected number of payloads. want=%d have=%d", numEntries, len(contents))
	}

	for i := 0; i < numEntries; i++ {
		var entry workerutil.ExecutionLogEntry
		if err := json.Unmarshal([]byte(contents[i]), &entry); err != nil {
			t.Fatalf("unexpected error decoding entry: %s", err)
		}

		expected := workerutil.ExecutionLogEntry{
			Command: []string{"ls", "-a", fmt.Sprintf("%d", i+1)},
			Out:     fmt.Sprintf("<load payload %d>", i+1),
		}
		if diff := cmp.Diff(expected, entry); diff != "" {
			t.Errorf("unexpected entry (-want +got):\n%s", diff)
		}
	}
}

func TestStoreMarkComplete(t *testing.T) {
	db := setupStoreTest(t)

	if _, err := db.ExecContext(context.Background(), `
		INSERT INTO workerutil_test (id, state)
		VALUES
			(1, 'processing')
	`); err != nil {
		t.Fatalf("unexpected error inserting records: %s", err)
	}

	marked, err := testStore(db, defaultTestStoreOptions(nil)).MarkComplete(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error marking record as completed: %s", err)
	}
	if !marked {
		t.Fatalf("expected record to be marked")
	}

	rows, err := db.QueryContext(context.Background(), `SELECT state, failure_message FROM workerutil_test WHERE id = 1`)
	if err != nil {
		t.Fatalf("unexpected error querying record: %s", err)
	}
	defer func() { _ = basestore.CloseRows(rows, nil) }()

	if !rows.Next() {
		t.Fatal("expected record to exist")
	}

	var state string
	var failureMessage *string
	if err := rows.Scan(&state, &failureMessage); err != nil {
		t.Fatalf("unexpected error scanning record: %s", err)
	}
	if state != "completed" {
		t.Errorf("unexpected state. want=%q have=%q", "completed", state)
	}
	if failureMessage != nil {
		t.Errorf("unexpected failure message. want=%v have=%v", nil, failureMessage)
	}
}

func TestStoreMarkCompleteNotProcessing(t *testing.T) {
	db := setupStoreTest(t)

	if _, err := db.ExecContext(context.Background(), `
		INSERT INTO workerutil_test (id, state, failure_message)
		VALUES
			(1, 'errored', 'old message')
	`); err != nil {
		t.Fatalf("unexpected error inserting records: %s", err)
	}

	marked, err := testStore(db, defaultTestStoreOptions(nil)).MarkComplete(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error marking record as completed: %s", err)
	}
	if marked {
		t.Fatalf("expected record not to be marked")
	}

	rows, err := db.QueryContext(context.Background(), `SELECT state, failure_message FROM workerutil_test WHERE id = 1`)
	if err != nil {
		t.Fatalf("unexpected error querying record: %s", err)
	}
	defer func() { _ = basestore.CloseRows(rows, nil) }()

	if !rows.Next() {
		t.Fatal("expected record to exist")
	}

	var state string
	var failureMessage *string
	if err := rows.Scan(&state, &failureMessage); err != nil {
		t.Fatalf("unexpected error scanning record: %s", err)
	}
	if state != "errored" {
		t.Errorf("unexpected state. want=%q have=%q", "errored", state)
	}
	if failureMessage == nil || *failureMessage != "old message" {
		t.Errorf("unexpected failure message. want=%v have=%v", "old message", failureMessage)
	}
}

func TestStoreMarkErrored(t *testing.T) {
	db := setupStoreTest(t)

	if _, err := db.ExecContext(context.Background(), `
		INSERT INTO workerutil_test (id, state)
		VALUES
			(1, 'processing')
	`); err != nil {
		t.Fatalf("unexpected error inserting records: %s", err)
	}

	marked, err := testStore(db, defaultTestStoreOptions(nil)).MarkErrored(context.Background(), 1, "new message")
	if err != nil {
		t.Fatalf("unexpected error marking record as errored: %s", err)
	}
	if !marked {
		t.Fatalf("expected record to be marked")
	}

	rows, err := db.QueryContext(context.Background(), `SELECT state, failure_message FROM workerutil_test WHERE id = 1`)
	if err != nil {
		t.Fatalf("unexpected error querying record: %s", err)
	}
	defer func() { _ = basestore.CloseRows(rows, nil) }()

	if !rows.Next() {
		t.Fatal("expected record to exist")
	}

	var state string
	var failureMessage *string
	if err := rows.Scan(&state, &failureMessage); err != nil {
		t.Fatalf("unexpected error scanning record: %s", err)
	}
	if state != "errored" {
		t.Errorf("unexpected state. want=%q have=%q", "errored", state)
	}
	if failureMessage == nil || *failureMessage != "new message" {
		t.Errorf("unexpected failure message. want=%v have=%v", "new message", failureMessage)
	}
}

func TestStoreMarkFailed(t *testing.T) {
	db := setupStoreTest(t)

	if _, err := db.ExecContext(context.Background(), `
		INSERT INTO workerutil_test (id, state)
		VALUES
			(1, 'processing')
	`); err != nil {
		t.Fatalf("unexpected error inserting records: %s", err)
	}

	marked, err := testStore(db, defaultTestStoreOptions(nil)).MarkFailed(context.Background(), 1, "new message")
	if err != nil {
		t.Fatalf("unexpected error marking upload as completed: %s", err)
	}
	if !marked {
		t.Fatalf("expected record to be marked")
	}

	rows, err := db.QueryContext(context.Background(), `SELECT state, failure_message FROM workerutil_test WHERE id = 1`)
	if err != nil {
		t.Fatalf("unexpected error querying record: %s", err)
	}
	defer func() { _ = basestore.CloseRows(rows, nil) }()

	if !rows.Next() {
		t.Fatal("expected record to exist")
	}

	var state string
	var failureMessage *string
	if err := rows.Scan(&state, &failureMessage); err != nil {
		t.Fatalf("unexpected error scanning record: %s", err)
	}
	if state != "failed" {
		t.Errorf("unexpected state. want=%q have=%q", "failed", state)
	}
	if failureMessage == nil || *failureMessage != "new message" {
		t.Errorf("unexpected failure message. want=%v have=%v", "new message", failureMessage)
	}
}

func TestStoreMarkErroredAlreadyCompleted(t *testing.T) {
	db := setupStoreTest(t)

	if _, err := db.ExecContext(context.Background(), `
		INSERT INTO workerutil_test (id, state)
		VALUES
			(1, 'completed')
	`); err != nil {
		t.Fatalf("unexpected error inserting records: %s", err)
	}

	marked, err := testStore(db, defaultTestStoreOptions(nil)).MarkErrored(context.Background(), 1, "new message")
	if err != nil {
		t.Fatalf("unexpected error marking record as errored: %s", err)
	}
	if !marked {
		t.Fatalf("expected record to be marked")
	}

	rows, err := db.QueryContext(context.Background(), `SELECT state, failure_message FROM workerutil_test WHERE id = 1`)
	if err != nil {
		t.Fatalf("unexpected error querying record: %s", err)
	}
	defer func() { _ = basestore.CloseRows(rows, nil) }()

	if !rows.Next() {
		t.Fatal("expected record to exist")
	}

	var state string
	var failureMessage *string
	if err := rows.Scan(&state, &failureMessage); err != nil {
		t.Fatalf("unexpected error scanning record: %s", err)
	}
	if state != "errored" {
		t.Errorf("unexpected state. want=%q have=%q", "errored", state)
	}
	if failureMessage == nil || *failureMessage != "new message" {
		t.Errorf("unexpected failure message. want=%v have=%v", "new message", failureMessage)
	}
}

func TestStoreMarkErroredAlreadyErrored(t *testing.T) {
	db := setupStoreTest(t)

	if _, err := db.ExecContext(context.Background(), `
		INSERT INTO workerutil_test (id, state, failure_message)
		VALUES
			(1, 'errored', 'old message')
	`); err != nil {
		t.Fatalf("unexpected error inserting records: %s", err)
	}

	marked, err := testStore(db, defaultTestStoreOptions(nil)).MarkErrored(context.Background(), 1, "new message")
	if err != nil {
		t.Fatalf("unexpected error marking record as errored: %s", err)
	}
	if marked {
		t.Fatalf("expected record not to be marked")
	}

	rows, err := db.QueryContext(context.Background(), `SELECT state, failure_message FROM workerutil_test WHERE id = 1`)
	if err != nil {
		t.Fatalf("unexpected error querying record: %s", err)
	}
	defer func() { _ = basestore.CloseRows(rows, nil) }()

	if !rows.Next() {
		t.Fatal("expected record to exist")
	}

	var state string
	var failureMessage *string
	if err := rows.Scan(&state, &failureMessage); err != nil {
		t.Fatalf("unexpected error scanning record: %s", err)
	}
	if state != "errored" {
		t.Errorf("unexpected state. want=%q have=%q", "errored", state)
	}
	if failureMessage == nil || *failureMessage != "old message" {
		t.Errorf("unexpected failure message. want=%v have=%v", "old message", failureMessage)
	}
}

func TestStoreMarkErroredRetriesExhausted(t *testing.T) {
	db := setupStoreTest(t)

	if _, err := db.ExecContext(context.Background(), `
		INSERT INTO workerutil_test (id, state, num_failures)
		VALUES
			(1, 'processing', 0),
			(2, 'processing', 1)
	`); err != nil {
		t.Fatalf("unexpected error inserting records: %s", err)
	}

	options := defaultTestStoreOptions(nil)
	options.MaxNumRetries = 2
	store := testStore(db, options)

	for i := 1; i < 3; i++ {
		marked, err := store.MarkErrored(context.Background(), i, "new message")
		if err != nil {
			t.Fatalf("unexpected error marking record as errored: %s", err)
		}
		if !marked {
			t.Fatalf("expected record to be marked")
		}
	}

	assertState := func(id int, wantState string) {
		q := fmt.Sprintf(`SELECT state FROM workerutil_test WHERE id = %d`, id)
		rows, err := db.QueryContext(context.Background(), q)
		if err != nil {
			t.Fatalf("unexpected error querying record: %s", err)
		}
		defer func() { _ = basestore.CloseRows(rows, nil) }()

		if !rows.Next() {
			t.Fatal("expected record to exist")
		}

		var state string
		if err := rows.Scan(&state); err != nil {
			t.Fatalf("unexpected error scanning record: %s", err)
		}
		if state != wantState {
			t.Errorf("record %d unexpected state. want=%q have=%q", id, wantState, state)
		}
	}

	assertState(1, "errored")
	assertState(2, "failed")
}

func TestStoreResetStalled(t *testing.T) {
	db := setupStoreTest(t)

	if _, err := db.ExecContext(context.Background(), `
		INSERT INTO workerutil_test (id, state, last_heartbeat_at, num_resets)
		VALUES
			(1, 'processing', NOW() - '6 second'::interval, 1),
			(2, 'processing', NOW() - '2 second'::interval, 0),
			(3, 'processing', NOW() - '3 second'::interval, 0),
			(4, 'processing', NOW() - '8 second'::interval, 0),
			(5, 'processing', NOW() - '8 second'::interval, 0),
			(6, 'processing', NOW() - '6 second'::interval, 5),
			(7, 'processing', NOW() - '8 second'::interval, 5)
	`); err != nil {
		t.Fatalf("unexpected error inserting records: %s", err)
	}

	tx, err := db.(dbutil.TxBeginner).BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()

	// Row lock record 5 in a transaction which should be skipped by ResetStalled
	if _, err := tx.Exec(`SELECT * FROM workerutil_test WHERE id = 5 FOR UPDATE`); err != nil {
		t.Fatal(err)
	}

	resetIDs, erroredIDs, err := testStore(db, defaultTestStoreOptions(nil)).ResetStalled(context.Background())
	if err != nil {
		t.Fatalf("unexpected error resetting stalled records: %s", err)
	}
	sort.Ints(resetIDs)
	sort.Ints(erroredIDs)

	if diff := cmp.Diff([]int{1, 4}, resetIDs); diff != "" {
		t.Errorf("unexpected reset ids (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff([]int{6, 7}, erroredIDs); diff != "" {
		t.Errorf("unexpected errored ids (-want +got):\n%s", diff)
	}

	rows, err := db.QueryContext(context.Background(), `SELECT state, num_resets FROM workerutil_test WHERE id = 1`)
	if err != nil {
		t.Fatalf("unexpected error querying record: %s", err)
	}
	defer func() { _ = basestore.CloseRows(rows, nil) }()

	if !rows.Next() {
		t.Fatal("expected record to exist")
	}

	var state string
	var numResets int
	if err := rows.Scan(&state, &numResets); err != nil {
		t.Fatalf("unexpected error scanning record: %s", err)
	}
	if state != "queued" {
		t.Errorf("unexpected state. want=%q have=%q", "queued", state)
	}
	if numResets != 2 {
		t.Errorf("unexpected num resets. want=%d have=%d", 2, numResets)
	}

	rows, err = db.QueryContext(context.Background(), `SELECT state FROM workerutil_test WHERE id = 6`)
	if err != nil {
		t.Fatalf("unexpected error querying record: %s", err)
	}
	defer func() { _ = basestore.CloseRows(rows, nil) }()

	if !rows.Next() {
		t.Fatal("expected record to exist")
	}

	if err := rows.Scan(&state); err != nil {
		t.Fatalf("unexpected error scanning record: %s", err)
	}
	if state != "errored" {
		t.Errorf("unexpected state. want=%q have=%q", "errored", state)
	}
}
