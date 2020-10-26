package indexmanager

import "sync"

// Manager is a synchronized set of index record identifiers which are currently
// being processed by an indexer process. This is read by the heartbeat process
// to construct the payload to the API to prevent the records being processed
// from being requeued.
type Manager struct {
	m        sync.RWMutex
	indexIDs map[int]struct{}
}

// New creates a new Manager.
func New() *Manager {
	return &Manager{
		indexIDs: map[int]struct{}{},
	}
}

// GetIDs returns a slice of the identifiers in the set.
func (i *Manager) GetIDs() (ids []int) {
	i.m.RLock()
	defer i.m.RUnlock()

	for id := range i.indexIDs {
		ids = append(ids, id)
	}

	return ids
}

// AddID adds an identifier to the set.
func (i *Manager) AddID(indexID int) {
	i.m.Lock()
	i.indexIDs[indexID] = struct{}{}
	i.m.Unlock()
}

// RemoveID removes an identifier from the set.
func (i *Manager) RemoveID(indexID int) {
	i.m.Lock()
	delete(i.indexIDs, indexID)
	i.m.Unlock()
}
