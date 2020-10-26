package main

import (
	"log"
	"strconv"
	"time"

	"github.com/sourcegraph/sourcegraph/internal/env"
)

var (
	rawBundleDir           = env.Get("PRECISE_CODE_INTEL_BUNDLE_DIR", "/lsif-storage", "Root dir containing uploads and converted bundles.")
	rawReaderDataCacheSize = env.Get("PRECISE_CODE_INTEL_READER_DATA_CACHE_CAPACITY", "1000000", "Maximum sum of (compressed and marshalled) source data (in bytes) that can be loaded into the SQLite reader data cache at once.")
	rawJanitorInterval     = env.Get("PRECISE_CODE_INTEL_JANITOR_INTERVAL", "1m", "Interval between cleanup runs.")
	rawMaxUploadAge        = env.Get("PRECISE_CODE_INTEL_MAX_UPLOAD_AGE", "24h", "The maximum time an upload can sit on disk.")
	rawMaxUploadPartAge    = env.Get("PRECISE_CODE_INTEL_MAX_UPLOAD_PART_AGE", "2h", "The maximum time an upload part file can sit on disk.")
	rawMaxDataAge          = env.Get("PRECISE_CODE_INTEL_MAX_DATA_AGE", "720h", "The maximum time LSIF data not visible from the tip of the default branch can remain in the database.")
	rawDisableJanitor      = env.Get("PRECISE_CODE_INTEL_DISABLE_JANITOR", "false", "Set to true to disable the janitor process during system migrations.")
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
	i, err := strconv.ParseInt(rawValue, 10, 64)
	if err != nil {
		log.Fatalf("invalid int %q for %s: %s", rawValue, name, err)
	}

	return int(i)
}

// mustParsePercent returns the integer percent (in range [0, 100]) version of the given raw
// value fatally logs on failure.
func mustParsePercent(rawValue, name string) int {
	p := mustParseInt(rawValue, name)
	if p < 0 || p > 100 {
		log.Fatalf("invalid percent %q for %s: must be 0 <= p <= 100", rawValue, name)
	}

	return p
}

// mustParseInterval returns the interval version of the given raw value fatally logs on failure.
func mustParseInterval(rawValue, name string) time.Duration {
	d, err := time.ParseDuration(rawValue)
	if err != nil {
		log.Fatalf("invalid duration %q for %s: %s", rawValue, name, err)
	}

	return d
}

// mustParseBool returns the boolean version of the given raw value fatally logs on failure.
func mustParseBool(rawValue, name string) bool {
	v, err := strconv.ParseBool(rawValue)
	if err != nil {
		log.Fatalf("invalid bool %q for %s: %s", rawValue, name, err)
	}

	return v
}
