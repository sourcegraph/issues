package conf

import (
	"context"
	"log"
	"math/rand"
	"net"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/conf/conftypes"
)

type client struct {
	store       *store
	passthrough ConfigurationSource
	watchersMu  sync.Mutex
	watchers    []chan struct{}
}

var (
	defaultClientOnce sync.Once
	defaultClientVal  *client
)

func defaultClient() *client {
	defaultClientOnce.Do(func() {
		defaultClientVal = initDefaultClient()
	})
	return defaultClientVal
}

// Raw returns a copy of the raw configuration.
func Raw() conftypes.RawUnified {
	return defaultClient().Raw()
}

// Get returns a copy of the configuration. The returned value should NEVER be
// modified.
//
// Important: The configuration can change while the process is running! Code
// should only call this in response to conf.Watch OR it should invoke it
// periodically or in direct response to a user action (e.g. inside an HTTP
// handler) to ensure it responds to configuration changes while the process
// is running.
//
// There are a select few configuration options that do restart the server, but these are the
// exception rather than the rule. In general, ANY use of configuration should
// be done in such a way that it responds to config changes while the process
// is running.
//
// Get is a wrapper around client.Get.
func Get() *Unified {
	return defaultClient().Get()
}

// Raw returns a copy of the raw configuration.
func (c *client) Raw() conftypes.RawUnified {
	return c.store.Raw()
}

// Get returns a copy of the configuration. The returned value should NEVER be
// modified.
//
// Important: The configuration can change while the process is running! Code
// should only call this in response to conf.Watch OR it should invoke it
// periodically or in direct response to a user action (e.g. inside an HTTP
// handler) to ensure it responds to configuration changes while the process
// is running.
//
// There are a select few configuration options that do restart the server but these are the
// exception rather than the rule. In general, ANY use of configuration should
// be done in such a way that it responds to config changes while the process
// is running.
func (c *client) Get() *Unified {
	return c.store.LastValid()
}

// Mock sets up mock data for the site configuration.
//
// Mock is a wrapper around client.Mock.
func Mock(mockery *Unified) {
	defaultClient().Mock(mockery)
}

// Mock sets up mock data for the site configuration.
func (c *client) Mock(mockery *Unified) {
	c.store.Mock(mockery)
}

// Watch calls the given function whenever the configuration has changed. The new configuration is
// accessed by calling conf.Get.
//
// Before Watch returns, it will invoke f to use the current configuration.
//
// Watch is a wrapper around client.Watch.
//
// IMPORTANT: Watch will block on config initialization. It therefore should *never* be called
// synchronously in `init` functions.
func Watch(f func()) {
	defaultClient().Watch(f)
}

// Cached will return a wrapper around f which caches the response. The value
// will be recomputed every time the config is updated.
//
// IMPORTANT: The first call to wrapped will block on config initialization.
func Cached(f func() interface{}) (wrapped func() interface{}) {
	return defaultClient().Cached(f)
}

// Watch calls the given function in a separate goroutine whenever the
// configuration has changed. The new configuration can be received by calling
// conf.Get.
//
// Before Watch returns, it will invoke f to use the current configuration.
func (c *client) Watch(f func()) {
	// Add the watcher channel now, rather than after invoking f below, in case
	// an update were to happen while we were invoking f.
	notify := make(chan struct{}, 1)
	c.watchersMu.Lock()
	c.watchers = append(c.watchers, notify)
	c.watchersMu.Unlock()

	// Call the function now, to use the current configuration.
	c.store.WaitUntilInitialized()
	f()

	go func() {
		// Invoke f when the configuration has changed.
		for {
			<-notify
			f()
		}
	}()
}

// Cached will return a wrapper around f which caches the response. The value
// will be recomputed every time the config is updated.
//
// The first call to wrapped will block on config initialization.
func (c *client) Cached(f func() interface{}) (wrapped func() interface{}) {
	var once sync.Once
	var val atomic.Value
	return func() interface{} {
		once.Do(func() {
			c.Watch(func() {
				val.Store(f())
			})
		})
		return val.Load()
	}
}

// notifyWatchers runs all the callbacks registered via client.Watch() whenever
// the configuration has changed.
func (c *client) notifyWatchers() {
	c.watchersMu.Lock()
	defer c.watchersMu.Unlock()
	for _, watcher := range c.watchers {
		// Perform a non-blocking send.
		//
		// Since the watcher channels that we are sending on have a
		// buffer of 1, it is guaranteed the watcher will
		// reconsider the config at some point in the future even
		// if this send fails.
		select {
		case watcher <- struct{}{}:
		default:
		}
	}
}

type continuousUpdateOptions struct {
	// delayBeforeUnreachableLog is how long to wait before logging an error upon initial startup
	// due to the frontend being unreachable. It is used to avoid log spam when other services (that
	// contact the frontend for configuration) start up before the frontend.
	delayBeforeUnreachableLog time.Duration

	log   func(format string, v ...interface{}) // log.Printf equivalent
	sleep func()                                // sleep between updates
}

// continuouslyUpdate runs (*client).fetchAndUpdate in an infinite loop, with error logging and
// random sleep intervals.
//
// The optOnlySetByTests parameter is ONLY customized by tests. All callers in main code should pass
// nil (so that the same defaults are used).
func (c *client) continuouslyUpdate(optOnlySetByTests *continuousUpdateOptions) {
	opt := optOnlySetByTests
	if opt == nil {
		// Apply defaults.
		opt = &continuousUpdateOptions{
			// This needs to be long enough to allow the frontend to fully migrate the PostgreSQL
			// database in most cases, to avoid log spam when running sourcegraph/server for the
			// first time.
			delayBeforeUnreachableLog: 15 * time.Second,
			log:                       log.Printf,
			sleep: func() {
				jitter := time.Duration(rand.Int63n(5 * int64(time.Second)))
				time.Sleep(jitter)
			},
		}
	}

	isFrontendUnreachableError := func(err error) bool {
		if urlErr, ok := errors.Cause(err).(*url.Error); ok {
			if netErr, ok := urlErr.Err.(*net.OpError); ok && netErr.Op == "dial" {
				return true
			}
		}
		return false
	}

	start := time.Now()
	for {
		err := c.fetchAndUpdate()
		if err != nil {
			// Suppress log messages for errors caused by the frontend being unreachable until we've
			// given the frontend enough time to initialize (in case other services start up before
			// the frontend), to reduce log spam.
			if time.Since(start) > opt.delayBeforeUnreachableLog || !isFrontendUnreachableError(err) {
				opt.log("received error during background config update, err: %s", err)
			}
		} else {
			// We successfully fetched the config, we reset the timer to give
			// frontend time if it needs to restart
			start = time.Now()
		}

		opt.sleep()
	}
}

func (c *client) fetchAndUpdate() error {
	ctx := context.Background()
	var (
		newConfig conftypes.RawUnified
		err       error
	)
	if c.passthrough != nil {
		newConfig, err = c.passthrough.Read(ctx)
	} else {
		newConfig, err = api.InternalClient.Configuration(ctx)
	}
	if err != nil {
		return errors.Wrap(err, "unable to fetch new configuration")
	}

	configChange, err := c.store.MaybeUpdate(newConfig)
	if err != nil {
		return errors.Wrap(err, "unable to update new configuration")
	}

	if configChange.Changed {
		c.notifyWatchers()
	}

	return nil
}
