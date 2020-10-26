package main

import (
	"fmt"
	"net/url"
	"strings"

	amconfig "github.com/prometheus/alertmanager/config"
	commoncfg "github.com/prometheus/common/config"
	"github.com/sourcegraph/sourcegraph/schema"
)

const (
	alertmanagerNoopReceiver     = "src-noop-receiver"
	alertmanagerWarningReceiver  = "src-warning-receiver"
	alertmanagerCriticalReceiver = "src-critical-receiver"
)

const (
	colorWarning  = "#FFFF00" // yellow
	colorCritical = "#FF0000" // red
	colorGood     = "#00FF00" // green
)

// commonLabels defines the set of labels we group alerts by, such that each alert falls in a unique group.
// These labels are available in Alertmanager templates as fields of `.CommonLabels`.
//
// Note that `alertname` is provided as a fallback grouping only - combinations of the other labels should be unique
// for alerts provided by the Sourcegraph generator.
//
// When changing this, make sure to update the webhook body documentation in /doc/admin/observability/alerting.md
var commonLabels = []string{"alertname", "level", "service_name", "name", "owner", "description"}

// Static alertmanager templates
var (
	// Alertmanager notification template reference: https://prometheus.io/docs/alerting/latest/notifications
	// All labels used in these templates should be included in route.GroupByStr
	alertSolutionsURLTemplate = `https://docs.sourcegraph.com/admin/observability/alert_solutions#{{ .CommonLabels.service_name }}-{{ .CommonLabels.name | reReplaceAll "(_low|_high)$" "" | reReplaceAll "_" "-" }}`

	// Title templates
	firingTitleTemplate       = "[{{ .CommonLabels.level | toUpper }}] {{ .CommonLabels.description }}"
	resolvedTitleTemplate     = "[RESOLVED] {{ .CommonLabels.description }}"
	notificationTitleTemplate = fmt.Sprintf(`{{ if eq .Status "firing" }}%s{{ else }}%s{{ end }}`, firingTitleTemplate, resolvedTitleTemplate)
)

