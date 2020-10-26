package cache

import (
	"context"
	"sync"
	"time"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence"
)

// StoreCache is a cache of store instances with very short TTL values. The primary purpose of this
// cache is to ensure that multiple concurrent requests for the same store do not launch multiple
// expensive migration operations, which occur when the store of an older bundle is first opened.
// Often, this results in one or more stores failing with a transaction error or a locked database.
//
// Stores in this cache are closed soon after they are first used, but any concurrent requests coming
// in for the same store will share the same cache entry and both will wait on the same store to
// complete initialization. It is guaranteed that a store will remain open while there are active
// users of that store instance.
type StoreCache interface {
	// WithStore calls the given handler function with a store argument. If the store has not yet
	// initialized and does not do so before context deadline, an error is returned. The store
	// initialization is not canceled due to a context deadline here and will continue to run in the
	// background until completion.
	WithStore(ctx context.Context, key string, f HandlerFunc) error
}

// StoreOpener initializes a new store for the given key.
type StoreOpener func(key string) (persistence.Store, error)

// Handler performs an operation on a store. The invocation of this function is a critical section that
// locks the given store argument so that it is not closed while in use.
type HandlerFunc func(store persistence.Store) error

type storeCache struct {
	opener StoreOpener

	m      sync.RWMutex
	stores map[string]*storeCacheEntry
}

// NewStoreCache creates a new store cache with the given store opener.
func NewStoreCache(opener StoreOpener) StoreCache {
	return newStoreCache(opener)
}

func newStoreCache(opener StoreOpener) *storeCache {
	return &storeCache{
		opener: opener,
		stores: map[string]*storeCacheEntry{},
	}
}

// WithStore calls the given handler function with a store argument. If the store has not yet
// initialized and does not do so before context deadline, an error is returned. The store
// initialization is not canceled due to a context deadline here and will continue to run in the
// background until completion.
func (c *storeCache) WithStore(ctx context.Context, key string, f HandlerFunc) error {
	entry, err := c.getLockedEntry(ctx, key)
	if err != nil {
		return err
	}
	defer entry.dec()

	select {
	case <-makeInitChannel(entry):
	case <-ctx.Done():
		return ctx.Err()
	}

	if entry.err != nil {
		return entry.err
	}

	return f(entry.store)
}

// getLockedEntry gets a non-disposed cache entry or creates a new one. The caller must ensure
// that the returned entry's refcount is decremented after use.
func (c *storeCache) getLockedEntry(ctx context.Context, key string) (*storeCacheEntry, error) {
	for {
		if entry, locked := c.getOrCreateEntry(key); locked || entry.inc() {
			return entry, nil
		}

		select {
		case <-time.After(time.Millisecond):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// getOrCreateEntry retrieves an existing store cache entry for the given key. If one does not
// exist, a new one is created. This method also returns a boolean flag indicating whether or not
// the refCount was incremented as part of construction. If this value is true, the caller is
// expected to decrement the entry's refcount after use (but not increment it).
func (c *storeCache) getOrCreateEntry(key string) (*storeCacheEntry, bool) {
	if entry := c.getEntry(key); entry != nil {
		return entry, false
	}

	return c.createEntry(key)
}

// getEntry attempts to return an existing entry for the given key. If such an entry does not exist,
// this method returns nil.
func (c *storeCache) getEntry(key string) *storeCacheEntry {
	c.m.RLock()
	defer c.m.RUnlock()
	return c.stores[key]
}

// createEntry attempts to create a new entry for the given key if such an entry does not already exist.
// This method also returns a boolean flag indicating whether or not the entry was just created. Newly
// created entries have a refcount of zero (and are already marked as in-use for the calling function).
func (c *storeCache) createEntry(key string) (*storeCacheEntry, bool) {
	c.m.Lock()
	defer c.m.Unlock()

	// Ensure entry doesn't exist after acquiring write lock
	if entry, ok := c.stores[key]; ok {
		return entry, false
	}

	// Create new entry and make it immediately visible to all other requests
	entry := newStoreCacheEntry()
	c.stores[key] = entry

	go func() {
		// Initialize the store in the background. This allows multiple concurrent requests
		// for the same store to share the same entry while it is initializing, which an take
		// a non-negligible amount of time for large, outdated bundles.
		entry.init(func() (persistence.Store, error) { return c.opener(key) })

		// Block until the entry is unused, then close the underlying store
		entry.closeOnceUnused()

		// Remove entry from map so that it's inaccessible to future requests
		c.removeEntry(key)
	}()

	return entry, true
}

// removeEntry removes the given key from the cache's store map.
func (c *storeCache) removeEntry(key string) {
	c.m.Lock()
	delete(c.stores, key)
	c.m.Unlock()
}

// makeInitChannel creates a channel that closes once the given entry has initialized.
func makeInitChannel(entry *storeCacheEntry) <-chan struct{} {
	ch := make(chan struct{})

	go func() {
		defer close(ch)
		entry.waitUntilInitialized()
	}()

	return ch
}

// storeCacheEntry wraps a store with its initialization state and its ref count.
type storeCacheEntry struct {
	m           *sync.Mutex       // protects all fields
	store       persistence.Store // shared store instance
	err         error             // construction error
	initialized bool              // set when store/err fields are set
	disposed    bool              // set when entry is no longer usable
	refCount    uint32            // number of references to entry

	// cond wraps the entry's mutex and broadcasts to waiting
	// goroutines when initialized is first set to true and when
	// refCount is decremented.
	cond *sync.Cond
}

// newStoreCacheEntry creates a new cache entry with a refcount of one.
func newStoreCacheEntry() *storeCacheEntry {
	m := &sync.Mutex{}

	return &storeCacheEntry{
		m:        m,
		cond:     sync.NewCond(m),
		refCount: 1,
	}
}

// init calls the given function to construct a store, then updates the store, err,
// and initialized fields of the entry.
func (e *storeCacheEntry) init(openStoreFunc func() (persistence.Store, error)) {
	store, err := openStoreFunc()

	e.m.Lock()
	e.store = store
	e.err = err
	e.initialized = true
	e.m.Unlock()

	// wake users waiting for initialization to complete
	e.cond.Broadcast()
}

// waitUntilInitialized blocks until the entry has completed initialization.
func (e *storeCacheEntry) waitUntilInitialized() {
	e.m.Lock()
	defer e.m.Unlock()

	for !e.initialized {
		e.cond.Wait()
	}
}

// closeOnceUnused blocks until the refcount of the entry goes to zero. The entry's disposed
// flag is set to indicate to future users that the underlying store is no longer usable. The
// entry's store is then closed (if it was initialized successfully).
func (e *storeCacheEntry) closeOnceUnused() {
	e.m.Lock()
	for e.refCount > 0 {
		e.cond.Wait()
	}
	e.disposed = true
	e.m.Unlock()

	if e.store != nil {
		e.store.Close(nil)
	}
}

// inc attempts to increase the refcount by one. If the entry is already disposed, this
// method returns false and the entry is not modified.
func (e *storeCacheEntry) inc() bool {
	e.m.Lock()
	defer e.m.Unlock()

	if e.disposed {
		return false
	}

	e.refCount++
	return true
}

// dec decreases the refcount by one.
func (e *storeCacheEntry) dec() {
	e.m.Lock()
	e.refCount--
	e.m.Unlock()
	e.cond.Broadcast()
}
