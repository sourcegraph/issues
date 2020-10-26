package conf

import (
	"fmt"
	"os"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/internal/conf/conftypes"
)

// store manages the in-memory storage, access,
// and updating of the site configuration in a threadsafe manner.
type store struct {
	configMu  sync.RWMutex
	lastValid *Unified
	mock      *Unified

	rawMu sync.RWMutex
	raw   conftypes.RawUnified

	ready chan struct{}
	once  sync.Once
}

// newStore returns a new configuration store.
func newStore() *store {
	return &store{
		ready: make(chan struct{}),
	}
}

// LastValid returns the last valid site configuration that this
// store was updated with.
func (s *store) LastValid() *Unified {
	s.WaitUntilInitialized()

	s.configMu.RLock()
	defer s.configMu.RUnlock()

	if s.mock != nil {
		return s.mock
	}

	return s.lastValid
}

// Raw returns the last raw configuration that this store was updated with.
func (s *store) Raw() conftypes.RawUnified {
	s.WaitUntilInitialized()

	s.rawMu.RLock()
	defer s.rawMu.RUnlock()
	return s.raw
}

// Mock sets up mock data for the site configuration. It uses the configuration
// mutex, to avoid possible races between test code and possible config watchers.
func (s *store) Mock(mockery *Unified) {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	s.mock = mockery
	s.initialize()
}

type updateResult struct {
	Changed bool
	Old     *Unified
	New     *Unified
}

// MaybeUpdate attempts to update the store with the supplied rawConfig.
//
// If the rawConfig isn't syntactically valid JSON, the store's LastValid field.
// won't be updating and a parsing error will be returned
// from the previous time that this function was called.
//
// configChange is defined iff the cache was actually updated.
// TODO@ggilmore: write a less-vague description
func (s *store) MaybeUpdate(rawConfig conftypes.RawUnified) (updateResult, error) {
	s.rawMu.Lock()
	defer s.rawMu.Unlock()

	s.configMu.Lock()
	defer s.configMu.Unlock()

	result := updateResult{
		Changed: false,
		Old:     s.lastValid,
		New:     s.lastValid,
	}

	if rawConfig.Site == "" {
		return result, errors.New("invalid site configuration (empty string)")
	}
	if s.raw.Equal(rawConfig) {
		return result, nil
	}

	s.raw = rawConfig

	newConfig, err := ParseConfig(rawConfig)
	if err != nil {
		return result, errors.Wrap(err, "when parsing rawConfig during update")
	}

	result.Changed = true
	result.New = newConfig
	s.lastValid = newConfig

	s.initialize()

	return result, nil
}

// WaitUntilInitialized blocks and only returns to the caller once the store
// has initialized with a syntactically valid configuration file (via MaybeUpdate() or Mock()).
func (s *store) WaitUntilInitialized() {
	if getMode() == modeServer {
		s.checkDeadlock()
	}

	<-s.ready
}

func (s *store) checkDeadlock() {
	select {
	// Frontend has initialized its configuration server, we can return early
	case <-configurationServerFrontendOnlyInitialized:
		return
	default:
	}

	deadlockTimeout := 5 * time.Minute
	if IsDev(DeployType()) {
		deadlockTimeout = 60 * time.Second
		disable, _ := strconv.ParseBool(os.Getenv("DISABLE_CONF_DEADLOCK_DETECTOR"))
		if disable {
			deadlockTimeout = 24 * 365 * time.Hour
		}
	}

	timer := time.NewTimer(deadlockTimeout)
	defer timer.Stop()

	select {
	// Frontend has initialized its configuration server.
	case <-configurationServerFrontendOnlyInitialized:
	// We assume that we're in an unrecoverable deadlock if frontend hasn't
	// started its configuration server after a while.
	case <-timer.C:
		// The running goroutine is not necessarily the cause of the
		// deadlock, so ask Go to dump all goroutine stack traces.
		debug.SetTraceback("all")
		if IsDev(DeployType()) {
			panic("potential deadlock detected: the frontend's configuration server hasn't started after 60s indicating a deadlock may be happening. A common cause of this is calling conf.Get or conf.Watch before the frontend has started fully (e.g. inside an init function) and if that is the case you may need to invoke those functions in a separate goroutine.")
		}
		panic(fmt.Sprintf("(bug) frontend configuration server failed to start after %v, this may indicate the DB is inaccessible", deadlockTimeout))
	}
}

func (s *store) initialize() {
	s.once.Do(func() {
		close(s.ready)
	})
}
