package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/rjeczalik/notify"

	// TODO - deduplicate me
	"github.com/sourcegraph/sourcegraph/dev/sg/root"
	"github.com/sourcegraph/sourcegraph/lib/output"
)

func run(ctx context.Context, cmds ...Command) error {
	chs := make([]<-chan struct{}, 0, len(cmds))
	monitor := &changeMonitor{}
	for _, cmd := range cmds {
		chs = append(chs, monitor.register(cmd))
	}

	pathChanges, err := watch()
	if err != nil {
		return err
	}
	go monitor.run(pathChanges)

	root, err := root.RepositoryRoot()
	if err != nil {
		return err
	}

	wg := sync.WaitGroup{}
	failures := make(chan failedRun, len(cmds))
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for i, cmd := range cmds {
		wg.Add(1)

		go func(cmd Command, ch <-chan struct{}) {
			defer wg.Done()

			if err := runWatch(ctx, cmd, root, ch); err != nil {
				if err != ctx.Err() {
					failures <- failedRun{cmdName: cmd.Name, err: err}
					cancel()
				}
			}
		}(cmd, chs[i])
	}

	wg.Wait()

	select {
	case failure := <-failures:
		printCmdError(out, failure.cmdName, failure.err)
		return failure
	default:
		return nil
	}
}

// failedRun is returned by run when a command failed to run and run exits
type failedRun struct {
	cmdName string
	err     error
}

func (e failedRun) Error() string {
	return fmt.Sprintf("failed to run %s", e.cmdName)
}

// installErr is returned by runWatch if the cmd.Install step fails.
type installErr struct {
	cmdName string
	output  string
}

func (e installErr) Error() string {
	return fmt.Sprintf("install of %s failed: %s", e.cmdName, e.output)
}

// reinstallErr is used internally by runWatch to print a message when a
// command failed to reinstall.
type reinstallErr struct {
	cmdName string
	output  string
}

func (e reinstallErr) Error() string {
	return fmt.Sprintf("reinstalling %s failed: %s", e.cmdName, e.output)
}

// runErr is used internally by runWatch to print a message when a
// command failed to reinstall.
type runErr struct {
	cmdName  string
	exitCode int
	stderr   string
	stdout   string
}

func (e runErr) Error() string {
	return fmt.Sprintf("failed to run %s.\nstderr:\n%s\nstdout:\n%s\n", e.cmdName, e.stderr, e.stdout)
}

func printCmdError(out *output.Output, cmdName string, err error) {
	var message, cmdOut string

	switch e := errors.Cause(err).(type) {
	case installErr:
		message = "Failed to build " + cmdName
		cmdOut = e.output
	case reinstallErr:
		message = "Failed to rebuild " + cmdName
		cmdOut = e.output
	case runErr:
		message = "Failed to run " + cmdName
		cmdOut = e.stderr

		formattedStdout := "\t" + strings.Join(strings.Split(e.stdout, "\n"), "\n\t")
		formattedStderr := "\t" + strings.Join(strings.Split(e.stderr, "\n"), "\n\t")

		cmdOut = fmt.Sprintf("Exit code: %d\n\nStandard out:\n%s\nStandard err:\n%s\n", e.exitCode, formattedStdout, formattedStderr)

	default:
		message = fmt.Sprintf("Failed to run %s: %s", cmdName, err)
	}

	separator := strings.Repeat("-", 80)
	if cmdOut != "" {
		line := output.Linef(
			"", output.StyleWarning,
			"%s\n%s%s:\n%s%s%s%s%s",
			separator, output.StyleBold, message, output.StyleReset,
			cmdOut, output.StyleWarning, separator, output.StyleReset,
		)
		out.WriteLine(line)
	} else {
		line := output.Linef(
			"", output.StyleWarning,
			"%s\n%s%s\n%s%s",
			separator, output.StyleBold, message,
			separator, output.StyleReset,
		)
		out.WriteLine(line)
	}
}

