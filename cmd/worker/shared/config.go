package shared

import (
	"strings"

	"github.com/cockroachdb/errors"

	"github.com/sourcegraph/sourcegraph/internal/env"
)

// Config is the configuration that controls what jobs will be initialized
// and monitored. By default, all jobs are enabled. Individual jobs can be
// explicit allowed or blocked from running on a particular instance.
type Config struct {
	env.BaseConfig
	names []string

	JobAllowlist []string
	JobBlocklist []string
}

var config = &Config{}

// Load reads from the environment and stores the transformed data on the config
// object for later retrieval.
func (c *Config) Load() {
	c.JobAllowlist = safeSplit(c.Get(
		"WORKER_JOB_ALLOWLIST",
		"all",
		`A comma-seprated list of names of jobs that should be enabled. The value "all" (the default) enables all jobs.`,
	), ",")

	c.JobBlocklist = safeSplit(c.Get(
		"WORKER_JOB_BLOCKLIST",
		"",
		"A comma-seprated list of names of jobs that should not be enabled. Values in this list take precedence over the allowlist.",
	), ",")
}

// Validate returns an error indicating if there was an invalid environment read
// during Load. The environment is invalid when a supplied job name is not recognized
// by the set of names registered to the worker (at compile time).
//
// This method assumes that the name field has been set externally.
func (c *Config) Validate() error {
	allowlist := map[string]struct{}{}
	for _, name := range c.names {
		allowlist[name] = struct{}{}
	}

	for _, name := range c.JobAllowlist {
		if _, ok := allowlist[name]; !ok && name != "all" {
			return errors.Errorf("unknown job %q", name)
		}
	}
	for _, name := range c.JobBlocklist {
		if _, ok := allowlist[name]; !ok {
			return errors.Errorf("unknown job %q", name)
		}
	}

	return nil
}

// shouldRunJob returns true if the given job should be run.
func shouldRunJob(name string) bool {
	for _, candidate := range config.JobBlocklist {
		if name == candidate {
			return false
		}
	}

	for _, candidate := range config.JobAllowlist {
		if candidate == "all" || name == candidate {
			return true
		}
	}

	return false
}

// safeSplit is strings.Split but returns nil (not a []string{""}) on empty input.
func safeSplit(text, sep string) []string {
	if text == "" {
		return nil
	}

	return strings.Split(text, sep)
}
