package productsubscription

import (
	"context"
	"testing"

	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
)

func TestProductLicenses_Create(t *testing.T) {
	db := dbtest.NewDB(t, "")
	ctx := context.Background()

	u, err := database.Users(db).Create(ctx, database.NewUser{Username: "u"})
	if err != nil {
		t.Fatal(err)
	}

	ps0, err := dbSubscriptions{db: db}.Create(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}

	pl0, err := dbLicenses{db: db}.Create(ctx, ps0, "k")
	if err != nil {
		t.Fatal(err)
	}

	got, err := dbLicenses{db: db}.GetByID(ctx, pl0)
	if err != nil {
		t.Fatal(err)
	}
	if want := pl0; got.ID != want {
		t.Errorf("got %v, want %v", got.ID, want)
	}
	if want := ps0; got.ProductSubscriptionID != want {
		t.Errorf("got %v, want %v", got.ProductSubscriptionID, want)
	}
	if want := "k"; got.LicenseKey != want {
		t.Errorf("got %q, want %q", got.LicenseKey, want)
	}

	ts, err := dbLicenses{db: db}.List(ctx, dbLicensesListOptions{ProductSubscriptionID: ps0})
	if err != nil {
		t.Fatal(err)
	}
	if want := 1; len(ts) != want {
		t.Errorf("got %d product licenses, want %d", len(ts), want)
	}

	ts, err = dbLicenses{db: db}.List(ctx, dbLicensesListOptions{ProductSubscriptionID: "69da12d5-323c-4e42-9d44-cc7951639bca" /* invalid */})
	if err != nil {
		t.Fatal(err)
	}
	if want := 0; len(ts) != want {
		t.Errorf("got %d product licenses, want %d", len(ts), want)
	}
}

func TestProductLicenses_List(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := dbtest.NewDB(t, "")
	ctx := context.Background()

	u1, err := database.Users(db).Create(ctx, database.NewUser{Username: "u1"})
	if err != nil {
		t.Fatal(err)
	}

	ps0, err := dbSubscriptions{db: db}.Create(ctx, u1.ID)
	if err != nil {
		t.Fatal(err)
	}
	ps1, err := dbSubscriptions{db: db}.Create(ctx, u1.ID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = dbLicenses{db: db}.Create(ctx, ps0, "k")
	if err != nil {
		t.Fatal(err)
	}
	_, err = dbLicenses{db: db}.Create(ctx, ps0, "n1")
	if err != nil {
		t.Fatal(err)
	}

	{
		// List all product licenses.
		ts, err := dbLicenses{db: db}.List(ctx, dbLicensesListOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if want := 2; len(ts) != want {
			t.Errorf("got %d product licenses, want %d", len(ts), want)
		}
		count, err := dbLicenses{db: db}.Count(ctx, dbLicensesListOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if want := 2; count != want {
			t.Errorf("got %d, want %d", count, want)
		}
	}

	{
		// List ps0's product licenses.
		ts, err := dbLicenses{db: db}.List(ctx, dbLicensesListOptions{ProductSubscriptionID: ps0})
		if err != nil {
			t.Fatal(err)
		}
		if want := 2; len(ts) != want {
			t.Errorf("got %d product licenses, want %d", len(ts), want)
		}
	}

	{
		// List ps1's product licenses.
		ts, err := dbLicenses{db: db}.List(ctx, dbLicensesListOptions{ProductSubscriptionID: ps1})
		if err != nil {
			t.Fatal(err)
		}
		if want := 0; len(ts) != want {
			t.Errorf("got %d product licenses, want %d", len(ts), want)
		}
	}
}