func runWatch(ctx context.Context, cmd Command, root string, reload <-chan struct{}) error {
	startedOnce := false

	var (
		md5hash    string
		md5changed bool
	)

	var wg sync.WaitGroup
	var cancelFuncs []context.CancelFunc

	errs := make(chan error, 1)
	defer func() {
		wg.Wait()
		close(errs)
	}()

	for {
		// Build it
		if cmd.Install != "" {
			out.WriteLine(output.Linef("", output.StylePending, "Installing %s...", cmd.Name))

			c := exec.CommandContext(ctx, "bash", "-c", cmd.Install)
			c.Dir = root
			c.Env = makeEnv(conf.Env, cmd.Env)

			cmdOut, err := c.CombinedOutput()
			if err != nil {
				if !startedOnce {
					return installErr{cmdName: cmd.Name, output: string(cmdOut)}
				} else {
					printCmdError(out, cmd.Name, reinstallErr{cmdName: cmd.Name, output: string(cmdOut)})
					// Now we wait for a reload signal before we start to build it again
					select {
					case <-reload:
						continue
					}
				}
			}

			// clear this signal before starting
			select {
			case <-reload:
			default:
			}

			out.WriteLine(output.Linef("", output.StyleSuccess, "%sSuccessfully installed %s%s", output.StyleBold, cmd.Name, output.StyleReset))

			if cmd.CheckBinary != "" {
				newHash, err := md5HashFile(filepath.Join(root, cmd.CheckBinary))
				if err != nil {
					return installErr{cmdName: cmd.Name, output: string(cmdOut)}
				}

				md5changed = md5hash != newHash
				md5hash = newHash
			}
		}

		if cmd.CheckBinary == "" || md5changed {
			for _, cancel := range cancelFuncs {
				cancel() // Stop command
				<-errs   // Wait for exit
			}
			cancelFuncs = nil

			// Run it
			out.WriteLine(output.Linef("", output.StylePending, "Running %s...", cmd.Name))

			sc, err := startCmd(ctx, root, cmd)
			defer sc.cancel()

			if err != nil {
				return err
			}

			cancelFuncs = append(cancelFuncs, sc.cancel)

			wg.Add(1)
			go func() {
				defer wg.Done()

				err := sc.Wait()

				var e *exec.ExitError
				if errors.As(err, &e) {
					err = runErr{
						cmdName:  cmd.Name,
						exitCode: e.ExitCode(),
						stderr:   sc.CapturedStderr(),
						stdout:   sc.CapturedStdout(),
					}
				}

				errs <- err
			}()

			// TODO: We should probably only set this after N seconds (or when
			// we're sure that the command has booted up -- maybe healthchecks?)
			startedOnce = true
		} else {
			out.WriteLine(output.Linef("", output.StylePending, "Binary did not change. Not restarting."))
		}

		select {
		case <-reload:
			out.WriteLine(output.Linef("", output.StylePending, "Change detected. Reloading %s...", cmd.Name))

			continue // Reinstall

		case err := <-errs:
			// Exited on its own or errored
			if err == nil {
				out.WriteLine(output.Linef("", output.StyleSuccess, "%s%s exited without error%s", output.StyleBold, cmd.Name, output.StyleReset))
			}
			return err
		}
	}
}

type startedCmd struct {
	*exec.Cmd

	cancel func()

	stdoutBuf *prefixSuffixSaver
	stderrBuf *prefixSuffixSaver
}

func (sc *startedCmd) CapturedStdout() string {
	if sc.stdoutBuf == nil {
		return ""
	}

	return string(sc.stdoutBuf.Bytes())
}

func (sc *startedCmd) CapturedStderr() string {
	if sc.stderrBuf == nil {
		return ""
	}

	return string(sc.stderrBuf.Bytes())
}

func startCmd(ctx context.Context, dir string, cmd Command) (*startedCmd, error) {
	sc := &startedCmd{
		stdoutBuf: &prefixSuffixSaver{N: 32 << 10},
		stderrBuf: &prefixSuffixSaver{N: 32 << 10},
	}

	commandCtx, cancel := context.WithCancel(ctx)
	sc.cancel = cancel

	sc.Cmd = exec.CommandContext(commandCtx, "bash", "-c", cmd.Cmd)
	sc.Cmd.Dir = dir
	sc.Cmd.Env = makeEnv(conf.Env, cmd.Env)

	logger := newCmdLogger(cmd.Name, out)
	if cmd.IgnoreStdout {
		out.WriteLine(output.Linef("", output.StyleSuggestion, "Ignoring stdout of %s", cmd.Name))
	} else {
		sc.Cmd.Stdout = io.MultiWriter(logger, sc.stdoutBuf)
	}
	if cmd.IgnoreStderr {
		out.WriteLine(output.Linef("", output.StyleSuggestion, "Ignoring stderr of %s", cmd.Name))
	} else {
		sc.Cmd.Stderr = io.MultiWriter(logger, sc.stderrBuf)
	}

	if err := sc.Start(); err != nil {
		return sc, err
	}

	return sc, nil
}

