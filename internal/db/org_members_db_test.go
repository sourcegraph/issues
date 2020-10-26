package db

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
)

func TestOrgMembers_CreateMembershipInOrgsForAllUsers(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()

	// Create fixtures.
	org1, err := Orgs.Create(ctx, "org1", nil)
	if err != nil {
		t.Fatal(err)
	}
	org2, err := Orgs.Create(ctx, "org2", nil)
	if err != nil {
		t.Fatal(err)
	}
	org3, err := Orgs.Create(ctx, "org3", nil)
	if err != nil {
		t.Fatal(err)
	}
	user1, err := Users.Create(ctx, NewUser{
		Email:                 "a1@example.com",
		Username:              "u1",
		Password:              "p",
		EmailVerificationCode: "c",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Users.Create(ctx, NewUser{
		Email:                 "a2@example.com",
		Username:              "u2",
		Password:              "p",
		EmailVerificationCode: "c",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := OrgMembers.Create(ctx, org1.ID, user1.ID); err != nil {
		t.Fatal(err)
	}

	check := func() error {
		want := map[string][]int32{
			"org1": {1, 2},
			"org2": {},
			"org3": {1, 2},
		}
		got := map[string][]int32{}
		for _, org := range []*types.Org{org1, org2, org3} {
			members, err := OrgMembers.GetByOrgID(ctx, org.ID)
			if err != nil {
				return err
			}
			if len(members) == 0 {
				got[org.Name] = []int32{}
			}
			for _, member := range members {
				got[org.Name] = append(got[org.Name], member.UserID)
			}
		}
		if !reflect.DeepEqual(got, want) {
			return fmt.Errorf("got membership %+v, want %+v", got, want)
		}
		return nil
	}

	// Try twice; it should be idempotent.
	if err := OrgMembers.CreateMembershipInOrgsForAllUsers(ctx, nil, []string{"org1", "org3"}); err != nil {
		t.Fatal(err)
	}
	if err := check(); err != nil {
		t.Fatal(err)
	}
	if err := OrgMembers.CreateMembershipInOrgsForAllUsers(ctx, nil, []string{"org1", "org3"}); err != nil {
		t.Fatal(err)
	}
	if err := check(); err != nil {
		t.Fatal(err)
	}

	// Passing an org that does not exist should not be an error.
	if err := OrgMembers.CreateMembershipInOrgsForAllUsers(ctx, nil, []string{"doesntexist"}); err != nil {
		t.Fatal(err)
	}
	if err := check(); err != nil {
		t.Fatal(err)
	}

	// An empty list shouldn't be an error.
	if err := OrgMembers.CreateMembershipInOrgsForAllUsers(ctx, nil, []string{}); err != nil {
		t.Fatal(err)
	}
	if err := check(); err != nil {
		t.Fatal(err)
	}
}
