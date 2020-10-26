// Package pubsub is a lightweight wrapper around pubsub which initializes
// pubsub based on environment variables.
package pubsub

import (
	"context"
	"errors"

	"cloud.google.com/go/pubsub"
	"github.com/inconshreveable/log15"
	"github.com/sourcegraph/sourcegraph/internal/env"
)

// PubSubProjectID is used to create a new pubsub client.
var PubSubProjectID = env.Get("PUBSUB_PROJECT_ID", "", "Pub/sub project ID is the id of the pubsub project.")

var client *pubsub.Client

// Enabled returns true if pubsub has been configured to run.
func Enabled() bool {
	return PubSubProjectID != ""
}

func init() {
	if !Enabled() {
		return
	}
	ctx := context.Background()
	pClient, err := pubsub.NewClient(ctx, PubSubProjectID)
	if err != nil {
		log15.Error("failed to create pubsub client.", "error", err)
		return
	}
	client = pClient
}

// Publish publishes msg to the topic asynchronously. Messages are batched and sent according to the topic's PublishSettings.
func Publish(topic string, msg string) error {
	if client == nil {
		return errors.New("failed to publish message: pubsub client is not initialized")
	}

	ctx := context.Background()
	t := client.Topic(topic)
	defer t.Stop()
	result := t.Publish(ctx, &pubsub.Message{
		Data: []byte(msg),
	})
	// Block until the result is returned and a server-generated
	// ID is returned for the published message.
	_, err := result.Get(ctx)
	if err != nil {
		return err
	}
	return nil
}
