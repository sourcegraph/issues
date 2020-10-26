package indexmanager

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/efritz/glock"
	"github.com/keegancsmith/sqlf"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/store"
	storemocks "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/store/mocks"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
	dbworkerstore "github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker/store"
	workerstoremocks "github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker/store/mocks"
)

func TestProcessSuccess(t *testing.T) {
	mockStore := storemocks.NewMockStore()
	mockWorkerStore := workerstoremocks.NewMockStore()
	mockWorkerStore.DequeueWithIndependentTransactionContextFunc.PushReturn(store.Index{ID: 42}, mockWorkerStore, true, nil)
	mockWorkerStore.MarkCompleteFunc.SetDefaultReturn(true, nil)
	clock := glock.NewMockClock()

	manager := newManager(mockStore, mockWorkerStore, ManagerOptions{
		MaximumTransactions:   10,
		RequeueDelay:          time.Second,
		UnreportedIndexMaxAge: time.Second,
		DeathThreshold:        time.Second,
	}, clock)

	index, dequeued, err := manager.Dequeue(context.Background(), "deadbeef")
	if err != nil {
		t.Fatalf("unexpected error dequeueing record: %s", err)
	}
	if !dequeued {
		t.Fatalf("expected a record")
	}
	if index.ID != 42 {
		t.Fatalf("unexpected record id. want=%d have=%d", 42, index.ID)
	}

	found, err := manager.Complete(context.Background(), "deadbeef", 42, "")
	if err != nil {
		t.Fatalf("unexpected error marking record as complete: %s", err)
	}
	if !found {
		t.Fatalf("expected record to be tracked: %s", err)
	}

	if callCount := len(mockWorkerStore.MarkCompleteFunc.History()); callCount != 1 {
		t.Errorf("unexpected mark complete call count. want=%d have=%d", 1, callCount)
	} else if id := mockWorkerStore.MarkCompleteFunc.History()[0].Arg1; id != 42 {
		t.Errorf("unexpected id argument to markge. want=%v have=%v", 42, id)
	}

	if callCount := len(mockWorkerStore.DoneFunc.History()); callCount != 1 {
		t.Errorf("unexpected done call count. want=%d have=%d", 1, callCount)
	} else if err := mockWorkerStore.DoneFunc.History()[0].Arg0; err != nil {
		t.Errorf("unexpected error argument to done. want=%v have=%v", nil, err)
	}
}

func TestProcessFailure(t *testing.T) {
	mockStore := storemocks.NewMockStore()
	mockWorkerStore := workerstoremocks.NewMockStore()
	mockWorkerStore.DequeueWithIndependentTransactionContextFunc.PushReturn(store.Index{ID: 42}, mockWorkerStore, true, nil)
	mockWorkerStore.MarkErroredFunc.SetDefaultReturn(true, nil)
	clock := glock.NewMockClock()

	manager := newManager(mockStore, mockWorkerStore, ManagerOptions{
		MaximumTransactions:   10,
		RequeueDelay:          time.Second,
		UnreportedIndexMaxAge: time.Second,
		DeathThreshold:        time.Second,
	}, clock)

	index, dequeued, err := manager.Dequeue(context.Background(), "deadbeef")
	if err != nil {
		t.Fatalf("unexpected error dequeueing record: %s", err)
	}
	if !dequeued {
		t.Fatalf("expected a record")
	}
	if index.ID != 42 {
		t.Fatalf("unexpected record id. want=%d have=%d", 42, index.ID)
	}

	found, err := manager.Complete(context.Background(), "deadbeef", 42, "oops")
	if err != nil {
		t.Fatalf("unexpected error marking record as complete: %s", err)
	}
	if !found {
		t.Fatalf("expected record to be tracked: %s", err)
	}

	if callCount := len(mockWorkerStore.MarkErroredFunc.History()); callCount != 1 {
		t.Errorf("unexpected mark errored call count. want=%d have=%d", 1, callCount)
	} else if id := mockWorkerStore.MarkErroredFunc.History()[0].Arg1; id != 42 {
		t.Errorf("unexpected id argument to mark errored. want=%v have=%v", 42, id)
	}

	if callCount := len(mockWorkerStore.DoneFunc.History()); callCount != 1 {
		t.Errorf("unexpected done call count. want=%d have=%d", 1, callCount)
	} else if err := mockWorkerStore.DoneFunc.History()[0].Arg0; err != nil {
		t.Errorf("unexpected error argument to done. want=%v have=%v", nil, err)
	}
}

