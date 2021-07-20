package database

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/go-cmp/cmp"

	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtesting"
	"github.com/sourcegraph/sourcegraph/internal/database/globalstatedb"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

// usernamesForTests is a list of test cases containing valid and invalid usernames and org names.
var usernamesForTests = []struct {
	name      string
	wantValid bool
}{
	{"nick", true},
	{"n1ck", true},
	{"Nick2", true},
	{"N-S", true},
	{"nick-s", true},
	{"renfred-xh", true},
	{"renfred-x-h", true},
	{"deadmau5", true},
	{"deadmau-5", true},
	{"3blindmice", true},
	{"nick.com", true},
	{"nick.com.uk", true},
	{"nick.com-put-er", true},
	{"nick-", true},
	{"777", true},
	{"7-7", true},
	{"long-butnotquitelongenoughtoreachlimit", true},

	{".nick", false},
	{"-nick", false},
	{"nick.", false},
	{"nick--s", false},
	{"nick--sny", false},
	{"nick..sny", false},
	{"nick.-sny", false},
	{"nick_s", false},
	{"_", false},
	{"_nick", false},
	{"nick_", false},
	{"ke$ha", false},
	{"ni%k", false},
	{"#nick", false},
	{"@nick", false},
	{"", false},
	{"nick s", false},
	{" ", false},
	{"-", false},
	{"--", false},
	{"-s", false},
	{"レンフレッド", false},
	{"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx", false},
}

func TestUsers_ValidUsernames(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Parallel()
	db := dbtest.NewDB(t, "")
	ctx := context.Background()

	for _, test := range usernamesForTests {
		t.Run(test.name, func(t *testing.T) {
			valid := true
			if _, err := Users(db).Create(ctx, NewUser{Username: test.name}); err != nil {
				var e errCannotCreateUser
				if errors.As(err, &e) && (e.Code() == "users_username_max_length" || e.Code() == "users_username_valid_chars") {
					valid = false
				} else {
					t.Fatal(err)
				}
			}
			if valid != test.wantValid {
				t.Errorf("%q: got valid %v, want %v", test.name, valid, test.wantValid)
			}
		})
	}
}

func TestUsers_Create_checkPasswordLength(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Parallel()
	db := dbtest.NewDB(t, "")
	ctx := context.Background()

	minPasswordRunes := conf.AuthMinPasswordLength()
	expErr := fmt.Sprintf("Password may not be less than %d or be more than %d characters.", minPasswordRunes, maxPasswordRunes)
	tests := []struct {
		name     string
		username string
		password string
		enforce  bool
		expErr   string
	}{
		{
			name:     "below minimum",
			username: "user1",
			password: strings.Repeat("x", minPasswordRunes-1),
			enforce:  true,
			expErr:   expErr,
		},
		{
			name:     "exceeds maximum",
			username: "user2",
			password: strings.Repeat("x", maxPasswordRunes+1),
			enforce:  true,
			expErr:   expErr,
		},

		{
			name:     "no problem at exact minimum",
			username: "user3",
			password: strings.Repeat("x", minPasswordRunes),
			enforce:  true,
			expErr:   "",
		},
		{
			name:     "no problem at exact maximum",
			username: "user4",
			password: strings.Repeat("x", maxPasswordRunes),
			enforce:  true,
			expErr:   "",
		},

		{
			name:     "does not enforce and below minimum",
			username: "user5",
			password: strings.Repeat("x", minPasswordRunes-1),
			enforce:  false,
			expErr:   "",
		},
		{
			name:     "does not enforce and exceeds maximum",
			username: "user6",
			password: strings.Repeat("x", maxPasswordRunes+1),
			enforce:  false,
			expErr:   "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Users(db).Create(ctx, NewUser{
				Username:              test.username,
				Password:              test.password,
				EnforcePasswordLength: test.enforce,
			})
			if pm := errcode.PresentationMessage(err); pm != test.expErr {
				t.Fatalf("err: want %q but got %q", test.expErr, pm)
			}
		})
	}
}

