package codemonitors

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/keegancsmith/sqlf"
)

func TestEnqueueActionEmailsForQueryIDInt64QueryByRecordID(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	ctx, s := newTestStore(t)
	_, _, _, userCTX := newTestUser(ctx, t, s.Handle().DB())
	_, err := s.insertTestMonitor(userCTX, t)
	if err != nil {
		t.Fatal(err)
	}
	err = s.EnqueueTriggerQueries(ctx)
	if err != nil {
		t.Fatal(err)
	}
	err = s.EnqueueActionEmailsForQueryIDInt64(ctx, 1, 1)
	if err != nil {
		t.Fatal(err)
	}

	var got *ActionJob
	got, err = s.ActionJobForIDInt(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	want := &ActionJob{
		Id:             1,
		Email:          1,
		TriggerEvent:   1,
		State:          "queued",
		FailureMessage: nil,
		StartedAt:      nil,
		FinishedAt:     nil,
		ProcessAfter:   nil,
		NumResets:      0,
		NumFailures:    0,
		LogContents:    nil,
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatalf("diff: %s", diff)
	}
}

func TestGetActionJobMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	ctx, s := newTestStore(t)
	_, _, _, userCTX := newTestUser(ctx, t, s.Handle().DB())
	_, err := s.insertTestMonitor(userCTX, t)
	if err != nil {
		t.Fatal(err)
	}
	err = s.EnqueueTriggerQueries(ctx)
	if err != nil {
		t.Fatal(err)
	}

	var (
		wantNumResults       = 42
		wantQuery            = testQuery + " after:\"" + s.Now().UTC().Format(time.RFC3339) + "\""
		wantMonitorID  int64 = 1
	)
	err = s.LogSearch(ctx, wantQuery, wantNumResults, 1)
	if err != nil {
		t.Fatal(err)
	}
	err = s.EnqueueActionEmailsForQueryIDInt64(ctx, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.GetActionJobMetadata(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	want := &ActionJobMetadata{
		Description: testDescription,
		Query:       wantQuery,
		NumResults:  &wantNumResults,
		MonitorID:   wantMonitorID,
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatalf("diff: %s", diff)
	}
}

func TestScanActionJobs(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	var (
		testRecordID             = 1
		testTriggerEventID       = 1
		testQueryID        int64 = 1
	)

	ctx, s := newTestStore(t)
	_, _, _, userCTX := newTestUser(ctx, t, s.Handle().DB())
	_, err := s.insertTestMonitor(userCTX, t)
	if err != nil {
		t.Fatal(err)
	}
	err = s.EnqueueTriggerQueries(ctx)
	if err != nil {
		t.Fatal(err)
	}
	err = s.EnqueueActionEmailsForQueryIDInt64(ctx, testQueryID, testTriggerEventID)
	if err != nil {
		t.Fatal(err)
	}
	var rows *sql.Rows
	rows, err = s.Query(ctx, sqlf.Sprintf(actionJobForIDFmtStr, testRecordID))
	record, _, err := ScanActionJobs(rows, err)
	if err != nil {
		t.Fatal(err)
	}

	if record.RecordID() != testRecordID {
		t.Fatalf("got %d, want %d", record.RecordID(), testRecordID)
	}
}
