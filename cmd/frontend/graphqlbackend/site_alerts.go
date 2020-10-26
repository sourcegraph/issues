package graphqlbackend

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/globals"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/app/updatecheck"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/env"
	"github.com/sourcegraph/sourcegraph/internal/prometheusutil"
	"github.com/sourcegraph/sourcegraph/internal/version"
	"github.com/sourcegraph/sourcegraph/schema"
)

// Alert implements the GraphQL type Alert.
type Alert struct {
	TypeValue                 string
	MessageValue              string
	IsDismissibleWithKeyValue string
}

func (r *Alert) Type() string    { return r.TypeValue }
func (r *Alert) Message() string { return r.MessageValue }
func (r *Alert) IsDismissibleWithKey() *string {
	if r.IsDismissibleWithKeyValue == "" {
		return nil
	}
	return &r.IsDismissibleWithKeyValue
}

// Constants for the GraphQL enum AlertType.
const (
	AlertTypeInfo    = "INFO"
	AlertTypeWarning = "WARNING"
	AlertTypeError   = "ERROR"
)

// AlertFuncs is a list of functions called to populate the GraphQL Site.alerts value. It may be
// appended to at init time.
//
// The functions are called each time the Site.alerts value is queried, so they must not block.
var AlertFuncs []func(AlertFuncArgs) []*Alert

// AlertFuncArgs are the arguments provided to functions in AlertFuncs used to populate the GraphQL
// Site.alerts value. They allow the functions to customize the returned alerts based on the
// identity of the viewer (without needing to query for that on their own, which would be slow).
type AlertFuncArgs struct {
	IsAuthenticated     bool             // whether the viewer is authenticated
	IsSiteAdmin         bool             // whether the viewer is a site admin
	ViewerFinalSettings *schema.Settings // the viewer's final user/org/global settings
}

func (r *siteResolver) Alerts(ctx context.Context) ([]*Alert, error) {
	settings, err := decodedViewerFinalSettings(ctx)
	if err != nil {
		return nil, err
	}

	args := AlertFuncArgs{
		IsAuthenticated:     actor.FromContext(ctx).IsAuthenticated(),
		IsSiteAdmin:         backend.CheckCurrentUserIsSiteAdmin(ctx) == nil,
		ViewerFinalSettings: settings,
	}

	var alerts []*Alert
	for _, f := range AlertFuncs {
		alerts = append(alerts, f(args)...)
	}
	return alerts, nil
}

// Intentionally named "DISABLE_SECURITY" and not something else, so that anyone considering
// disabling this thinks twice about the risks associated with disabling these and considers
// keeping up-to-date more frequently instead.
var disableSecurity, _ = strconv.ParseBool(env.Get("DISABLE_SECURITY", "false", "disables security upgrade notices"))

func init() {
	conf.ContributeWarning(func(c conf.Unified) (problems conf.Problems) {
		if c.ExternalURL == "" {
			problems = append(problems, conf.NewSiteProblem("`externalURL` is required to be set for many features of Sourcegraph to work correctly."))
		} else if conf.DeployType() != conf.DeployDev && strings.HasPrefix(c.ExternalURL, "http://") {
			problems = append(problems, conf.NewSiteProblem("Your connection is not private. We recommend [configuring Sourcegraph to use HTTPS/SSL](https://docs.sourcegraph.com/admin/nginx)"))
		}

		return problems
	})

	if !disableSecurity {
		// Warn about Sourcegraph being out of date.
		AlertFuncs = append(AlertFuncs, outOfDateAlert)
	} else {
		log15.Warn("WARNING: SECURITY NOTICES DISABLED: this is not recommended, please unset DISABLE_SECURITY=true")
	}

	// Notify when updates are available, if the instance can access the public internet.
	AlertFuncs = append(AlertFuncs, updateAvailableAlert)

	// Notify admins if critical alerts are firing.
	AlertFuncs = append(AlertFuncs, observabilityActiveAlertsAlert(prometheusutil.PrometheusURL))

	// Warn about invalid site configuration.
	AlertFuncs = append(AlertFuncs, func(args AlertFuncArgs) []*Alert {
		// 🚨 SECURITY: Only the site admin should care about the site configuration being invalid, as they
		// are the only one who can take action on that. Additionally, it may be unsafe to expose information
		// about the problems with the configuration (e.g. if the error message contains sensitive information).
		if !args.IsSiteAdmin {
			return nil
		}

		problems, err := conf.Validate(globals.ConfigurationServerFrontendOnly.Raw())
		if err != nil {
			return []*Alert{
				{
					TypeValue:    AlertTypeError,
					MessageValue: `Update [**site configuration**](/site-admin/configuration) to resolve problems: ` + err.Error(),
				},
			}
		}

		warnings, err := conf.GetWarnings()
		if err != nil {
			return []*Alert{
				{
					TypeValue:    AlertTypeError,
					MessageValue: `Update [**site configuration**](/site-admin/configuration) to resolve problems: ` + err.Error(),
				},
			}
		}
		problems = append(problems, warnings...)

		if len(problems) == 0 {
			return nil
		}
		alerts := make([]*Alert, 0, 2)

		siteProblems := problems.Site()
		if len(siteProblems) > 0 {
			alerts = append(alerts, &Alert{
				TypeValue: AlertTypeWarning,
				MessageValue: `[**Update site configuration**](/site-admin/configuration) to resolve problems:` +
					"\n* " + strings.Join(siteProblems.Messages(), "\n* "),
			})
		}

		externalServiceProblems := problems.ExternalService()
		if len(externalServiceProblems) > 0 {
			alerts = append(alerts, &Alert{
				TypeValue: AlertTypeWarning,
				MessageValue: `[**Update external service configuration**](/site-admin/external-services) to resolve problems:` +
					"\n* " + strings.Join(externalServiceProblems.Messages(), "\n* "),
			})
		}
		return alerts
	})
}