func TestProcessSetLogContents(t *testing.T) {
	mockStore := storemocks.NewMockStore()
	mockStore.WithFunc.SetDefaultReturn(mockStore)
	mockWorkerStore := workerstoremocks.NewMockStore()
	mockWorkerStore.DequeueWithIndependentTransactionContextFunc.PushReturn(store.Index{ID: 42}, mockWorkerStore, true, nil)
	mockWorkerStore.MarkCompleteFunc.SetDefaultReturn(true, nil)
	clock := glock.NewMockClock()

	manager := newManager(mockStore, mockWorkerStore, ManagerOptions{
		MaximumTransactions:   10,
		RequeueDelay:          time.Second,
		UnreportedIndexMaxAge: time.Second,
		DeathThreshold:        time.Second,
	}, clock)

	index, dequeued, err := manager.Dequeue(context.Background(), "deadbeef")
	if err != nil {
		t.Fatalf("unexpected error dequeueing record: %s", err)
	}
	if !dequeued {
		t.Fatalf("expected a record")
	}
	if index.ID != 42 {
		t.Fatalf("unexpected record id. want=%d have=%d", 42, index.ID)
	}

	if err := manager.SetLogContents(context.Background(), "deadbeef", 42, "test payload"); err != nil {
		t.Fatalf("unexpected error setting log contents: %s", err)
	}

	found, err := manager.Complete(context.Background(), "deadbeef", 42, "")
	if err != nil {
		t.Fatalf("unexpected error marking record as complete: %s", err)
	}
	if !found {
		t.Fatalf("expected record to be tracked: %s", err)
	}

	if callCount := len(mockWorkerStore.MarkCompleteFunc.History()); callCount != 1 {
		t.Errorf("unexpected mark complete call count. want=%d have=%d", 1, callCount)
	} else if id := mockWorkerStore.MarkCompleteFunc.History()[0].Arg1; id != 42 {
		t.Errorf("unexpected id argument to markge. want=%v have=%v", 42, id)
	}

	if callCount := len(mockWorkerStore.DoneFunc.History()); callCount != 1 {
		t.Errorf("unexpected done call count. want=%d have=%d", 1, callCount)
	} else if err := mockWorkerStore.DoneFunc.History()[0].Arg0; err != nil {
		t.Errorf("unexpected error argument to done. want=%v have=%v", nil, err)
	}

	if callCount := len(mockStore.WithFunc.History()); callCount != 1 {
		t.Errorf("unexpected with call count. want=%d have=%d", 1, callCount)
	}

	if callCount := len(mockStore.SetIndexLogContentsFunc.History()); callCount != 1 {
		t.Errorf("unexpected set index log contents call count. want=%d have=%d", 1, callCount)
	} else if value := mockStore.SetIndexLogContentsFunc.History()[0].Arg2; value != "test payload" {
		t.Errorf("unexpected error argument to done. want=%v have=%v", "test payload", value)
	}
}

