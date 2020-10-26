package resolvers

import "github.com/dgraph-io/ristretto"

// HunkCache is a LRU cache that holds git diff hunks.
type HunkCache interface {
	// Get returns the value (if any) and a boolean representing whether the value was
	// found or not.
	Get(key interface{}) (interface{}, bool)

	// Set attempts to add the key-value item to the cache with the given cost. If it
	// returns false, then the value as dropped and the item isn't added to the cache.
	Set(key, value interface{}, cost int64) bool
}

// NewHunkCache creates a data cache instance with the given maximum capacity.
func NewHunkCache(size int) (HunkCache, error) {
	return ristretto.NewCache(&ristretto.Config{
		NumCounters: int64(size) * 10,
		MaxCost:     int64(size),
		BufferItems: 64,
	})
}
