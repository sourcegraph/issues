package graphqlbackend

import (
	"context"
	"fmt"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"strings"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/db"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/globals"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/conf"
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
	IsAuthenticated bool // whether the viewer is authenticated
	IsSiteAdmin     bool // whether the viewer is a site admin
}

func (r *siteResolver) Alerts(ctx context.Context) ([]*Alert, error) {
	args := AlertFuncArgs{
		IsAuthenticated: actor.FromContext(ctx).IsAuthenticated(),
		IsSiteAdmin:     backend.CheckCurrentUserIsSiteAdmin(ctx) == nil,
	}

	var alerts []*Alert
	for _, f := range AlertFuncs {
		alerts = append(alerts, f(args)...)
	}
	return alerts, nil
}

func checkDuplicateRateLimits() (problems conf.Problems) {
	externalServices, err := db.ExternalServices.List(context.Background(), db.ExternalServicesListOptions{})
	if err != nil {
		problems = append(problems, conf.NewExternalServiceProblem(fmt.Sprintf("Could not load external services: %v", err)))
		return problems
	}

	common := make([]extsvc.Common, len(externalServices))
	for i := range externalServices {
		common[i].Config = externalServices[i].Config
		common[i].Kind = externalServices[i].Kind
		common[i].DisplayName = externalServices[i].DisplayName
	}

	rateLimits, err := extsvc.RateLimits(common)
	if err != nil {
		problems = append(problems, conf.NewExternalServiceProblem(fmt.Sprintf("Could not get rate limit config: %v", err)))
		return problems
	}

	// BaseURL -> DisplayName
	byURL := make(map[string][]string)
	// Warn if more than one service for the same code host has a non default rate limiter set
	for _, r := range rateLimits {
		if r.IsDefault {
			continue
		}
		byURL[r.BaseURL] = append(byURL[r.BaseURL], r.DisplayName)
	}

	for _, duplicates := range byURL {
		msg := fmt.Sprintf("Multiple rate limiters configured for the same code host: %s", strings.Join(duplicates, ", "))
		problems = append(problems, conf.NewExternalServiceProblem(msg))
	}

	return problems
}

func init() {
	conf.ContributeWarning(func(c conf.Unified) (problems conf.Problems) {
		if c.ExternalURL == "" {
			problems = append(problems, conf.NewSiteProblem("`externalURL` is required to be set for many features of Sourcegraph to work correctly."))
		} else if conf.DeployType() != conf.DeployDev && strings.HasPrefix(c.ExternalURL, "http://") {
			problems = append(problems, conf.NewSiteProblem("Your connection is not private. We recommend [configuring Sourcegraph to use HTTPS/SSL](https://docs.sourcegraph.com/admin/nginx)"))
		}

		problems = append(problems, checkDuplicateRateLimits()...)

		return problems
	})

	// Warn about invalid site configuration.
	AlertFuncs = append(AlertFuncs, func(args AlertFuncArgs) []*Alert {
		// 🚨 SECURITY: Only the site admin cares about this. Leaking a boolean wouldn't be a
		// security vulnerability, but just in case this method is changed to return more
		// information, let's lock it down.
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
					MessageValue: `Update [**critical configuration**](/help/admin/management_console) to resolve problems: ` + err.Error(),
				},
			}
		}
		problems = append(problems, warnings...)

		if len(problems) == 0 {
			return nil
		}

		alerts := make([]*Alert, 0, 2)

		criticalProblems := problems.Critical()
		if len(criticalProblems) > 0 {
			alerts = append(alerts, &Alert{
				TypeValue: AlertTypeWarning,
				MessageValue: `[**Update critical configuration**](/help/admin/management_console) to resolve problems:` +
					"\n* " + strings.Join(criticalProblems.Messages(), "\n* "),
			})
		}

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
