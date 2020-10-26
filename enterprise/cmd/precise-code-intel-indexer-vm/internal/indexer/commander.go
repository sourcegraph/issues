package indexer

import "context"

// Commander abstracts running processes on the host machine.
type Commander interface {
	// Run invokes the given command on the host machine.
	Run(ctx context.Context, command ...string) error
}

// CommanderFunc is a function version of the Commander interface.
type CommanderFunc func(ctx context.Context, command ...string) error

// Run invokes the given command on the host machine. See the Commander interface for additional details.
func (f CommanderFunc) Run(ctx context.Context, command ...string) error {
	return f(ctx, command...)
}

func NewDefaultCommander(logger *IndexJobLogger) Commander {
	return CommanderFunc(func(ctx context.Context, command ...string) error {
		return runCommand(ctx, logger, command...)
	})
}
