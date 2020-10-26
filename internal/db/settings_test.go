package db

import (
	"context"
	"testing"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
)

func TestSettings_ListAll(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()

	user1, err := Users.Create(ctx, NewUser{Username: "u1"})
	if err != nil {
		t.Fatal(err)
	}
	user2, err := Users.Create(ctx, NewUser{Username: "u2"})
	if err != nil {
		t.Fatal(err)
	}

	// Try creating both with non-nil author and nil author.
	if _, err := Settings.CreateIfUpToDate(ctx, api.SettingsSubject{User: &user1.ID}, nil, &user1.ID, `{"abc": 1}`); err != nil {
		t.Fatal(err)
	}
	if _, err := Settings.CreateIfUpToDate(ctx, api.SettingsSubject{User: &user2.ID}, nil, nil, `{"xyz": 2}`); err != nil {
		t.Fatal(err)
	}

	t.Run("all", func(t *testing.T) {
		settings, err := Settings.ListAll(ctx, "")
		if err != nil {
			t.Fatal(err)
		}
		if want := 2; len(settings) != want {
			t.Errorf("got %d settings, want %d", len(settings), want)
		}
	})

	t.Run("impreciseSubstring", func(t *testing.T) {
		settings, err := Settings.ListAll(ctx, "xyz")
		if err != nil {
			t.Fatal(err)
		}
		if want := 1; len(settings) != want {
			t.Errorf("got %d settings, want %d", len(settings), want)
		}
		if want := `{"xyz": 2}`; settings[0].Contents != want {
			t.Errorf("got contents %q, want %q", settings[0].Contents, want)
		}
	})
}
