package usagestats

import (
	"context"
	"encoding/json"
	"time"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/envvar"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/env"
	"github.com/sourcegraph/sourcegraph/internal/pubsub"
	"github.com/sourcegraph/sourcegraph/internal/version"
)

// pubSubDotComEventsTopicID is the topic ID of the topic that forwards messages to Sourcegraph.com events' pub/sub subscribers.
var pubSubDotComEventsTopicID = env.Get("PUBSUB_DOTCOM_EVENTS_TOPIC_ID", "", "Pub/sub dotcom events topic ID is the pub/sub topic id where Sourcegraph.com events are published.")

// Event represents a request to log telemetry.
type Event struct {
	EventName    string
	UserID       int32
	UserCookieID string
	URL          string
	Source       string
	Argument     json.RawMessage
}

// LogBackendEvent is a convenience function for logging backend events.
func LogBackendEvent(userID int32, eventName string, argument json.RawMessage) error {
	return LogEvent(context.Background(), Event{
		EventName:    eventName,
		UserID:       userID,
		UserCookieID: "backend", // Use a non-empty string here to avoid the event_logs table's user existence constraint causing issues
		URL:          "",
		Source:       "BACKEND",
		Argument:     argument,
	})
}

// LogEvent logs an event.
func LogEvent(ctx context.Context, args Event) error {
	if !conf.EventLoggingEnabled() {
		return nil
	}
	if envvar.SourcegraphDotComMode() {
		err := publishSourcegraphDotComEvent(args)
		if err != nil {
			return err
		}
	}
	return logLocalEvent(ctx, args.EventName, args.URL, args.UserID, args.UserCookieID, args.Source, args.Argument)
}

type bigQueryEvent struct {
	EventName       string `json:"name"`
	AnonymousUserID string `json:"anonymous_user_id"`
	UserID          int    `json:"user_id"`
	URL             string `json:"url"`
	Source          string `json:"source"`
	Timestamp       string `json:"timestamp"`
	Version         string `json:"version"`
	Argument        string `json:"argument"`
}

// publishSourcegraphDotComEvent publishes Sourcegraph.com events to BigQuery.
func publishSourcegraphDotComEvent(args Event) error {
	if !envvar.SourcegraphDotComMode() {
		return nil
	}
	if pubSubDotComEventsTopicID == "" {
		return nil
	}
	event, err := json.Marshal(bigQueryEvent{
		EventName:       args.EventName,
		UserID:          int(args.UserID),
		AnonymousUserID: args.UserCookieID,
		URL:             args.URL,
		Source:          args.Source,
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		Version:         version.Version(),
		Argument:        string(args.Argument),
	})
	if err != nil {
		return err
	}
	return pubsub.Publish(pubSubDotComEventsTopicID, string(event))
}

// logLocalEvent logs users events.
func logLocalEvent(ctx context.Context, name, url string, userID int32, userCookieID, source string, argument json.RawMessage) error {
	if name == "SearchResultsQueried" {
		err := logSiteSearchOccurred()
		if err != nil {
			return err
		}
	}
	if name == "findReferences" {
		err := logSiteFindRefsOccurred()
		if err != nil {
			return err
		}
	}

	info := &db.Event{
		Name:            name,
		URL:             url,
		UserID:          uint32(userID),
		AnonymousUserID: userCookieID,
		Source:          source,
		Argument:        argument,
		Timestamp:       timeNow().UTC(),
	}
	return db.EventLogs.Insert(ctx, info)
}
