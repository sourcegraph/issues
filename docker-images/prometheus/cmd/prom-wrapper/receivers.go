package main

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/cockroachdb/errors"
	amconfig "github.com/prometheus/alertmanager/config"
	commoncfg "github.com/prometheus/common/config"

	"github.com/sourcegraph/sourcegraph/internal/version"
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

const docsURL = "https://docs.sourcegraph.com"
const alertSolutionsPagePath = "admin/observability/alert_solutions"

// alertSolutionsURL generates a link to the alert solutions page that embeds the appropriate version
// if it is available and it is a semantic version.
func alertSolutionsURL() string {
	maybeSemver := "v" + version.Version()
	_, semverErr := semver.NewVersion(maybeSemver)
	if semverErr == nil && !version.IsDev(version.Version()) {
		return fmt.Sprintf("%s/@%s/%s", docsURL, maybeSemver, alertSolutionsPagePath)
	}
	return fmt.Sprintf("%s/%s", docsURL, alertSolutionsPagePath)
}

// commonLabels defines the set of labels we group alerts by, such that each alert falls in a unique group.
// These labels are available in Alertmanager templates as fields of `.CommonLabels`.
//
// Note that `alertname` is provided as a fallback grouping only - combinations of the other labels should be unique
// for alerts provided by the Sourcegraph generator.
//
// When changing this, make sure to update the webhook body documentation in /doc/admin/observability/alerting.md
var commonLabels = []string{"alertname", "level", "service_name", "name", "owner", "description"}

// Static alertmanager templates. Templating reference: https://prometheus.io/docs/alerting/latest/notifications
//
// All `.CommonLabels` labels used in these templates should be included in `route.GroupByStr` in order for them to be available.
var (
	// observableDocAnchorTemplate must match anchors generated in `monitoring/monitoring/documentation.go`.
	observableDocAnchorTemplate = `{{ .CommonLabels.service_name }}-{{ .CommonLabels.name | reReplaceAll "_" "-" }}`
	alertSolutionsURLTemplate   = fmt.Sprintf(`%s#%s`, alertSolutionsURL(), observableDocAnchorTemplate)

	// Title templates
	firingTitleTemplate       = "[{{ .CommonLabels.level | toUpper }}] {{ .CommonLabels.description }}"
	resolvedTitleTemplate     = "[RESOLVED] {{ .CommonLabels.description }}"
	notificationTitleTemplate = fmt.Sprintf(`{{ if eq .Status "firing" }}%s{{ else }}%s{{ end }}`, firingTitleTemplate, resolvedTitleTemplate)

	tagsTemplateDefault = "{{ range $key, $value := .CommonLabels }}{{$key}}={{$value}},{{end}}"
)

