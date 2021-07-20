package monitoring

import (
	"fmt"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/prometheus/common/model"
)

const (
	alertRulesFileSuffix = "_alert_rules.yml"
)

// prometheusAlertName creates an alertname that is unique given the combination of parameters
func prometheusAlertName(level, service, name string) string {
	return fmt.Sprintf("%s_%s_%s", level, service, name)
}

// promRule is a subset of a Prometheus recording or alert rule definition.
type promRule struct {
	// either Record or Alert
	Record string `yaml:",omitempty"` // https://prometheus.io/docs/prometheus/latest/configuration/recording_rules/
	Alert  string `yaml:",omitempty"` // https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/

	Labels map[string]string
	Expr   string

	// for Alert only
	For *model.Duration `yaml:",omitempty"`
}

func (r *promRule) validate() error {
	if r.Record != "" && r.Alert != "" {
		return errors.Errorf("promRule cannot be both a record (%q) and an alert (%q)", r.Record, r.Alert)
	}
	if r.Alert == "" && r.For != nil {
		return errors.Errorf("promRule can only have a 'for' (%q) if it is an alert", r.For.String())
	}
	return nil
}

// promRulesFile represents a Prometheus recording rules file (which we use for defining our alerts)
// see:
//
// https://prometheus.io/docs/prometheus/latest/configuration/recording_rules/
//
type promRulesFile struct {
	Groups []promGroup
}

type promGroup struct {
	Name  string
	Rules []promRule
}

func (g *promGroup) validate() error {
	if g.Name == "" {
		return errors.New("promGroup requires name")
	}
	for _, r := range g.Rules {
		if err := r.validate(); err != nil {
			return errors.Errorf("promGroup has invalid rule: %w", err)
		}
	}
	return nil
}

func (g *promGroup) appendRow(alertQuery string, labels map[string]string, duration time.Duration) {
	labels["alert_type"] = "builtin" // indicate alert is generated
	var forDuration *model.Duration
	if duration > 0 {
		d := model.Duration(duration)
		forDuration = &d
	}

	alertName := prometheusAlertName(labels["level"], labels["service_name"], labels["name"])
	g.Rules = append(g.Rules,
		// Native prometheus alert, based on alertQuery which returns 0 if not firing or 1 if firing.
		promRule{
			Alert:  alertName,
			Labels: labels,
			Expr:   alertQuery,
			For:    forDuration,
		},
		// Record for generated alert, useful for indicating in Grafana dashboards if this alert
		// is defined at all. Prometheus's ALERTS metric does not track alerts with alertstate="inactive".
		//
		// Since ALERTS{alertname="value"} does not exist if the alert has never fired, we add set
		// the series to vector(0) instead.
		promRule{
			Record: "alert_count",
			Labels: labels,
			Expr:   fmt.Sprintf(`max(ALERTS{alertname=%q,alertstate="firing"} OR on() vector(0))`, alertName),
		})
}
