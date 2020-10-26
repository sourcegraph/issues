package rcache

import (
	"context"
	"testing"
)

func TestTryAcquireMutex(t *testing.T) {
	SetupForTest(t)

	options := MutexOptions{
		// Make mutex fail faster
		Tries: 1,
	}

	ctx, release, ok := TryAcquireMutex(context.Background(), "test", options)
	if !ok {
		t.Fatalf("expected to acquire mutex")
	}

	if _, _, ok = TryAcquireMutex(context.Background(), "test", options); ok {
		t.Fatalf("expected to fail to acquire mutex")
	}

	release()
	if ctx.Err() == nil {
		t.Errorf("expected to cancel context")
	}

	// Test out if cancelling the parent context allows us to still release
	ctx, cancel := context.WithCancel(context.Background())
	_, release, ok = TryAcquireMutex(ctx, "test", options)
	if !ok {
		t.Fatalf("expected to acquire mutex")
	}
	cancel()
	release()

	_, release, ok = TryAcquireMutex(context.Background(), "test", options)
	if !ok {
		t.Fatalf("expected to acquire mutex")
	}
	release()
}
