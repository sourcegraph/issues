// Package srcprometheus defines an API to interact with Sourcegraph Prometheus, including
// prom-wrapper. See https://docs.sourcegraph.com/dev/background-information/observability/prometheus
package srcprometheus

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"syscall"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/internal/env"
)

// ErrPrometheusUnavailable is raised specifically when prometheusURL is unset or when
// prometheus API access times out, both of which indicate that the server API has likely
// been configured to explicitly disallow access to prometheus, or that prometheus is not
// deployed at all. The website checks for this error in `fetchMonitoringStats`, for example.
var ErrPrometheusUnavailable = errors.New("prometheus API is unavailable")

// PrometheusURL is the configured Prometheus instance.
var PrometheusURL = env.Get("PROMETHEUS_URL", "", "prometheus server URL")

// Client provides the interface for interacting with Sourcegraph Prometheus, including
// prom-wrapper. See https://docs.sourcegraph.com/dev/background-information/observability/prometheus
type Client interface {
	GetAlertsStatus(ctx context.Context) (*AlertsStatus, error)
	GetAlertsHistory(ctx context.Context, timespan time.Duration) (*AlertsHistory, error)
	GetConfigStatus(ctx context.Context) (*ConfigStatus, error)
}

type client struct {
	http    http.Client
	promURL url.URL
}

// NewClient provides a client for interacting with Sourcegraph Prometheus. It errors if
// the target Prometheus URL is invalid, or if no Prometheus URL is configured at all.
// Users should check for the latter case by asserting against `ErrPrometheusUnavailable`
// to avoid rendering an error.
//
// See https://docs.sourcegraph.com/dev/background-information/observability/prometheus
func NewClient(prometheusURL string) (Client, error) {
	if prometheusURL == "" {
		return nil, ErrPrometheusUnavailable
	}
	promURL, err := url.Parse(prometheusURL)
	if err != nil {
		return nil, errors.Errorf("invalid URL: %w", err)
	}
	return &client{
		http: http.Client{
			Transport: &roundTripper{},
		},
		promURL: *promURL,
	}, nil
}

func (c *client) newRequest(endpoint string, query url.Values) (*http.Request, error) {
	target := c.promURL
	target.Path = endpoint
	if query != nil {
		target.RawQuery = query.Encode()
	}
	req, err := http.NewRequest(http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, errors.Errorf("prometheus misconfigured: %w", err)
	}
	return req, nil
}

func (c *client) do(ctx context.Context, req *http.Request) (*http.Response, error) {
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, errors.Errorf("src-prometheus: %w", err)
	}
	if resp.StatusCode != 200 {
		log15.Error("src-prometheus request made but failed with non-zero status", "request", req, "resp", resp)
		return nil, errors.Errorf("src-prometheus: %s %q: failed with status %d", req.Method, req.URL.String(), resp.StatusCode)
	}
	return resp, nil
}

const EndpointAlertsStatus = "/prom-wrapper/alerts-status"

// GetAlertsStatus retrieves an overview of current alerts
func (c *client) GetAlertsStatus(ctx context.Context) (*AlertsStatus, error) {
	req, err := c.newRequest(EndpointAlertsStatus, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(ctx, req)
	if err != nil {
		return nil, err
	}

	var alertsStatus AlertsStatus
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&alertsStatus); err != nil {
		return nil, err
	}
	return &alertsStatus, nil
}

const EndpointAlertsStatusHistory = EndpointAlertsStatus + "/history"

// GetAlertsHistory retrieves a historical summary of all alerts
func (c *client) GetAlertsHistory(ctx context.Context, timespan time.Duration) (*AlertsHistory, error) {
	query := make(url.Values)
	query.Add("timespan", timespan.String())
	req, err := c.newRequest(EndpointAlertsStatusHistory, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(ctx, req)
	if err != nil {
		return nil, err
	}

	var alertsHistory AlertsHistory
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&alertsHistory); err != nil {
		return nil, err
	}
	return &alertsHistory, nil
}

const EndpointConfigSubscriber = "/prom-wrapper/config-subscriber"

func (c *client) GetConfigStatus(ctx context.Context) (*ConfigStatus, error) {
	req, err := c.newRequest(EndpointConfigSubscriber, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(ctx, req)
	if err != nil {
		return nil, err
	}

	var status ConfigStatus
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, err
	}
	return &status, nil
}

// roundTripper treats certain connection errors as `ErrPrometheusUnavailable` which can be
// handled explicitly for environments without Prometheus available.
type roundTripper struct{}

func (r *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := http.DefaultTransport.RoundTrip(req)

	// Check for specific syscall errors to detect if the provided prometheus server is
	// not accessible in this deployment. Treat deadline exceeded as an indicator as well.
	//
	// See https://github.com/golang/go/issues/9424
	if errors.IsAny(err, context.DeadlineExceeded, syscall.ECONNREFUSED, syscall.EHOSTUNREACH) {
		err = ErrPrometheusUnavailable
	}

	return resp, err
}
