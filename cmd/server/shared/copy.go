package shared

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// copyConfigs will copy /etc/sourcegraph/{netrc,gitconfig} to locations read
// by other tools.
func copyConfigs() error {
	paths := map[string]string{
		"netrc":     "$HOME/.netrc",
		"gitconfig": "$HOME/.gitconfig",
	}
	for src, dst := range paths {
		src = filepath.Join(os.Getenv("CONFIG_DIR"), src)
		dst = os.ExpandEnv(dst)

		data, err := ioutil.ReadFile(src)
		if os.IsNotExist(err) {
			continue
		} else if err != nil {
			return errors.Wrapf(err, "failed to copy %s -> %s", src, dst)
		}

		if err := ioutil.WriteFile(dst, data, 0600); err != nil {
			return errors.Wrapf(err, "failed to copy %s -> %s", src, dst)
		}
	}

	return nil
}

// copySSH will copy the files at /etc/sourcegraph/ssh and put them into
// ~/.ssh
func copySSH() error {
	from := filepath.Join(os.Getenv("CONFIG_DIR"), "ssh")
	fi, err := os.Stat(from)
	if err != nil {
		if os.IsNotExist(err) {
			if verbose {
				log.Printf("%s does not exist, so only repos that do not require SSH will be accessible.", from)
			}
			return nil
		}
		return errors.Wrap(err, "failed to setup SSH auth")
	}
	if !fi.IsDir() {
		return errors.Errorf("%s is not a directory", from)
	}

	// Easiest way to recursive copy and update perm is via shell
	to := os.ExpandEnv("$HOME/.ssh")
	e := execer{}
	e.Command("cp", "-r", from+"/", to)
	e.Command("find", to, "-type", "f", "-exec", "chmod", "600", "{}", ";")
	e.Command("find", to, "-type", "d", "-exec", "chmod", "700", "{}", ";")
	return e.Error()
}

// execer wraps exec.Command, but acts like "set -x". If a command fails, all
// future commands will return the original error.
type execer struct {
	// Out if set will write the command, stdout and stderr to it
	Out io.Writer
	// Working directory of the command.
	Dir string

	err error
}

// Command creates an exec.Command connected to stdout/stderr and runs it.
func (e *execer) Command(name string, arg ...string) {
	e.CommandWithFilter(defaultErrorFilter, name, arg...)
}

func (e *execer) Run(cmd *exec.Cmd) {
	e.RunWithFilter(defaultErrorFilter, cmd)
}

type errorFilter func(err error, out string) bool

func defaultErrorFilter(err error, out string) bool {
	return true
}

// CommandWithFilter is like Command but will not set an error on the command
// object if the given error filter returns false. The command filter is given
// both the (non-nil) error value and the output of the command.
func (e *execer) CommandWithFilter(errorFilter errorFilter, name string, arg ...string) {
	e.RunWithFilter(errorFilter, exec.Command(name, arg...))
}

// RunWithFilter is like Run but will not set an error on the command object
// if the given error filter returns false. The command filter is given both
// the (non-nil) error value and the output of the command.
func (e *execer) RunWithFilter(errorFilter errorFilter, cmd *exec.Cmd) {
	if e.err != nil {
		return
	}

	if cmd.Dir == "" {
		cmd.Dir = e.Dir
	}

	if verbose {
		log.Printf("$ %s %s", cmd.Path, strings.Join(cmd.Args, " "))
	}

	if e.Out != nil {
		_, _ = e.Out.Write([]byte(fmt.Sprintf("\n$ %s %s\n", cmd.Path, strings.Join(cmd.Args, " "))))
	}

	if cmd.Stdout == nil {
		if e.Out != nil {
			cmd.Stdout = e.Out
		} else {
			cmd.Stdout = os.Stdout
		}
	}
	if cmd.Stderr == nil {
		if e.Out != nil {
			cmd.Stderr = e.Out
		} else {
			cmd.Stderr = os.Stderr
		}
	}

	out := &bytes.Buffer{}
	cmd.Stdout = io.MultiWriter(cmd.Stdout, out)
	cmd.Stderr = io.MultiWriter(cmd.Stderr, out)

	if err := cmd.Run(); err != nil && errorFilter(err, out.String()) {
		e.err = err
	}
}

// Error returns the first error encountered.
func (e execer) Error() error {
	return e.err
}