func TestUsers_Create_SiteAdmin(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := dbtesting.GetDB(t)
	ctx := context.Background()

	if _, err := globalstatedb.Get(ctx); err != nil {
		t.Fatal(err)
	}

	// Create site admin.
	user, err := Users(db).Create(ctx, NewUser{
		Email:                 "a@a.com",
		Username:              "u",
		Password:              "p",
		EmailVerificationCode: "c",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !user.SiteAdmin {
		t.Fatal("!user.SiteAdmin")
	}

	// Creating a non-site-admin now that the site has already been initialized.
	u2, err := Users(db).Create(ctx, NewUser{
		Email:                 "a2@a2.com",
		Username:              "u2",
		Password:              "p2",
		EmailVerificationCode: "c2",
	})
	if err != nil {
		t.Fatal(err)
	}
	if u2.SiteAdmin {
		t.Fatal("want u2 not site admin because site is already initialized")
	}
	// Similar to the above, but expect an error because we pass FailIfNotInitialUser: true.
	_, err = Users(db).Create(ctx, NewUser{
		Email:                 "a3@a3.com",
		Username:              "u3",
		Password:              "p3",
		EmailVerificationCode: "c3",
		FailIfNotInitialUser:  true,
	})
	if want := (errCannotCreateUser{"site_already_initialized"}); err != want {
		t.Fatalf("got error %v, want %v", err, want)
	}

	// Delete the site admin.
	if err := Users(db).Delete(ctx, user.ID); err != nil {
		t.Fatal(err)
	}

	// Disallow creating a site admin when a user already exists (even if the site is not yet initialized).
	if _, err := db.ExecContext(ctx, "UPDATE site_config SET initialized=false"); err != nil {
		t.Fatal(err)
	}
	u4, err := Users(db).Create(ctx, NewUser{
		Email:                 "a4@a4.com",
		Username:              "u4",
		Password:              "p4",
		EmailVerificationCode: "c4",
	})
	if err != nil {
		t.Fatal(err)
	}
	if u4.SiteAdmin {
		t.Fatal("want u4 not site admin because site is already initialized")
	}
	// Similar to the above, but expect an error because we pass FailIfNotInitialUser: true.
	if _, err := db.ExecContext(ctx, "UPDATE site_config SET initialized=false"); err != nil {
		t.Fatal(err)
	}
	_, err = Users(db).Create(ctx, NewUser{
		Email:                 "a5@a5.com",
		Username:              "u5",
		Password:              "p5",
		EmailVerificationCode: "c5",
		FailIfNotInitialUser:  true,
	})
	if want := (errCannotCreateUser{"initial_site_admin_must_be_first_user"}); err != want {
		t.Fatalf("got error %v, want %v", err, want)
	}
}

func TestUsers_CheckAndDecrementInviteQuota(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Parallel()
	db := dbtest.NewDB(t, "")
	ctx := context.Background()

	user, err := Users(db).Create(ctx, NewUser{
		Email:                 "a@a.com",
		Username:              "u",
		Password:              "p",
		EmailVerificationCode: "c",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Check default invite quota.
	var inviteQuota int
	row := db.QueryRowContext(ctx, "SELECT invite_quota FROM users WHERE id=$1", user.ID)
	if err := row.Scan(&inviteQuota); err != nil {
		t.Fatal(err)
	}
	// Check that it's within some reasonable bounds. The upper bound number here can increased
	// if we increase the default.
	if lo, hi := 0, 15; inviteQuota <= lo || inviteQuota > hi {
		t.Fatalf("got default user invite quota %d, want in [%d,%d)", inviteQuota, lo, hi)
	}

	// Decrementing should succeed while we have remaining quota. Keep going until we exhaust it.
	// Since the quota is fairly low, this isn't too slow.
	for inviteQuota > 0 {
		if ok, err := Users(db).CheckAndDecrementInviteQuota(ctx, user.ID); !ok || err != nil {
			t.Fatal("initial CheckAndDecrementInviteQuota failed:", err)
		}
		inviteQuota--
	}

	// Now our quota is exhausted, and CheckAndDecrementInviteQuota should fail.
	if ok, err := Users(db).CheckAndDecrementInviteQuota(ctx, user.ID); ok || err != nil {
		t.Fatalf("over-limit CheckAndDecrementInviteQuota #1: got error %v", err)
	}

	// Check again that we're still over quota, just in case.
	if ok, err := Users(db).CheckAndDecrementInviteQuota(ctx, user.ID); ok || err != nil {
		t.Fatalf("over-limit CheckAndDecrementInviteQuota #2: got error %v", err)
	}
}

func TestUsers_ListCount(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Parallel()
	db := dbtest.NewDB(t, "")
	ctx := context.Background()

	user, err := Users(db).Create(ctx, NewUser{
		Email:                 "a@a.com",
		Username:              "u",
		Password:              "p",
		EmailVerificationCode: "c",
	})
	if err != nil {
		t.Fatal(err)
	}
	user.Tags = []string{}

	if count, err := Users(db).Count(ctx, &UsersListOptions{}); err != nil {
		t.Fatal(err)
	} else if want := 1; count != want {
		t.Errorf("got %d, want %d", count, want)
	}
	if users, err := Users(db).List(ctx, &UsersListOptions{}); err != nil {
		t.Fatal(err)
	} else if users, want := normalizeUsers(users), normalizeUsers([]*types.User{user}); !reflect.DeepEqual(users, want) {
		t.Errorf("got %+v, want %+v", users, want)
	}

	if count, err := Users(db).Count(ctx, &UsersListOptions{UserIDs: []int32{}}); err != nil {
		t.Fatal(err)
	} else if want := 0; count != want {
		t.Errorf("got %d, want %d", count, want)
	}
	if users, err := Users(db).List(ctx, &UsersListOptions{UserIDs: []int32{}}); err != nil {
		t.Fatal(err)
	} else if len(users) > 0 {
		t.Errorf("got %d, want empty", len(users))
	}

	if users, err := Users(db).List(ctx, &UsersListOptions{}); err != nil {
		t.Fatal(err)
	} else if users, want := normalizeUsers(users), normalizeUsers([]*types.User{user}); !reflect.DeepEqual(users, want) {
		t.Errorf("got %+v, want %+v", users[0], user)
	}

	if err := Users(db).Delete(ctx, user.ID); err != nil {
		t.Fatal(err)
	}

	if count, err := Users(db).Count(ctx, &UsersListOptions{}); err != nil {
		t.Fatal(err)
	} else if want := 0; count != want {
		t.Errorf("got %d, want %d", count, want)
	}
}

func TestUsers_Update(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Parallel()
	db := dbtest.NewDB(t, "")
	ctx := context.Background()

	user, err := Users(db).Create(ctx, NewUser{
		Email:                 "a@a.com",
		Username:              "u",
		Password:              "p",
		EmailVerificationCode: "c",
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := Users(db).Update(ctx, user.ID, UserUpdate{
		Username:    "u1",
		DisplayName: strptr("d1"),
		AvatarURL:   strptr("a1"),
	}); err != nil {
		t.Fatal(err)
	}
	user, err = Users(db).GetByID(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if want := "u1"; user.Username != want {
		t.Errorf("got username %q, want %q", user.Username, want)
	}
	if want := "d1"; user.DisplayName != want {
		t.Errorf("got display name %q, want %q", user.DisplayName, want)
	}
	if want := "a1"; user.AvatarURL != want {
		t.Errorf("got avatar URL %q, want %q", user.AvatarURL, want)
	}

	if err := Users(db).Update(ctx, user.ID, UserUpdate{
		DisplayName: strptr(""),
	}); err != nil {
		t.Fatal(err)
	}
	user, err = Users(db).GetByID(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if want := "u1"; user.Username != want {
		t.Errorf("got username %q, want %q", user.Username, want)
	}
	if user.DisplayName != "" {
		t.Errorf("got display name %q, want nil", user.DisplayName)
	}
	if want := "a1"; user.AvatarURL != want {
		t.Errorf("got avatar URL %q, want %q", user.AvatarURL, want)
	}

	// Can't update to duplicate username.
	user2, err := Users(db).Create(ctx, NewUser{
		Email:                 "a2@a.com",
		Username:              "u2",
		Password:              "p2",
		EmailVerificationCode: "c2",
	})
	if err != nil {
		t.Fatal(err)
	}
	err = Users(db).Update(ctx, user2.ID, UserUpdate{Username: "u1"})
	if diff := cmp.Diff(err.Error(), "Username is already in use."); diff != "" {
		t.Fatal(diff)
	}

	// Can't update nonexistent user.
	if err := Users(db).Update(ctx, 12345, UserUpdate{Username: "u12345"}); err == nil {
		t.Fatal("want error when updating nonexistent user")
	}
}

func TestUsers_GetByVerifiedEmail(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Parallel()
	db := dbtest.NewDB(t, "")
	ctx := context.Background()

	user, err := Users(db).Create(ctx, NewUser{
		Email:                 "a@a.com",
		Username:              "u",
		Password:              "p",
		EmailVerificationCode: "c",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := Users(db).GetByVerifiedEmail(ctx, "a@a.com"); !errcode.IsNotFound(err) {
		t.Errorf("for unverified email, got error %v, want IsNotFound", err)
	}

	if err := UserEmails(db).SetVerified(ctx, user.ID, "a@a.com", true); err != nil {
		t.Fatal(err)
	}

	gotUser, err := Users(db).GetByVerifiedEmail(ctx, "a@a.com")
	if err != nil {
		t.Fatal(err)
	}
	if gotUser.ID != user.ID {
		t.Errorf("got user %d, want %d", gotUser.ID, user.ID)
	}
}

func TestUsers_GetByUsernames(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Parallel()
	db := dbtest.NewDB(t, "")
	ctx := context.Background()

	newUsers := []NewUser{
		{
			Email:           "alice@example.com",
			Username:        "alice",
			EmailIsVerified: true,
		},
		{
			Email:           "bob@example.com",
			Username:        "bob",
			EmailIsVerified: true,
		},
	}

	for _, newUser := range newUsers {
		_, err := Users(db).Create(ctx, newUser)
		if err != nil {
			t.Fatal(err)
		}
	}

	users, err := Users(db).GetByUsernames(ctx, "alice", "bob", "cindy")
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 {
		t.Fatalf("got %d users, but want 2", len(users))
	}
	for i := range users {
		if users[i].Username != newUsers[i].Username {
			t.Errorf("got %s, but want %s", users[i].Username, newUsers[i].Username)
		}
	}
}

func TestUsers_Delete(t *testing.T) {
	for name, hard := range map[string]bool{"soft": false, "hard": true} {
		t.Run(name, func(t *testing.T) {
			if testing.Short() {
				t.Skip()
			}
			t.Parallel()
			db := dbtest.NewDB(t, "")
			ctx := context.Background()
			ctx = actor.WithActor(ctx, &actor.Actor{UID: 1, Internal: true})

			otherUser, err := Users(db).Create(ctx, NewUser{Username: "other"})
			if err != nil {
				t.Fatal(err)
			}

			user, err := Users(db).Create(ctx, NewUser{
				Email:                 "a@a.com",
				Username:              "u",
				Password:              "p",
				EmailVerificationCode: "c",
			})
			if err != nil {
				t.Fatal(err)
			}

			// Create external service owned by the user
			confGet := func() *conf.Unified {
				return &conf.Unified{}
			}
			err = ExternalServices(db).Create(ctx, confGet, &types.ExternalService{
				Kind:            extsvc.KindGitHub,
				DisplayName:     "GITHUB #1",
				Config:          `{"url": "https://github.com", "repositoryQuery": ["none"], "token": "abc", "authorization": {}}`,
				NamespaceUserID: user.ID,
			})
			if err != nil {
				t.Fatal(err)
			}

			// Create settings for the user, and for another user authored by this user.
			if _, err := Settings(db).CreateIfUpToDate(ctx, api.SettingsSubject{User: &user.ID}, nil, &user.ID, "{}"); err != nil {
				t.Fatal(err)
			}
			if _, err := Settings(db).CreateIfUpToDate(ctx, api.SettingsSubject{User: &otherUser.ID}, nil, &user.ID, "{}"); err != nil {
				t.Fatal(err)
			}

			// Create a repository to comply with the postgres repo constraint.
			if err := Repos(db).Upsert(ctx, InsertRepoOp{Name: "myrepo", Description: "", Fork: false}); err != nil {
				t.Fatal(err)
			}

			// Create a saved search owned by the user.
			if _, err := SavedSearches(db).Create(ctx, &types.SavedSearch{
				Description: "desc",
				Query:       "foo",
				UserID:      &user.ID,
			}); err != nil {
				t.Fatal(err)
			}

			// Create an event log
			err = EventLogs(db).Insert(ctx, &Event{
				Name:            "something",
				URL:             "http://example.com",
				UserID:          uint32(user.ID),
				AnonymousUserID: "",
				Source:          "Test",
				Timestamp:       time.Now(),
			})
			if err != nil {
				t.Fatal(err)
			}

			if hard {
				// Hard delete user.
				if err := Users(db).HardDelete(ctx, user.ID); err != nil {
					t.Fatal(err)
				}
			} else {
				// Delete user.
				if err := Users(db).Delete(ctx, user.ID); err != nil {
					t.Fatal(err)
				}
			}

			// User no longer exists.
			_, err = Users(db).GetByID(ctx, user.ID)
			if !errcode.IsNotFound(err) {
				t.Errorf("got error %v, want ErrUserNotFound", err)
			}
			users, err := Users(db).List(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(users) > 1 {
				// The otherUser should still exist, which is why we check for 1 not 0.
				t.Errorf("got %d users, want 1", len(users))
			}

			// User's settings no longer exist.
			if settings, err := Settings(db).GetLatest(ctx, api.SettingsSubject{User: &user.ID}); err != nil {
				t.Error(err)
			} else if settings != nil {
				t.Errorf("got settings %+v, want nil", settings)
			}
			// Settings authored by user still exist but have nil author.
			if settings, err := Settings(db).GetLatest(ctx, api.SettingsSubject{User: &otherUser.ID}); err != nil {
				t.Fatal(err)
			} else if settings.AuthorUserID != nil {
				t.Errorf("got author %v, want nil", *settings.AuthorUserID)
			}

			// User's external services no longer exist
			ess, err := ExternalServices(db).List(ctx, ExternalServicesListOptions{
				NamespaceUserID: user.ID,
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(ess) > 0 {
				t.Errorf("got %d external services, want 0", len(ess))
			}

			// Can't delete already-deleted user.
			err = Users(db).Delete(ctx, user.ID)
			if !errcode.IsNotFound(err) {
				t.Errorf("got error %v, want ErrUserNotFound", err)
			}

			// Check event logs
			eventLogs, err := EventLogs(db).ListAll(ctx, EventLogsListOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if len(eventLogs) != 1 {
				t.Fatal("Expected 1 event log")
			}
			eventLog := eventLogs[0]
			if hard {
				// Event logs should now be anonymous
				if *eventLog.UserID != 0 {
					t.Error("After hard delete user id should be 0")
				}
				if len(eventLog.AnonymousUserID) == 0 {
					t.Error("After hard anonymous user id should not be blank")
				}
			} else {
				// Event logs are unchanged
				if *eventLog.UserID != user.ID {
					t.Error("After soft delete user id should be non zero")
				}
				if len(eventLog.AnonymousUserID) != 0 {
					t.Error("After soft delete anonymous user id should be blank")
				}
			}
		})
	}
}

func TestUsers_HasTag(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Parallel()
	db := dbtest.NewDB(t, "")
	ctx := context.Background()

	var id int32
	if err := db.QueryRowContext(ctx, "INSERT INTO users (username, tags) VALUES ('karim', '{\"foo\", \"bar\"}') RETURNING id").Scan(&id); err != nil {
		t.Fatal(err)
	}

	// lookup existing tag
	ok, err := Users(db).HasTag(ctx, id, "foo")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected tag to be found")
	}

	// lookup non-existing tag
	ok, err = Users(db).HasTag(ctx, id, "baz")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected tag to be not found")
	}

	// lookup non-existing user
	ok, err = Users(db).HasTag(ctx, id+1, "bar")
	if err == nil || ok {
		t.Fatal("expected user to be not found")
	}
}

func TestUsers_InvalidateSessions(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Parallel()
	db := dbtest.NewDB(t, "")
	ctx := context.Background()

	newUsers := []NewUser{
		{
			Email:           "alice@example.com",
			Username:        "alice",
			EmailIsVerified: true,
		},
		{
			Email:           "bob@example.com",
			Username:        "bob",
			EmailIsVerified: true,
		},
	}

	for _, newUser := range newUsers {
		_, err := Users(db).Create(ctx, newUser)
		if err != nil {
			t.Fatal(err)
		}
	}

	users, err := Users(db).GetByUsernames(ctx, "alice", "bob")
	if err != nil {
		t.Fatal(err)
	}

	if len(users) != 2 {
		t.Fatalf("got %d users, but want 2", len(users))
	}
	for i := range users {
		if err := Users(db).InvalidateSessionsByID(ctx, users[i].ID); err != nil {
			t.Fatal(err)
		}
	}
}

func TestUsers_SetTag(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Parallel()
	db := dbtest.NewDB(t, "")
	ctx := context.Background()

	// Create user.
	u, err := Users(db).Create(ctx, NewUser{Username: "u"})
	if err != nil {
		t.Fatal(err)
	}

	checkTags := func(t *testing.T, userID int32, wantTags []string) {
		t.Helper()
		u, err := Users(db).GetByID(ctx, userID)
		if err != nil {
			t.Fatal(err)
		}
		sort.Strings(u.Tags)
		sort.Strings(wantTags)
		if !reflect.DeepEqual(u.Tags, wantTags) {
			t.Errorf("got tags %v, want %v", u.Tags, wantTags)
		}
	}
	checkUsersWithTag := func(t *testing.T, tag string, wantUsers []int32) {
		t.Helper()
		users, err := Users(db).List(ctx, &UsersListOptions{Tag: tag})
		if err != nil {
			t.Fatal(err)
		}
		userIDs := make([]int32, len(users))
		for i, u := range users {
			userIDs[i] = u.ID
		}
		if !reflect.DeepEqual(userIDs, wantUsers) {
			t.Errorf("got user IDs %v, want %v", userIDs, wantUsers)
		}
	}

	t.Run("fails on nonexistent user", func(t *testing.T) {
		if err := Users(db).SetTag(ctx, 1234 /* doesn't exist */, "t", true); !errcode.IsNotFound(err) {
			t.Errorf("got err %v, want errcode.IsNotFound", err)
		}
		if err := Users(db).SetTag(ctx, 1234 /* doesn't exist */, "t", false); !errcode.IsNotFound(err) {
			t.Errorf("got err %v, want errcode.IsNotFound", err)
		}
	})

	t.Run("tags begins empty", func(t *testing.T) {
		checkTags(t, u.ID, []string{})
		checkUsersWithTag(t, "t1", []int32{})
	})

	t.Run("adds and removes tag", func(t *testing.T) {
		if err := Users(db).SetTag(ctx, u.ID, "t1", true); err != nil {
			t.Fatal(err)
		}
		checkTags(t, u.ID, []string{"t1"})
		checkUsersWithTag(t, "t1", []int32{u.ID})

		t.Run("deduplicates", func(t *testing.T) {
			if err := Users(db).SetTag(ctx, u.ID, "t1", true); err != nil {
				t.Fatal(err)
			}
			checkTags(t, u.ID, []string{"t1"})
		})

		if err := Users(db).SetTag(ctx, u.ID, "t2", true); err != nil {
			t.Fatal(err)
		}
		checkTags(t, u.ID, []string{"t1", "t2"})
		checkUsersWithTag(t, "t1", []int32{u.ID})
		checkUsersWithTag(t, "t2", []int32{u.ID})

		if err := Users(db).SetTag(ctx, u.ID, "t1", false); err != nil {
			t.Fatal(err)
		}
		checkTags(t, u.ID, []string{"t2"})
		checkUsersWithTag(t, "t1", []int32{})
		checkUsersWithTag(t, "t2", []int32{u.ID})

		t.Run("removing nonexistent tag is noop", func(t *testing.T) {
			if err := Users(db).SetTag(ctx, u.ID, "t1", false); err != nil {
				t.Fatal(err)
			}
			checkTags(t, u.ID, []string{"t2"})
		})

		if err := Users(db).SetTag(ctx, u.ID, "t2", false); err != nil {
			t.Fatal(err)
		}
		checkTags(t, u.ID, []string{})
		checkUsersWithTag(t, "t2", []int32{})
	})
}

func normalizeUsers(users []*types.User) []*types.User {
	for _, u := range users {
		u.CreatedAt = u.CreatedAt.Local().Round(time.Second)
		u.UpdatedAt = u.UpdatedAt.Local().Round(time.Second)
		u.InvalidatedSessionsAt = u.InvalidatedSessionsAt.Local().Round(time.Second)
	}
	return users
}
