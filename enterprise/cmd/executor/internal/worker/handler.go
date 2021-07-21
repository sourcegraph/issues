package worker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/honeycombio/libhoney-go"
	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/enterprise/cmd/executor/internal/command"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/executor"
	"github.com/sourcegraph/sourcegraph/internal/honey"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
)

type handler struct {
	idSet         *IDSet
	store         workerutil.Store
	options       Options
	operations    *command.Operations
	runnerFactory func(dir string, logger *command.Logger, options command.Options, operations *command.Operations) command.Runner
}

var _ workerutil.Handler = &handler{}

// ErrJobAlreadyExists occurs when a duplicate job identifier is dequeued.
var ErrJobAlreadyExists = errors.New("job already exists")

// Handle clones the target code into a temporary directory, invokes the target indexer in a
// fresh docker container, and uploads the results to the external frontend API.
func (h *handler) Handle(ctx context.Context, record workerutil.Record) (err error) {
	job := record.(executor.Job)
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(h.options.MaximumRuntimePerJob))
	defer cancel()

	wrapError := func(err error, message string) error {
		if errors.Is(err, context.DeadlineExceeded) {
			err = errors.Errorf("job exceeded maximum execution time of %s", h.options.MaximumRuntimePerJob)
		}

		return errors.Wrap(err, message)
	}

	start := time.Now()
	defer func() {
		if honey.Enabled() {
			_ = createHoneyEvent(ctx, job, err, time.Since(start)).Send()
		}
	}()

	if !h.idSet.Add(job.ID, cancel) {
		return ErrJobAlreadyExists
	}
	defer h.idSet.Remove(job.ID)

	// 🚨 SECURITY: The job logger must be supplied with all sensitive values that may appear
	// in a command constructed and run in the following function. Note that the command and
	// its output may both contain sensitive values, but only values which we directly
	// interpolate into the command. No command that we run on the host leaks environment
	// variables, and the user-specified commands (which could leak their environment) are
	// run in a clean VM.
	logger := command.NewLogger(union(h.options.RedactedValues, job.RedactedValues))

	done := make(chan struct{})
	defer func() {
		close(logger.EntryC)

		<-done
	}()

	go func() {
		for entry := range logger.EntryC {
			log15.Info("Writing log entry", "jobID", job.ID, "repositoryName", job.RepositoryName, "commit", job.Commit)
			// Perform this outside of the task execution context. If there is a timeout or
			// cancellation error we don't want to skip uploading these logs as users will
			// often want to see how far something progressed prior to a timeout.
			if err := h.store.AddExecutionLogEntry(context.Background(), record.RecordID(), entry); err != nil {
				log15.Warn("Failed to upload executor log entry for job", "id", record.RecordID(), "repositoryName", job.RepositoryName, "commit", job.Commit, "error", err)
			}
		}
		close(done)
	}()

	// Create a working directory for this job which will be removed once the job completes.
	// If a repository is supplied as part of the job configuration, it will be cloned into
	// the working directory.

	log15.Info("Creating workspace", "jobID", job.ID, "repositoryName", job.RepositoryName, "commit", job.Commit)

	hostRunner := h.runnerFactory("", logger, command.Options{}, h.operations)
	workingDirectory, err := h.prepareWorkspace(ctx, hostRunner, job.RepositoryName, job.Commit)
	if err != nil {
		return wrapError(err, "failed to prepare workspace")
	}
	defer func() {
		_ = os.RemoveAll(workingDirectory)
	}()

	// Copy the file contents from the job record into the working directory
	for relativePath, content := range job.VirtualMachineFiles {
		path, err := filepath.Abs(filepath.Join(workingDirectory, relativePath))
		if err != nil {
			return err
		}

		if !strings.HasPrefix(path, workingDirectory) {
			return errors.Errorf("refusing to write outside of working directory")
		}

		if err := os.WriteFile(path, []byte(content), os.ModePerm); err != nil {
			return err
		}
	}

	name, err := uuid.NewRandom()
	if err != nil {
		return err
	}

	options := command.Options{
		ExecutorName:       name.String(),
		FirecrackerOptions: h.options.FirecrackerOptions,
		ResourceOptions:    h.options.ResourceOptions,
	}
	runner := h.runnerFactory(workingDirectory, logger, options, h.operations)

	// Deduplicate and sort (for testing)
	imageMap := map[string]struct{}{}
	for _, dockerStep := range job.DockerSteps {
		imageMap[dockerStep.Image] = struct{}{}
	}

	imageNames := make([]string, 0, len(imageMap))
	for image := range imageMap {
		imageNames = append(imageNames, image)
	}
	sort.Strings(imageNames)

	scriptNames := make([]string, 0, len(job.DockerSteps))
	for i, dockerStep := range job.DockerSteps {
		scriptName := scriptNameFromJobStep(job, i)
		scriptPath := filepath.Join(workingDirectory, command.ScriptsPath, scriptName)

		if err := os.WriteFile(scriptPath, buildScript(dockerStep), os.ModePerm); err != nil {
			return err
		}

		scriptNames = append(scriptNames, scriptName)
	}

	log15.Info("Setting up VM", "jobID", job.ID, "repositoryName", job.RepositoryName, "commit", job.Commit)

	// Setup Firecracker VM (if enabled)
	if err := runner.Setup(ctx, imageNames, nil); err != nil {
		return wrapError(err, "failed to setup virtual machine")
	}
	defer func() {
		// Perform this outside of the task execution context. If there is a timeout or
		// cancellation error we don't want to skip cleaning up the resources that we've
		// allocated for the current task.
		if teardownErr := runner.Teardown(context.Background()); teardownErr != nil {
			err = multierror.Append(err, teardownErr)
		}
	}()

	// Invoke each docker step sequentially
	for i, dockerStep := range job.DockerSteps {
		dockerStepCommand := command.CommandSpec{
			Key:        fmt.Sprintf("step.docker.%d", i),
			Image:      dockerStep.Image,
			ScriptPath: scriptNames[i],
			Dir:        dockerStep.Dir,
			Env:        dockerStep.Env,
			Operation:  h.operations.Exec,
		}

		log15.Info(fmt.Sprintf("Running docker step #%d", i), "jobID", job.ID, "repositoryName", job.RepositoryName, "commit", job.Commit)

		if err := runner.Run(ctx, dockerStepCommand); err != nil {
			return wrapError(err, "failed to perform docker step")
		}
	}

	// Invoke each src-cli step sequentially
	for i, cliStep := range job.CliSteps {
		log15.Info(fmt.Sprintf("Running src-cli step #%d", i), "jobID", job.ID, "repositoryName", job.RepositoryName, "commit", job.Commit)

		env := cliStep.Env
		if !options.FirecrackerOptions.Enabled {
			for _, k := range srcCliHostEnvVars {
				env = append(env, fmt.Sprintf("%s=%s", k, os.Getenv(k)))
			}
		}

		cliStepCommand := command.CommandSpec{
			Key:       fmt.Sprintf("step.src.%d", i),
			Command:   append([]string{"src"}, cliStep.Commands...),
			Dir:       cliStep.Dir,
			Env:       env,
			Operation: h.operations.Exec,
		}

		if err := runner.Run(ctx, cliStepCommand); err != nil {
			return wrapError(err, "failed to perform src-cli step")
		}
	}

	return nil
}

