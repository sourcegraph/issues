// Package tracer initializes distributed tracing and log15 behavior. It also updates distributed
// tracing behavior in response to changes in site configuration. When the Init function of this
// package is invoked, opentracing.SetGlobalTracer is called (and subsequently called again after
// every Sourcegraph site configuration change). Importing programs should not invoke
// opentracing.SetGlobalTracer anywhere else.
package tracer

import (
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"

	"github.com/inconshreveable/log15"
	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/env"
	"github.com/sourcegraph/sourcegraph/internal/trace"
	"github.com/sourcegraph/sourcegraph/internal/trace/ot"
	"go.uber.org/automaxprocs/maxprocs"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jaegerlog "github.com/uber/jaeger-client-go/log"
	jaegermetrics "github.com/uber/jaeger-lib/metrics"
)

func init() {
	// Tune GOMAXPROCS for kubernetes. All our binaries import this package,
	// so we tune for all of them.
	//
	// TODO it is surprising that we do this here. We should create a standard
	// import for sourcegraph binaries which would have less surprising
	// behaviour.
	if _, err := maxprocs.Set(); err != nil {
		log15.Error("automaxprocs failed", "error", err)
	}
}

// Options control the behavior of a tracer.
type Options struct {
	serviceName string
}

// If this idiom seems strange:
// https://github.com/tmrts/go-patterns/blob/master/idiom/functional-options.md
type Option func(*Options)

func ServiceName(s string) Option {
	return func(o *Options) {
		o.serviceName = s
	}
}

func Init(options ...Option) {
	opts := &Options{}
	for _, setter := range options {
		setter(opts)
	}
	if opts.serviceName == "" {
		opts.serviceName = env.MyName
	}

	initTracer(opts.serviceName)
}

type jaegerOpts struct {
	ServiceName string
	ExternalURL string
	Enabled     bool
	Debug       bool
}

// initTracer is a helper that should be called exactly once (from Init).
func initTracer(serviceName string) {
	globalTracer := newSwitchableTracer()
	opentracing.SetGlobalTracer(globalTracer)

	// initial tracks if its our first run of conf.Watch. This is used to
	// prevent logging "changes" when its the first run.
	initial := true

	// Initially everything is disabled since we haven't read conf yet.
	oldOpts := jaegerOpts{
		ServiceName: serviceName,
		Enabled:     false,
		Debug:       false,
	}

	// Watch loop
	go conf.Watch(func() {
		siteConfig := conf.Get()

		// Set sampling strategy
		samplingStrategy := ot.TraceNone
		shouldLog := false
		if tracingConfig := siteConfig.ObservabilityTracing; tracingConfig != nil {
			switch tracingConfig.Sampling {
			case "all":
				samplingStrategy = ot.TraceAll
			case "selective":
				samplingStrategy = ot.TraceSelective
			}
			shouldLog = tracingConfig.Debug
		} else if siteConfig.UseJaeger {
			samplingStrategy = ot.TraceAll
		}
		if tracePolicy := ot.GetTracePolicy(); tracePolicy != samplingStrategy && !initial {
			log15.Info("opentracing: TracePolicy", "oldValue", tracePolicy, "newValue", samplingStrategy)
		}
		initial = false
		ot.SetTracePolicy(samplingStrategy)

		opts := jaegerOpts{
			ServiceName: serviceName,
			ExternalURL: siteConfig.ExternalURL,
			Enabled:     samplingStrategy == ot.TraceAll || samplingStrategy == ot.TraceSelective,
			Debug:       shouldLog,
		}

		if opts == oldOpts {
			// Nothing changed
			return
		}

		oldOpts = opts

		tracer, urlFunc, closer, err := newTracer(&opts)
		if err != nil {
			log15.Warn("Could not initialize jaeger tracer", "error", err.Error())
			return
		}

		globalTracer.set(tracer, closer, opts.Debug)
		trace.SetSpanURLFunc(urlFunc)
	})
}

func newTracer(opts *jaegerOpts) (opentracing.Tracer, func(span opentracing.Span) string, io.Closer, error) {
	if !opts.Enabled {
		log15.Info("opentracing: Jaeger disabled")
		return opentracing.NoopTracer{}, nil, nil, nil
	}

	log15.Info("opentracing: Jaeger enabled")
	cfg, err := jaegercfg.FromEnv()
	cfg.ServiceName = opts.ServiceName
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "jaegercfg.FromEnv failed")
	}
	if reflect.DeepEqual(cfg.Sampler, &jaegercfg.SamplerConfig{}) {
		// Default sampler configuration for when it is not specified via
		// JAEGER_SAMPLER_* env vars. In most cases, this is sufficient
		// enough to connect Sourcegraph to Jaeger without any env vars.
		cfg.Sampler.Type = jaeger.SamplerTypeConst
		cfg.Sampler.Param = 1
	}
	tracer, closer, err := cfg.NewTracer(
		jaegercfg.Logger(jaegerlog.StdLogger),
		jaegercfg.Metrics(jaegermetrics.NullFactory),
	)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "jaegercfg.NewTracer failed")
	}

	// We proxy jaeger so we can construct URLs to traces.
	jaegerURL := strings.TrimSuffix(opts.ExternalURL, "/") + "/-/debug/jaeger/trace/"

	spanURL := func(span opentracing.Span) string {
		if span == nil {
			return tracingNotEnabledURL
		}
		spanCtx, ok := span.Context().(jaeger.SpanContext)
		if !ok {
			return tracingNotEnabledURL
		}
		return jaegerURL + spanCtx.TraceID().String()
	}

	return tracer, spanURL, closer, nil
}

// switchableTracer implements opentracing.Tracer. The underlying tracer used is switchable (set via
// the `set` method).
type switchableTracer struct {
	mu           sync.RWMutex
	tracer       opentracing.Tracer
	tracerCloser io.Closer
	log          bool
}

func newSwitchableTracer() *switchableTracer {
	return &switchableTracer{tracer: opentracing.NoopTracer{}}
}

func (t *switchableTracer) StartSpan(operationName string, opts ...opentracing.StartSpanOption) opentracing.Span {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.log {
		log15.Info("opentracing: StartSpan", "operationName", operationName, "tracer", fmt.Sprintf("%T", t.tracer))
	}
	return t.tracer.StartSpan(operationName, opts...)
}

func (t *switchableTracer) Inject(sm opentracing.SpanContext, format interface{}, carrier interface{}) error {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.log {
		log15.Info("opentracing: Inject", "tracer", fmt.Sprintf("%T", t.tracer))
	}
	return t.tracer.Inject(sm, format, carrier)
}

func (t *switchableTracer) Extract(format interface{}, carrier interface{}) (opentracing.SpanContext, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.log {
		log15.Info("opentracing: Extract", "tracer", fmt.Sprintf("%T", t.tracer))
	}
	return t.tracer.Extract(format, carrier)
}

func (t *switchableTracer) set(tracer opentracing.Tracer, tracerCloser io.Closer, log bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if tc := t.tracerCloser; tc != nil {
		// Close the old tracerCloser outside the critical zone
		go tc.Close()
	}

	t.tracerCloser = tracerCloser
	t.tracer = tracer
	t.log = log
}

func (t *switchableTracer) get() (tracer opentracing.Tracer, log bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.tracer, t.log
}

const tracingNotEnabledURL = "#tracing_not_enabled_for_this_request_add_?trace=1_to_url_to_enable"