func TestProcessIndexerMismatch(t *testing.T) {
	mockStore := storemocks.NewMockStore()
	mockWorkerStore := workerstoremocks.NewMockStore()
	mockWorkerStore.DequeueWithIndependentTransactionContextFunc.PushReturn(store.Index{ID: 42}, mockWorkerStore, true, nil)
	clock := glock.NewMockClock()

	manager := newManager(mockStore, mockWorkerStore, ManagerOptions{
		MaximumTransactions:   10,
		RequeueDelay:          time.Second,
		UnreportedIndexMaxAge: time.Second,
		DeathThreshold:        time.Second,
	}, clock)

	index, dequeued, err := manager.Dequeue(context.Background(), "deadbeef")
	if err != nil {
		t.Fatalf("unexpected error dequeueing record: %s", err)
	}
	if !dequeued {
		t.Fatalf("expected a record")
	}
	if index.ID != 42 {
		t.Fatalf("unexpected record id. want=%d have=%d", 42, index.ID)
	}

	found, err := manager.Complete(context.Background(), "livebeef", 42, "oops")
	if err != nil {
		t.Fatalf("unexpected error marking record as complete: %s", err)
	}
	if found {
		t.Fatalf("expected record to belong to a different indexer: %s", err)
	}

	if callCount := len(mockWorkerStore.DoneFunc.History()); callCount != 0 {
		t.Errorf("unexpected done call count. want=%d have=%d", 0, callCount)
	}
}

func TestBoundedTransactions(t *testing.T) {
	mockStore := storemocks.NewMockStore()
	mockWorkerStore := workerstoremocks.NewMockStore()
	mockWorkerStore.MarkCompleteFunc.SetDefaultReturn(true, nil)
	clock := glock.NewMockClock()

	calls := 0
	mockWorkerStore.DequeueWithIndependentTransactionContextFunc.SetDefaultHook(func(ctx context.Context, conds []*sqlf.Query) (workerutil.Record, dbworkerstore.Store, bool, error) {
		calls++
		return store.Index{ID: calls + 10}, mockWorkerStore, true, nil
	})

	manager := newManager(mockStore, mockWorkerStore, ManagerOptions{
		MaximumTransactions:   10,
		RequeueDelay:          time.Second,
		UnreportedIndexMaxAge: time.Second,
		DeathThreshold:        time.Second,
	}, clock)

	for i := 1; i <= 10; i++ {
		index, dequeued, err := manager.Dequeue(context.Background(), "deadbeef")
		if err != nil {
			t.Fatalf("unexpected error dequeueing record: %s", err)
		}
		if !dequeued {
			t.Fatalf("expected a record")
		}
		if index.ID != i+10 {
			t.Fatalf("unexpected record id. want=%d have=%d", i+10, index.ID)
		}
	}

	_, dequeued, err := manager.Dequeue(context.Background(), "deadbeef")
	if err != nil {
		t.Fatalf("unexpected error dequeueing record: %s", err)
	}
	if dequeued {
		t.Fatalf("expected to hit dequeue limit")
	}

	// Complete one outstanding record
	found, err := manager.Complete(context.Background(), "deadbeef", 15, "")
	if err != nil {
		t.Fatalf("unexpected error marking record as complete: %s", err)
	}
	if !found {
		t.Fatalf("expected record to be tracked: %s", err)
	}

	_, dequeued, err = manager.Dequeue(context.Background(), "deadbeef")
	if err != nil {
		t.Fatalf("unexpected error dequeueing record: %s", err)
	}
	if !dequeued {
		t.Fatalf("expected complete to free up a record slot")
	}
}

