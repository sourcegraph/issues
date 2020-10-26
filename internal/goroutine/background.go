package goroutine

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// StartableRoutine represents a component of a binary that consists of a long
// running process.
type StartableRoutine interface {
	// Start begins the long-running process. The Stop method should signal to
	// this process that that application is beginnign to shut down.
	Start()
}

// BackgroundRoutine represents a component of a binary that consists of a long
// running process and a graceful shutdown mechanism.
type BackgroundRoutine interface {
	StartableRoutine

	// Stop signals the Start method to stop accepting new work and complete its
	// current work. This method can but is not required to block until Start has
	// returned.
	Stop()
}

// MonitorBackgroundRoutines will start the given background routines in their own
// (safe) goroutine (via this package's Go method). If a signal is received, the
// Stop method of each routine will be called. This method unblocks once both Start
// and Stop methods of each routine has returned. A second signal will cause the
// app to shutdown immediately.
func MonitorBackgroundRoutines(routines ...BackgroundRoutine) {
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGHUP)
	monitorBackgroundRoutines(signals, routines...)
}

func monitorBackgroundRoutines(signals <-chan os.Signal, routines ...BackgroundRoutine) {
	var wg sync.WaitGroup
	startAll(&wg, routines...)
	waitForSignal(signals)
	stopAll(&wg, routines...)
	wg.Wait()
}

// startAll calls each routine's Start method in its own goroutine and and registers
// each running goroutine with the given waitgroup.
func startAll(wg *sync.WaitGroup, routines ...BackgroundRoutine) {
	for _, r := range routines {
		t := r
		wg.Add(1)
		Go(func() { defer wg.Done(); t.Start() })
	}
}

// stopAll calls each routine's Stop method in its own goroutine and and registers
// each running goroutine with the given waitgroup.
func stopAll(wg *sync.WaitGroup, routines ...BackgroundRoutine) {
	for _, r := range routines {
		t := r
		wg.Add(1)
		Go(func() { defer wg.Done(); t.Stop() })
	}
}

// exiter exits the process with a status code of zero. This is declared here
// so it can be replaced by tests without risk of aborting the tests without
// a good indication to the calling program that the tests didn't in fact pass.
var exiter = func() { os.Exit(0) }

// waitForSignal blocks until a signal has been received. This will call os.Exit(0)
// if a second signal is received.
func waitForSignal(signals <-chan os.Signal) {
	<-signals

	go func() {
		// Shutdown immediately on a second signal
		<-signals
		exiter()
	}()
}

// CombinedRoutine is a list of background routines which are stopped and
// started in unison.
type CombinedRoutine []BackgroundRoutine

func (r CombinedRoutine) Start() {
	var wg sync.WaitGroup
	startAll(&wg, r...)
	wg.Wait()
}

func (r CombinedRoutine) Stop() {
	var wg sync.WaitGroup
	stopAll(&wg, r...)
	wg.Wait()
}
