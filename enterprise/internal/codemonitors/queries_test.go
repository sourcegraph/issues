package codemonitors

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestQueryByRecordID(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	ctx, s := newTestStore(t)
	_, id, _, userCTX := newTestUser(ctx, t, s.Handle().DB())
	m, err := s.insertTestMonitor(userCTX, t)
	if err != nil {
		t.Fatal(err)
	}
	err = s.EnqueueTriggerQueries(ctx)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.GetQueryByRecordID(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	now := s.Now()
	want := &MonitorQuery{
		Id:           1,
		Monitor:      m.ID,
		QueryString:  testQuery,
		NextRun:      now,
		LatestResult: &now,
		CreatedBy:    id,
		CreatedAt:    now,
		ChangedBy:    id,
		ChangedAt:    now,
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatalf("diff: %s", diff)
	}
}

func TestTriggerQueryNextRun(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	ctx, s := newTestStore(t)
	_, id, _, userCTX := newTestUser(ctx, t, s.Handle().DB())
	m, err := s.insertTestMonitor(userCTX, t)
	if err != nil {
		t.Fatal(err)
	}
	err = s.EnqueueTriggerQueries(ctx)
	if err != nil {
		t.Fatal(err)
	}

	wantLatestResult := s.Now().Add(time.Minute)
	wantNextRun := s.Now().Add(time.Hour)

	err = s.SetTriggerQueryNextRun(ctx, 1, wantNextRun, wantLatestResult)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.GetQueryByRecordID(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	want := &MonitorQuery{
		Id:           1,
		Monitor:      m.ID,
		QueryString:  testQuery,
		NextRun:      wantNextRun,
		LatestResult: &wantLatestResult,
		CreatedBy:    id,
		CreatedAt:    s.Now(),
		ChangedBy:    id,
		ChangedAt:    s.Now(),
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatalf("diff: %s", diff)
	}
}

func TestResetTriggerQueryTimestamps(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	ctx, s := newTestStore(t)
	_, id, _, userCTX := newTestUser(ctx, t, s.Handle().DB())
	m, err := s.insertTestMonitor(userCTX, t)
	if err != nil {
		t.Fatal(err)
	}
	now := s.Now()
	want := &MonitorQuery{
		Id:           1,
		Monitor:      m.ID,
		QueryString:  testQuery,
		NextRun:      now,
		LatestResult: &now,
		CreatedBy:    id,
		CreatedAt:    now,
		ChangedBy:    id,
		ChangedAt:    now,
	}
	got, err := s.triggerQueryByIDInt64(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatalf("diff: %s", diff)
	}

	err = s.ResetTriggerQueryTimestamps(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	got, err = s.triggerQueryByIDInt64(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	want = &MonitorQuery{
		Id:           1,
		Monitor:      m.ID,
		QueryString:  testQuery,
		NextRun:      now,
		LatestResult: nil,
		CreatedBy:    id,
		CreatedAt:    now,
		ChangedBy:    id,
		ChangedAt:    now,
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatalf("diff: %s", diff)
	}
}