func makeEnv(envs ...map[string]string) []string {
	combined := os.Environ()

	expandedEnv := map[string]string{}

	for _, env := range envs {
		for k, v := range env {
			// Expand env vars and keep track of previously set env vars
			// so they can be used when expanding too.
			// TODO: using range to iterate over the env is not stable and thus
			// this won't work
			expanded := os.Expand(v, func(lookup string) string {
				// If we're looking up the key that we're trying to define, we
				// skip the self-reference and look in the OS
				if lookup == k {
					return os.Getenv(lookup)
				}

				if e, ok := env[lookup]; ok {
					return e
				}
				return os.Getenv(lookup)
			})
			expandedEnv[k] = expanded
			combined = append(combined, fmt.Sprintf("%s=%s", k, expanded))
		}
	}

	return combined
}

func md5HashFile(filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return string(h.Sum(nil)), nil
}

//
//

type changeMonitor struct {
	subscriptions []subscription
}

type subscription struct {
	cmd Command
	ch  chan struct{}
}

func (m *changeMonitor) run(paths <-chan string) {
	for path := range paths {
		for _, sub := range m.subscriptions {
			m.notify(sub, path)
		}
	}
}

func (m *changeMonitor) notify(sub subscription, path string) {
	found := false
	for _, prefix := range sub.cmd.Watch {
		if strings.HasPrefix(path, prefix) {
			found = true
		}
	}
	if !found {
		return
	}

	select {
	case sub.ch <- struct{}{}:
	default:
	}
}

func (m *changeMonitor) register(cmd Command) <-chan struct{} {
	ch := make(chan struct{}, 0)
	m.subscriptions = append(m.subscriptions, subscription{cmd, ch})
	return ch
}

//
//

var watchIgnorePatterns = []*regexp.Regexp{
	regexp.MustCompile(`_test\.go$`),
	regexp.MustCompile(`^.bin/`),
	regexp.MustCompile(`^.git/`),
	regexp.MustCompile(`^dev/`),
	regexp.MustCompile(`^node_modules/`),
}

func watch() (<-chan string, error) {
	root, err := root.RepositoryRoot()
	if err != nil {
		return nil, err
	}

	paths := make(chan string)
	events := make(chan notify.EventInfo, 1)

	if err := notify.Watch(root+"/...", events, notify.All); err != nil {
		return nil, err
	}

	go func() {
		defer close(events)
		defer notify.Stop(events)

	outer:
		for event := range events {
			path := strings.TrimPrefix(strings.TrimPrefix(event.Path(), root), "/")

			for _, pattern := range watchIgnorePatterns {
				if pattern.MatchString(path) {
					continue outer
				}
			}

			paths <- path
		}
	}()

	return paths, nil
}

func runTest(ctx context.Context, cmd Command, args []string) error {
	root, err := root.RepositoryRoot()
	if err != nil {
		return err
	}

	out.WriteLine(output.Linef("", output.StylePending, "Starting testsuite %q.", cmd.Name))
	if len(args) != 0 {
		out.WriteLine(output.Linef("", output.StylePending, "\tAdditional arguments: %s", args))
	}
	commandCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	cmdArgs := []string{cmd.Cmd}
	if len(args) != 0 {
		cmdArgs = append(cmdArgs, args...)
	} else {
		cmdArgs = append(cmdArgs, cmd.DefaultArgs)
	}

	c := exec.CommandContext(commandCtx, "bash", "-c", strings.Join(cmdArgs, " "))
	c.Dir = root
	c.Env = makeEnv(conf.Env, cmd.Env)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	out.WriteLine(output.Linef("", output.StylePending, "Running %s in %q...", c, root))

	return c.Run()
}

