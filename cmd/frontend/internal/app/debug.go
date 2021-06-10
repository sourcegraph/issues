package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/app/debugproxies"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/debugserver"
	"github.com/sourcegraph/sourcegraph/internal/env"
	srcprometheus "github.com/sourcegraph/sourcegraph/internal/src-prometheus"
)

var grafanaURLFromEnv = env.Get("GRAFANA_SERVER_URL", "", "URL at which Grafana can be reached")
var jaegerURLFromEnv = env.Get("JAEGER_SERVER_URL", "", "URL at which Jaeger UI can be reached")

func init() {
	conf.ContributeWarning(newPrometheusValidator(srcprometheus.NewClient(srcprometheus.PrometheusURL)))
}

func addNoK8sClientHandler(r *mux.Router, db dbutil.DB) {
	noHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `Cluster information not available`)
		fmt.Fprintf(w, `<br><br><a href="headers">headers</a><br>`)
	})
	r.Handle("/", adminOnly(noHandler, db))
}

// addDebugHandlers registers the reverse proxies to each services debug
// endpoints.
func addDebugHandlers(r *mux.Router, db dbutil.DB) {
	addGrafana(r, db)
	addJaeger(r, db)

	rph := debugproxies.ReverseProxyHandler{DB: db}

	if len(debugserver.Services) > 0 {
		peps := make([]debugproxies.Endpoint, 0, len(debugserver.Services))
		for _, s := range debugserver.Services {
			peps = append(peps, debugproxies.Endpoint{
				Service: s.Name,
				Addr:    s.Host,
			})
		}
		rph.Populate(peps)
	} else if conf.IsDeployTypeKubernetes(conf.DeployType()) {
		err := debugproxies.StartClusterScanner(rph.Populate)
		if err != nil {
			// we ended up here because cluster is not a k8s cluster
			addNoK8sClientHandler(r, db)
			return
		}
	} else {
		addNoK8sClientHandler(r, db)
	}

	rph.AddToRouter(r)
}

// PreMountGrafanaHook (if set) is invoked as a hook prior to mounting a
// the Grafana endpoint to the debug router.
var PreMountGrafanaHook func() error

// This error is returned if the current license does not support monitoring.
const errMonitoringNotLicensed = `The feature "monitoring" is not activated in your Sourcegraph license. Upgrade your Sourcegraph subscription to use this feature.`

func addNoGrafanaHandler(r *mux.Router, db dbutil.DB) {
	noGrafana := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `Grafana endpoint proxying: Please set env var GRAFANA_SERVER_URL`)
	})
	r.Handle("/grafana", adminOnly(noGrafana, db))
}

func addGrafanaNotLicensedHandler(r *mux.Router, db dbutil.DB) {
	notLicensed := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, errMonitoringNotLicensed, http.StatusUnauthorized)
	})
	r.Handle("/grafana", adminOnly(notLicensed, db))
}

// addReverseProxyForService registers a reverse proxy for the specified service.
func addGrafana(r *mux.Router, db dbutil.DB) {
	if PreMountGrafanaHook != nil {
		if err := PreMountGrafanaHook(); err != nil {
			addGrafanaNotLicensedHandler(r, db)
			return
		}
	}
	if len(grafanaURLFromEnv) > 0 {
		grafanaURL, err := url.Parse(grafanaURLFromEnv)
		if err != nil {
			log.Printf("failed to parse GRAFANA_SERVER_URL=%s: %v",
				grafanaURLFromEnv, err)
			addNoGrafanaHandler(r, db)
		} else {
			prefix := "/grafana"
			// 🚨 SECURITY: Only admins have access to Grafana dashboard
			r.PathPrefix(prefix).Handler(adminOnly(&httputil.ReverseProxy{
				Director: func(req *http.Request) {
					req.URL.Scheme = "http"
					req.URL.Host = grafanaURL.Host
					if i := strings.Index(req.URL.Path, prefix); i >= 0 {
						req.URL.Path = req.URL.Path[i+len(prefix):]
					}
				},
				ErrorLog: log.New(env.DebugOut, fmt.Sprintf("%s debug proxy: ", "grafana"), log.LstdFlags),
			}, db))
		}
	} else {
		addNoGrafanaHandler(r, db)
	}
}

func addNoJaegerHandler(r *mux.Router, db dbutil.DB) {
	noJaeger := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `Jaeger endpoint proxying: Please set env var JAEGER_SERVER_URL`)
	})
	r.Handle("/jaeger", adminOnly(noJaeger, db))
}

func addJaeger(r *mux.Router, db dbutil.DB) {
	if len(jaegerURLFromEnv) > 0 {
		fmt.Println("Jaeger URL from env ", jaegerURLFromEnv)
		jaegerURL, err := url.Parse(jaegerURLFromEnv)
		if err != nil {
			log.Printf("failed to parse JAEGER_SERVER_URL=%s: %v", jaegerURLFromEnv, err)
			addNoJaegerHandler(r, db)
		} else {
			prefix := "/jaeger"
			// 🚨 SECURITY: Only admins have access to Jaeger dashboard
			r.PathPrefix(prefix).Handler(adminOnly(&httputil.ReverseProxy{
				Director: func(req *http.Request) {
					req.URL.Scheme = "http"
					req.URL.Host = jaegerURL.Host
				},
				ErrorLog: log.New(env.DebugOut, fmt.Sprintf("%s debug proxy: ", "jaeger"), log.LstdFlags),
			}, db))
		}

	} else {
		addNoJaegerHandler(r, db)
	}
}

// adminOnly is a HTTP middleware which only allows requests by admins.
func adminOnly(next http.Handler, db dbutil.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := backend.CheckCurrentUserIsSiteAdmin(r.Context(), db); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// newPrometheusValidator renders problems with the Prometheus deployment and relevant site configuration
// as reported by `prom-wrapper` inside the `sourcegraph/prometheus` container if Prometheus is enabled.
//
// It also accepts the error from creating `srcprometheus.Client` as an parameter, to validate
// Prometheus configuration.
func newPrometheusValidator(prom srcprometheus.Client, promErr error) conf.Validator {
	return func(c conf.Unified) conf.Problems {
		// surface new prometheus client error if it was unexpected
		prometheusUnavailable := promErr != nil && errors.Is(promErr, srcprometheus.ErrPrometheusUnavailable)
		if promErr != nil && !prometheusUnavailable {
			return conf.NewSiteProblems(fmt.Sprintf("Prometheus (`PROMETHEUS_URL`) might be misconfigured: %v", promErr))
		}

		// no need to validate prometheus config if no `observability.*` settings are configured
		observabilityNotConfigured := len(c.ObservabilityAlerts) == 0 && len(c.ObservabilitySilenceAlerts) == 0
		if observabilityNotConfigured {
			// no observability configuration, no checks to make
			return nil
		} else if prometheusUnavailable {
			// no prometheus, but observability is configured
			return conf.NewSiteProblems("`observability.alerts` or `observability.silenceAlerts` are configured, but Prometheus is not available")
		}

		// use a short timeout to avoid having this block problems from loading
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		// get reported problems
		status, err := prom.GetConfigStatus(ctx)
		if err != nil {
			return conf.NewSiteProblems(fmt.Sprintf("`observability`: failed to fetch alerting configuration status: %v", err))
		}
		return status.Problems
	}
}
