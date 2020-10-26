package main

import (
	"log"
	"strconv"
	"time"

	"github.com/sourcegraph/sourcegraph/internal/env"
)

var (
	rawFrontendURL              = env.Get("PRECISE_CODE_INTEL_EXTERNAL_URL", "", "The external URL of the sourcegraph instance.")
	rawFrontendURLFromDocker    = env.Get("PRECISE_CODE_INTEL_EXTERNAL_URL_FROM_DOCKER", "", "The external URL of the sourcegraph instance used form within an index container.")
	rawInternalProxyAuthToken   = env.Get("PRECISE_CODE_INTEL_INTERNAL_PROXY_AUTH_TOKEN", "", "The auth token supplied to the frontend.")
	rawIndexerPollInterval      = env.Get("PRECISE_CODE_INTEL_INDEXER_POLL_INTERVAL", "1s", "Interval between queries to the precise-code-intel-index-manager.")
	rawIndexerHeartbeatInterval = env.Get("PRECISE_CODE_INTEL_INDEXER_HEARTBEAT_INTERVAL", "1s", "Interval between heartbeat requests.")
	rawMaxContainers            = env.Get("PRECISE_CODE_INTEL_MAXIMUM_CONTAINERS", "1", "Number of virtual machines or containers that can be running at once.")
	rawFirecrackerImage         = env.Get("PRECISE_CODE_INTEL_FIRECRACKER_IMAGE", "sourcegraph/ignite-ubuntu:insiders", "The base image to use for virtual machines.")
	rawUseFirecracker           = env.Get("PRECISE_CODE_INTEL_USE_FIRECRACKER", "true", "Whether to isolate index containers in virtual machines.")
	rawFirecrackerNumCPUs       = env.Get("PRECISE_CODE_INTEL_FIRECRACKER_NUM_CPUS", "4", "How many CPUs to allocate to each virtual machine or container.")
	rawFirecrackerMemory        = env.Get("PRECISE_CODE_INTEL_FIRECRACKER_MEMORY", "12G", "How much memory to allocate to each virtual machine or container.")
	rawFirecrackerDiskSpace     = env.Get("PRECISE_CODE_INTEL_FIRECRACKER_DISK_SPACE", "20G", "How much disk space to allocate to each virtual machine or container.")
	rawImageArchivePath         = env.Get("PRECISE_CODE_INTEL_IMAGE_ARCHIVE_PATH", "", "Where to store tar archives of docker images shared by virtual machines.")
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
