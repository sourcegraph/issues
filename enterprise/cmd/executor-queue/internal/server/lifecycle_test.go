package server

import (
	"context"
	"testing"
	"time"

	"github.com/derision-test/glock"
	"github.com/google/go-cmp/cmp"

	apiclient "github.com/sourcegraph/sourcegraph/enterprise/internal/executor"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
	workerstoremocks "github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker/store/mocks"
)

func TestHeartbeat(t *testing.T) {
	store1 := workerstoremocks.NewMockStore()
	store2 := workerstoremocks.NewMockStore()
	recordTransformer := func(ctx context.Context, record workerutil.Record) (apiclient.Job, error) {
		return apiclient.Job{ID: record.RecordID()}, nil
	}

	var cancel1, cancel2 int
	store1.DequeueFunc.PushReturn(testRecord{ID: 41}, func() { cancel1++ } /* store1 */, true, nil)
	store1.DequeueFunc.PushReturn(testRecord{ID: 42}, func() { cancel1++ } /* store1 */, true, nil)
	store2.DequeueFunc.PushReturn(testRecord{ID: 43}, func() { cancel2++ } /* store2 */, true, nil)
	store2.DequeueFunc.PushReturn(testRecord{ID: 44}, func() { cancel2++ } /* store2 */, true, nil)

	options := Options{
		QueueOptions: map[string]QueueOptions{
			"q1": {Store: store1, RecordTransformer: recordTransformer},
			"q2": {Store: store2, RecordTransformer: recordTransformer},
		},
		MaximumNumTransactions: 10,
		UnreportedMaxAge:       time.Second,
	}
	clock := glock.NewMockClock()
	handler := newHandler(options, clock)

	_, dequeued1, _ := handler.dequeue(context.Background(), "q1", "deadbeef", "test")
	_, dequeued2, _ := handler.dequeue(context.Background(), "q1", "deadveal", "test")
	_, dequeued3, _ := handler.dequeue(context.Background(), "q2", "deadbeef", "test")
	_, dequeued4, _ := handler.dequeue(context.Background(), "q2", "deadveal", "test")
	if !dequeued1 || !dequeued2 || !dequeued3 || !dequeued4 {
		t.Fatalf("failed to dequeue records")
	}

	assertDoneCounts := func(c1, c2 int) {
		if cancel1 != c1 {
			t.Fatalf("unexpected number of calls to cancel (store 1). want=%d have=%d", c1, cancel1)
		}
		if cancel2 != c2 {
			t.Fatalf("unexpected number of calls to cancel (store 2). want=%d have=%d", c2, cancel2)
		}
	}

	// missing all jobs, but they're less than UnreportedMaxAge
	clock.Advance(time.Second / 2)
	if _, err := handler.heartbeat(context.Background(), "deadbeef", []int{}); err != nil {
		t.Fatalf("unexpected error performing heartbeat: %s", err)
	}
	if _, err := handler.heartbeat(context.Background(), "deadveal", []int{}); err != nil {
		t.Fatalf("unexpected error performing heartbeat: %s", err)
	}
	assertDoneCounts(0, 0)

	// missing no jobs
	clock.Advance(time.Minute * 2)
	if _, err := handler.heartbeat(context.Background(), "deadbeef", []int{41, 43}); err != nil {
		t.Fatalf("unexpected error performing heartbeat: %s", err)
	}
	if _, err := handler.heartbeat(context.Background(), "deadveal", []int{42, 44}); err != nil {
		t.Fatalf("unexpected error performing heartbeat: %s", err)
	}
	assertDoneCounts(0, 0)

	// missing one deadbeef jobs
	clock.Advance(time.Minute * 2)
	if _, err := handler.heartbeat(context.Background(), "deadbeef", []int{41}); err != nil {
		t.Fatalf("unexpected error performing heartbeat: %s", err)
	}
	if _, err := handler.heartbeat(context.Background(), "deadveal", []int{42, 44}); err != nil {
		t.Fatalf("unexpected error performing heartbeat: %s", err)
	}
	assertDoneCounts(0, 1)

	// missing two deadveal jobs
	clock.Advance(time.Minute * 2)
	if _, err := handler.heartbeat(context.Background(), "deadbeef", []int{41}); err != nil {
		t.Fatalf("unexpected error performing heartbeat: %s", err)
	}
	if _, err := handler.heartbeat(context.Background(), "deadveal", []int{}); err != nil {
		t.Fatalf("unexpected error performing heartbeat: %s", err)
	}
	assertDoneCounts(1, 2)

	// unknown jobs
	clock.Advance(time.Minute * 2)
	if unknownIDs, err := handler.heartbeat(context.Background(), "deadbeef", []int{41, 43, 45}); err != nil {
		t.Fatalf("unexpected error performing heartbeat: %s", err)
	} else if diff := cmp.Diff([]int{43, 45}, unknownIDs); diff != "" {
		t.Errorf("unexpected unknown ids (-want +got):\n%s", diff)
	}
	if unknownIDs, err := handler.heartbeat(context.Background(), "deadveal", []int{42, 44, 45}); err != nil {
		t.Fatalf("unexpected error performing heartbeat: %s", err)
	} else if diff := cmp.Diff([]int{42, 44, 45}, unknownIDs); diff != "" {
		t.Errorf("unexpected unknown ids (-want +got):\n%s", diff)
	}
}

