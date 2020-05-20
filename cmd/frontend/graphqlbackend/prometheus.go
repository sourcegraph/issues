package graphqlbackend

import (
	"context"
	"errors"
	"fmt"

	prometheusAPI "github.com/prometheus/client_golang/api"
	prometheus "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/sourcegraph/sourcegraph/internal/env"
)

var prometheusURL = env.Get("PROMETHEUS_URL", "http://prometheus:9090", "prometheus server URL")

// prometheusQuerier provides a shim around prometheus.API
type prometheusQuerier interface {
	// QueryRange performs a query for the given range.
	QueryRange(ctx context.Context, query string, r prometheus.Range) (model.Value, prometheus.Warnings, error)
}

// errPrometheusUnavailable is raised specifically when prometheusURL is unset or when
// prometheus API access times out, both of which indicate that the server API has likely
// been configured to explicitly disallow access to prometheus, or that prometheus is not
// deployed at all. The website checks for this error in `fetchMonitoringStats`, for example.
var errPrometheusUnavailable = errors.New("prometheus API is unavailable")

func newPrometheusQuerier() (prometheusQuerier, error) {
	if prometheusURL == "" {
		return nil, errPrometheusUnavailable
	}
	c, err := prometheusAPI.NewClient(prometheusAPI.Config{
		Address: prometheusURL,
	})
	if err != nil {
		return nil, fmt.Errorf("prometheus configuration malformed: %w", err)
	}
	return prometheus.NewAPI(c), nil
}