func TestHeartbeatRemovesUnknownIndexes(t *testing.T) {
	mockStore := storemocks.NewMockStore()
	mockWorkerStore := workerstoremocks.NewMockStore()
	mockWorkerStore.MarkCompleteFunc.SetDefaultReturn(true, nil)
	clock := glock.NewMockClock()

	calls := 0
	mockWorkerStore.DequeueWithIndependentTransactionContextFunc.SetDefaultHook(func(ctx context.Context, conds []*sqlf.Query) (workerutil.Record, dbworkerstore.Store, bool, error) {
		calls++
		return store.Index{ID: calls + 10}, mockWorkerStore, true, nil
	})

	manager := newManager(mockStore, mockWorkerStore, ManagerOptions{
		MaximumTransactions:   10,
		RequeueDelay:          time.Second,
		UnreportedIndexMaxAge: time.Second,
		DeathThreshold:        time.Second,
	}, clock)

	for i := 0; i < 5; i++ {
		_, dequeued, err := manager.Dequeue(context.Background(), "deadbeef")
		if err != nil {
			t.Fatalf("unexpected error dequeueing record: %s", err)
		}
		if !dequeued {
			t.Fatalf("expected a record")
		}
	}

	// Advance by UnreportedIndexMaxAge
	clock.Advance(time.Second)

	if err := manager.Heartbeat(context.Background(), "deadbeef", []int{12, 14, 15}); err != nil {
		t.Fatalf("unexpected error performing heartbeat: %s", err)
	}

	if callCount := len(mockWorkerStore.RequeueFunc.History()); callCount != 2 {
		t.Errorf("unexpected requeue call count. want=%d have=%d", 2, callCount)
	}
	if callCount := len(mockWorkerStore.DoneFunc.History()); callCount != 2 {
		t.Errorf("unexpected done call count. want=%d have=%d", 2, callCount)
	}

	testCases := map[int]bool{
		11: false,
		12: true,
		13: false,
		14: true,
		15: true,
	}

	for id, expected := range testCases {
		name := fmt.Sprintf("id=%d", id)

		t.Run(name, func(t *testing.T) {
			found, err := manager.Complete(context.Background(), "deadbeef", id, "")
			if err != nil {
				t.Fatalf("unexpected error marking record as complete: %s", err)
			}
			if found != expected {
				t.Errorf("unexpected flag value. want=%v have=%v", expected, found)
			}
		})
	}

}

func TestUnresponsiveIndexer(t *testing.T) {
	mockStore := storemocks.NewMockStore()
	mockWorkerStore := workerstoremocks.NewMockStore()
	mockWorkerStore.MarkCompleteFunc.SetDefaultReturn(true, nil)
	clock := glock.NewMockClock()

	calls := 0
	mockWorkerStore.DequeueWithIndependentTransactionContextFunc.SetDefaultHook(func(ctx context.Context, conds []*sqlf.Query) (workerutil.Record, dbworkerstore.Store, bool, error) {
		calls++
		return store.Index{ID: calls + 10}, mockWorkerStore, true, nil
	})

	manager := newManager(mockStore, mockWorkerStore, ManagerOptions{
		MaximumTransactions:   10,
		RequeueDelay:          time.Second,
		UnreportedIndexMaxAge: time.Second,
		DeathThreshold:        time.Second,
	}, clock)

	for i := 0; i < 5; i++ {
		_, dequeued, err := manager.Dequeue(context.Background(), "deadbeef")
		if err != nil {
			t.Fatalf("unexpected error dequeueing record: %s", err)
		}
		if !dequeued {
			t.Fatalf("expected a record")
		}

		_, dequeued, err = manager.Dequeue(context.Background(), "livebeef")
		if err != nil {
			t.Fatalf("unexpected error dequeueing record: %s", err)
		}
		if !dequeued {
			t.Fatalf("expected a record")
		}
	}

	// Advance by 75% of DeathThreshold
	clock.Advance(time.Second * 3 / 4)

	// Keep one indexer alive
	if err := manager.Heartbeat(context.Background(), "livebeef", []int{12, 14, 16, 18, 20}); err != nil {
		t.Fatalf("unexpected error performing heartbeat: %s", err)
	}

	// Advance by 75% of DeathThreshold
	clock.Advance(time.Second * 3 / 4)

	// Perform a cleanup
	_ = manager.Handle(context.Background())

	if callCount := len(mockWorkerStore.RequeueFunc.History()); callCount != 5 {
		t.Errorf("unexpected requeue call count. want=%d have=%d", 5, callCount)
	}
	if callCount := len(mockWorkerStore.DoneFunc.History()); callCount != 5 {
		t.Errorf("unexpected done call count. want=%d have=%d", 5, callCount)
	}
}
