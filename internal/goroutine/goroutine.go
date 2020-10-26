package goroutine

import (
	"log"
	"runtime/debug"
)

// Go runs the given function in a goroutine and catches and logs panics.
//
// This prevents a single panicking goroutine from crashing the entire binary,
// which is undesirable for services with many different components, like our
// frontend service, where one location of code panicking could be catastrophic.
//
// More advanced use cases should copy this implementation and modify it.
func Go(f func()) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				stack := debug.Stack()
				log.Printf("goroutine panic: %v\n%s", err, stack)
			}
		}()

		f()
	}()
}
