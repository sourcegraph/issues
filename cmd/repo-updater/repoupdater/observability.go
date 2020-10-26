package repoupdater

import (
	"net/http"
	"strconv"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/opentracing-contrib/go-stdlib/nethttp"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sourcegraph/sourcegraph/internal/metrics"
	"github.com/sourcegraph/sourcegraph/internal/trace/ot"
)

// HandlerMetrics encapsulates the Prometheus metrics of an http.Handler.
type HandlerMetrics struct {
	ServeHTTP *metrics.OperationMetrics
}

// NewHandlerMetrics returns HandlerMetrics that need to be registered
// in a Prometheus registry.
func NewHandlerMetrics() HandlerMetrics {
	return HandlerMetrics{
		ServeHTTP: &metrics.OperationMetrics{
			Duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Name: "src_repoupdater_http_handler_duration_seconds",
				Help: "Time spent handling an HTTP request",
			}, []string{"path", "code"}),
			Count: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_repoupdater_http_handler_requests_total",
				Help: "Total number of HTTP requests",
			}, []string{"path", "code"}),
			Errors: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_repoupdater_http_handler_errors_total",
				Help: "Total number of HTTP error responses (code >= 400)",
			}, []string{"path", "code"}),
		},
	}
}

// MustRegister registers all metrics in HandlerMetrics in the given
// prometheus.Registerer. It panics in case of failure.
func (m HandlerMetrics) MustRegister(r prometheus.Registerer) {
	r.MustRegister(m.ServeHTTP.Count)
	r.MustRegister(m.ServeHTTP.Duration)
	r.MustRegister(m.ServeHTTP.Errors)
}

// ObservedHandler returns a decorator that wraps an http.Handler
// with logging, Prometheus metrics and tracing.
func ObservedHandler(
	log log15.Logger,
	m HandlerMetrics,
	tr opentracing.Tracer,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return ot.MiddlewareWithTracer(tr,
			&observedHandler{
				next:    next,
				log:     log,
				metrics: m,
			},
			nethttp.OperationNameFunc(func(r *http.Request) string {
				return "HTTP " + r.Method + ":" + r.URL.Path
			}),
			nethttp.MWComponentName("repo-updater"),
			nethttp.MWSpanObserver(func(sp opentracing.Span, r *http.Request) {
				sp.SetTag("http.uri", r.URL.EscapedPath())
			}),
		)
	}
}

type observedHandler struct {
	next    http.Handler
	log     log15.Logger
	metrics HandlerMetrics
}

func (h *observedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rr := &responseRecorder{w, http.StatusOK, 0}

	defer func(begin time.Time) {
		took := time.Since(begin)

		h.log.Debug(
			"http.request",
			"method", r.Method,
			"route", r.URL.Path,
			"code", rr.code,
			"duration", took,
		)

		var err error
		if rr.code >= 400 {
			err = errors.New(http.StatusText(rr.code))
		}

		h.metrics.ServeHTTP.Observe(
			took.Seconds(),
			1,
			&err,
			r.URL.Path,
			strconv.Itoa(rr.code),
		)
	}(time.Now())

	h.next.ServeHTTP(rr, r)
}

type responseRecorder struct {
	http.ResponseWriter
	code    int
	written int64
}

// WriteHeader may not be explicitly called, so care must be taken to
// initialize w.code to its default value of http.StatusOK.
func (w *responseRecorder) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseRecorder) Write(p []byte) (int, error) {
	n, err := w.ResponseWriter.Write(p)
	w.written += int64(n)
	return n, err
}
