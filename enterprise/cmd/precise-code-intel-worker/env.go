package main

import (
	"log"
	"strconv"
	"time"

	"github.com/sourcegraph/sourcegraph/internal/env"
)

var (
	rawBundleManagerURL      = env.Get("PRECISE_CODE_INTEL_BUNDLE_MANAGER_URL", "", "HTTP address for internal LSIF bundle manager server.")
	rawWorkerPollInterval    = env.Get("PRECISE_CODE_INTEL_WORKER_POLL_INTERVAL", "1s", "Interval between queries to the upload queue.")
	rawWorkerConcurrency     = env.Get("PRECISE_CODE_INTEL_WORKER_CONCURRENCY", "1", "The maximum number of indexes that can be processed concurrently.")
	rawWorkerBudget          = env.Get("PRECISE_CODE_INTEL_WORKER_BUDGET", "0", "The amount of compressed input data (in bytes) a worker can process concurrently. Zero acts as an infinite budget.")
	rawResetInterval         = env.Get("PRECISE_CODE_INTEL_RESET_INTERVAL", "1m", "How often to reset stalled uploads.")
	rawCommitUpdaterInterval = env.Get("PRECISE_CODE_INTEL_COMMIT_UPDATER_INTERVAL", "5s", "How often to update commits for dirty repositories.")
)

// mustGet returns the non-empty version of the given raw value fatally logs on failure.
func mustGet(rawValue, name string) string {
	if rawValue == "" {
		log.Fatalf("invalid value %q for %s: no value supplied", rawValue, name)
	}

	return rawValue
}

// mustParseInt returns the integer version of the given raw value fatally logs on failure.
func mustParseInt(rawValue, name string) int {
	return int(mustParseInt64(rawValue, name))
}

// mustParseInt64 returns the integer version of the given raw value fatally logs on failure.
func mustParseInt64(rawValue, name string) int64 {
	i, err := strconv.ParseInt(rawValue, 10, 64)
	if err != nil {
		log.Fatalf("invalid int %q for %s: %s", rawValue, name, err)
	}

	return i
}

// mustParseInterval returns the interval version of the given raw value fatally logs on failure.
func mustParseInterval(rawValue, name string) time.Duration {
	d, err := time.ParseDuration(rawValue)
	if err != nil {
		log.Fatalf("invalid duration %q for %s: %s", rawValue, name, err)
	}

	return d
}