func updateAvailableAlert(args AlertFuncArgs) []*Alert {
	// We only show update alerts to admins. This is not for security reasons, as we already
	// expose the version number of the instance to all users via the user settings page.
	if !args.IsSiteAdmin {
		return nil
	}

	globalUpdateStatus := updatecheck.Last()
	if globalUpdateStatus == nil || updatecheck.IsPending() || !globalUpdateStatus.HasUpdate() || globalUpdateStatus.Err != nil {
		return nil
	}
	// ensure the user has opted in to receiving notifications for minor updates and there is one available
	if !args.ViewerFinalSettings.AlertsShowPatchUpdates && !isMinorUpdateAvailable(version.Version(), globalUpdateStatus.UpdateVersion) {
		return nil
	}
	message := fmt.Sprintf("An update is available: [Sourcegraph v%s](https://about.sourcegraph.com/blog) - [changelog](https://about.sourcegraph.com/changelog)", globalUpdateStatus.UpdateVersion)

	// dismission key includes the version so after it is dismissed the alert comes back for the next update.
	key := "update-available-" + globalUpdateStatus.UpdateVersion
	return []*Alert{{TypeValue: AlertTypeInfo, MessageValue: message, IsDismissibleWithKeyValue: key}}
}

// isMinorUpdateAvailable tells if upgrading from the current version to the specified upgrade
// candidate would be a major/minor update and NOT a patch update.
func isMinorUpdateAvailable(currentVersion, updateVersion string) bool {
	// If either current or update versions aren't semvers (e.g., a user is on a date-based build version, or "dev"),
	// always return true and allow any alerts to be shown. This has the effect of simply deferring to the response
	// from Sourcegraph.com about whether an update alert is needed.
	cv, err := semver.NewVersion(currentVersion)
	if err != nil {
		return true
	}
	uv, err := semver.NewVersion(updateVersion)
	if err != nil {
		return true
	}
	return cv.Major() != uv.Major() || cv.Minor() != uv.Minor()
}

func outOfDateAlert(args AlertFuncArgs) []*Alert {
	globalUpdateStatus := updatecheck.Last()
	if globalUpdateStatus == nil || updatecheck.IsPending() {
		return nil
	}
	offline := globalUpdateStatus.Err != nil // Whether or not instance can connect to Sourcegraph.com for update checks
	now := time.Now()
	monthsOutOfDate, err := version.HowLongOutOfDate(now)
	if err != nil {
		log15.Error("failed to determine how out of date Sourcegraph is", "error", err)
		return nil
	}
	alert := determineOutOfDateAlert(args.IsSiteAdmin, monthsOutOfDate, offline)
	if alert == nil {
		return nil
	}
	return []*Alert{alert}
}

