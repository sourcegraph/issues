package bg

import (
	"context"

	"github.com/inconshreveable/log15"
	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/jsonc"
)

type savedQuery struct {
	Key         string
	Description string
	Query       string
	Notify      bool
	NotifySlack bool
}

type savedQueryField struct {
	SavedQueries []savedQuery `json:"search.savedQueries"`
}

// MigrateSavedQueriesAndSlackWebhookURLsFromSettingsToDatabase migrates saved searches from the `search.SavedQueries` value in
// site settings into the `saved_searches` PostgreSQL table, and migrates organization Slack webhook URLs from the
// `notifications.slack` value in site settings to the `slack_webhook_url` column in the `saved_searches` table.
//
// This migration does NOT remove the existing "search.savedQueries" and "notifications.slack" values from the user or org's settings.
// This is to be done in a future iteration after all customer instances have upgraded and ran the migration successfully.
func MigrateSavedQueriesAndSlackWebhookURLsFromSettingsToDatabase(ctx context.Context) {
	savedSearchTableIsEmpty, err := db.SavedSearches.IsEmpty(ctx)
	if err != nil {
		log15.Error("migrate.saved-queries: unable to migrate search.savedQueries because cannot tell if saved searches table is empty. Please report this issue.", "error", err)
	}
	if savedSearchTableIsEmpty {
		if err := doMigrateSavedQueriesAndSlackWebhookURLsFromSettingsToDatabase(ctx); err != nil {
			log15.Error(`migrate.saved-queries: unable to migrate "search.savedQueries" settings to database. Please report this issue.`, "error", err)
		}
	}
}

func doMigrateSavedQueriesAndSlackWebhookURLsFromSettingsToDatabase(ctx context.Context) error {
	// List all settings that include a `search.savedQueries` field.
	settings, err := db.Settings.ListAll(ctx, "search.savedQueries")
	if err != nil {
		// Don't continue the migration if we can't list settings.
		return errors.WithMessagef(err, "migrate.saved-queries: unable to migrate saved queries to database. Error listing settings. Please report this issue")
	}

	for _, s := range settings {
		if _, err := migrateSavedQueryIntoDB(ctx, s); err != nil {
			// Do not fail completely if one settings JSON is invalid.
			log15.Error("migrate.saved-queries: ignoring invalid settings for subject", "subject", s.Subject, "error", err)
			continue
		}
	}

	// After migrating all saved searches into the DB, migrate organization Slack webhook URLs into the
	// slack_webhook_url column in the saved_searches table.
	err = migrateSlackWebhookUrlsToSavedSearches(ctx)
	if err != nil {
		return err
	}
	return nil
}

func migrateSavedQueryIntoDB(ctx context.Context, s *api.Settings) (bool, error) {
	var sq savedQueryField
	err := jsonc.Unmarshal(s.Contents, &sq)
	if err != nil {
		return false, errors.WithMessagef(err, `migrate.saved-queries: unable to migrate saved queries from settings subject %s. Unable to unmarshal JSON value. Please report this issue.`, s.Subject)
	}
	err = insertSavedQueryIntoDB(ctx, s, &sq)
	if err != nil {
		return false, err
	}

	return true, nil
}

// getFirstSiteAdminID gets the database ID of the first site admin available in the users table.
func getFirstSiteAdminID(ctx context.Context) *int32 {
	var siteAdminID *int32
	err := dbconn.Global.QueryRowContext(ctx, "WITH site_admins AS (SELECT id, username FROM users WHERE site_admin=true) SELECT min(id) FROM site_admins;").Scan(&siteAdminID)
	if err != nil {
		log15.Error(`migrate.saved-queries: unable to migrate saved query into database. No site admin ID found.`, err.Error())
	}
	return siteAdminID
}

