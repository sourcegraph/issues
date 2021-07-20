package metrics

import (
	"net/http"
	"net/url"
	"strconv"
	"syscall"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/inconshreveable/log15"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/sourcegraph/sourcegraph/internal/httpcli"
)

type testRegisterer struct{}

func (testRegisterer) Register(prometheus.Collector) error  { return nil }
func (testRegisterer) MustRegister(...prometheus.Collector) {}
func (testRegisterer) Unregister(prometheus.Collector) bool { return true }

// TestRegisterer is a behaviorless Prometheus Registerer usable for unit tests.
var TestRegisterer prometheus.Registerer = testRegisterer{}

// registerer exists so we can override it in tests
var registerer = prometheus.DefaultRegisterer

// RequestMeter wraps a Prometheus request meter (counter + duration histogram) updated by requests made by derived
// http.RoundTrippers.
type RequestMeter struct {
	counter   *prometheus.CounterVec
	duration  *prometheus.HistogramVec
	subsystem string
}

// NewRequestMeter creates a new request meter.
func NewRequestMeter(subsystem, help string) *RequestMeter {
	requestCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "src",
		Subsystem: subsystem,
		Name:      "requests_total",
		Help:      help,
	}, []string{"category", "code", "host"})
	registerer.MustRegister(requestCounter)

	// TODO(uwedeportivo):
	// A prometheus histogram has a request counter built in.
	// It will have the suffix _count (ie src_subsystem_request_duration_count).
	// See if we can get rid of requestCounter (if it hasn't been used by a customer yet) and use this counter instead.
	requestDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "src",
		Subsystem: subsystem,
		Name:      "request_duration_seconds",
		Help:      "Time (in seconds) spent on request.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"category", "code", "host"})
	registerer.MustRegister(requestDuration)

	return &RequestMeter{counter: requestCounter, duration: requestDuration, subsystem: subsystem}
}

// Transport returns an http.RoundTripper that updates rm for each request. The categoryFunc is called to
// determine the category label for each request.
func (rm *RequestMeter) Transport(transport http.RoundTripper, categoryFunc func(*url.URL) string) http.RoundTripper {
	return &requestCounterMiddleware{
		meter:        rm,
		transport:    transport,
		categoryFunc: categoryFunc,
	}
}

// Doer returns an httpcli.Doer that updates rm for each request. The categoryFunc is called to
// determine the category label for each request.
func (rm *RequestMeter) Doer(cli httpcli.Doer, categoryFunc func(*url.URL) string) httpcli.Doer {
	return &requestCounterMiddleware{
		meter:        rm,
		cli:          cli,
		categoryFunc: categoryFunc,
	}
}

type requestCounterMiddleware struct {
	meter        *RequestMeter
	cli          httpcli.Doer
	transport    http.RoundTripper
	categoryFunc func(*url.URL) string
}

func (t *requestCounterMiddleware) RoundTrip(r *http.Request) (resp *http.Response, err error) {
	start := time.Now()
	if t.transport != nil {
		resp, err = t.transport.RoundTrip(r)
	} else if t.cli != nil {
		resp, err = t.cli.Do(r)
	}

	category := t.categoryFunc(r.URL)

	var code string
	if err != nil {
		code = "error"
	} else {
		code = strconv.Itoa(resp.StatusCode)
	}

	d := time.Since(start)
	t.meter.counter.WithLabelValues(category, code, r.URL.Host).Inc()
	t.meter.duration.WithLabelValues(category, code, r.URL.Host).Observe(d.Seconds())
	log15.Debug("TRACE "+t.meter.subsystem, "host", r.URL.Host, "path", r.URL.Path, "code", code, "duration", d)
	return
}

func (t *requestCounterMiddleware) Do(req *http.Request) (*http.Response, error) {
	return t.RoundTrip(req)
}

// MustRegisterDiskMonitor exports two prometheus metrics
// "src_disk_space_available_bytes{path=$path}" and
// "src_disk_space_total_bytes{path=$path}". The values exported are for the
// filesystem that path is on.
//
// It is safe to call this function more than once for the same path.
func MustRegisterDiskMonitor(path string) {
	mustRegisterOnce(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name:        "src_disk_space_available_bytes",
		Help:        "Amount of free space disk space.",
		ConstLabels: prometheus.Labels{"path": path},
	}, func() float64 {
		var stat syscall.Statfs_t
		_ = syscall.Statfs(path, &stat)
		return float64(stat.Bavail * uint64(stat.Bsize))
	}))

	mustRegisterOnce(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name:        "src_disk_space_total_bytes",
		Help:        "Amount of total disk space.",
		ConstLabels: prometheus.Labels{"path": path},
	}, func() float64 {
		var stat syscall.Statfs_t
		_ = syscall.Statfs(path, &stat)
		return float64(stat.Blocks * uint64(stat.Bsize))
	}))
}

func mustRegisterOnce(c prometheus.Collector) {
	err := registerer.Register(c)
	if err != nil && !errors.HasType(err, prometheus.AlreadyRegisteredError{}) {
		panic(err)
	}
}
