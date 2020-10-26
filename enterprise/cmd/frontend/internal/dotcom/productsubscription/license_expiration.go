package productsubscription

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/efritz/glock"
	"github.com/gomodule/redigo/redis"
	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/licensing"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/redispool"
	"github.com/sourcegraph/sourcegraph/internal/slack"
)

const lastLicenseExpirationCheckKey = "last_license_expiration_check"

var licenseExpirationCheckers uint32

// StartCheckForUpcomingLicenseExpirations checks for upcoming license expirations once per day.
func StartCheckForUpcomingLicenseExpirations() {
	if atomic.AddUint32(&licenseExpirationCheckers, 1) != 1 {
		panic("StartCheckForUpcomingLicenseExpirations called more than once")
	}

	dotcom := conf.Get().Dotcom
	if dotcom == nil {
		return
	}
	client := slack.New(dotcom.SlackLicenseExpirationWebhook)

	t := time.NewTicker(1 * time.Hour)
	for range t.C {
		checkLicensesIfNeeded(client)
	}
}

type slackClient interface {
	Post(ctx context.Context, payload *slack.Payload) error
}

// checkLicensesIfNeeded checks whether a day has passed since the last license check, and if so, initiates one.
func checkLicensesIfNeeded(client slackClient) {
	c := redispool.Store.Get()
	defer func() { _ = c.Close() }()

	today := time.Now().UTC().Format("2006-01-02")

	lastCheckDate, err := redis.String(c.Do("GETSET", lastLicenseExpirationCheckKey, today))
	if err == redis.ErrNil {
		// If the redis key hasn't been set yet, do so and leave lastCheckDate as nil
		_, setErr := c.Do("SET", lastLicenseExpirationCheckKey, today)
		if setErr != nil {
			log15.Error("startCheckForUpcomingLicenseExpirations: error SET last license expiration check date", "error", err)
			return
		}
	} else if err != nil {
		log15.Error("startCheckForUpcomingLicenseExpirations: error GETSET last license expiration check date", "error", err)
		return
	}

	if today != lastCheckDate {
		checkForUpcomingLicenseExpirations(glock.NewRealClock(), client)
	}
}

func checkForUpcomingLicenseExpirations(clock glock.Clock, client slackClient) {
	if conf.Get().Dotcom == nil || conf.Get().Dotcom.SlackLicenseExpirationWebhook == "" {
		return
	}

	ctx := context.Background()

	allDBSubscriptions, err := dbSubscriptions{}.List(ctx, dbSubscriptionsListOptions{
		IncludeArchived: false,
	})
	if err != nil {
		log15.Error("startCheckForUpcomingLicenseExpirations: error listing subscriptions", "error", err)
		return
	}

	for _, dbSubscription := range allDBSubscriptions {
		// Get the active (i.e., latest created) license.
		licenses, err := dbLicenses{}.List(ctx, dbLicensesListOptions{ProductSubscriptionID: dbSubscription.ID, LimitOffset: &db.LimitOffset{Limit: 1}})
		if err != nil {
			log15.Error("startCheckForUpcomingLicenseExpirations: error listing licenses", "error", err)
			return
		}
		// Skip if subscription has no licenses.
		if len(licenses) < 1 {
			return
		}

		user, err := db.Users.GetByID(ctx, dbSubscription.UserID)
		if err != nil {
			log15.Error("startCheckForUpcomingLicenseExpirations: error looking up user", "error", err)
			return
		}

		info, _, err := licensing.ParseProductLicenseKeyWithBuiltinOrGenerationKey(licenses[0].LicenseKey)
		if err != nil {
			log15.Error("startCheckForUpcomingLicenseExpirations: error parsing license key", "error", err)
			return
		}

		weekAway := clock.Now().Add(7 * 24 * time.Hour)
		dayAway := clock.Now().Add(24 * time.Hour)

		if info.ExpiresAt.After(weekAway) && info.ExpiresAt.Before(weekAway.Add(24*time.Hour)) {
			err = client.Post(context.Background(), &slack.Payload{
				Text: fmt.Sprintf("The license for user `%s` <https://sourcegraph.com/site-admin/dotcom/product/subscriptions/%s|will expire *in 7 days*>", user.Username, dbSubscription.ID),
			})
			if err != nil {
				log15.Error("startCheckForUpcomingLicenseExpirations: error sending Slack message", "error", err)
				return
			}
		} else if info.ExpiresAt.After(dayAway) && info.ExpiresAt.Before(dayAway.Add(24*time.Hour)) {
			err = client.Post(context.Background(), &slack.Payload{
				Text: fmt.Sprintf("The license for user `%s` <https://sourcegraph.com/site-admin/dotcom/product/subscriptions/%s|will expire *in the next 24 hours*> :rotating_light:", user.Username, dbSubscription.ID),
			})
			if err != nil {
				log15.Error("startCheckForUpcomingLicenseExpirations: error sending Slack message", "error", err)
				return
			}
		}
	}
}
