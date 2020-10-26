package bg

import (
	"context"
	"testing"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
)

func TestMigrateAllSettingsMOTDToNotices(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()

	// In TestMigrateSettingsMOTDToNotices below, we test the actual migration in more detail. Here
	// we add 2 settings document that need migration (1 valid, 1 with an error). This makes this
	// test into an integration test, albeit one that's fast and simple. This is better than truly
	// mocking out the underlying migration func, which would add more complexity to the impl than
	// is warranted.

	t.Run("valid", func(t *testing.T) {
		u, err := db.Users.Create(ctx, db.NewUser{Username: "u1"})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Settings.CreateIfUpToDate(ctx, api.SettingsSubject{User: &u.ID}, nil, nil, `{"motd": ["a",], "notices": []}`); err != nil {
			t.Fatal(err)
		}
		if err := doMigrateAllSettingsMOTDToNotices(ctx, 0); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("no changes needed", func(t *testing.T) {
		u, err := db.Users.Create(ctx, db.NewUser{Username: "u2"})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Settings.CreateIfUpToDate(ctx, api.SettingsSubject{User: &u.ID}, nil, nil, `{"other": 123}`); err != nil {
			t.Fatal(err)
		}
		db.Mocks.Settings.CreateIfUpToDate = func(ctx context.Context, subject api.SettingsSubject, lastID, authorUserID *int32, contents string) (latestSetting *api.Settings, err error) {
			t.Fatal("want no settings changes")
			panic("unreachable")
		}
		defer func() { db.Mocks.Settings.CreateIfUpToDate = nil }()
		if err := doMigrateAllSettingsMOTDToNotices(ctx, 0); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("invalid", func(t *testing.T) {
		u, err := db.Users.Create(ctx, db.NewUser{Username: "u3"})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Settings.CreateIfUpToDate(ctx, api.SettingsSubject{User: &u.ID}, nil, nil, `{"motd": [1]}`); err != nil {
			t.Fatal(err)
		}
		if err := doMigrateAllSettingsMOTDToNotices(ctx, 0); err == nil {
			t.Error("want error due to invalid settings JSON")
		}
	})
}

func TestMigrateSettingsMOTDToNotices(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()

	check := func(t *testing.T, subject api.SettingsSubject, wantChanged bool, wantContents string) {
		t.Helper()
		if changed, err := migrateSettingsMOTDToNotices(ctx, subject); err != nil {
			t.Fatal(err)
		} else if changed != wantChanged {
			t.Errorf("got changed %v, want %v", changed, wantChanged)
		}
		if !wantChanged {
			return
		}
		s, err := db.Settings.GetLatest(ctx, subject)
		if err != nil {
			t.Fatal(err)
		}
		if s.Contents != wantContents {
			t.Errorf("got contents %q, want %q", s.Contents, wantContents)
		}
	}

	t.Run("user needing migration", func(t *testing.T) {
		u, err := db.Users.Create(ctx, db.NewUser{Username: "u1"})
		if err != nil {
			t.Fatal(err)
		}
		subject := api.SettingsSubject{User: &u.ID}
		if _, err := db.Settings.CreateIfUpToDate(ctx, subject, nil, nil, `{"motd": ["a",], "notices": []}`); err != nil {
			t.Fatal(err)
		}
		check(t, subject, true, `{
  "notices": [
    {
      "dismissible": true,
      "location": "top",
      "message": "a"
    }
  ]}`)
	})
	t.Run("user not needing migration", func(t *testing.T) {
		u, err := db.Users.Create(ctx, db.NewUser{Username: "u2"})
		if err != nil {
			t.Fatal(err)
		}
		subject := api.SettingsSubject{User: &u.ID}
		if _, err := db.Settings.CreateIfUpToDate(ctx, subject, nil, nil, `{}`); err != nil {
			t.Fatal(err)
		}
		check(t, subject, false, "")
	})
	t.Run("user with invalid settings JSON", func(t *testing.T) {
		u, err := db.Users.Create(ctx, db.NewUser{Username: "u3"})
		if err != nil {
			t.Fatal(err)
		}
		subject := api.SettingsSubject{User: &u.ID}
		const invalidSettingsJSON = `{"motd": [1]}`
		if _, err := db.Settings.CreateIfUpToDate(ctx, subject, nil, nil, invalidSettingsJSON); err != nil {
			t.Fatal(err)
		}
		if changed, err := migrateSettingsMOTDToNotices(ctx, subject); err == nil {
			t.Fatal("want error")
		} else if changed {
			t.Error("want unchanged")
		}
		// Ensure the settings were not changed (and remain in the original invalid state).
		s, err := db.Settings.GetLatest(ctx, subject)
		if err != nil {
			t.Fatal(err)
		}
		if s.Contents != invalidSettingsJSON {
			t.Errorf("got contents %q, want to be unchanged from %q", s.Contents, invalidSettingsJSON)
		}
	})
	t.Run("org needing migration", func(t *testing.T) {
		org, err := db.Orgs.Create(ctx, "myorg", nil)
		if err != nil {
			t.Fatal(err)
		}
		subject := api.SettingsSubject{Org: &org.ID}
		if _, err := db.Settings.CreateIfUpToDate(ctx, subject, nil, nil, `{"motd": ["b",]}`); err != nil {
			t.Fatal(err)
		}
		check(t, subject, true, `{
  "notices": [
    {
      "dismissible": true,
      "location": "top",
      "message": "b"
    }
  ]
}`)
	})
	t.Run("global settings needing migration", func(t *testing.T) {
		subject := api.SettingsSubject{} // global settings subject
		if _, err := db.Settings.CreateIfUpToDate(ctx, subject, nil, nil, `{"motd": ["c",], "notices": []}`); err != nil {
			t.Fatal(err)
		}
		check(t, subject, true, `{
  "notices": [
    {
      "dismissible": true,
      "location": "top",
      "message": "c"
    }
  ]}`)
	})
}

func TestMigrateSettingsMOTDToNoticesJSON(t *testing.T) {
	tests := map[string]struct {
		input string
		want  string
	}{
		"empty": {
			input: `{}`,
			want:  `{}`,
		},
		"neither motd nor notices": {
			input: `/* z */{"abc": 1 /* c */}`,
			want:  `/* z */{"abc": 1 /* c */}`,
		},
		"motd only": {
			input: `{"motd": ["x"]}`,
			want: `{
  "notices": [
    {
      "dismissible": true,
      "location": "top",
      "message": "x"
    }
  ]
}`,
		},
		"2 motds": {
			input: `{"motd": ["x", "z"]}`,
			want: `{
  "notices": [
    {
      "dismissible": true,
      "location": "top",
      "message": "x"
    },
    {
      "dismissible": true,
      "location": "top",
      "message": "z"
    }
  ]
}`,
		},
		"motd and notices": {
			input: `{"motd": ["x"], "notices": [{"location": "top", "message": "y"}]}`,
			want: `{
  "notices": [
    {
      "location": "top",
      "message": "y"
    },
    {
      "dismissible": true,
      "location": "top",
      "message": "x"
    }
  ]}`,
		},
		"2 motds and notices": {
			input: `{"motd": ["x", "z"], "notices": [{"location": "top", "message": "y"}]}`,
			want: `{
  "notices": [
    {
      "location": "top",
      "message": "y"
    },
    {
      "dismissible": true,
      "location": "top",
      "message": "x"
    },
    {
      "dismissible": true,
      "location": "top",
      "message": "z"
    }
  ]}`,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := migrateSettingsMOTDToNoticesJSON(test.input)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Errorf("got:\n%s\n\nwant:\n%s", got, test.want)
			}
		})
	}

	t.Run("error", func(t *testing.T) {
		_, err := migrateSettingsMOTDToNoticesJSON(`{`)
		if err == nil {
			t.Fatal()
		}
	})
}