func determineOutOfDateAlert(isAdmin bool, months int, offline bool) *Alert {
	if months <= 0 {
		return nil
	}
	// online instances will still be prompt site admins to upgrade via site_update_check
	if months < 3 && !offline {
		return nil
	}

	if isAdmin {
		key := fmt.Sprintf("months-out-of-date-%d", months)
		switch {
		case months < 3:
			message := fmt.Sprintf("Sourcegraph is %d+ months out of date, for the latest features and bug fixes please upgrade ([changelog](http://about.sourcegraph.com/changelog))", months)
			return &Alert{TypeValue: AlertTypeInfo, MessageValue: message, IsDismissibleWithKeyValue: key}
		case months == 3:
			message := "Sourcegraph is 3+ months out of date, you may be missing important security or bug fixes. Users will be notified at 4+ months. ([changelog](http://about.sourcegraph.com/changelog))"
			return &Alert{TypeValue: AlertTypeWarning, MessageValue: message}
		case months == 4:
			message := "Sourcegraph is 4+ months out of date, you may be missing important security or bug fixes. A notice is shown to users. ([changelog](http://about.sourcegraph.com/changelog))"
			return &Alert{TypeValue: AlertTypeWarning, MessageValue: message}
		case months == 5:
			message := "Sourcegraph is 5+ months out of date, you may be missing important security or bug fixes. A notice is shown to users. ([changelog](http://about.sourcegraph.com/changelog))"
			return &Alert{TypeValue: AlertTypeError, MessageValue: message}
		default:
			message := fmt.Sprintf("Sourcegraph is %d+ months out of date, you may be missing important security or bug fixes. A notice is shown to users. ([changelog](http://about.sourcegraph.com/changelog))", months)
			return &Alert{TypeValue: AlertTypeError, MessageValue: message}
		}
	}

	key := fmt.Sprintf("months-out-of-date-%d", months)
	switch months {
	case 0, 1, 2, 3:
		return nil
	case 4, 5:
		message := fmt.Sprintf("Sourcegraph is %d+ months out of date, ask your site administrator to upgrade for the latest features and bug fixes. ([changelog](http://about.sourcegraph.com/changelog))", months)
		return &Alert{TypeValue: AlertTypeWarning, MessageValue: message, IsDismissibleWithKeyValue: key}
	default:
		alertType := AlertTypeWarning
		if months > 12 {
			alertType = AlertTypeError
		}
		message := fmt.Sprintf("Sourcegraph is %d+ months out of date, you may be missing important security or bug fixes. Ask your site administrator to upgrade. ([changelog](http://about.sourcegraph.com/changelog))", months)
		return &Alert{TypeValue: alertType, MessageValue: message, IsDismissibleWithKeyValue: key}
	}
}

// observabilityActiveAlertsAlert directs admins to check Grafana if critical alerts are firing
func observabilityActiveAlertsAlert(prometheusURL string) func(AlertFuncArgs) []*Alert {
	return func(args AlertFuncArgs) []*Alert {
		// true by default - change settings.schema.json if this changes
		observabilitySiteAlertsDisabled := true
		if args.ViewerFinalSettings != nil && args.ViewerFinalSettings.AlertsHideObservabilitySiteAlerts != nil {
			observabilitySiteAlertsDisabled = *args.ViewerFinalSettings.AlertsHideObservabilitySiteAlerts
		}

		if !args.IsSiteAdmin || prometheusURL == "" || observabilitySiteAlertsDisabled {
			return nil
		}

		// set up request to fetch status from prom-wrapper
		promURL, err := url.Parse(prometheusURL)
		if err != nil {
			return []*Alert{{TypeValue: AlertTypeWarning, MessageValue: fmt.Sprintf("Prometheus misconfigured: %s", err)}}
		}
		promURL.Path = "/prom-wrapper/alerts-status"
		req, err := http.NewRequest("GET", promURL.String(), nil)
		if err != nil {
			return []*Alert{{TypeValue: AlertTypeWarning, MessageValue: fmt.Sprintf("Prometheus misconfigured: %s", err)}}
		}

		// use a short timeout to avoid having this block problems from loading
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		resp, err := http.DefaultClient.Do(req.WithContext(ctx))
		if err != nil {
			return []*Alert{{TypeValue: AlertTypeWarning, MessageValue: fmt.Sprintf("Unable to fetch alerts status: %s", err)}}
		}
		if resp.StatusCode != 200 {
			return []*Alert{{TypeValue: AlertTypeWarning, MessageValue: fmt.Sprintf("Unable to fetch alerts status: status %d", resp.StatusCode)}}
		}

		var alertsStatus map[string]int
		defer resp.Body.Close()
		if err := json.NewDecoder(resp.Body).Decode(&alertsStatus); err != nil {
			return []*Alert{{TypeValue: AlertTypeWarning, MessageValue: err.Error()}}
		}
		criticalAlerts := alertsStatus["critical"]
		servicesCritical := alertsStatus["services_critical"]
		if criticalAlerts == 0 {
			return nil
		}
		msg := fmt.Sprintf("%s across %s currently firing - [view alerts](/-/debug/grafana)",
			pluralize(criticalAlerts, "critical alert", "critical alerts"),
			pluralize(servicesCritical, "service", "services"))
		return []*Alert{{TypeValue: AlertTypeError, MessageValue: msg}}
	}
}

func pluralize(v int, singular, plural string) string {
	if v == 1 {
		return fmt.Sprintf("%d %s", v, singular)
	}
	return fmt.Sprintf("%d %s", v, plural)
}
