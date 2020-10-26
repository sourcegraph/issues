package rcache

import (
	"fmt"
	"time"

	"context"

	"github.com/go-redsync/redsync"
)

const (
	DefaultMutexExpiry = time.Minute
	// We make it low since we want to give up quickly. Failing to acquire the lock will be
	// unrelated to failing to reach quorum.
	DefaultMutexTries = 3
	DefaultMutexDelay = 512 * time.Millisecond
)

// MutexOptions hold options passed to TryAcquireMutex. It is safe to
// pass zero values in which case defaults will be used instead.
type MutexOptions struct {
	// Expiry sets how long a lock should be held. Under normal
	// operation it will be extended on an interval of (Expiry / 2)
	Expiry time.Duration
	// Tries is how many tries we have before we give up acquiring a
	// lock.
	Tries int
	// RetryDelay is how long to sleep between attempts to lock
	RetryDelay time.Duration
}

// TryAcquireMutex tries to Lock a distributed mutex. If the mutex is already
// locked, it will return `ctx, nil, false`. Otherwise it returns `ctx,
// release, true`. Release must be called to free the lock.
// The lock has a 1 minute lifetime, but a background routine extends it every 30 seconds.
// If, on release, we are unable to unlock the mutex it will continue to be locked until
// it is expired by Redis.
// The returned context will be cancelled if any of the following occur:
// * The parent context in cancelled
// * The release function is called
// * There is an error extending the lock expiry or the expiry can't be extended because
//   they key no longer exists in Redis
// A caller can therefore assume that they are the sole holder of the lock as long as the
// context has not been cancelled.
func TryAcquireMutex(ctx context.Context, name string, options MutexOptions) (context.Context, func(), bool) {
	// We return a canceled context if we fail, so create the context here
	ctx, cancel := context.WithCancel(ctx)

	if options.Expiry == 0 {
		options.Expiry = DefaultMutexExpiry
	}
	if options.Tries == 0 {
		options.Tries = DefaultMutexTries
	}
	if options.RetryDelay == 0 {
		options.RetryDelay = DefaultMutexDelay
	}

	name = fmt.Sprintf("%s:mutex:%s", globalPrefix, name)
	mu := redsync.New([]redsync.Pool{pool}).NewMutex(
		name,
		redsync.SetExpiry(options.Expiry),
		redsync.SetTries(options.Tries),
		redsync.SetRetryDelay(options.RetryDelay),
	)

	err := mu.Lock()
	if err != nil {
		cancel()
		return ctx, nil, false
	}
	unlockedC := make(chan struct{})
	go func() {
		ticker := time.NewTicker(options.Expiry / 2)
		for {
			select {
			case <-ctx.Done():
				// An error here means we may not have released the lock.
				// It's OK to ignore as we'll stop extending the lock anyway
				// and it will expire.
				_, _ = mu.Unlock()
				ticker.Stop()
				close(unlockedC)
				return
			case <-ticker.C:
				// We do not retry on error as we should cancel the context
				// as soon as we are not 100% sure we hold the lock. This minimises
				// the chance of more than one instance thinking they hold it.
				if ok, err := mu.Extend(); !ok || err != nil {
					cancel()
				}
			}
		}
	}()
	return ctx, func() {
		cancel()
		<-unlockedC
	}, true
}