// srcCliHostEnvVars is a list of environment variable names that Job.CliSteps
// inherit when executing directly on the host.
//
// That only happens if Firecracker is disabled (which is only on
// developer macOS machines).
var srcCliHostEnvVars = []string{"HOME", "PATH"}

var scriptPreamble = `
set -x
`

func buildScript(dockerStep executor.DockerStep) []byte {
	return []byte(strings.Join(append([]string{scriptPreamble, ""}, dockerStep.Commands...), "\n") + "\n")
}

func union(a, b map[string]string) map[string]string {
	c := make(map[string]string, len(a)+len(b))

	for k, v := range a {
		c[k] = v
	}
	for k, v := range b {
		c[k] = v
	}

	return c
}

func scriptNameFromJobStep(job executor.Job, i int) string {
	return fmt.Sprintf("%d.%d_%s@%s.sh", job.ID, i, strings.ReplaceAll(job.RepositoryName, "/", "_"), job.Commit)
}

func createHoneyEvent(ctx context.Context, job executor.Job, err error, duration time.Duration) *libhoney.Event {
	fields := map[string]interface{}{
		"duration_ms":    duration.Milliseconds(),
		"recordID":       job.RecordID(),
		"repositoryName": job.RepositoryName,
		"commit":         job.Commit,
		"numDockerSteps": len(job.DockerSteps),
		"numCliSteps":    len(job.CliSteps),
	}

	if err != nil {
		fields["error"] = err.Error()
	}
	// Currently disabled as the import pulls in conf packages
	// if spanURL := trace.SpanURLFromContext(ctx); spanURL != "" {
	// 	fields["trace"] = spanURL
	// }

	return honey.EventWithFields("executor", fields)
}