// insertSavedQueryIntoDB inserts an existing saved query from site settings into the saved_searches database table.
// Global saved queries will be associated with the first site admin's profile. It will be added with the UserID set to the first site admin's user ID.
func insertSavedQueryIntoDB(ctx context.Context, s *api.Settings, sq *savedQueryField) error {
	for _, query := range sq.SavedQueries {
		// Add case for global settings. It should make a site admin user the owner of that saved search.
		if s.Subject.User != nil {
			_, err := db.SavedSearches.Create(ctx, &types.SavedSearch{Description: query.Description, Query: query.Query, Notify: query.Notify, NotifySlack: query.NotifySlack, UserID: s.Subject.User})
			if err != nil {
				return errors.WithMessagef(err, `migrate.saved-queries: unable to insert user saved query into database.`)
			}
		} else if s.Subject.Org != nil {
			_, err := db.SavedSearches.Create(ctx, &types.SavedSearch{Description: query.Description, Query: query.Query, Notify: query.Notify, NotifySlack: query.NotifySlack, OrgID: s.Subject.Org})
			if err != nil {
				return errors.WithMessagef(err, `migrate.saved-queries: unable to migrate org saved query into database.`)
			}
		} else if s.Subject.Site || s.Subject.Default {
			siteAdminID := getFirstSiteAdminID(ctx)
			_, err := db.SavedSearches.Create(ctx, &types.SavedSearch{Description: query.Description, Query: query.Query, Notify: query.Notify, NotifySlack: query.NotifySlack, UserID: siteAdminID})
			if err != nil {
				return errors.WithMessagef(err, `migrate.saved-queries: unable to migrate global saved query into database.`)
			}
		}
	}

	return nil
}

type notificationsSlackField struct {
	NotificationsSlack webhookURL `json:"notifications.slack"`
}

type webhookURL struct {
	WebhookURL string `json:"webhookURL"`
}

// migrateSlackWebhookUrlsToSavedSearches migrates Slack webhook URLs from site settings into the
// slack_webhook_url column of the saved_searches database table. As a result, Slack webhook URLs
// will be associated with individual saved searches rather than user or organization profiles.
func migrateSlackWebhookUrlsToSavedSearches(ctx context.Context) error {
	settings, err := db.Settings.ListAll(ctx, "notifications.slack")
	if err != nil {
		return errors.WithMessagef(err, `migrate.saved-queries: unable to migrate "saved queries" to database). Please report this issue`)
	}

	for _, s := range settings {
		var notifsField notificationsSlackField
		err := jsonc.Unmarshal(s.Contents, &notifsField)
		if err != nil {
			// Do not fail completely if one settings JSON is invalid.
			log15.Error(`migrate.saved-queries: Unable to migrate Slack webhook URL to saved search table. Error unmarshaling notifications.slack setting.`, err)
			continue
		}
		if s.Subject.Org != nil {
			insertSlackWebhookURLIntoSavedSearchesTable(ctx, "org", s.Subject.Org, notifsField.NotificationsSlack.WebhookURL)
		} else if s.Subject.User != nil {
			insertSlackWebhookURLIntoSavedSearchesTable(ctx, "user", s.Subject.User, notifsField.NotificationsSlack.WebhookURL)
		} else if s.Subject.Site || s.Subject.Default {
			siteAdminID := getFirstSiteAdminID(ctx)
			insertSlackWebhookURLIntoSavedSearchesTable(ctx, "user", siteAdminID, notifsField.NotificationsSlack.WebhookURL)
		}
	}
	return nil
}

// insertSlackWebhookURLIntoSavedSearchesTable inserts an existing Slack webhook URL into the slack_webhook_url column of the
// saved_searches table.
func insertSlackWebhookURLIntoSavedSearchesTable(ctx context.Context, location string, id *int32, webhookURL string) {
	if location == "org" {
		_, err := dbconn.Global.ExecContext(ctx, "UPDATE saved_searches SET slack_webhook_url=$1 WHERE org_id=$2", webhookURL, *id)
		if err != nil {
			log15.Error("`migrate.saved-queries: unable to migrate Slack webhook URL into saved search table. Error inserting webhook URL.", err)
		}
	} else if location == "user" {
		_, err := dbconn.Global.ExecContext(ctx, "UPDATE saved_searches SET slack_webhook_url=$1 WHERE user_id=$2", webhookURL, *id)
		if err != nil {
			log15.Error("`migrate.saved-queries: unable to migrate Slack webhook URL into saved search table. Error inserting webhook URL.", err)
		}
	}
}