// newRoutesAndReceivers converts the given alerts from Sourcegraph site configuration into Alertmanager receivers
// and routes with the following strategy:
//
// * Each alert level has a receiver, which has configuration for all channels for that level.
// * Each alert level and owner combination has a receiver and route, which has configuration for all channels for that filter.
// * Additional routes can route alerts based on `alerts.on`, but all alerts still fall through to the per-level receivers.
func newRoutesAndReceivers(newAlerts []*schema.ObservabilityAlerts, externalURL string, newProblem func(error)) ([]*amconfig.Receiver, []*amconfig.Route) {
	// Receivers must be uniquely named. They route
	var (
		warningReceiver     = &amconfig.Receiver{Name: alertmanagerWarningReceiver}
		criticalReceiver    = &amconfig.Receiver{Name: alertmanagerCriticalReceiver}
		additionalReceivers = map[string]*amconfig.Receiver{
			// stub receiver, for routes that do not have a configured receiver
			alertmanagerNoopReceiver: {
				Name: alertmanagerNoopReceiver,
			},
		}
	)

	// Routes
	var (
		defaultRoutes = []*amconfig.Route{
			{
				Receiver: alertmanagerWarningReceiver,
				Match: map[string]string{
					"level": "warning",
				},
			}, {
				Receiver: alertmanagerCriticalReceiver,
				Match: map[string]string{
					"level": "critical",
				},
			},
		}
		additionalRoutes []*amconfig.Route
	)

	// Parameterized alertmanager templates
	var (
		// link to grafana dashboard, based on external URL configuration and alert labels
		dashboardURLTemplate = strings.TrimSuffix(externalURL, "/") + `/-/debug/grafana/d/` +
			// link to service dashboard
			`{{ .CommonLabels.service_name }}/{{ .CommonLabels.service_name }}` +
			// link directly to the relevant panel
			"?viewPanel={{ .CommonLabels.grafana_panel_id }}" +
			// link to a time frame relevant to the alert.
			// we add 000 to adapt prometheus unix to grafana milliseconds for URL parameters.
			// this template is weird due to lack of Alertmanager functionality: https://github.com/prometheus/alertmanager/issues/1188
			"{{ $start := (index .Alerts 0).StartsAt.Unix }}{{ $end := (index .Alerts 0).EndsAt.Unix }}" + // start var decls
			"{{ if gt $end 0 }}&from={{ $start }}000&end={{ $end }}000" + // if $end is valid, link to start and end
			"{{ else }}&time={{ $start }}000&time.window=3600000{{ end }}" // if $end is invalid, link to start and window of 1 hour

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

	// Convert site configuration alerts to Alertmanager configuration
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

		// Generate receiver and route for alerts with 'Owners'
		if len(alert.Owners) > 0 {
			owners := strings.Join(alert.Owners, "|")
			ownerRegexp, err := amconfig.NewRegexp(fmt.Sprintf("^(%s)$", owners))
			if err != nil {
				newProblem(errors.Errorf("failed to apply alert %d: %w", i, err))
				continue
			}

			receiverName := fmt.Sprintf("src-%s-on-%s", alert.Level, owners)
			if r, exists := additionalReceivers[receiverName]; exists {
				receiver = r
			} else {
				receiver = &amconfig.Receiver{Name: receiverName}
				additionalReceivers[receiverName] = receiver
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
					newProblem(errors.Errorf("failed to apply notifier %d: %w", i, err))
					continue
				}
				apiURL = &amconfig.URL{URL: u}
			}

			var apiKEY amconfig.Secret
			if notifier.Opsgenie.ApiKey != "" {
				apiKEY = amconfig.Secret(notifier.Opsgenie.ApiKey)
			} else {
				apiKEY = amconfig.Secret(opsGenieAPIKey)
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

			var priority string

			switch alert.Level {
			case "critical":
				priority = "P1"
			case "warning":
				priority = "P2"
			case "info":
				priority = "P3"
			default:
				priority = "P4"
			}

			if notifier.Opsgenie.Priority != "" {
				priority = notifier.Opsgenie.Priority
			}

			tags := tagsTemplateDefault
			if notifier.Opsgenie.Tags != "" {
				tags = notifier.Opsgenie.Tags
			}

			receiver.OpsGenieConfigs = append(receiver.OpsGenieConfigs, &amconfig.OpsGenieConfig{
				APIKey: apiKEY,
				APIURL: apiURL,

				Message:     notificationTitleTemplate,
				Description: notificationBodyTemplateWithoutLinks,
				Priority:    priority,
				Tags:        tags,
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
					newProblem(errors.Errorf("failed to apply notifier %d: %w", i, err))
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
				newProblem(errors.Errorf("failed to apply notifier %d: %w", i, err))
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
				newProblem(errors.Errorf("failed to apply notifier %d: %w", i, err))
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
			newProblem(errors.Errorf("failed to apply notifier %d: no configuration found", i))
		}
	}

	var additionalReceiversSlice []*amconfig.Receiver
	for _, r := range additionalReceivers {
		additionalReceiversSlice = append(additionalReceiversSlice, r)
	}
	return append(additionalReceiversSlice, warningReceiver, criticalReceiver),
		append(additionalRoutes, defaultRoutes...)
}

// newRootRoute generates a base Route required by Alertmanager to wrap all routes
func newRootRoute(routes []*amconfig.Route) *amconfig.Route {
	return &amconfig.Route{
		GroupByStr: commonLabels,

		// How long to initially wait to send a notification for a group - each group matches exactly one alert, so fire immediately
		GroupWait: duration(1 * time.Second),

		// How long to wait before sending a notification about new alerts that are added to a group of alerts - in this case,
		// equivalent to how long to wait until notifying about an alert re-firing
		GroupInterval:  duration(1 * time.Minute),
		RepeatInterval: duration(7 * 24 * time.Hour),

		// Route alerts to notifications
		Routes: routes,

		// Fallback to do nothing for alerts not compatible with our receivers
		Receiver: alertmanagerNoopReceiver,
	}
}
