package executor

import "github.com/sourcegraph/sourcegraph/internal/workerutil"

// Job describes a series of steps to perform within an executor.
type Job struct {
	// ID is the unique identifier of a job within the source queue. Note
	// that different queues can share identifiers.
	ID int `json:"id"`

	// RepositoryName is the name of the repository to be cloned into the
	// workspace prior to job execution.
	RepositoryName string `json:"repositoryName,omitempty"`

	// Workspace replaces RepositoryName, and defines a repository and optional
	// path that describes a workspace.
	Workspace Workspace `json:"workspace,omitempty"`

	// Commit is the revhash that should be checked out prior to job execution.
	Commit string `json:"commit"`

	// VirtualMachineFiles is a map from file names to content. Each entry in
	// this map will be written into the workspace prior to job execution.
	VirtualMachineFiles map[string]string `json:"files"`

	// DockerSteps describe a series of docker run commands to be invoked in the
	// workspace. This may be done inside or outside of a Firecracker virtual
	// machine.
	DockerSteps []DockerStep `json:"dockerSteps"`

	// CliSteps describe a series of src commands to be invoked in the workspace.
	// These run after all docker commands have been completed successfully. This
	// may be done inside or outside of a Firecracker virtual machine.
	CliSteps []CliStep `json:"cliSteps"`

	// RedactedValues is a map from strings to replace to their replacement in the command
	// output before sending it to the underlying job store. This should contain all worker
	// environment variables, as well as secret values passed along with the dequeued job
	// payload, which may be sensitive (e.g. shared API tokens, URLs with credentials).
	RedactedValues map[string]string `json:"redactedValues"`
}

func (j Job) RecordID() int {
	return j.ID
}

type Workspace struct {
	// RepositoryName is the name of the repository to be cloned into the
	// workspace prior to job execution.
	RepositoryName string `json:"repositoryName"`

	// Path is the name of the path relative to the repository root in which
	// jobs should be executed.
	Path string `json:"path"`

	// OnlyFetchWorkspace will cause _only_ the path within the repository to be
	// downloaded if enabled. If Path is empty, then this field has no effect.
	OnlyFetchWorkspace bool `json:"onlyFetchWorkspace"`
}

type DockerStep struct {
	// Image specifies the docker image.
	Image string `json:"image"`

	// Commands specifies the arguments supplied to the end of a docker run command.
	Commands []string `json:"commands"`

	// Dir specifies the target working directory.
	Dir string `json:"dir"`

	// Env specifies a set of NAME=value pairs to supply to the docker command.
	Env []string `json:"env"`
}

type CliStep struct {
	// Commands specifies the arguments supplied to the src command.
	Commands []string `json:"command"`

	// Dir specifies the target working directory.
	Dir string `json:"dir"`

	// Env specifies a set of NAME=value pairs to supply to the src command.
	Env []string `json:"env"`
}

type DequeueRequest struct {
	ExecutorName string `json:"executorName"`
}

type AddExecutionLogEntryRequest struct {
	ExecutorName string `json:"executorName"`
	JobID        int    `json:"jobId"`
	workerutil.ExecutionLogEntry
}

type MarkCompleteRequest struct {
	ExecutorName string `json:"executorName"`
	JobID        int    `json:"jobId"`
}

type MarkErroredRequest struct {
	ExecutorName string `json:"executorName"`
	JobID        int    `json:"jobId"`
	ErrorMessage string `json:"errorMessage"`
}

type HeartbeatRequest struct {
	ExecutorName string `json:"executorName"`
	JobIDs       []int  `json:"jobIds"`
}
