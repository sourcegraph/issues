package cache

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence"
	persistencemocks "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence/mocks"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/types"
)

func TestStoreCache(t *testing.T) {
	opener := func(filename string) (persistence.Store, error) {
		store := persistencemocks.NewMockStore()
		store.ReadMetaFunc.SetDefaultReturn(types.MetaData{NumResultChunks: 4}, nil)
		return store, nil
	}

	var meta types.MetaData
	handler := func(store persistence.Store) (err error) {
		meta, err = store.ReadMeta(context.Background())
		return err
	}

	cache := newStoreCache(opener)

	if err := cache.WithStore(context.Background(), "test", handler); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if meta.NumResultChunks != 4 {
		t.Errorf("unexpected store result. want=%d have=%d", 4, meta.NumResultChunks)
	}
}

func TestStoreCacheError(t *testing.T) {
	opener := func(filename string) (persistence.Store, error) {
		return persistencemocks.NewMockStore(), nil
	}

	expectedErr := fmt.Errorf("oops")
	handler := func(store persistence.Store) error {
		return expectedErr
	}

	cache := newStoreCache(opener)

	if err := cache.WithStore(context.Background(), "test", handler); err != expectedErr {
		t.Fatalf("unexpected error. want=%q have=%q", expectedErr, err)
	}
}

func TestStoreCacheInitError(t *testing.T) {
	expectedErr := fmt.Errorf("oops")
	returnedErr := expectedErr
	opener := func(filename string) (persistence.Store, error) {
		temp := returnedErr
		returnedErr = nil
		return persistencemocks.NewMockStore(), temp
	}

	handler := func(store persistence.Store) error {
		return nil
	}

	cache := newStoreCache(opener)

	if err := cache.WithStore(context.Background(), "test", handler); err != expectedErr {
		t.Fatalf("unexpected error. want=%q have=%q", expectedErr, err)
	}

	// Wait for entry to drain
	waitForCondition(func() bool {
		cache.m.RLock()
		defer cache.m.RUnlock()
		return len(cache.stores) == 0
	})

	// Ensure we can recreate the store without error
	if err := cache.WithStore(context.Background(), "test", handler); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestStoreCacheConcurrentRequests(t *testing.T) {
	t.Skip("Skipping because there seems to be race condition where the reported number of calls is wrong")

	calls := 0
	wait := make(chan struct{}) // blocks opener

	opener := func(filename string) (persistence.Store, error) {
		calls++
		<-wait

		store := persistencemocks.NewMockStore()
		store.ReadMetaFunc.SetDefaultReturn(types.MetaData{NumResultChunks: 4}, nil)
		return store, nil
	}

	n := 10
	errs := make(chan error)                // errors from withStore
	sync := make(chan struct{})             // signals goroutine has been scheduled
	values := make(chan types.MetaData, 10) // values from ReadMeta within handler

	handler := func(store persistence.Store) error {
		meta, err := store.ReadMeta(context.Background())
		values <- meta
		return err
	}

	cache := newStoreCache(opener)

	for i := 0; i < n; i++ {
		go func() {
			sync <- struct{}{}
			errs <- cache.WithStore(context.Background(), "test", handler)
		}()
	}

	// Wait until all routines have scheduled
	for i := 0; i < n; i++ {
		<-sync
	}
	close(wait)

	// Read all errors
	for i := 0; i < n; i++ {
		if err := <-errs; err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
	}
	close(errs)

	// Read all handler results
	for i := 0; i < n; i++ {
		if meta := <-values; meta.NumResultChunks != 4 {
			t.Errorf("unexpected store result. want=%d have=%d", 4, meta.NumResultChunks)
		}
	}
	close(values)

	if calls != 1 {
		t.Errorf("unexpected number of calls. want=%d have=%d", 1, calls)
	}
}

func TestStoreCacheDisposedEntry(t *testing.T) {
	wait := make(chan struct{}) // blocks close from returning
	sync := make(chan struct{}) // signals close routine has been scheduled

	store := persistencemocks.NewMockStore()
	store.CloseFunc.PushHook(func(err error) error {
		close(sync)
		<-wait
		return err
	})

	opener := func(filename string) (persistence.Store, error) {
		return store, nil
	}

	handler := func(store persistence.Store) error {
		return nil
	}

	cache := newStoreCache(opener)

	if err := cache.WithStore(context.Background(), "test", handler); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	<-sync

	go func() {
		// Make disposed entry available for 25ms
		<-time.After(TestTickDuration)
		close(wait)
	}()

	if err := cache.WithStore(context.Background(), "test", handler); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestStoreCacheDisposedEntryContextCanceled(t *testing.T) {
	sync := make(chan struct{}) // signals close routine has been scheduled

	store := persistencemocks.NewMockStore()
	store.CloseFunc.PushHook(func(err error) error {
		close(sync)
		select {} // block forever
	})

	opener := func(filename string) (persistence.Store, error) {
		return store, nil
	}

	handler := func(store persistence.Store) error {
		return nil
	}

	cache := newStoreCache(opener)

	if err := cache.WithStore(context.Background(), "test", handler); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	<-sync

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := cache.WithStore(ctx, "test", handler); err != context.Canceled {
		t.Fatalf("unexpected error. want=%q have=%q", context.Canceled, err)
	}
}

func TestStoreContextCanceled(t *testing.T) {
	wait := make(chan struct{})
	defer close(wait)

	opener := func(filename string) (persistence.Store, error) {
		<-wait
		return persistencemocks.NewMockStore(), nil
	}

	handler := func(store persistence.Store) error {
		return nil
	}

	cache := newStoreCache(opener)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := cache.WithStore(ctx, "test", handler); err != context.Canceled {
		t.Fatalf("unexpected error. want=%q have=%q", context.Canceled, err)
	}
}
