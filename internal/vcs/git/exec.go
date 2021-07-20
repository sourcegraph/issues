package git

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/cockroachdb/errors"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/trace/ot"
)

// checkSpecArgSafety returns a non-nil err if spec begins with a "-", which could
// cause it to be interpreted as a git command line argument.
func checkSpecArgSafety(spec string) error {
	if strings.HasPrefix(spec, "-") {
		return errors.Errorf("invalid git revision spec %q (begins with '-')", spec)
	}
	return nil
}

// ExecSafe executes a Git subcommand iff it is allowed according to a allowlist.
//
// An error is only returned when there is a failure unrelated to the actual command being
// executed. If the executed command exits with a nonzero exit code, err == nil. This is similar to
// how http.Get returns a nil error for HTTP non-2xx responses.
func ExecSafe(ctx context.Context, repo api.RepoName, params []string) (stdout, stderr []byte, exitCode int, err error) {
	if Mocks.ExecSafe != nil {
		return Mocks.ExecSafe(params)
	}

	span, ctx := ot.StartSpanFromContext(ctx, "Git: ExecSafe")
	defer span.Finish()

	if len(params) == 0 {
		return nil, nil, 0, errors.New("at least one argument required")
	}

	if !isAllowedGitCmd(params) {
		return nil, nil, 0, errors.Errorf("command failed: %q is not a allowed git command", params)
	}

	cmd := gitserver.DefaultClient.Command("git", params...)
	cmd.Repo = repo
	stdout, stderr, err = cmd.DividedOutput(ctx)
	exitCode = cmd.ExitStatus
	if exitCode != 0 && err != nil {
		err = nil // the error must just indicate that the exit code was nonzero
	}
	return stdout, stderr, exitCode, err
}

// ExecReader executes an arbitrary `git` command (`git [args...]`) and returns a reader connected
// to its stdout.
func ExecReader(ctx context.Context, repo api.RepoName, args []string) (io.ReadCloser, error) {
	if Mocks.ExecReader != nil {
		return Mocks.ExecReader(args)
	}

	span, ctx := ot.StartSpanFromContext(ctx, "Git: ExecReader")
	span.SetTag("args", args)
	defer span.Finish()

	if !isAllowedGitCmd(args) {
		return nil, errors.Errorf("command failed: %v is not a allowed git command", args)
	}
	cmd := gitserver.DefaultClient.Command("git", args...)
	cmd.Repo = repo
	return gitserver.StdoutReader(ctx, cmd)
}

func readUntilTimeout(ctx context.Context, cmd *gitserver.Cmd) ([]byte, bool, error) {
	stdout, err := cmd.Output(ctx)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return stdout, false, nil
		}

		stdout = bytes.TrimSpace(stdout)
		if len(stdout) > 100 {
			stdout = append(stdout[:100], []byte("... (truncated)")...)
		}
		return nil, false, errors.WithMessage(err, fmt.Sprintf("git command %v failed (output: %q)", cmd.Args, stdout))
	}

	return stdout, true, nil
}

var (
	// gitCmdAllowlist are commands and arguments that are allowed to execute when calling ExecSafe.
	gitCmdAllowlist = map[string][]string{
		"log":    append([]string{}, gitCommonAllowlist...),
		"show":   append([]string{}, gitCommonAllowlist...),
		"remote": {"-v"},
		"diff":   append([]string{}, gitCommonAllowlist...),
		"blame":  {"--root", "--incremental", "-w", "-p", "--porcelain", "--"},
		"branch": {"-r", "-a", "--contains"},

		"rev-parse":    {"--abbrev-ref", "--symbolic-full-name"},
		"rev-list":     {"--max-parents", "--reverse", "--max-count"},
		"ls-remote":    {"--get-url"},
		"symbolic-ref": {"--short"},
	}

	// `git log`, `git show`, `git diff`, etc., share a large common set of allowed args.
	gitCommonAllowlist = []string{
		"--name-status", "--full-history", "-M", "--date", "--format", "-i", "-n1", "-m", "--", "-n200", "-n2", "--follow", "--author", "--grep", "--date-order", "--decorate", "--skip", "--max-count", "--numstat", "--pretty", "--parents", "--topo-order", "--raw", "--follow", "--all", "--before", "--no-merges",
		"--patch", "--unified", "-S", "-G", "--pickaxe-all", "--pickaxe-regex", "--function-context", "--branches", "--source", "--src-prefix", "--dst-prefix", "--no-prefix",
		"--regexp-ignore-case", "--glob", "--cherry", "-z",
		"--until", "--since", "--author", "--committer",
		"--all-match", "--invert-grep", "--extended-regexp",
		"--no-color", "--decorate", "--no-patch", "--exclude",
		"--no-merges",
		"--full-index",
		"--find-copies",
		"--find-renames",
		"--inter-hunk-context",
	}
)

// isAllowedGitArg checks if the arg is allowed.
func isAllowedGitArg(allowedArgs []string, arg string) bool {
	// Split the arg at the first equal sign and check the LHS against the allowlist args.
	splitArg := strings.Split(arg, "=")[0]
	for _, allowedArg := range allowedArgs {
		if splitArg == allowedArg {
			return true
		}
	}
	return false
}

// isAllowedGitCmd checks if the cmd and arguments are allowed.
func isAllowedGitCmd(args []string) bool {
	// check if the supplied command is a allowed cmd
	if len(gitCmdAllowlist) == 0 {
		return false
	}
	cmd := args[0]
	allowedArgs, ok := gitCmdAllowlist[cmd]
	if !ok {
		// Command not allowed
		return false
	}
	for _, arg := range args[1:] {
		if strings.HasPrefix(arg, "-") {
			// Special-case `git log -S` and `git log -G`, which interpret any characters
			// after their 'S' or 'G' as part of the query. There is no long form of this
			// flags (such as --something=query), so if we did not special-case these,
			// there would be no way to safely express a query that began with a '-'
			// character. (Same for `git show`, where the flag has the same meaning.)
			if (cmd == "log" || cmd == "show") && (strings.HasPrefix(arg, "-S") || strings.HasPrefix(arg, "-G")) {
				continue // this arg is OK
			}

			if !isAllowedGitArg(allowedArgs, arg) {
				return false
			}
		}
	}
	return true
}

func gitserverCmdFunc(repo api.RepoName) cmdFunc {
	return func(args []string) cmd {
		cmd := gitserver.DefaultClient.Command("git", args...)
		cmd.Repo = repo
		return cmd
	}
}

// cmdFunc is a func that creates a new executable Git command.
type cmdFunc func(args []string) cmd

// cmd is an executable Git command.
type cmd interface {
	Output(context.Context) ([]byte, error)
	String() string
}