// newRoutesAndReceivers converts the given alerts from Sourcegraph site configuration into Alertmanager receivers
// and routes. Each alert level has a receiver, which has configuration for all channels for that level. Additional
// routes can route alerts based on `alerts.on`, but all alerts still fall through to the per-level receivers.
func newRoutesAndReceivers(newAlerts []*schema.ObservabilityAlerts, externalURL string, newProblem func(error)) ([]*amconfig.Receiver, []*amconfig.Route) {
	var (
		warningReceiver     = &amconfig.Receiver{Name: alertmanagerWarningReceiver}
		criticalReceiver    = &amconfig.Receiver{Name: alertmanagerCriticalReceiver}
		additionalReceivers []*amconfig.Receiver
		additionalRoutes    []*amconfig.Route
	)

	// Parameterized alertmanager templates
	var (
		dashboardURLTemplate = strings.TrimSuffix(externalURL, "/") + `/-/debug/grafana/d/{{ .CommonLabels.service_name }}/{{ .CommonLabels.service_name }}`

		// messages for different states
		firingBodyTemplate          = `{{ .CommonLabels.level | title }} alert '{{ .CommonLabels.name }}' is firing for service '{{ .CommonLabels.service_name }}' ({{ .CommonLabels.owner }}).`
		firingBodyTemplateWithLinks = fmt.Sprintf(`%s

For possible solutions, please refer to our documentation: %s
For more details, please refer to the service dashboard: %s`, firingBodyTemplate, alertSolutionsURLTemplate, dashboardURLTemplate)
		resolvedBodyTemplate = `{{ .CommonLabels.level | title }} alert '{{ .CommonLabels.name }}' for service '{{ .CommonLabels.service_name }}' has resolved.`

		// use for notifiers that provide fields for links
		notificationBodyTemplateWithoutLinks = fmt.Sprintf(`{{ if eq .Status "firing" }}%s{{ else }}%s{{ end }}`, firingBodyTemplate, resolvedBodyTemplate)
		// use for notifiers that don't provide fields for links
		notificationBodyTemplateWithLinks = fmt.Sprintf(`{{ if eq .Status "firing" }}%s{{ else }}%s{{ end }}`, firingBodyTemplateWithLinks, resolvedBodyTemplate)
	)

	for i, alert := range newAlerts {
		var receiver *amconfig.Receiver
		var activeColor string
		if alert.Level == "critical" {
			receiver = criticalReceiver
			activeColor = colorCritical
		} else {
			receiver = warningReceiver
			activeColor = colorWarning
		}
		colorTemplate := fmt.Sprintf(`{{ if eq .Status "firing" }}%s{{ else }}%s{{ end }}`, activeColor, colorGood)

		// generate a new receiver and route for alerts with 'Owners'
		if len(alert.Owners) > 0 {
			owners := strings.Join(alert.Owners, "|")
			ownerRegexp, err := amconfig.NewRegexp(fmt.Sprintf("^(%s)$", owners))
			if err != nil {
				newProblem(fmt.Errorf("failed to apply alert %d: %w", i, err))
				continue
			}

			receiverName := fmt.Sprintf("src-%s-on-%s", alert.Level, owners)
			receiver = &amconfig.Receiver{Name: receiverName}
			additionalReceivers = append(additionalReceivers, receiver)
			additionalRoutes = append(additionalRoutes, &amconfig.Route{
				Receiver: receiverName,
				Match: map[string]string{
					"level": alert.Level,
				},
				MatchRE: amconfig.MatchRegexps{
					"owner": *ownerRegexp,
				},
				// Generated routes are set up as siblings. Generally, Alertmanager
				// matches on exactly one route, but for additionalRoutes we don't
				// want to prevent other routes from getting this alert, so we configure
				// this route with 'continue: true'
				//
				// Also see https://prometheus.io/docs/alerting/latest/configuration/#route
				Continue: true,
			})
		}

		notifierConfig := amconfig.NotifierConfig{
			VSendResolved: !alert.DisableSendResolved,
		}
		notifier := alert.Notifier
		switch {
		// https://prometheus.io/docs/alerting/latest/configuration/#email_config
		case notifier.Email != nil:
			receiver.EmailConfigs = append(receiver.EmailConfigs, &amconfig.EmailConfig{
				To: notifier.Email.Address,

				Headers: map[string]string{
					"subject": notificationTitleTemplate,
				},
				HTML: fmt.Sprintf(`<body>%s</body>`, notificationBodyTemplateWithLinks),
				Text: notificationBodyTemplateWithLinks,

				// SMTP configuration is applied globally by changeSMTP

				NotifierConfig: notifierConfig,
			})

		// https://prometheus.io/docs/alerting/latest/configuration/#opsgenie_config
		case notifier.Opsgenie != nil:
			var apiURL *amconfig.URL
			if notifier.Opsgenie.ApiUrl != "" {
				u, err := url.Parse(notifier.Opsgenie.ApiUrl)
				if err != nil {
					newProblem(fmt.Errorf("failed to apply notifier %d: %w", i, err))
					continue
				}
				apiURL = &amconfig.URL{URL: u}
			}
			responders := make([]amconfig.OpsGenieConfigResponder, len(notifier.Opsgenie.Responders))
			for i, resp := range notifier.Opsgenie.Responders {
				responders[i] = amconfig.OpsGenieConfigResponder{
					Type:     resp.Type,
					ID:       resp.Id,
					Name:     resp.Name,
					Username: resp.Username,
				}
			}
			receiver.OpsGenieConfigs = append(receiver.OpsGenieConfigs, &amconfig.OpsGenieConfig{
				APIKey: amconfig.Secret(notifier.Opsgenie.ApiKey),
				APIURL: apiURL,

				Message:     notificationTitleTemplate,
				Description: notificationBodyTemplateWithoutLinks,
				Priority:    notifier.Opsgenie.Priority,
				Responders:  responders,
				Source:      dashboardURLTemplate,
				Details: map[string]string{
					"Solutions": alertSolutionsURLTemplate,
				},

				NotifierConfig: notifierConfig,
			})

		// https://prometheus.io/docs/alerting/latest/configuration/#pagerduty_config
		case notifier.Pagerduty != nil:
			var apiURL *amconfig.URL
			if notifier.Pagerduty.ApiUrl != "" {
				u, err := url.Parse(notifier.Pagerduty.ApiUrl)
				if err != nil {
					newProblem(fmt.Errorf("failed to apply notifier %d: %w", i, err))
					continue
				}
				apiURL = &amconfig.URL{URL: u}
			}
			receiver.PagerdutyConfigs = append(receiver.PagerdutyConfigs, &amconfig.PagerdutyConfig{
				RoutingKey: amconfig.Secret(notifier.Pagerduty.IntegrationKey),
				Severity:   notifier.Pagerduty.Severity,
				URL:        apiURL,

				Description: notificationTitleTemplate,
				Links: []amconfig.PagerdutyLink{{
					Text: "Solutions",
					Href: alertSolutionsURLTemplate,
				}, {
					Text: "Dashboard",
					Href: dashboardURLTemplate,
				}},

				NotifierConfig: notifierConfig,
			})

		// https://prometheus.io/docs/alerting/latest/configuration/#slack_config
		case notifier.Slack != nil:
			u, err := url.Parse(notifier.Slack.Url)
			if err != nil {
				newProblem(fmt.Errorf("failed to apply notifier %d: %w", i, err))
				continue
			}

			// set a default username if none is provided
			if notifier.Slack.Username == "" {
				notifier.Slack.Username = "Sourcegraph Alerts"
			}

			receiver.SlackConfigs = append(receiver.SlackConfigs, &amconfig.SlackConfig{
				APIURL:    &amconfig.SecretURL{URL: u},
				Username:  notifier.Slack.Username,
				Channel:   notifier.Slack.Recipient,
				IconEmoji: notifier.Slack.Icon_emoji,
				IconURL:   notifier.Slack.Icon_url,

				Title:     notificationTitleTemplate,
				TitleLink: alertSolutionsURLTemplate,

				Text: notificationBodyTemplateWithoutLinks,
				Actions: []*amconfig.SlackAction{{
					Text: "Solutions",
					Type: "button",
					URL:  alertSolutionsURLTemplate,
				}, {
					Text: "Dashboard",
					Type: "button",
					URL:  dashboardURLTemplate,
				}},
				Color: colorTemplate,

				NotifierConfig: notifierConfig,
			})

		// https://prometheus.io/docs/alerting/latest/configuration/#webhook_config
		case notifier.Webhook != nil:
			u, err := url.Parse(notifier.Webhook.Url)
			if err != nil {
				newProblem(fmt.Errorf("failed to apply notifier %d: %w", i, err))
				continue
			}
			receiver.WebhookConfigs = append(receiver.WebhookConfigs, &amconfig.WebhookConfig{
				URL: &amconfig.URL{URL: u},
				HTTPConfig: &commoncfg.HTTPClientConfig{
					BasicAuth: &commoncfg.BasicAuth{
						Username: notifier.Webhook.Username,
						Password: commoncfg.Secret(notifier.Webhook.Password),
					},
					BearerToken: commoncfg.Secret(notifier.Webhook.BearerToken),
				},

				NotifierConfig: notifierConfig,
			})

		// define new notifiers to support in site.schema.json
		default:
			newProblem(fmt.Errorf("failed to apply notifier %d: no configuration found", i))
		}
	}

	return append(additionalReceivers, warningReceiver, criticalReceiver),
		append(additionalRoutes, &amconfig.Route{
			Receiver: alertmanagerWarningReceiver,
			Match: map[string]string{
				"level": "warning",
			},
		}, &amconfig.Route{
			Receiver: alertmanagerCriticalReceiver,
			Match: map[string]string{
				"level": "critical",
			},
		})
}
