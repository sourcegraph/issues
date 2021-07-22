package worker

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/enterprise/cmd/executor/internal/apiclient"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/executor/internal/command"
	"github.com/sourcegraph/sourcegraph/internal/goroutine"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
)

type Options struct {
	// QueueName is the name of the queue to process work from. Having this configurable
	// allows us to have multiple worker pools with different resource requirements and
	// horizontal scaling factors while still uniformly processing events.
	QueueName string

	// HeartbeatInterval denotes the time between heartbeat requests to the queue API.
	HeartbeatInterval time.Duration

	// GitServicePath is the path to the internal git service API proxy in the frontend.
	// This path should contain the endpoints info/refs and git-upload-pack.
	GitServicePath string

	// RedactedValues is a map from strings to replace to their replacement in the command
	// output before sending it to the underlying job store. This should contain all worker
	// environment variables, as well as secret values passed along with the dequeued job
	// payload, which may be sensitive (e.g. shared API tokens, URLs with credentials).
	RedactedValues map[string]string

	// WorkerOptions configures the worker behavior.
	WorkerOptions workerutil.WorkerOptions

	// ClientOptions configures the client that interacts with the queue API.
	ClientOptions apiclient.Options

	// FirecrackerOptions configures the behavior of Firecracker virtual machine creation.
	FirecrackerOptions command.FirecrackerOptions

	// ResourceOptions configures the resource limits of docker container and Firecracker
	// virtual machines running on the executor.
	ResourceOptions command.ResourceOptions

	// MaximumRuntimePerJob is the maximum wall time that can be spent on a single job.
	MaximumRuntimePerJob time.Duration
}

// NewWorker creates a worker that polls a remote job queue API for work. The returned
// routine contains both a worker that periodically polls for new work to perform, as well
// as a heartbeat routine that will periodically hit the remote API with the work that is
// currently being performed, which is necessary so the job queue API doesn't hand out jobs
// it thinks may have been dropped.
func NewWorker(options Options, observationContext *observation.Context) goroutine.BackgroundRoutine {
	idSet := newIDSet()
	queueStore := apiclient.New(options.ClientOptions, observationContext)
	store := &storeShim{queueName: options.QueueName, queueStore: queueStore}

	if !connectToFrontend(queueStore, options) {
		os.Exit(1)
	}

	handler := &handler{
		idSet:         idSet,
		store:         store,
		options:       options,
		operations:    command.NewOperations(observationContext),
		runnerFactory: command.NewRunner,
	}

	indexer := workerutil.NewWorker(context.Background(), store, handler, options.WorkerOptions)
	heartbeat := goroutine.NewHandlerWithErrorMessage("heartbeat", func(ctx context.Context) error {
		unknownIDs, err := queueStore.Heartbeat(ctx, options.QueueName, idSet.Slice())
		if err != nil {
			return err
		}

		for _, id := range unknownIDs {
			idSet.Remove(id)
		}

		return nil
	})

	return goroutine.CombinedRoutine{
		indexer,
		goroutine.NewPeriodicGoroutine(context.Background(), options.HeartbeatInterval, heartbeat),
	}
}

// connectToFrontend will ping the configured Sourcegraph instance until it receives a 200 response.
// For the first minute, "connection refused" errors will not be emitted. This is to stop log spam
// in dev environments where the executor may start up before the frontend. This method returns true
// after a ping is successful and returns false if a user signal is received.
func connectToFrontend(queueStore *apiclient.Client, options Options) bool {
	start := time.Now()
	log15.Info("Connecting to Sourcegraph instance", "url", options.ClientOptions.EndpointOptions.URL)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGHUP, syscall.SIGINT)
	defer signal.Stop(signals)

	for {
		err := queueStore.Ping(context.Background(), options.QueueName, nil)
		if err == nil {
			log15.Info("Connected to Sourcegraph instance")
			return true
		}

		var e *os.SyscallError
		if errors.As(err, &e) && e.Syscall == "connect" && time.Since(start) < time.Minute {
			// Hide initial connection logs due to services starting up in an nondeterminstic order.
			// Logs occurring one minute after startup or later are not filtered, nor are non-expected
			// connection errors during app startup.
		} else {
			log15.Error("Failed to connect to Sourcegraph instance", "error", err)
		}

		select {
		case <-ticker.C:
		case <-signals:
			log15.Error("Signal received while connecting to Sourcegraph")
			return false
		}
	}
}
