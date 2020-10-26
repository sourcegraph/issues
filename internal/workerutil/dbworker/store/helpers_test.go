package store

import (
	"database/sql"
	"testing"
	"time"

	"github.com/keegancsmith/sqlf"
	"github.com/sourcegraph/sourcegraph/internal/db/basestore"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
)

type TestWorkRecord struct {
	ID int
}

func (r TestWorkRecord) RecordID() int {
	return r.ID
}

func testStore(options StoreOptions) *store {
	return newStore(basestore.NewHandleWithDB(dbconn.Global, sql.TxOptions{}), options)
}

type TestRecord struct {
	ID    int
	State string
}

func (v TestRecord) RecordID() int {
	return v.ID
}

func testScanFirstRecord(rows *sql.Rows, queryErr error) (v workerutil.Record, _ bool, err error) {
	if queryErr != nil {
		return nil, false, queryErr
	}
	defer func() { err = basestore.CloseRows(rows, err) }()

	if rows.Next() {
		var record TestRecord
		if err := rows.Scan(&record.ID, &record.State); err != nil {
			return nil, false, err
		}

		return record, true, nil
	}

	return nil, false, nil
}

type TestRecordView struct {
	ID       int
	State    string
	NewField int
}

func (v TestRecordView) RecordID() int {
	return v.ID
}

func testScanFirstRecordView(rows *sql.Rows, queryErr error) (v workerutil.Record, exists bool, err error) {
	if queryErr != nil {
		return nil, false, queryErr
	}
	defer func() { err = basestore.CloseRows(rows, err) }()

	if rows.Next() {
		var record TestRecordView
		if err := rows.Scan(&record.ID, &record.State, &record.NewField); err != nil {
			return nil, false, err
		}

		return record, true, nil
	}

	return nil, false, nil
}

type TestRecordRetry struct {
	ID        int
	State     string
	NumResets int
}

func (v TestRecordRetry) RecordID() int {
	return v.ID
}

func testScanFirstRecordRetry(rows *sql.Rows, queryErr error) (v workerutil.Record, exists bool, err error) {
	if queryErr != nil {
		return nil, false, queryErr
	}
	defer func() { err = basestore.CloseRows(rows, err) }()

	if rows.Next() {
		var record TestRecordRetry
		if err := rows.Scan(&record.ID, &record.State, &record.NumResets); err != nil {
			return nil, false, err
		}

		return record, true, nil
	}

	return nil, false, nil
}

func setupStoreTest(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)

	if _, err := dbconn.Global.Exec(`
		CREATE TABLE IF NOT EXISTS workerutil_test (
			id              integer NOT NULL,
			state           text NOT NULL,
			failure_message text,
			started_at      timestamp with time zone,
			finished_at     timestamp with time zone,
			process_after   timestamp with time zone,
			num_resets      integer NOT NULL default 0,
			num_failures    integer NOT NULL default 0,
			uploaded_at     timestamp with time zone NOT NULL default NOW(),
			log_contents    text
		)
	`); err != nil {
		t.Fatalf("unexpected error creating test table: %s", err)
	}

	if _, err := dbconn.Global.Exec(`
		CREATE OR REPLACE VIEW workerutil_test_view AS (
			SELECT w.*, (w.id * 7) as new_field FROM workerutil_test w
		)
	`); err != nil {
		t.Fatalf("unexpected error creating test table: %s", err)
	}
}

var defaultTestStoreOptions = StoreOptions{
	TableName:         "workerutil_test w",
	Scan:              testScanFirstRecord,
	OrderByExpression: sqlf.Sprintf("w.uploaded_at"),
	ColumnExpressions: []*sqlf.Query{
		sqlf.Sprintf("w.id"),
		sqlf.Sprintf("w.state"),
	},
	StalledMaxAge: time.Second * 5,
	MaxNumResets:  5,
	MaxNumRetries: 3,
}

func assertDequeueRecordResult(t *testing.T, expectedID int, record interface{}, tx Store, ok bool, err error) {
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !ok {
		t.Fatalf("expected a dequeueable record")
	}
	defer func() { _ = tx.Done(nil) }()

	if val := record.(TestRecord).ID; val != expectedID {
		t.Errorf("unexpected id. want=%d have=%d", expectedID, val)
	}
	if val := record.(TestRecord).State; val != "processing" {
		t.Errorf("unexpected state. want=%s have=%s", "processing", val)
	}
}

func assertDequeueRecordViewResult(t *testing.T, expectedID, expectedNewField int, record interface{}, tx Store, ok bool, err error) {
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !ok {
		t.Fatalf("expected a dequeueable record")
	}
	defer func() { _ = tx.Done(nil) }()

	if val := record.(TestRecordView).ID; val != expectedID {
		t.Errorf("unexpected id. want=%d have=%d", expectedID, val)
	}
	if val := record.(TestRecordView).State; val != "processing" {
		t.Errorf("unexpected state. want=%s have=%s", "processing", val)
	}
	if val := record.(TestRecordView).NewField; val != expectedNewField {
		t.Errorf("unexpected new field. want=%d have=%d", expectedNewField, val)
	}
}

func assertDequeueRecordRetryResult(t *testing.T, expectedID, record interface{}, tx Store, ok bool, err error) {
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !ok {
		t.Fatalf("expected a dequeueable record")
	}
	defer func() { _ = tx.Done(nil) }()

	if val := record.(TestRecordRetry).ID; val != expectedID {
		t.Errorf("unexpected id. want=%d have=%d", expectedID, val)
	}
	if val := record.(TestRecordRetry).State; val != "processing" {
		t.Errorf("unexpected state. want=%s have=%s", "processing", val)
	}
}

func testNow() time.Time {
	return time.Now().UTC().Truncate(time.Second)
}
