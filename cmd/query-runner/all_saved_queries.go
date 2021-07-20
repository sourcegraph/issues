package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/cmd/query-runner/queryrunnerapi"
	"github.com/sourcegraph/sourcegraph/internal/api"
)

// diffSavedQueryConfigs takes the old and new saved queries configurations.
//
// It returns maps whose keys represent the old value and value represent the
// new value, i.e. a map of the saved query in the oldList and what its new
// value is in the newList for each respective category. For deleted, the new
// value will be an empty struct.
func diffSavedQueryConfigs(oldList, newList map[api.SavedQueryIDSpec]api.ConfigSavedQuery) (deleted, updated, created map[api.SavedQuerySpecAndConfig]api.SavedQuerySpecAndConfig) {
	deleted = map[api.SavedQuerySpecAndConfig]api.SavedQuerySpecAndConfig{}
	updated = map[api.SavedQuerySpecAndConfig]api.SavedQuerySpecAndConfig{}
	created = map[api.SavedQuerySpecAndConfig]api.SavedQuerySpecAndConfig{}

	// Because the api.SavedqueryIDSpec contains pointers, we should use its
	// unique string key.
	//
	// TODO(slimsag/farhan): long term: let's make these
	// api.SavedQuery Spec types more sane / remove them (in reality, this will
	// be easy to do once we move query runner to frontend later.)
	oldByKey := make(map[string]api.SavedQuerySpecAndConfig, len(oldList))
	for k, v := range oldList {
		oldByKey[k.Key] = api.SavedQuerySpecAndConfig{Spec: k, Config: v}
	}
	newByKey := make(map[string]api.SavedQuerySpecAndConfig, len(newList))
	for k, v := range newList {
		newByKey[k.Key] = api.SavedQuerySpecAndConfig{Spec: k, Config: v}
	}
	// Detect deleted entries
	for k, oldVal := range oldByKey {
		if _, ok := newByKey[k]; !ok {
			deleted[oldVal] = api.SavedQuerySpecAndConfig{}
		}
	}

	for k, newVal := range newByKey {
		// Detect created entries
		if oldVal, ok := oldByKey[k]; !ok {
			created[oldVal] = newVal
			continue
		}
		// Detect updated entries
		oldVal := oldByKey[k]
		if ok := reflect.DeepEqual(newVal, oldVal); !ok {
			updated[oldVal] = newVal
		}
	}
	return deleted, updated, created
}

func sendNotificationsForCreatedOrUpdatedOrDeleted(oldList, newList map[api.SavedQueryIDSpec]api.ConfigSavedQuery) {
	deleted, updated, created := diffSavedQueryConfigs(oldList, newList)
	for oldVal, newVal := range deleted {
		oldVal := oldVal
		newVal := newVal
		go func() {
			if err := notifySavedQueryWasCreatedOrUpdated(oldVal, newVal); err != nil {
				log15.Error("Failed to handle deleted saved search.", "query", oldVal.Config.Query, "error", err)
			}
		}()
	}
	for oldVal, newVal := range created {
		oldVal := oldVal
		newVal := newVal
		go func() {
			if err := notifySavedQueryWasCreatedOrUpdated(oldVal, newVal); err != nil {
				log15.Error("Failed to handle created saved search.", "query", oldVal.Config.Query, "error", err)
			}
		}()
	}
	for oldVal, newVal := range updated {
		oldVal := oldVal
		newVal := newVal
		go func() {
			if err := notifySavedQueryWasCreatedOrUpdated(oldVal, newVal); err != nil {
				log15.Error("Failed to handle updated saved search.", "query", oldVal.Config.Query, "error", err)
			}
		}()
	}
}

func notifySavedQueryWasCreatedOrUpdated(oldValue, newValue api.SavedQuerySpecAndConfig) error {
	ctx := context.Background()

	oldRecipients, err := getNotificationRecipients(ctx, oldValue.Spec, oldValue.Config)
	if err != nil {
		return err
	}
	newRecipients, err := getNotificationRecipients(ctx, newValue.Spec, newValue.Config)
	if err != nil {
		return err
	}

	removedRecipients, addedRecipients := diffNotificationRecipients(oldRecipients, newRecipients)
	log15.Debug("Notifying for created/updated saved search", "removed", removedRecipients, "added", addedRecipients)
	for _, removedRecipient := range removedRecipients {
		if removedRecipient.email {
			if err := emailNotifySubscribeUnsubscribe(ctx, removedRecipient, oldValue, notifyUnsubscribedTemplate); err != nil {
				log15.Error("Failed to send unsubscribed email notification.", "recipient", removedRecipient, "error", err)
			}
		}
		if removedRecipient.slack {
			if err := slackNotifyUnsubscribed(ctx, removedRecipient, oldValue); err != nil {
				log15.Error("Failed to send unsubscribed Slack notification.", "recipient", removedRecipient, "error", err)
			}
		}
	}
	for _, addedRecipient := range addedRecipients {
		if addedRecipient.email {
			if err := emailNotifySubscribeUnsubscribe(ctx, addedRecipient, newValue, notifySubscribedTemplate); err != nil {
				log15.Error("Failed to send subscribed email notification.", "recipient", addedRecipient, "error", err)
			}
		}
		if addedRecipient.slack {
			if err := slackNotifySubscribed(ctx, addedRecipient, newValue); err != nil {
				log15.Error("Failed to send subscribed Slack notification.", "recipient", addedRecipient, "error", err)
			}
		}
	}
	return nil
}

var testNotificationMu sync.Mutex

func serveTestNotification(w http.ResponseWriter, r *http.Request) {
	testNotificationMu.Lock()
	defer testNotificationMu.Unlock()

	var args *queryrunnerapi.TestNotificationArgs
	if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
		writeError(w, errors.Wrap(err, "decoding JSON arguments"))
		return
	}

	recipients, err := getNotificationRecipients(r.Context(), args.SavedSearch.Spec, args.SavedSearch.Config)
	if err != nil {
		writeError(w, errors.Errorf("error computing recipients: %s", err))
		return
	}

	for _, recipient := range recipients {
		if err := emailNotifySubscribeUnsubscribe(r.Context(), recipient, args.SavedSearch, notifySubscribedTemplate); err != nil {
			writeError(w, errors.Errorf("error sending email notifications to %s: %s", recipient.spec, err))
			return
		}
		testNotificationAlert := fmt.Sprintf(`It worked! This is a test notification for the Sourcegraph saved search <%s|"%s">.`, searchURL(args.SavedSearch.Config.Query, utmSourceSlack), args.SavedSearch.Config.Description)
		if err := slackNotify(context.Background(), recipient,
			testNotificationAlert, args.SavedSearch.Config.SlackWebhookURL); err != nil {
			writeError(w, errors.Errorf("error sending slack notifications to %s: %s", recipient.spec, err))
			return
		}
	}

	log15.Info("saved query test notification sent", "spec", args.SavedSearch.Spec, "key", args.SavedSearch.Spec.Key)
}