func TestCleanup(t *testing.T) {
	store1 := workerstoremocks.NewMockStore()
	store2 := workerstoremocks.NewMockStore()
	recordTransformer := func(ctx context.Context, record workerutil.Record) (apiclient.Job, error) {
		return apiclient.Job{ID: record.RecordID()}, nil
	}

	var cancel1, cancel2 int
	store1.DequeueFunc.PushReturn(testRecord{ID: 41}, func() { cancel1++ } /* store1 */, true, nil)
	store1.DequeueFunc.PushReturn(testRecord{ID: 42}, func() { cancel1++ } /* store1 */, true, nil)
	store2.DequeueFunc.PushReturn(testRecord{ID: 43}, func() { cancel2++ } /* store2 */, true, nil)
	store2.DequeueFunc.PushReturn(testRecord{ID: 44}, func() { cancel2++ } /* store2 */, true, nil)

	options := Options{
		QueueOptions: map[string]QueueOptions{
			"q1": {Store: store1, RecordTransformer: recordTransformer},
			"q2": {Store: store2, RecordTransformer: recordTransformer},
		},
		MaximumNumTransactions: 10,
		DeathThreshold:         time.Minute * 5,
	}
	clock := glock.NewMockClock()
	handler := newHandler(options, clock)

	_, dequeued1, _ := handler.dequeue(context.Background(), "q1", "deadbeef", "test")
	_, dequeued2, _ := handler.dequeue(context.Background(), "q1", "deadveal", "test")
	_, dequeued3, _ := handler.dequeue(context.Background(), "q2", "deadbeef", "test")
	_, dequeued4, _ := handler.dequeue(context.Background(), "q2", "deadveal", "test")
	if !dequeued1 || !dequeued2 || !dequeued3 || !dequeued4 {
		t.Fatalf("failed to dequeue records")
	}

	for i := 0; i < 6; i++ {
		clock.Advance(time.Minute)

		if _, err := handler.heartbeat(context.Background(), "deadbeef", []int{41, 43}); err != nil {
			t.Fatalf("unexpected error performing heartbeat: %s", err)
		}
	}

	if err := handler.cleanup(context.Background()); err != nil {
		t.Fatalf("unexpected error performing cleanup: %s", err)
	}

	if cancel1 != 1 {
		t.Fatalf("unexpected number of calls to cancel (store 1). want=%d have=%d", 1, cancel1)
	}
	if cancel2 != 1 {
		t.Fatalf("unexpected number of calls to cancel (store 2). want=%d have=%d", 1, cancel2)
	}
}
