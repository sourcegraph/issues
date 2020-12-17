package command

import (
	"context"
	"testing"
)

func TestRunCommandEmptyCommand(t *testing.T) {
	command := command{
		Commands:  []string{},
		Operation: makeTestOperation(),
	}
	if err := runCommand(context.Background(), command, nil); err != ErrIllegalCommand {
		t.Errorf("unexpected error. want=%q have=%q", ErrIllegalCommand, err)
	}
}

func TestRunCommandIllegalCommand(t *testing.T) {
	command := command{
		Commands:  []string{"kill"},
		Operation: makeTestOperation(),
	}
	if err := runCommand(context.Background(), command, nil); err != ErrIllegalCommand {
		t.Errorf("unexpected error. want=%q have=%q", ErrIllegalCommand, err)
	}
}
