package shared

import (
	"os"

	"github.com/inconshreveable/log15"
)

const (
	prometheusProcLine = `prometheus: env STORAGE_PATH=/var/opt/sourcegraph/prometheus /bin/prom-wrapper >> /var/opt/sourcegraph/prometheus.log 2>&1`
	grafanaProcLine    = `grafana: /usr/share/grafana/bin/grafana-server -config /sg_config_grafana/grafana-single-container.ini -homepath /usr/share/grafana >> /var/opt/sourcegraph/grafana.log 2>&1`
	jaegerProcLine     = `jaeger: env QUERY_BASE_PATH=/-/debug/jaeger jaeger --memory.max-traces=20000 >> /var/opt/sourcegraph/jaeger.log 2>&1`
)

func maybeMonitoring() []string {
	if os.Getenv("DISABLE_OBSERVABILITY") != "" {
		log15.Info("WARNING: Running with monitoring disabled")
		return []string{}
	}
	return []string{
		prometheusProcLine,
		grafanaProcLine,
		jaegerProcLine,
	}
}
