package goroutine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/efritz/glock"
)

func TestPeriodicGoroutine(t *testing.T) {
	clock := glock.NewMockClock()
	handler := NewMockHandler()

	goroutine := newPeriodicGoroutine(context.Background(), time.Second, handler, clock)
	go goroutine.Start()
	clock.BlockingAdvance(time.Second)
	clock.BlockingAdvance(time.Second)
	clock.BlockingAdvance(time.Second)
	goroutine.Stop()

	if calls := len(handler.HandleFunc.History()); calls != 4 {
		t.Errorf("unexpected number of handler invocations. want=%d have=%d", 4, calls)
	}
}

func TestPeriodicGoroutineError(t *testing.T) {
	clock := glock.NewMockClock()
	handler := NewMockHandlerWithErrorHandler()
	handler.HandleFunc.PushReturn(errors.New("oops"))

	goroutine := newPeriodicGoroutine(context.Background(), time.Second, handler, clock)
	go goroutine.Start()
	clock.BlockingAdvance(time.Second)
	clock.BlockingAdvance(time.Second)
	clock.BlockingAdvance(time.Second)
	goroutine.Stop()

	if calls := len(handler.HandleFunc.History()); calls != 4 {
		t.Errorf("unexpected number of handler invocations. want=%d have=%d", 4, calls)
	}

	if calls := len(handler.HandleErrorFunc.History()); calls != 1 {
		t.Errorf("unexpected number of error handler invocations. want=%d have=%d", 1, calls)
	}
}

func TestPeriodicGoroutineContextError(t *testing.T) {
	clock := glock.NewMockClock()
	handler := NewMockHandlerWithErrorHandler()
	handler.HandleFunc.SetDefaultHook(func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})

	goroutine := newPeriodicGoroutine(context.Background(), time.Second, handler, clock)
	go goroutine.Start()
	goroutine.Stop()

	if calls := len(handler.HandleFunc.History()); calls != 1 {
		t.Errorf("unexpected number of handler invocations. want=%d have=%d", 1, calls)
	}

	if calls := len(handler.HandleErrorFunc.History()); calls != 0 {
		t.Errorf("unexpected number of error handler invocations. want=%d have=%d", 0, calls)
	}
}

func TestPeriodicGoroutineFinalizer(t *testing.T) {
	clock := glock.NewMockClock()
	handler := NewMockHandlerWithFinalizer()

	goroutine := newPeriodicGoroutine(context.Background(), time.Second, handler, clock)
	go goroutine.Start()
	clock.BlockingAdvance(time.Second)
	clock.BlockingAdvance(time.Second)
	clock.BlockingAdvance(time.Second)
	goroutine.Stop()

	if calls := len(handler.HandleFunc.History()); calls != 4 {
		t.Errorf("unexpected number of handler invocations. want=%d have=%d", 4, calls)
	}

	if calls := len(handler.OnShutdownFunc.History()); calls != 1 {
		t.Errorf("unexpected number of finalizer invocations. want=%d have=%d", 1, calls)
	}
}

type MockHandlerWithErrorHandler struct {
	*MockHandler
	*MockErrorHandler
}

func NewMockHandlerWithErrorHandler() *MockHandlerWithErrorHandler {
	return &MockHandlerWithErrorHandler{
		MockHandler:      NewMockHandler(),
		MockErrorHandler: NewMockErrorHandler(),
	}
}

type MockHandlerWithFinalizer struct {
	*MockHandler
	*MockFinalizer
}

func NewMockHandlerWithFinalizer() *MockHandlerWithFinalizer {
	return &MockHandlerWithFinalizer{
		MockHandler:   NewMockHandler(),
		MockFinalizer: NewMockFinalizer(),
	}
}
