package bg

import (
	"context"
	"reflect"
	"testing"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
)

func TestMigrateSavedQueriesAndSlackWebhookURLsFromSettingsToDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()

	t.Run("migrate user saved query", func(t *testing.T) {
		u, err := db.Users.Create(ctx, db.NewUser{Username: "u"})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Settings.CreateIfUpToDate(ctx, api.SettingsSubject{User: &u.ID}, nil, nil, `{"search.savedQueries": [{"key": "1a2b3c", "description": "test query", "query": "test type:diff"}]}`); err != nil {
			t.Fatal(err)
		}
		MigrateSavedQueriesAndSlackWebhookURLsFromSettingsToDatabase(ctx)
		ss, err := db.SavedSearches.ListSavedSearchesByUserID(ctx, u.ID)
		if err != nil {
			t.Fatal(err)
		}
		want := []api.SavedQuerySpecAndConfig{{Spec: api.SavedQueryIDSpec{Subject: api.SettingsSubject{User: &u.ID}, Key: "1"}, Config: api.ConfigSavedQuery{Key: "1", Description: "test query", Query: "test type:diff", Notify: false, NotifySlack: false, UserID: &u.ID, OrgID: nil, SlackWebhookURL: nil}}}
		if reflect.DeepEqual(want, ss) {
			t.Errorf("want %v, got %v", want, ss)
		}
	})

	t.Run("migrate org saved query", func(t *testing.T) {
		o, err := db.Orgs.Create(ctx, "test-org", nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Settings.CreateIfUpToDate(ctx, api.SettingsSubject{Org: &o.ID}, nil, nil, `{"search.savedQueries": [{"key": "1a2b3c", "description": "test query", "query": "test type:diff"}]}`); err != nil {
			t.Fatal(err)
		}
		MigrateSavedQueriesAndSlackWebhookURLsFromSettingsToDatabase(ctx)
		ss, err := db.SavedSearches.ListSavedSearchesByOrgID(ctx, o.ID)
		if err != nil {
			t.Fatal(err)
		}
		want := []api.SavedQuerySpecAndConfig{{Spec: api.SavedQueryIDSpec{Subject: api.SettingsSubject{Org: &o.ID}, Key: "1"}, Config: api.ConfigSavedQuery{Key: "1", Description: "test query", Query: "test type:diff", Notify: false, NotifySlack: false, UserID: nil, OrgID: &o.ID, SlackWebhookURL: nil}}}
		if reflect.DeepEqual(want, ss) {
			t.Errorf("want %v, got %v", want, ss)
		}
	})

	t.Run("migrate global saved query", func(t *testing.T) {
		u, err := db.Users.GetByUsername(ctx, "u")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Settings.CreateIfUpToDate(ctx, api.SettingsSubject{}, nil, nil, `{"search.savedQueries": [{"key": "1a2b3c", "description": "test query global", "query": "test type:diff"}]}`); err != nil {
			t.Fatal(err)
		}
		MigrateSavedQueriesAndSlackWebhookURLsFromSettingsToDatabase(ctx)
		ss, err := db.SavedSearches.ListSavedSearchesByUserID(ctx, u.ID)
		if err != nil {
			t.Fatal(err)
		}
		want := []api.SavedQuerySpecAndConfig{{Spec: api.SavedQueryIDSpec{Subject: api.SettingsSubject{User: &u.ID}, Key: "1"}, Config: api.ConfigSavedQuery{Key: "1", Description: "test query", Query: "test type:diff", Notify: false, NotifySlack: false, UserID: &u.ID, OrgID: nil, SlackWebhookURL: nil}}, {Spec: api.SavedQueryIDSpec{Subject: api.SettingsSubject{}, Key: "1"}, Config: api.ConfigSavedQuery{Key: "1", Description: "test query global", Query: "test type:diff", Notify: false, NotifySlack: false, UserID: &u.ID, OrgID: nil, SlackWebhookURL: nil}}}
		if reflect.DeepEqual(want, ss) {
			t.Errorf("want %v, got %v", want, ss)
		}
	})
}

