package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/app/debugproxies"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/debugserver"
	"github.com/sourcegraph/sourcegraph/internal/env"
	"github.com/sourcegraph/sourcegraph/internal/prometheusutil"
)

var grafanaURLFromEnv = env.Get("GRAFANA_SERVER_URL", "", "URL at which Grafana can be reached")
var jaegerURLFromEnv = env.Get("JAEGER_SERVER_URL", "", "URL at which Jaeger UI can be reached")

func init() {
	conf.ContributeWarning(newPrometheusValidator(prometheusutil.PrometheusURL))
}

func addNoK8sClientHandler(r *mux.Router) {
	noHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `Cluster information not available`)
		fmt.Fprintf(w, `<br><br><a href="headers">headers</a><br>`)
	})
	r.Handle("/", adminOnly(noHandler))
}

// addDebugHandlers registers the reverse proxies to each services debug
// endpoints.
func addDebugHandlers(r *mux.Router) {
	addGrafana(r)
	addJaeger(r)

	var rph debugproxies.ReverseProxyHandler

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
			addNoK8sClientHandler(r)
			return
		}
	} else {
		addNoK8sClientHandler(r)
	}

	rph.AddToRouter(r)
}

func addNoGrafanaHandler(r *mux.Router) {
	noGrafana := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `Grafana endpoint proxying: Please set env var GRAFANA_SERVER_URL`)
	})
	r.Handle("/grafana", adminOnly(noGrafana))
}

// addReverseProxyForService registers a reverse proxy for the specified service.
func addGrafana(r *mux.Router) {
	if len(grafanaURLFromEnv) > 0 {
		grafanaURL, err := url.Parse(grafanaURLFromEnv)
		if err != nil {
			log.Printf("failed to parse GRAFANA_SERVER_URL=%s: %v",
				grafanaURLFromEnv, err)
			addNoGrafanaHandler(r)
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
			}))
		}
	} else {
		addNoGrafanaHandler(r)
	}
}

func addNoJaegerHandler(r *mux.Router) {
	noJaeger := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `Jaeger endpoint proxying: Please set env var JAEGER_SERVER_URL`)
	})
	r.Handle("/jaeger", adminOnly(noJaeger))
}

func addJaeger(r *mux.Router) {
	if len(jaegerURLFromEnv) > 0 {
		fmt.Println("Jaeger URL from env ", jaegerURLFromEnv)
		jaegerURL, err := url.Parse(jaegerURLFromEnv)
		if err != nil {
			log.Printf("failed to parse JAEGER_SERVER_URL=%s: %v", jaegerURLFromEnv, err)
			addNoJaegerHandler(r)
		} else {
			prefix := "/jaeger"
			// 🚨 SECURITY: Only admins have access to Jaeger dashboard
			r.PathPrefix(prefix).Handler(adminOnly(&httputil.ReverseProxy{
				Director: func(req *http.Request) {
					req.URL.Scheme = "http"
					req.URL.Host = jaegerURL.Host
				},
				ErrorLog: log.New(env.DebugOut, fmt.Sprintf("%s debug proxy: ", "jaeger"), log.LstdFlags),
			}))
		}

	} else {
		addNoJaegerHandler(r)
	}
}

// adminOnly is a HTTP middleware which only allows requests by admins.
func adminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := backend.CheckCurrentUserIsSiteAdmin(r.Context()); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// newPrometheusValidator renders problems with the Prometheus deployment and configuration
// as reported by `prom-wrapper` inside the `sourcegraph/prometheus` container if Prometheus is enabled.
func newPrometheusValidator(prometheusURL string) conf.Validator {
	return func(c conf.Unified) (problems conf.Problems) {
		// no need to validate prometheus config if no `observability.*`` settings are configured
		observabilityNotConfigured := len(c.ObservabilityAlerts) == 0 && len(c.ObservabilitySilenceAlerts) == 0
		if prometheusURL == "" || observabilityNotConfigured {
			return
		}

		// set up request to fetch status from grafana-wrapper
		promURL, err := url.Parse(prometheusURL)
		if err != nil {
			return // don't report problem, since activeAlertsAlert will report this
		}
		promURL.Path = "/prom-wrapper/config-subscriber"
		req, err := http.NewRequest("GET", promURL.String(), nil)
		if err != nil {
			return // don't report problem, since activeAlertsAlert will report this
		}

		// use a short timeout to avoid having this block problems from loading
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		resp, err := http.DefaultClient.Do(req.WithContext(ctx))
		if err != nil {
			problems = append(problems, conf.NewSiteProblem(fmt.Sprintf("`observability.alerts`: Unable to fetch configuration status: %v", err)))
			return
		}
		if resp.StatusCode != 200 {
			problems = append(problems, conf.NewSiteProblem(fmt.Sprintf("`observability.alerts`: Unable to fetch configuration status: status code %d", resp.StatusCode)))
			return
		}

		var promConfigStatus struct {
			Problems conf.Problems `json:"problems"`
		}
		defer resp.Body.Close()
		if err := json.NewDecoder(resp.Body).Decode(&promConfigStatus); err != nil {
			problems = append(problems, conf.NewSiteProblem(fmt.Sprintf("`observability.alerts`: unable to read Prometheus status: %v", err)))
			return
		}

		return promConfigStatus.Problems
	}
}
