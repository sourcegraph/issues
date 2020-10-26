package billing

import (
	"context"
	"fmt"
	"testing"

	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
)

func TestGetOrAssignUserCustomerID(t *testing.T) {
	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()

	c := 0
	mockCreateCustomerID = func(userID int32) (string, error) {
		c++
		return fmt.Sprintf("cust%d", c), nil
	}
	defer func() { mockCreateCustomerID = nil }()

	u, err := db.Users.Create(ctx, db.NewUser{Username: "u"})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("assigns and retrieves", func(t *testing.T) {
		custID1, err := GetOrAssignUserCustomerID(ctx, u.ID)
		if err != nil {
			t.Fatal(err)
		}
		custID2, err := GetOrAssignUserCustomerID(ctx, u.ID)
		if err != nil {
			t.Fatal(err)
		}
		if custID2 != custID1 {
			t.Errorf("got custID %q, want %q", custID2, custID2)
		}
	})

	t.Run("fails on nonexistent users", func(t *testing.T) {
		if _, err := GetOrAssignUserCustomerID(ctx, 123 /* no such user */); err == nil {
			t.Fatal("err == nil")
		}
	})
}
