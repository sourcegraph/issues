package leader

import (
	"context"
	"testing"
	"time"

	"github.com/sourcegraph/sourcegraph/internal/rcache"
)

func TestDoWhileLeader(t *testing.T) {
	rcache.SetupForTest(t)

	key := "test-leader"
	ctx, cancel := context.WithCancel(context.Background())
	// In case we don't make it to cancel lower down
	t.Cleanup(cancel)

	var count int64

	fn := func(ctx context.Context) {
		select {
		case <-ctx.Done():
			return
		default:
		}
		count++
		<-ctx.Done()
	}

	options := Options{
		AcquireInterval: 50 * time.Millisecond,
		MutexOptions: rcache.MutexOptions{
			Tries:      1,
			RetryDelay: 10 * time.Millisecond,
		},
	}

	cancelled := make(chan struct{})

	go func() {
		Do(ctx, key, options, fn)
		cancelled <- struct{}{}
	}()
	go func() {
		Do(ctx, key, options, fn)
		cancelled <- struct{}{}
	}()

	time.Sleep(500 * time.Millisecond)
	cancel()

	if count != 1 {
		t.Fatalf("Count > 1: %d", count)
	}

	// Check that Do exits after cancelled
	for i := 0; i < 2; i++ {
		select {
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Timeout")
		case <-cancelled:
		}
	}

}
