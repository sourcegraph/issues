package main

import (
	"context"
	"fmt"

	"github.com/cockroachdb/errors"
	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/slack"
)

func (n *notifier) slackNotify(ctx context.Context) {
	plural := ""
	if n.results.Data.Search.Results.ApproximateResultCount != "1" {
		plural = "s"
	}

	text := fmt.Sprintf(`*%s* new result%s found for saved search <%s|"%s">`,
		n.results.Data.Search.Results.ApproximateResultCount,
		plural,
		searchURL(n.newQuery, utmSourceSlack),
		n.query.Description,
	)
	for _, recipient := range n.recipients {
		if err := slackNotify(ctx, recipient, text, n.query.SlackWebhookURL); err != nil {
			log15.Error("Failed to post Slack notification message.", "recipient", recipient, "text", text, "error", err)
		}
	}
	// TODO(Dan): find all users in the recipient list and log events for all of them
	logEvent(0, "SavedSearchSlackNotificationSent", "results")
}

func slackNotifySubscribed(ctx context.Context, recipient *recipient, query api.SavedQuerySpecAndConfig) error {
	text := fmt.Sprintf(`Slack notifications enabled for the saved search <%s|"%s">. Notifications will be sent here when new results are available.`,
		searchURL(query.Config.Query, utmSourceSlack),
		query.Config.Description,
	)
	if err := slackNotify(ctx, recipient, text, query.Config.SlackWebhookURL); err != nil {
		return err
	}
	// TODO(Dan): find all users in the recipient list and log events for all of them
	logEvent(0, "SavedSearchSlackNotificationSent", "enabled")
	return nil
}

func slackNotifyUnsubscribed(ctx context.Context, recipient *recipient, query api.SavedQuerySpecAndConfig) error {
	text := fmt.Sprintf(`Slack notifications for the saved search <%s|"%s"> disabled.`,
		searchURL(query.Config.Query, utmSourceSlack),
		query.Config.Description,
	)
	if err := slackNotify(ctx, recipient, text, query.Config.SlackWebhookURL); err != nil {
		return err
	}
	// TODO(Dan): find all users in the recipient list and log events for all of them
	logEvent(0, "SavedSearchSlackNotificationSent", "disabled")
	return nil
}

func slackNotify(ctx context.Context, recipient *recipient, text string, slackWebhookURL *string) error {
	if !recipient.slack {
		return nil
	}

	if slackWebhookURL == nil || *slackWebhookURL == "" {
		return errors.Errorf("unable to send Slack notification because recipient (%s) has no Slack webhook URL configured", recipient.spec)
	}

	payload := &slack.Payload{
		Username:    "saved-search-bot",
		IconEmoji:   ":mag:",
		UnfurlLinks: false,
		UnfurlMedia: false,
		Text:        text,
	}
	client := slack.New(*slackWebhookURL)
	return client.Post(ctx, payload)
}
