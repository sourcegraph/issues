package productsubscription

import (
	"context"
	"database/sql"
	"testing"

	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
)

func TestProductSubscriptions_Create(t *testing.T) {
	db := dbtest.NewDB(t, "")
	ctx := context.Background()

	u, err := database.Users(db).Create(ctx, database.NewUser{Username: "u"})
	if err != nil {
		t.Fatal(err)
	}

	sub0, err := dbSubscriptions{db: db}.Create(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}

	got, err := dbSubscriptions{db: db}.GetByID(ctx, sub0)
	if err != nil {
		t.Fatal(err)
	}
	if want := sub0; got.ID != want {
		t.Errorf("got %v, want %v", got.ID, want)
	}
	if want := u.ID; got.UserID != want {
		t.Errorf("got %v, want %v", got.UserID, want)
	}
	if got.BillingSubscriptionID != nil {
		t.Errorf("got %v, want nil", got.BillingSubscriptionID)
	}

	ts, err := dbSubscriptions{db: db}.List(ctx, dbSubscriptionsListOptions{UserID: u.ID})
	if err != nil {
		t.Fatal(err)
	}
	if want := 1; len(ts) != want {
		t.Errorf("got %d product subscriptions, want %d", len(ts), want)
	}

	ts, err = dbSubscriptions{db: db}.List(ctx, dbSubscriptionsListOptions{UserID: 123 /* invalid */})
	if err != nil {
		t.Fatal(err)
	}
	if want := 0; len(ts) != want {
		t.Errorf("got %d product subscriptions, want %d", len(ts), want)
	}
}

func TestProductSubscriptions_List(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := dbtest.NewDB(t, "")
	ctx := context.Background()

	u1, err := database.Users(db).Create(ctx, database.NewUser{Username: "u1"})
	if err != nil {
		t.Fatal(err)
	}
	u2, err := database.Users(db).Create(ctx, database.NewUser{Username: "u2"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = dbSubscriptions{db: db}.Create(ctx, u1.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = dbSubscriptions{db: db}.Create(ctx, u1.ID)
	if err != nil {
		t.Fatal(err)
	}

	{
		// List all product subscriptions.
		ts, err := dbSubscriptions{db: db}.List(ctx, dbSubscriptionsListOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if want := 2; len(ts) != want {
			t.Errorf("got %d product subscriptions, want %d", len(ts), want)
		}
		count, err := dbSubscriptions{db: db}.Count(ctx, dbSubscriptionsListOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if want := 2; count != want {
			t.Errorf("got %d, want %d", count, want)
		}
	}

	{
		// List u1's product subscriptions.
		ts, err := dbSubscriptions{db: db}.List(ctx, dbSubscriptionsListOptions{UserID: u1.ID})
		if err != nil {
			t.Fatal(err)
		}
		if want := 2; len(ts) != want {
			t.Errorf("got %d product subscriptions, want %d", len(ts), want)
		}
	}

	{
		// List u2's product subscriptions.
		ts, err := dbSubscriptions{db: db}.List(ctx, dbSubscriptionsListOptions{UserID: u2.ID})
		if err != nil {
			t.Fatal(err)
		}
		if want := 0; len(ts) != want {
			t.Errorf("got %d product subscriptions, want %d", len(ts), want)
		}
	}
}

func TestProductSubscriptions_Update(t *testing.T) {
	db := dbtest.NewDB(t, "")
	ctx := context.Background()

	u, err := database.Users(db).Create(ctx, database.NewUser{Username: "u"})
	if err != nil {
		t.Fatal(err)
	}

	sub0, err := dbSubscriptions{db: db}.Create(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := (dbSubscriptions{db: db}).GetByID(ctx, sub0); err != nil {
		t.Fatal(err)
	} else if got.BillingSubscriptionID != nil {
		t.Errorf("got %q, want nil", *got.BillingSubscriptionID)
	}

	// Set non-null value.
	if err := (dbSubscriptions{db: db}).Update(ctx, sub0, dbSubscriptionUpdate{
		billingSubscriptionID: &sql.NullString{
			String: "x",
			Valid:  true,
		},
	}); err != nil {
		t.Fatal(err)
	}
	if got, err := (dbSubscriptions{db: db}).GetByID(ctx, sub0); err != nil {
		t.Fatal(err)
	} else if want := "x"; got.BillingSubscriptionID == nil || *got.BillingSubscriptionID != want {
		t.Errorf("got %v, want %q", got.BillingSubscriptionID, want)
	}

	// Update no fields.
	if err := (dbSubscriptions{db: db}).Update(ctx, sub0, dbSubscriptionUpdate{billingSubscriptionID: nil}); err != nil {
		t.Fatal(err)
	}
	if got, err := (dbSubscriptions{db: db}).GetByID(ctx, sub0); err != nil {
		t.Fatal(err)
	} else if want := "x"; got.BillingSubscriptionID == nil || *got.BillingSubscriptionID != want {
		t.Errorf("got %v, want %q", got.BillingSubscriptionID, want)
	}

	// Set null value.
	if err := (dbSubscriptions{db: db}).Update(ctx, sub0, dbSubscriptionUpdate{
		billingSubscriptionID: &sql.NullString{Valid: false},
	}); err != nil {
		t.Fatal(err)
	}
	if got, err := (dbSubscriptions{db: db}).GetByID(ctx, sub0); err != nil {
		t.Fatal(err)
	} else if got.BillingSubscriptionID != nil {
		t.Errorf("got %q, want nil", *got.BillingSubscriptionID)
	}
}
