package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/inconshreveable/log15"
	amclient "github.com/prometheus/alertmanager/api/v2/client"
	"github.com/prometheus/alertmanager/api/v2/client/silence"
	"github.com/prometheus/alertmanager/api/v2/models"
	amconfig "github.com/prometheus/alertmanager/config"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/schema"
)

type ChangeContext struct {
	AMConfig *amconfig.Config // refer to https://prometheus.io/docs/alerting/latest/configuration/
	AMClient *amclient.Alertmanager
}

// ChangeResult indicates output from a Change
type ChangeResult struct {
	Problems conf.Problems
}

// Change implements a change to configuration
type Change func(ctx context.Context, log log15.Logger, change ChangeContext, newConfig *subscribedSiteConfig) (result ChangeResult)

// changeReceivers applies `observability.alerts` as Alertmanager receivers.
func changeReceivers(ctx context.Context, log log15.Logger, change ChangeContext, newConfig *subscribedSiteConfig) (result ChangeResult) {
	// convenience function for creating a prefixed problem - this reflects the relevant site configuration fields
	newProblem := func(err error) {
		result.Problems = append(result.Problems, conf.NewSiteProblem(fmt.Sprintf("`observability.alerts`: %v", err)))
	}

	// reset and generate new notifiers configuration
	receivers, routes := newRoutesAndReceivers(newConfig.Alerts, newConfig.ExternalURL, newProblem)
	change.AMConfig.Receivers = append(receivers, &amconfig.Receiver{
		// stub receiver
		Name: alertmanagerNoopReceiver,
	})
	change.AMConfig.Route = &amconfig.Route{
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

	return result
}

// changeSMTP applies SMTP server configuration.
func changeSMTP(ctx context.Context, log log15.Logger, change ChangeContext, newConfig *subscribedSiteConfig) (result ChangeResult) {
	if change.AMConfig.Global == nil {
		change.AMConfig.Global = &amconfig.GlobalConfig{}
	}

	email := newConfig.Email
	change.AMConfig.Global.SMTPFrom = email.Address

	// assign zero-values to AMConfig SMTP fields if email.SMTP is nil
	if email.SMTP == nil {
		email.SMTP = &schema.SMTPServerConfig{}
	}
	change.AMConfig.Global.SMTPHello = email.SMTP.Domain
	change.AMConfig.Global.SMTPSmarthost = amconfig.HostPort{
		Host: email.SMTP.Host,
		Port: strconv.Itoa(email.SMTP.Port),
	}
	change.AMConfig.Global.SMTPAuthUsername = email.SMTP.Username
	switch email.SMTP.Authentication {
	case "PLAIN":
		change.AMConfig.Global.SMTPAuthPassword = amconfig.Secret(email.SMTP.Password)
	case "CRAM-MD5":
		change.AMConfig.Global.SMTPAuthSecret = amconfig.Secret(email.SMTP.Password)
	}
	change.AMConfig.Global.SMTPRequireTLS = !email.SMTP.DisableTLS

	return
}

// changeSilences syncs Alertmanager silences with silences configured in observability.silenceAlerts
func changeSilences(ctx context.Context, log log15.Logger, change ChangeContext, newConfig *subscribedSiteConfig) (result ChangeResult) {
	// convenience function for creating a prefixed problem - this reflects the relevant site configuration fields
	newProblem := func(err error) {
		result.Problems = append(result.Problems, conf.NewSiteProblem(fmt.Sprintf("`observability.silenceAlerts`: %v", err)))
	}

	var (
		createdBy = "src-prom-wrapper"
		comment   = "Applied via `observability.silenceAlerts` in site configuration"
		startTime = strfmt.DateTime(time.Now())
		// set 10 year expiry (expiry required, but we don't want it to expire)
		// silences removed from config will be removed from alertmanager
		endTime = strfmt.DateTime(time.Now().Add(10 * 365 * 24 * time.Hour))
		// map configured silences to alertmanager silence IDs
		activeSilences = map[string]string{}
	)

	for _, s := range newConfig.SilencedAlerts {
		activeSilences[s] = ""
	}

	// delete existing silences that should no longer be silenced
	existingSilences, err := change.AMClient.Silence.GetSilences(&silence.GetSilencesParams{Context: ctx})
	if err != nil {
		newProblem(fmt.Errorf("failed to get existing silences: %w", err))
		return
	}
	for _, s := range existingSilences.Payload {
		if *s.CreatedBy != createdBy || *s.Status.State != "active" {
			continue
		}

		// if this silence should not exist, delete
		silencedAlert := newSilenceFromMatchers(s.Matchers)
		if _, shouldBeActive := activeSilences[silencedAlert]; shouldBeActive {
			activeSilences[silencedAlert] = *s.ID
		} else {
			uid := strfmt.UUID(*s.ID)
			if _, err := change.AMClient.Silence.DeleteSilence(&silence.DeleteSilenceParams{
				Context:   ctx,
				SilenceID: uid,
			}); err != nil {
				newProblem(fmt.Errorf("failed to delete existing silence %q: %w", *s.ID, err))
				return
			}
		}
	}
	log.Info("updating alert silences", "silences", activeSilences)

	// create or update silences
	for alert, existingSilence := range activeSilences {
		s := models.Silence{
			CreatedBy: &createdBy,
			Comment:   &comment,
			StartsAt:  &startTime,
			EndsAt:    &endTime,
			Matchers:  newMatchersFromSilence(alert),
		}
		var err error
		if existingSilence != "" {
			_, err = change.AMClient.Silence.PostSilences(&silence.PostSilencesParams{
				Context: ctx,
				Silence: &models.PostableSilence{
					ID:      existingSilence,
					Silence: s,
				},
			})
		} else {
			_, err = change.AMClient.Silence.PostSilences(&silence.PostSilencesParams{
				Context: ctx,
				Silence: &models.PostableSilence{
					Silence: s,
				},
			})
		}
		if err != nil {
			silenceData, _ := json.Marshal(s)
			log.Error("failed to update silence", "error", err, "silence", string(silenceData), "existingSilence", existingSilence)
			newProblem(fmt.Errorf("failed to update silence: %w", err))
			return
		}
	}

	return result
}