func TestDoMigrateSavedQueriesAndSlackWebhookURLsFromSettingsToDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()

	t.Run("valid", func(t *testing.T) {
		u, err := db.Users.Create(ctx, db.NewUser{Username: "u1"})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Settings.CreateIfUpToDate(ctx, api.SettingsSubject{User: &u.ID}, nil, nil, `{"search.savedQueries": [{"key": "1a2b3c", "description": "test query", "query": "test type:diff"}]}`); err != nil {
			t.Fatal(err)
		}
		if err := doMigrateSavedQueriesAndSlackWebhookURLsFromSettingsToDatabase(ctx); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("no saved queries", func(t *testing.T) {
		u, err := db.Users.Create(ctx, db.NewUser{Username: "u2"})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Settings.CreateIfUpToDate(ctx, api.SettingsSubject{User: &u.ID}, nil, nil, `{}`); err != nil {
			t.Fatal(err)
		}
		if err := doMigrateSavedQueriesAndSlackWebhookURLsFromSettingsToDatabase(ctx); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("do not throw error for an invalid settings JSON", func(t *testing.T) {
		u, err := db.Users.Create(ctx, db.NewUser{Username: "u3"})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Settings.CreateIfUpToDate(ctx, api.SettingsSubject{User: &u.ID}, nil, nil, `{"search.savedQueries": [1]}`); err != nil {
			t.Fatal(err)
		}
		err = doMigrateSavedQueriesAndSlackWebhookURLsFromSettingsToDatabase(ctx)
		if err != nil {
			t.Error(err, "there should not be an error for an invalid settings json")
		}
	})

	t.Run("do not throw error for an invalid settings JSON and properly migrate saved queries from valid settings", func(t *testing.T) {
		u, err := db.Users.Create(ctx, db.NewUser{Username: "u4"})
		if err != nil {
			t.Fatal(err)
		}
		u2, err := db.Users.Create(ctx, db.NewUser{Username: "u5"})
		if err != nil {
			t.Fatal(err)
		}

		if _, err := db.Settings.CreateIfUpToDate(ctx, api.SettingsSubject{User: &u.ID}, nil, nil, `{"search.savedQueries": [{"key": "1a2b3c", "description": "test query", "query": "test type:diff"}]}`); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Settings.CreateIfUpToDate(ctx, api.SettingsSubject{User: &u2.ID}, nil, nil, `{"search.savedQueries": [1]}`); err != nil {
			t.Fatal(err)
		}
		err = doMigrateSavedQueriesAndSlackWebhookURLsFromSettingsToDatabase(ctx)
		if err != nil {
			t.Error(err, "there should not be an error for an invalid settings json")
		}
		ss, err := db.SavedSearches.ListSavedSearchesByUserID(ctx, u.ID)
		if len(ss) == 0 {
			t.Fatal("Error: user saved query not created")
		}
		if err != nil {
			t.Fatal(err)
		}
		got := ss[0]
		t.Logf("%v+", ss)
		want := &types.SavedSearch{ID: 5, Description: "test query", Query: "test type:diff", Notify: false, NotifySlack: false, UserID: &u.ID, OrgID: nil, SlackWebhookURL: nil}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})
}

func TestMigrateSavedQueryIntoDB(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()

	check := func(t *testing.T, settings *api.Settings, wantChanged bool) {
		t.Helper()
		if changed, err := migrateSavedQueryIntoDB(ctx, settings); err != nil {
			t.Fatal(err)
		} else if changed != wantChanged {
			t.Errorf("got changed %v, want %v", changed, wantChanged)
		}
	}

	t.Run("user needing migration", func(t *testing.T) {
		u, err := db.Users.Create(ctx, db.NewUser{Username: "u1"})
		if err != nil {
			t.Fatal(err)
		}
		settings, err := db.Settings.CreateIfUpToDate(ctx, api.SettingsSubject{User: &u.ID}, nil, nil, `{"search.savedQueries": [{"key": "1a2b3c", "description": "test query", "query": "test type:diff"}]}`)
		if err != nil {
			t.Fatal(err)
		}
		check(t, settings, true)
	})

	t.Run("org needing migration", func(t *testing.T) {
		org, err := db.Orgs.Create(ctx, "myorg", nil)
		if err != nil {
			t.Fatal(err)
		}
		settings, err := db.Settings.CreateIfUpToDate(ctx, api.SettingsSubject{Org: &org.ID}, nil, nil, `{"search.savedQueries": [{"key": "1a2b3c", "description": "test query", "query": "test type:diff"}]}`)
		if err != nil {
			t.Fatal(err)
		}
		if err := doMigrateSavedQueriesAndSlackWebhookURLsFromSettingsToDatabase(ctx); err != nil {
			t.Fatal(err)
		}
		check(t, settings, true)
	})

	t.Run("global settings needing migration", func(t *testing.T) {
		settings, err := db.Settings.CreateIfUpToDate(ctx, api.SettingsSubject{}, nil, nil, `{"search.savedQueries": [{"key": "1a2b3c", "description": "test query", "query": "test type:diff"}]}`)
		if err != nil {
			t.Fatal(err)
		}
		if err := doMigrateSavedQueriesAndSlackWebhookURLsFromSettingsToDatabase(ctx); err != nil {
			t.Fatal(err)
		}
		check(t, settings, true)
	})
}

func TestInsertSavedQueryIntoDB(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()

	t.Run("user saved search inserted into db table", func(t *testing.T) {
		u, err := db.Users.Create(ctx, db.NewUser{Username: "u1"})
		if err != nil {
			t.Fatal(err)
		}
		settings, err := db.Settings.CreateIfUpToDate(ctx, api.SettingsSubject{User: &u.ID}, nil, nil, `{"search.savedQueries": [{"key": "1a2b3c", "description": "test query", "query": "test type:diff"}]}`)
		if err != nil {
			t.Fatal(err)
		}

		if err := insertSavedQueryIntoDB(ctx, settings, &savedQueryField{SavedQueries: []savedQuery{{Key: "1a2b3c", Description: "test query", Query: "test type:diff"}}}); err != nil {
			t.Fatal(err)
		}
		ss, err := db.SavedSearches.ListSavedSearchesByUserID(ctx, u.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(ss) == 0 {
			t.Fatal("Error: user saved query not created")
		}
		got := ss[0]
		want := &types.SavedSearch{ID: 1, Description: "test query", Query: "test type:diff", Notify: false, NotifySlack: false, UserID: &u.ID, OrgID: nil, SlackWebhookURL: nil}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("org saved search inserted into db table", func(t *testing.T) {
		org, err := db.Orgs.Create(ctx, "myorg", nil)
		if err != nil {
			t.Fatal(err)
		}
		settings, err := db.Settings.CreateIfUpToDate(ctx, api.SettingsSubject{Org: &org.ID}, nil, nil, `{"search.savedQueries": [{"key": "1a2b3c", "description": "test query", "query": "test type:diff"}]}`)
		if err != nil {
			t.Fatal(err)
		}

		if err := insertSavedQueryIntoDB(ctx, settings, &savedQueryField{SavedQueries: []savedQuery{{Key: "1a2b3c", Description: "test query", Query: "test type:diff"}}}); err != nil {
			t.Fatal(err)
		}
		ss, err := db.SavedSearches.ListSavedSearchesByOrgID(ctx, org.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(ss) == 0 {
			t.Fatal("Error: organization saved query not created")
		}
		got := ss[0]
		want := &types.SavedSearch{ID: 2, Description: "test query", Query: "test type:diff", Notify: false, NotifySlack: false, UserID: nil, OrgID: &org.ID, SlackWebhookURL: nil}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %+v, want %+v", got, want)
		}
	})

	t.Run("global saved search inserted into db table under site admin user", func(t *testing.T) {
		// The site-admin is the first user we created in this DB, and will have an ID of 1.
		user, err := db.Users.GetByID(ctx, 1)
		if err != nil {
			t.Fatal(err)
		}
		settings, err := db.Settings.CreateIfUpToDate(ctx, api.SettingsSubject{User: &user.ID}, nil, nil, `{"search.savedQueries": [{"key": "1a2b3c", "description": "test query", "query": "test type:diff"}]}`)
		if err != nil {
			t.Fatal(err)
		}

		if err := insertSavedQueryIntoDB(ctx, settings, &savedQueryField{SavedQueries: []savedQuery{{Key: "1a2b3c", Description: "test query", Query: "test type:diff"}}}); err != nil {
			t.Fatal(err)
		}
		ss, err := db.SavedSearches.ListSavedSearchesByUserID(ctx, user.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(ss) == 0 {
			t.Fatal("Error: global saved query not created")
		}
		got := ss[0]
		want := &types.SavedSearch{ID: 1, Description: "test query", Query: "test type:diff", Notify: false, NotifySlack: false, UserID: &user.ID, OrgID: nil, SlackWebhookURL: nil}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
		t.Log(db.SavedSearches.ListSavedSearchesByUserID(ctx, user.ID))
	})
}

func TestMigrateSlackWebhookURL(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()

	t.Run("migrate user slack webhook URL", func(t *testing.T) {
		u, err := db.Users.Create(ctx, db.NewUser{Username: "u1"})
		if err != nil {
			t.Fatal(err)
		}

		// Migrate saved query without Slack webhook URL into DB.
		latest, err := db.Settings.CreateIfUpToDate(ctx, api.SettingsSubject{User: &u.ID}, nil, nil, `{"search.savedQueries": [{"key": "1a2b3c", "description": "test query", "query": "test type:diff"}]}`)
		if err != nil {
			t.Fatal(err)
		}
		if err := doMigrateSavedQueriesAndSlackWebhookURLsFromSettingsToDatabase(ctx); err != nil {
			t.Fatal(err)
		}

		// Create settings with notifications.slack for the same user.
		_, err = db.Settings.CreateIfUpToDate(ctx, api.SettingsSubject{User: &u.ID}, &latest.ID, nil, `{"notifications.slack": {"webhookURL": "https://test.slackwebhook.com"}}`)
		if err != nil {
			t.Fatal(err)
		}

		err = migrateSlackWebhookUrlsToSavedSearches(ctx)
		if err != nil {
			t.Fatal(err)
		}

		ss, err := db.SavedSearches.ListSavedSearchesByUserID(ctx, u.ID)
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range ss {
			t.Logf("%v+", s)
		}

		got := ss[0]
		webhookURL := "https://test.slackwebhook.com"
		want := &types.SavedSearch{ID: 1, Description: "test query", Query: "test type:diff", Notify: false, NotifySlack: false, UserID: &u.ID, OrgID: nil, SlackWebhookURL: &webhookURL}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("invalid slack webhook URL", func(t *testing.T) {
		u, err := db.Users.Create(ctx, db.NewUser{Username: "u2"})
		if err != nil {
			t.Fatal(err)
		}

		// Migrate saved query without Slack webhook URL into DB.
		latest, err := db.Settings.CreateIfUpToDate(ctx, api.SettingsSubject{User: &u.ID}, nil, nil, `{"search.savedQueries": [{"key": "1a2b3c", "description": "test query", "query": "test type:diff"}]}`)
		if err != nil {
			t.Fatal(err)
		}
		if err := doMigrateSavedQueriesAndSlackWebhookURLsFromSettingsToDatabase(ctx); err != nil {
			t.Fatal(err)
		}

		// Create invalid JSON settings with notifications.slack for the same user.
		_, err = db.Settings.CreateIfUpToDate(ctx, api.SettingsSubject{User: &u.ID}, &latest.ID, nil, `{"notifications.slack": {"webhookURL": [1]}}`)
		if err != nil {
			t.Fatal(err)
		}

		err = migrateSlackWebhookUrlsToSavedSearches(ctx)
		if err != nil {
			t.Fatal(err)
		}

		ss, err := db.SavedSearches.ListSavedSearchesByUserID(ctx, u.ID)
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range ss {
			t.Logf("%v+", s)
		}

		got := ss[0]
		want := &types.SavedSearch{ID: 2, Description: "test query", Query: "test type:diff", Notify: false, NotifySlack: false, UserID: &u.ID, OrgID: nil, SlackWebhookURL: nil}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("migrate org slack webhook URL", func(t *testing.T) {
		org, err := db.Orgs.Create(ctx, "myorg", nil)
		if err != nil {
			t.Fatal(err)
		}

		// Migrate saved query without Slack webhook URL into DB.
		latest, err := db.Settings.CreateIfUpToDate(ctx, api.SettingsSubject{Org: &org.ID}, nil, nil, `{"search.savedQueries": [{"key": "1a2b3c", "description": "test query", "query": "test type:diff"}]}`)
		if err != nil {
			t.Fatal(err)
		}
		if err := doMigrateSavedQueriesAndSlackWebhookURLsFromSettingsToDatabase(ctx); err != nil {
			t.Fatal(err)
		}

		// Create settings with notifications.slack for the same org.
		_, err = db.Settings.CreateIfUpToDate(ctx, api.SettingsSubject{Org: &org.ID}, &latest.ID, nil, `{"notifications.slack": {"webhookURL": "https://test.slackwebhook.com"}}`)
		if err != nil {
			t.Fatal(err)
		}

		err = migrateSlackWebhookUrlsToSavedSearches(ctx)
		if err != nil {
			t.Fatal(err)
		}

		ss, err := db.SavedSearches.ListSavedSearchesByOrgID(ctx, org.ID)
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range ss {
			t.Logf("%v+", s)
		}

		got := ss[0]
		webhookURL := "https://test.slackwebhook.com"
		want := &types.SavedSearch{ID: 3, Description: "test query", Query: "test type:diff", Notify: false, NotifySlack: false, UserID: nil, OrgID: &org.ID, SlackWebhookURL: &webhookURL}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("migrate global slack webhook URL", func(t *testing.T) {
		// Get the site admin user.
		user, err := db.Users.GetByUsername(ctx, "u1")
		if err != nil {
			t.Fatal(err)
		}

		// Create global settings with notifications.slack.
		_, err = db.Settings.CreateIfUpToDate(ctx, api.SettingsSubject{}, nil, nil, `{"notifications.slack": {"webhookURL": "https://test.slackwebhook.com"}}`)
		if err != nil {
			t.Fatal(err)
		}

		err = migrateSlackWebhookUrlsToSavedSearches(ctx)
		if err != nil {
			t.Fatal(err)
		}

		ss, err := db.SavedSearches.ListSavedSearchesByUserID(ctx, user.ID)
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range ss {
			t.Logf("%v+", s)
		}

		got := ss[0]
		webhookURL := "https://test.slackwebhook.com"
		want := &types.SavedSearch{ID: 1, Description: "test query", Query: "test type:diff", Notify: false, NotifySlack: false, UserID: &user.ID, OrgID: nil, SlackWebhookURL: &webhookURL}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("invalid slack webhook URL", func(t *testing.T) {
		u, err := db.Users.Create(ctx, db.NewUser{Username: "u3"})
		if err != nil {
			t.Fatal(err)
		}
		u2, err := db.Users.Create(ctx, db.NewUser{Username: "u4"})
		if err != nil {
			t.Fatal(err)
		}

		// Migrate saved query without Slack webhook URL into DB.
		latest, err := db.Settings.CreateIfUpToDate(ctx, api.SettingsSubject{User: &u.ID}, nil, nil, `{"search.savedQueries": [{"key": "1a2b3c", "description": "test query", "query": "test type:diff"}]}`)
		if err != nil {
			t.Fatal(err)
		}
		latest2, err := db.Settings.CreateIfUpToDate(ctx, api.SettingsSubject{User: &u2.ID}, nil, nil, `{"search.savedQueries": [{"key": "1a2b3c4d", "description": "test query2", "query": "test2 type:diff"}]}`)
		if err != nil {
			t.Fatal(err)
		}

		if err := doMigrateSavedQueriesAndSlackWebhookURLsFromSettingsToDatabase(ctx); err != nil {
			t.Fatal(err)
		}
		// Create valid JSON settings with notifications.slack for one user.
		_, err = db.Settings.CreateIfUpToDate(ctx, api.SettingsSubject{User: &u.ID}, &latest.ID, nil, `{"notifications.slack": {"webhookURL": "https://test.slackwebhook.com"}}`)
		if err != nil {
			t.Fatal(err)
		}
		// Create invalid JSON settings with notifications.slack for one user.
		_, err = db.Settings.CreateIfUpToDate(ctx, api.SettingsSubject{User: &u2.ID}, &latest2.ID, nil, `{"notifications.slack": {"webhookURL": [1]}}`)
		if err != nil {
			t.Fatal(err)
		}

		err = migrateSlackWebhookUrlsToSavedSearches(ctx)
		if err != nil {
			t.Fatal(err)
		}

		ss, err := db.SavedSearches.ListSavedSearchesByUserID(ctx, u.ID)
		if err != nil {
			t.Fatal(err)
		}

		got := ss[0]
		webhookURL := "https://test.slackwebhook.com"
		want := &types.SavedSearch{ID: 4, Description: "test query", Query: "test type:diff", Notify: false, NotifySlack: false, UserID: &u.ID, OrgID: nil, SlackWebhookURL: &webhookURL}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}

		ss2, err := db.SavedSearches.ListSavedSearchesByUserID(ctx, u2.ID)
		if err != nil {
			t.Fatal(err)
		}

		got2 := ss2[0]
		want2 := &types.SavedSearch{ID: 5, Description: "test query2", Query: "test2 type:diff", Notify: false, NotifySlack: false, UserID: &u2.ID, OrgID: nil, SlackWebhookURL: nil}
		if !reflect.DeepEqual(got2, want2) {
			t.Fatalf("got2 %v, want2 %v", got2, want2)
		}
	})
}
