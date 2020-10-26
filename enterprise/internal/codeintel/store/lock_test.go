package store

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
)

func TestLock(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	key := rand.Intn(1000)

	// Start txn before acquiring locks outside of txn
	tx, err := store.Transact(context.Background())
	if err != nil {
		t.Fatalf("unexpected error starting transaction: %s", err)
	}

	acquired, unlock, err := store.Lock(context.Background(), key, true)
	if err != nil {
		t.Fatalf("unexpected error attempting to acquire lock: %s", err)
	}
	if !acquired {
		t.Errorf("expected lock to be acquired")
	}

	acquired, _, err = tx.Lock(context.Background(), key, false)
	if err != nil {
		t.Fatalf("unexpected error attempting to acquire lock: %s", err)
	}
	if acquired {
		t.Errorf("expected lock to be held by other process")
	}

	unlock(nil)

	acquired, _, err = tx.Lock(context.Background(), key, false)
	if err != nil {
		t.Fatalf("unexpected error attempting to acquire lock: %s", err)
	}
	if !acquired {
		t.Errorf("expected lock to be acquired after release")
	}
}

func TestLockBlockingAcquire(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	key := rand.Intn(1000)

	// Start txn before acquiring locks outside of txn
	tx, err := store.Transact(context.Background())
	if err != nil {
		t.Errorf("unexpected error starting transaction: %s", err)
		return
	}

	acquired, unlock, err := store.Lock(context.Background(), key, true)
	if err != nil {
		t.Fatalf("unexpected error attempting to acquire lock: %s", err)
	}
	if !acquired {
		t.Errorf("expected lock to be acquired")
	}

	sync := make(chan struct{})
	go func() {
		defer close(sync)

		acquired, unlock, err := tx.Lock(context.Background(), key, true)
		if err != nil {
			t.Errorf("unexpected error attempting to acquire lock: %s", err)
			return
		}
		defer unlock(nil)

		if !acquired {
			t.Errorf("expected lock to be acquired")
			return
		}
	}()

	select {
	case <-sync:
		t.Errorf("lock acquired before release")
	case <-time.After(time.Millisecond * 100):
	}

	unlock(nil)

	select {
	case <-sync:
	case <-time.After(time.Millisecond * 100):
		t.Errorf("lock not acquired before release")
	}
}
