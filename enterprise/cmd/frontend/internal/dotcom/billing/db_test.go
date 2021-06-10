package billing

import (
	"context"
	"testing"

	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
)

func TestDBUsersBillingCustomerID(t *testing.T) {
	db := dbtest.NewDB(t, "")
	ctx := context.Background()

	t.Run("existing user", func(t *testing.T) {
		u, err := database.Users(db).Create(ctx, database.NewUser{Username: "u"})
		if err != nil {
			t.Fatal(err)
		}

		if custID, err := (dbBilling{db: db}).getUserBillingCustomerID(ctx, u.ID); err != nil {
			t.Fatal(err)
		} else if custID != nil {
			t.Errorf("got %q, want nil", *custID)
		}

		t.Run("set to non-nil", func(t *testing.T) {
			if err := (dbBilling{db: db}).setUserBillingCustomerID(ctx, u.ID, strptr("x")); err != nil {
				t.Fatal(err)
			}
			if custID, err := (dbBilling{db: db}).getUserBillingCustomerID(ctx, u.ID); err != nil {
				t.Fatal(err)
			} else if want := "x"; custID == nil || *custID != want {
				t.Errorf("got %v, want %q", custID, want)
			}
		})

		t.Run("set to nil", func(t *testing.T) {
			if err := (dbBilling{db: db}).setUserBillingCustomerID(ctx, u.ID, nil); err != nil {
				t.Fatal(err)
			}
			if custID, err := (dbBilling{db: db}).getUserBillingCustomerID(ctx, u.ID); err != nil {
				t.Fatal(err)
			} else if custID != nil {
				t.Errorf("got %q, want nil", *custID)
			}
		})
	})

	t.Run("nonexistent user", func(t *testing.T) {
		if _, err := (dbBilling{db: db}).getUserBillingCustomerID(ctx, 123 /* doesn't exist */); !errcode.IsNotFound(err) {
			t.Errorf("got %v, want errcode.IsNotFound(err) == true", err)
		}
		if err := (dbBilling{db: db}).setUserBillingCustomerID(ctx, 123 /* doesn't exist */, strptr("x")); !errcode.IsNotFound(err) {
			t.Errorf("got %v, want errcode.IsNotFound(err) == true", err)
		}
	})
}

func strptr(s string) *string {
	return &s
}
