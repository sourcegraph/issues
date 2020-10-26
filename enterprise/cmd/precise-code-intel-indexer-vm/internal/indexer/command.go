package indexer

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"

	"github.com/inconshreveable/log15"
)

// runCommand invokes the given command on the host machine.
func runCommand(ctx context.Context, logger *IndexJobLogger, command ...string) error {
	if len(command) == 0 {
		return fmt.Errorf("no command supplied")
	}

	switch command[0] {
	case "git":
	case "docker":
	case "ignite":
	default:
		return fmt.Errorf("illegal command: %s", strings.Join(command, " "))
	}

	cmd, stdout, stderr, err := makeCommand(ctx, command...)
	if err != nil {
		return err
	}

	log15.Info(fmt.Sprintf("Running command: %s", strings.Join(command, " ")))

	wg := wgWrap(func() {
		logger.RecordCommand(command, stdout, stderr)
	})

	if err := cmd.Start(); err != nil {
		return err
	}

	wg.Wait()

	if err := cmd.Wait(); err != nil {
		return err
	}

	return nil
}

// makeCommand returns a new exec.Cmd and pipes to its stdout/stderr streams.
func makeCommand(ctx context.Context, command ...string) (_ *exec.Cmd, stdout, stderr io.Reader, err error) {
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)

	stdout, err = cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, err
	}

	stderr, err = cmd.StderrPipe()
	if err != nil {
		return nil, nil, nil, err
	}

	return cmd, stdout, stderr, nil
}

func wgWrap(f func()) *sync.WaitGroup {
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		f()
	}()

	return &wg
}
