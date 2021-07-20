package email

import (
	"context"
	"fmt"
	"net/url"

	"github.com/cockroachdb/errors"
	"github.com/graph-gophers/graphql-go/relay"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/codemonitors"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/txemail/txtypes"
)

var externalURL *url.URL

// To avoid a circular dependency with the codemonitors/resolvers package
// we have to redeclare the MonitorKind.
const MonitorKind = "CodeMonitor"
const utmSourceEmail = "code-monitoring-email"
const priorityCritical = "CRITICAL"

var MockSendEmailForNewSearchResult func(ctx context.Context, userID int32, data *TemplateDataNewSearchResults) error
var MockExternalURL func() *url.URL

func SendEmailForNewSearchResult(ctx context.Context, userID int32, data *TemplateDataNewSearchResults) error {
	if MockSendEmailForNewSearchResult != nil {
		return MockSendEmailForNewSearchResult(ctx, userID, data)
	}
	return sendEmail(ctx, userID, newSearchResultsEmailTemplates, data)
}

type TemplateDataNewSearchResults struct {
	Priority                  string
	CodeMonitorURL            string
	SearchURL                 string
	Description               string
	NumberOfResultsWithDetail string
	IsTest                    bool
}

func NewTemplateDataForNewSearchResults(ctx context.Context, monitorDescription, queryString string, email *codemonitors.MonitorEmail, numResults int) (d *TemplateDataNewSearchResults, err error) {
	var (
		searchURL                 string
		codeMonitorURL            string
		priority                  string
		numberOfResultsWithDetail string
	)
	searchURL, err = getSearchURL(ctx, queryString, utmSourceEmail)
	if err != nil {
		return nil, err
	}

	codeMonitorURL, err = getCodeMonitorURL(ctx, email.Monitor, utmSourceEmail)
	if err != nil {
		return nil, err
	}

	if email.Priority == priorityCritical {
		priority = "Critical"
	} else {
		priority = "New"
	}

	if numResults == 1 {
		numberOfResultsWithDetail = fmt.Sprintf("There was %d new search result for your query", numResults)
	} else {
		numberOfResultsWithDetail = fmt.Sprintf("There were %d new search results for your query", numResults)
	}

	return &TemplateDataNewSearchResults{
		Priority:                  priority,
		CodeMonitorURL:            codeMonitorURL,
		SearchURL:                 searchURL,
		Description:               monitorDescription,
		NumberOfResultsWithDetail: numberOfResultsWithDetail,
	}, nil
}

func NewTestTemplateDataForNewSearchResults(ctx context.Context, monitorDescription string) *TemplateDataNewSearchResults {
	return &TemplateDataNewSearchResults{
		Priority:                  "New",
		Description:               monitorDescription,
		NumberOfResultsWithDetail: "There was 1 new search result for your query",
		IsTest:                    true,
	}
}

func sendEmail(ctx context.Context, userID int32, template txtypes.Templates, data interface{}) error {
	email, err := api.InternalClient.UserEmailsGetEmail(ctx, userID)
	if err != nil {
		return errors.Errorf("InternalClient.UserEmailsGetEmail for userID=%d: %w", userID, err)
	}
	if email == nil {
		return errors.Errorf("unable to send email to user ID %d with unknown email address", userID)
	}
	if err := api.InternalClient.SendEmail(ctx, txtypes.Message{
		To:       []string{*email},
		Template: template,
		Data:     data,
	}); err != nil {
		return errors.Errorf("InternalClient.SendEmail to email=%q userID=%d: %w", *email, userID, err)
	}
	return nil
}

func getSearchURL(ctx context.Context, query, utmSource string) (string, error) {
	return sourcegraphURL(ctx, "search", query, utmSource)
}

func getCodeMonitorURL(ctx context.Context, monitorID int64, utmSource string) (string, error) {
	return sourcegraphURL(ctx, fmt.Sprintf("code-monitoring/%s", relay.MarshalID(MonitorKind, monitorID)), "", utmSource)
}

func sourcegraphURL(ctx context.Context, path, query, utmSource string) (string, error) {
	if MockExternalURL != nil {
		externalURL = MockExternalURL()
	}
	if externalURL == nil {
		// Determine the external URL.
		externalURLStr, err := api.InternalClient.ExternalURL(ctx)
		if err != nil {
			return "", errors.Errorf("failed to get ExternalURL: %w", err)
		}
		externalURL, err = url.Parse(externalURLStr)
		if err != nil {

			return "", errors.Errorf("failed to get ExternalURL: %w", err)
		}
	}

	// Construct URL to the search query.
	u := externalURL.ResolveReference(&url.URL{Path: path})
	q := u.Query()
	if query != "" {
		q.Set("q", query)
	}
	q.Set("utm_source", utmSource)
	u.RawQuery = q.Encode()
	return u.String(), nil
}
