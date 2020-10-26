package main

import (
	"encoding/json"
	"net/http"

	"github.com/inconshreveable/log15"
	amclient "github.com/prometheus/alertmanager/api/v2/client"
	"github.com/prometheus/alertmanager/api/v2/client/alert"
)

// AlertsStatusReporter summarizes alert activity from Alertmanager
type AlertsStatusReporter struct {
	log          log15.Logger
	alertmanager *amclient.Alertmanager
}

func NewAlertsStatusReporter(logger log15.Logger, alertmanager *amclient.Alertmanager) *AlertsStatusReporter {
	return &AlertsStatusReporter{
		log:          logger.New("logger", "alerts-status"),
		alertmanager: alertmanager,
	}
}

func (s *AlertsStatusReporter) Handler() http.Handler {
	handler := http.NewServeMux()
	handler.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		t := true
		f := false
		results, err := s.alertmanager.Alert.GetAlerts(&alert.GetAlertsParams{
			Active:    &t,
			Inhibited: &f,
			Context:   req.Context(),
		})
		if err != nil {
			w.WriteHeader(500)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		var criticalAlerts, warningAlerts, silencedAlerts int
		servicesWithCriticalAlerts := map[string]struct{}{}
		for _, a := range results.GetPayload() {
			if len(a.Status.SilencedBy) > 0 {
				silencedAlerts++
				continue
			}
			level := a.Labels["level"]
			switch level {
			case "warning":
				warningAlerts++
			case "critical":
				criticalAlerts++
				svc := a.Labels["service_name"]
				servicesWithCriticalAlerts[svc] = struct{}{}
			}
		}
		// summarize alerts status
		b, err := json.Marshal(map[string]int{
			"critical":          criticalAlerts,
			"services_critical": len(servicesWithCriticalAlerts),
			"warning":           warningAlerts,
			"silenced":          silencedAlerts,
		})
		if err != nil {
			w.WriteHeader(500)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		_, _ = w.Write(b)
	})
	return handler
}
