//go:generate go build -o /tmp/monitoring-generator
//go:generate /tmp/monitoring-generator
package main

import (
	"os"
	"strconv"

	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/internal/logging"
	"github.com/sourcegraph/sourcegraph/monitoring/definitions"
	"github.com/sourcegraph/sourcegraph/monitoring/monitoring"
)

func optsFromEnv() monitoring.GenerateOptions {
	return monitoring.GenerateOptions{
		DisablePrune: getEnvBool("NO_PRUNE", false),
		Reload:       getEnvBool("RELOAD", false),

		GrafanaDir:    getEnvStr("GRAFANA_DIR", "../docker-images/grafana/config/provisioning/dashboards/sourcegraph/"),
		PrometheusDir: getEnvStr("PROMETHEUS_DIR", "../docker-images/prometheus/config/"),
		DocsDir:       getEnvStr("DOCS_DIR", "../doc/admin/observability/"),
	}
}

func main() {
	// Use standard Sourcegraph logging options and flags.
	logging.Init()
	logger := log15.Root()

	// Runs the monitoring generator. Ensure that any dashboards created or removed are
	// updated in the arguments here as required.
	if err := monitoring.Generate(logger, optsFromEnv(),
		definitions.Frontend(),
		definitions.GitServer(),
		definitions.GitHubProxy(),
		definitions.Postgres(),
		definitions.PreciseCodeIntelWorker(),
		definitions.QueryRunner(),
		definitions.Worker(),
		definitions.RepoUpdater(),
		definitions.Searcher(),
		definitions.Symbols(),
		definitions.SyntectServer(),
		definitions.ZoektIndexServer(),
		definitions.ZoektWebServer(),
		definitions.Prometheus(),
		definitions.ExecutorQueue(),
		definitions.Executor(),
	); err != nil {
		// Rely on the Generate function doing logging, so just exit with an appropriate
		// error code here.
		os.Exit(1)
	}
}

func getEnvStr(key, defaultValue string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if v, ok := os.LookupEnv(key); ok {
		b, err := strconv.ParseBool(v)
		if err != nil {
			panic(err)
		}
		return b
	}
	return defaultValue
}
