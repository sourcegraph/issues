package bg

import (
	"context"
	"math/rand"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/pkg/errors"
	"github.com/sourcegraph/jsonx"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/jsonc"
	"github.com/sourcegraph/sourcegraph/schema"
)

// MigrateAllSettingsMOTDToNotices migrates the deprecated "motd" settings property to the new
// "notices" settings property.
//
// TODO: Remove this migration code in Sourcegraph 3.4, which is one minor release after Sourcegraph
// 3.3 (which introduces the "motd" deprecation and this migration).
func MigrateAllSettingsMOTDToNotices(ctx context.Context) {
	// It is not necessary to run this immediately because the old "motd" property is still
	// supported. In case the DB query is computationally intensive, wait for a random delay to
	// avoid a thundering herd.
	time.Sleep(time.Duration(rand.Intn(300)) * time.Second)

	if err := doMigrateAllSettingsMOTDToNotices(ctx, 60*time.Second); err != nil {
		log15.Error(`Warning: unable to migrate settings ("motd" to "notices"). Please report this issue. The "motd" settings property has been deprecated in favor of "notices", and future versions of Sourcegraph will remove support for "motd".`, "error", err)
	}
}

func doMigrateAllSettingsMOTDToNotices(ctx context.Context, iterationDelay time.Duration) (err error) {
rerun:
	settingsWithMOTD, err := db.Settings.ListAll(ctx, "motd")
	if err != nil {
		return err
	}
	count := 0
	for _, s := range settingsWithMOTD {
		changed, err := migrateSettingsMOTDToNotices(ctx, s.Subject)
		if err != nil {
			return errors.WithMessagef(err, "in settings for %s", s.Subject)
		}
		if changed {
			count++
		}
	}

	// To reduce the (small) chance of a race condition whereby a new settings document can have an
	// "motd" added without reporting a validation error (e.g., by an older deployed version while
	// this migration is running), rerun until we have nothing else to do.
	if count > 0 {
		log15.Info(`Migrated settings "motd" to "notices".`, "count", count)
		time.Sleep(iterationDelay)
		goto rerun
	}

	return nil
}

// migrateSettingsMOTDToNotices migrates a single settings document (for the subject) from using
// "motd" to "notices". It reports whether the settings actually were changed.
func migrateSettingsMOTDToNotices(ctx context.Context, settingsSubject api.SettingsSubject) (changed bool, err error) {
	// Refetch the settings to reduce the risk of a race condition where the user edited their
	// settings between the time we fetched the entire list and when we are processing migrating
	// this one.
	settings, err := db.Settings.GetLatest(ctx, settingsSubject)
	if err != nil {
		return false, err
	}

	newContents, err := migrateSettingsMOTDToNoticesJSON(settings.Contents)
	if err != nil {
		return false, err
	}
	if newContents == settings.Contents {
		return false, nil // nothing to do
	}

	// Write migrated settings to DB.
	var lastID *int32
	if settings != nil {
		lastID = &settings.ID
	}
	_, err = db.Settings.CreateIfUpToDate(ctx, settingsSubject, lastID, nil, newContents)
	return true, err
}

// migrateSettingsMOTDToNoticesJSON returns the migrated JSON from the input, with the "motd"
// property migrated to "notices".
func migrateSettingsMOTDToNoticesJSON(text string) (string, error) {
	var parsed schema.Settings
	if err := jsonc.Unmarshal(text, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Motd) == 0 {
		return text, nil // no change needed
	}
	migratedNotices := make([]schema.Notice, len(parsed.Motd))
	for i, motd := range parsed.Motd {
		migratedNotices[i] = schema.Notice{
			Location:    "top",
			Message:     motd,
			Dismissible: true,
		}
	}

	// Remove "motd".
	edits, _, err := jsonx.ComputePropertyRemoval(text, jsonx.MakePath("motd"), conf.FormatOptions)
	if err != nil {
		return "", err
	}
	text, err = jsonx.ApplyEdits(text, edits...)
	if err != nil {
		return "", err
	}

	// Append notices one-by-one (because the ComputePropertyEdit API does not let you append
	// multiple array elements in one call).
	for _, migratedNotice := range migratedNotices {
		edits, _, err := jsonx.ComputePropertyEdit(text, jsonx.MakePath("notices", -1), migratedNotice, nil, conf.FormatOptions)
		if err != nil {
			return "", err
		}
		text, err = jsonx.ApplyEdits(text, edits...)
		if err != nil {
			return "", err
		}
	}
	return text, nil
}