func runChecks(ctx context.Context, checks map[string]Check) error {
	root, err := root.RepositoryRoot()
	if err != nil {
		return err
	}

	for _, check := range checks {
		commandCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		c := exec.CommandContext(commandCtx, "bash", "-c", check.Cmd)
		c.Dir = root
		c.Env = makeEnv(conf.Env)

		p := out.Pending(output.Linef(output.EmojiLightbulb, output.StylePending, "Running check %q...", check.Name))

		if cmdOut, err := c.CombinedOutput(); err != nil {
			p.Complete(output.Linef(output.EmojiFailure, output.StyleWarning, "Check %q failed: %s", check.Name, err))

			out.WriteLine(output.Linef("", output.StyleWarning, "%s", check.FailMessage))
			if len(cmdOut) != 0 {
				out.WriteLine(output.Linef("", output.StyleWarning, "Check produced the following output:"))
				separator := strings.Repeat("-", 80)
				line := output.Linef(
					"", output.StyleWarning,
					"%s\n%s%s%s%s%s",
					separator, output.StyleReset, cmdOut, output.StyleWarning, separator, output.StyleReset,
				)
				out.WriteLine(line)
			}
		} else {
			p.Complete(output.Linef(output.EmojiSuccess, output.StyleSuccess, "Check %q success!", check.Name))
		}
	}

	return nil
}

// prefixSuffixSaver is an io.Writer which retains the first N bytes
// and the last N bytes written to it. The Bytes() methods reconstructs
// it with a pretty error message.
//
// Copy of https://sourcegraph.com/github.com/golang/go@3b770f2ccb1fa6fecc22ea822a19447b10b70c5c/-/blob/src/os/exec/exec.go#L661-729
type prefixSuffixSaver struct {
	N         int // max size of prefix or suffix
	prefix    []byte
	suffix    []byte // ring buffer once len(suffix) == N
	suffixOff int    // offset to write into suffix
	skipped   int64

	// TODO(bradfitz): we could keep one large []byte and use part of it for
	// the prefix, reserve space for the '... Omitting N bytes ...' message,
	// then the ring buffer suffix, and just rearrange the ring buffer
	// suffix when Bytes() is called, but it doesn't seem worth it for
	// now just for error messages. It's only ~64KB anyway.
}

func (w *prefixSuffixSaver) Write(p []byte) (n int, err error) {
	lenp := len(p)
	p = w.fill(&w.prefix, p)

	// Only keep the last w.N bytes of suffix data.
	if overage := len(p) - w.N; overage > 0 {
		p = p[overage:]
		w.skipped += int64(overage)
	}
	p = w.fill(&w.suffix, p)

	// w.suffix is full now if p is non-empty. Overwrite it in a circle.
	for len(p) > 0 { // 0, 1, or 2 iterations.
		n := copy(w.suffix[w.suffixOff:], p)
		p = p[n:]
		w.skipped += int64(n)
		w.suffixOff += n
		if w.suffixOff == w.N {
			w.suffixOff = 0
		}
	}
	return lenp, nil
}

// fill appends up to len(p) bytes of p to *dst, such that *dst does not
// grow larger than w.N. It returns the un-appended suffix of p.
func (w *prefixSuffixSaver) fill(dst *[]byte, p []byte) (pRemain []byte) {
	if remain := w.N - len(*dst); remain > 0 {
		add := minInt(len(p), remain)
		*dst = append(*dst, p[:add]...)
		p = p[add:]
	}
	return p
}

func (w *prefixSuffixSaver) Bytes() []byte {
	if w.suffix == nil {
		return w.prefix
	}
	if w.skipped == 0 {
		return append(w.prefix, w.suffix...)
	}
	var buf bytes.Buffer
	buf.Grow(len(w.prefix) + len(w.suffix) + 50)
	buf.Write(w.prefix)
	buf.WriteString("\n... omitting ")
	buf.WriteString(strconv.FormatInt(w.skipped, 10))
	buf.WriteString(" bytes ...\n")
	buf.Write(w.suffix[w.suffixOff:])
	buf.Write(w.suffix[:w.suffixOff])
	return buf.Bytes()
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
