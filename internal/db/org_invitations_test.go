package db

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
)

// 🚨 SECURITY: This tests the routine that creates org invitations and returns the invitation secret value
// to the user.
func TestOrgInvitations(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()

	sender, err := Users.Create(ctx, NewUser{
		Email:                 "a1@example.com",
		Username:              "u1",
		Password:              "p1",
		EmailVerificationCode: "c1",
	})
	if err != nil {
		t.Fatal(err)
	}

	recipient, err := Users.Create(ctx, NewUser{
		Email:                 "a2@example.com",
		Username:              "u2",
		Password:              "p2",
		EmailVerificationCode: "c2",
	})
	if err != nil {
		t.Fatal(err)
	}

	org1, err := Orgs.Create(ctx, "o1", nil)
	if err != nil {
		t.Fatal(err)
	}
	org2, err := Orgs.Create(ctx, "o2", nil)
	if err != nil {
		t.Fatal(err)
	}

	oi1, err := OrgInvitations.Create(ctx, org1.ID, sender.ID, recipient.ID)
	if err != nil {
		t.Fatal(err)
	}
	oi2, err := OrgInvitations.Create(ctx, org2.ID, sender.ID, recipient.ID)
	if err != nil {
		t.Fatal(err)
	}

	testGetByID := func(t *testing.T, id int64, want *OrgInvitation) {
		t.Helper()
		if oi, err := OrgInvitations.GetByID(ctx, id); err != nil {
			t.Fatal(err)
		} else if !reflect.DeepEqual(oi, want) {
			t.Errorf("got %+v, want %+v", oi, want)
		}
	}
	t.Run("GetByID", func(t *testing.T) {
		testGetByID(t, oi1.ID, oi1)
		testGetByID(t, oi2.ID, oi2)
		if _, err := OrgInvitations.GetByID(ctx, 12345 /* doesn't exist */); !errcode.IsNotFound(err) {
			t.Errorf("got err %v, want errcode.IsNotFound", err)
		}
	})

	testListCount := func(t *testing.T, opt OrgInvitationsListOptions, want []*OrgInvitation) {
		t.Helper()
		if ois, err := OrgInvitations.List(ctx, opt); err != nil {
			t.Fatal(err)
		} else if !reflect.DeepEqual(ois, want) {
			t.Errorf("got %+v, want %+v", ois, want)
		}
		if n, err := OrgInvitations.Count(ctx, opt); err != nil {
			t.Fatal(err)
		} else if want := len(want); n != want {
			t.Errorf("got %d, want %d", n, want)
		}
	}
	t.Run("List/Count all", func(t *testing.T) {
		testListCount(t, OrgInvitationsListOptions{}, []*OrgInvitation{oi1, oi2})
	})
	t.Run("List/Count by OrgID", func(t *testing.T) {
		testListCount(t, OrgInvitationsListOptions{OrgID: org1.ID}, []*OrgInvitation{oi1})
	})
	t.Run("List/Count by RecipientUserID", func(t *testing.T) {
		testListCount(t, OrgInvitationsListOptions{RecipientUserID: recipient.ID}, []*OrgInvitation{oi1, oi2})
	})

	t.Run("UpdateEmailSentTimestamp", func(t *testing.T) {
		if oi1.NotifiedAt != nil {
			t.Fatalf("failed precondition: oi.NotifiedAt == %q, want nil", *oi1.NotifiedAt)
		}
		if err := OrgInvitations.UpdateEmailSentTimestamp(ctx, oi1.ID); err != nil {
			t.Fatal(err)
		}
		oi, err := OrgInvitations.GetByID(ctx, oi1.ID)
		if err != nil {
			t.Fatal(err)
		}
		if oi.NotifiedAt == nil || time.Since(*oi.NotifiedAt) > 1*time.Minute {
			t.Fatalf("got NotifiedAt %v, want recent", oi.NotifiedAt)
		}

		// Update it again.
		prevNotifiedAt := *oi.NotifiedAt
		if err := OrgInvitations.UpdateEmailSentTimestamp(ctx, oi1.ID); err != nil {
			t.Fatal(err)
		}
		oi, err = OrgInvitations.GetByID(ctx, oi1.ID)
		if err != nil {
			t.Fatal(err)
		}
		if oi.NotifiedAt == nil || !oi.NotifiedAt.After(prevNotifiedAt) {
			t.Errorf("got NotifiedAt %v, want after %v", oi.NotifiedAt, prevNotifiedAt)
		}
	})

	testRespond := func(t *testing.T, oi *OrgInvitation, accepted bool) {
		if oi.RespondedAt != nil {
			t.Fatalf("failed precondition: oi.RespondedAt == %q, want nil", *oi.RespondedAt)
		}
		if got, err := OrgInvitations.GetPending(ctx, oi.OrgID, oi.RecipientUserID); err != nil {
			t.Fatal(err)
		} else if got.ID != oi.ID {
			t.Errorf("got %d, want %d", got.ID, oi.ID)
		}

		// Try responding with the wrong recipient user ID, which should fail.
		if _, err := OrgInvitations.Respond(ctx, oi.ID, 12345 /* invalid user */, accepted); !errcode.IsNotFound(err) {
			t.Errorf("got err %v, want errcode.IsNotFound", err)
		}

		if orgID, err := OrgInvitations.Respond(ctx, oi.ID, oi.RecipientUserID, accepted); err != nil {
			t.Fatal(err)
		} else if want := oi.OrgID; orgID != want {
			t.Errorf("got %v, want %v", orgID, want)
		}
		oi, err := OrgInvitations.GetByID(ctx, oi.ID)
		if err != nil {
			t.Fatal(err)
		}
		if oi.RespondedAt == nil || time.Since(*oi.RespondedAt) > 1*time.Minute {
			t.Errorf("got RespondedAt %v, want recent", oi.RespondedAt)
		}
		if oi.ResponseType == nil || *oi.ResponseType != accepted {
			t.Errorf("got ResponseType %v, want %v", oi.ResponseType, accepted)
		}

		// After responding, these should fail.
		if _, err := OrgInvitations.GetPending(ctx, oi.OrgID, oi.RecipientUserID); !errcode.IsNotFound(err) {
			t.Errorf("got err %v, want errcode.IsNotFound", err)
		}
		if _, err := OrgInvitations.Respond(ctx, oi.ID, oi.RecipientUserID, accepted); !errcode.IsNotFound(err) {
			t.Errorf("got err %v, want errcode.IsNotFound", err)
		}
	}
	t.Run("Respond true", func(t *testing.T) {
		testRespond(t, oi1, true)
	})
	t.Run("Respond true", func(t *testing.T) {
		testRespond(t, oi2, false)
	})

	t.Run("Revoke", func(t *testing.T) {
		org3, err := Orgs.Create(ctx, "o3", nil)
		if err != nil {
			t.Fatal(err)
		}
		oi3, err := OrgInvitations.Create(ctx, org3.ID, sender.ID, recipient.ID)
		if err != nil {
			t.Fatal(err)
		}

		if err := OrgInvitations.Revoke(ctx, oi3.ID); err != nil {
			t.Fatal(err)
		}

		// After revoking, these should fail.
		if _, err := OrgInvitations.GetPending(ctx, oi3.OrgID, oi3.RecipientUserID); !errcode.IsNotFound(err) {
			t.Errorf("got err %v, want errcode.IsNotFound", err)
		}
		if _, err := OrgInvitations.Respond(ctx, oi3.ID, recipient.ID, true); !errcode.IsNotFound(err) {
			t.Errorf("got err %v, want errcode.IsNotFound", err)
		}
	})
}
