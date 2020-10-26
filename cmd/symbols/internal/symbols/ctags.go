package symbols

import (
	"fmt"
	"log"
	"os"
	"strconv"

	ctags "github.com/sourcegraph/go-ctags"
	"github.com/sourcegraph/sourcegraph/internal/env"
)

const debugLogs = false

var logErrors = os.Getenv("DEPLOY_TYPE") == "dev"

var ctagsCommand = env.Get("CTAGS_COMMAND", "universal-ctags", "ctags command (should point to universal-ctags executable compiled with JSON and seccomp support)")

// Increasing this value may increase the size of the symbols cache, but will also stop long lines containing symbols from
// being highlighted improperly. See https://github.com/sourcegraph/sourcegraph/issues/7668.
var rawPatternLengthLimit = env.Get("CTAGS_PATTERN_LENGTH_LIMIT", "250", "the maximum length of the patterns output by ctags")

// New runs the ctags command from the CTAGS_COMMAND environment
// variable, falling back to `universal-ctags`.
func NewParser() (ctags.Parser, error) {
	patternLengthLimit, err := strconv.Atoi(rawPatternLengthLimit)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern length limit: %s", rawPatternLengthLimit)
	}

	var info *log.Logger
	if logErrors {
		info = log.New(os.Stderr, "ctags: ", log.LstdFlags)
	}

	var debug *log.Logger
	if debugLogs {
		debug = log.New(os.Stderr, "DBUG ctags: ", log.LstdFlags)
	}

	return ctags.New(ctags.Options{
		Bin:                ctagsCommand,
		PatternLengthLimit: patternLengthLimit,
		Info:               info,
		Debug:              debug,
	})
}
