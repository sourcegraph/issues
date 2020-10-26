package main

import (
	"context"
	"fmt"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/txemail"
	"github.com/sourcegraph/sourcegraph/internal/txemail/txtypes"
)

func canSendEmail(ctx context.Context) error {
	canSendEmail, err := api.InternalClient.CanSendEmail(ctx)
	if err != nil {
		return errors.Wrap(err, "InternalClient.CanSendEmail")
	}
	if !canSendEmail {
		return errors.New("SMTP server not set in site configuration")
	}
	return nil
}

func (n *notifier) emailNotify(ctx context.Context) {
	if err := canSendEmail(ctx); err != nil {
		log15.Error("Failed to send email notification for saved search.", "error", err)
		return
	}

	// Send tx emails asynchronously.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		for _, recipient := range n.recipients {
			ownership := "the" // example: "new search results have been found for {{.Ownership}} saved search"
			if n.spec.Subject.User != nil && *n.spec.Subject.User == recipient.spec.userID {
				ownership = "your"
			}
			if n.spec.Subject.Org != nil {
				ownership = "your organization's"
			}

			plural := ""
			if n.results.Data.Search.Results.ApproximateResultCount != "1" {
				plural = "s"
			}
			if err := sendEmail(ctx, recipient.spec.userID, "results", newSearchResultsEmailTemplates, struct {
				URL                    string
				SavedSearchPageURL     string
				Description            string
				Query                  string
				ApproximateResultCount string
				Ownership              string
				PluralResults          string
			}{
				URL:                    searchURL(n.newQuery, utmSourceEmail),
				SavedSearchPageURL:     savedSearchListPageURL(utmSourceEmail),
				Description:            n.query.Description,
				Query:                  n.query.Query,
				ApproximateResultCount: n.results.Data.Search.Results.ApproximateResultCount,
				Ownership:              ownership,
				PluralResults:          plural,
			}); err != nil {
				log15.Error("Failed to send email notification for new saved search results.", "userID", recipient.spec.userID, "error", err)
			}
		}
	}()
}

var newSearchResultsEmailTemplates = txemail.MustValidate(txtypes.Templates{
	Subject: `[{{.ApproximateResultCount}} new result{{.PluralResults}}] {{.Description}}`,
	Text: `
{{.ApproximateResultCount}} new search result{{.PluralResults}} found for {{.Ownership}} saved search:

  "{{.Description}}"

View the new result{{.PluralResults}} on Sourcegraph: {{.URL}}
`,
	HTML: `
<strong>{{.ApproximateResultCount}}</strong> new search result{{.PluralResults}} found for {{.Ownership}} saved search:

<p style="padding-left: 16px">&quot;{{.Description}}&quot;</p>

<p><a href="{{.URL}}">View the new result{{.PluralResults}} on Sourcegraph</a></p>

<p><a href="{{.SavedSearchPageURL}}">Edit your saved searches on Sourcegraph</a></p>
`,
})

func emailNotifySubscribeUnsubscribe(ctx context.Context, recipient *recipient, query api.SavedQuerySpecAndConfig, template txtypes.Templates) error {
	if !recipient.email {
		return nil
	}

	if err := canSendEmail(ctx); err != nil {
		return err
	}

	var eventType string
	switch {
	case template == notifySubscribedTemplate:
		eventType = "subscribed"
	case template == notifyUnsubscribedTemplate:
		eventType = "unsubscribed"
	default:
		eventType = "unknown"
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	ownership := "the" // example: "new search results have been found for {{.Ownership}} saved search"
	if query.Spec.Subject.User != nil && *query.Spec.Subject.User == recipient.spec.userID {
		ownership = "your"
	}
	if query.Spec.Subject.Org != nil {
		ownership = "your organization's"
	}

	return sendEmail(ctx, recipient.spec.userID, eventType, template, struct {
		Ownership   string
		Description string
	}{
		Ownership:   ownership,
		Description: query.Config.Description,
	})
}

func sendEmail(ctx context.Context, userID int32, eventType string, template txtypes.Templates, data interface{}) error {
	email, err := api.InternalClient.UserEmailsGetEmail(ctx, userID)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("InternalClient.UserEmailsGetEmail for userID=%d", userID))
	}
	if email == nil {
		return fmt.Errorf("unable to send email to user ID %d with unknown email address", userID)
	}

	if err := api.InternalClient.SendEmail(ctx, txtypes.Message{
		To:       []string{*email},
		Template: template,
		Data:     data,
	}); err != nil {
		return errors.Wrap(err, fmt.Sprintf("InternalClient.SendEmail to email=%q userID=%d", *email, userID))
	}
	logEvent(userID, "SavedSearchEmailNotificationSent", eventType)
	return nil
}

var notifySubscribedTemplate = txemail.MustValidate(txtypes.Templates{
	Subject: `[Subscribed] {{.Description}}`,
	Text: `
You are now receiving notifications for {{.Ownership}} saved search:

  "{{.Description}}"

When new search results become available, we will notify you.
`,
	HTML: `
<p>You are now receiving notifications for {{.Ownership}} saved search:</p>

<p style="padding-left: 16px">&quot;{{.Description}}&quot;</p>

<p>When new search results become available, we will notify you.</p>
`,
})

var notifyUnsubscribedTemplate = txemail.MustValidate(txtypes.Templates{
	Subject: `[Unsubscribed] {{.Description}}`,
	Text: `
You will no longer receive notifications for {{.Ownership}} saved search:

  "{{.Description}}"

(either you were removed as a person to notify, or the saved search was deleted)
`,
	HTML: `
<p>You will no longer receive notifications for {{.Ownership}} saved search:</p>

<p style="padding-left: 16px">&quot;{{.Description}}&quot;</p>

<p>(either you were removed as a person to notify, or the saved search was deleted)</p>
`,
})
