package auth

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
)

func init() {
	spew.Config.DisablePointerAddresses = true
	spew.Config.SortKeys = true
	spew.Config.SpewKeys = true
}

// TestGetAndSaveUser ensures the correctness of the GetAndSaveUser function.
//
// 🚨 SECURITY: This guarantees the integrity of the identity resolution process (ensuring that new
// external accounts are linked to the appropriate user account)
func TestGetAndSaveUser(t *testing.T) {
	type innerCase struct {
		description string
		actorUID    int32
		op          GetAndSaveUserOp

		// if true, then will expect same output if op.CreateIfNotExist is true or false
		createIfNotExistIrrelevant bool

		// expected return values
		expUserID  int32
		expSafeErr string
		expErr     error

		// expected side effects
		expSavedExtAccts                 map[int32][]extsvc.AccountSpec
		expUpdatedUsers                  map[int32][]db.UserUpdate
		expCreatedUsers                  map[int32]db.NewUser
		expCalledGrantPendingPermissions bool
	}
	type outerCase struct {
		description string
		mock        mockParams
		innerCases  []innerCase
	}

	unexpectedErr := errors.New("unexpected err")

	oneUser := []userInfo{{
		user: types.User{ID: 1, Username: "u1"},
		extAccts: []extsvc.AccountSpec{
			ext("st1", "s1", "c1", "s1/u1"),
		},
		emails: []string{"u1@example.com"},
	}}
	getOneUserOp := GetAndSaveUserOp{
		ExternalAccount: ext("st1", "s1", "c1", "s1/u1"),
		UserProps:       userProps("u1", "u1@example.com", true),
	}
	getNonExistentUserCreateIfNotExistOp := GetAndSaveUserOp{
		ExternalAccount:  ext("st1", "s1", "c1", "nonexistent"),
		UserProps:        userProps("nonexistent", "nonexistent@example.com", true),
		CreateIfNotExist: true,
	}

	mainCase := outerCase{
		description: "no unexpected errors",
		mock: mockParams{
			userInfos: []userInfo{
				{
					user: types.User{ID: 1, Username: "u1"},
					extAccts: []extsvc.AccountSpec{
						ext("st1", "s1", "c1", "s1/u1"),
					},
					emails: []string{"u1@example.com"},
				},
				{
					user: types.User{ID: 2, Username: "u2"},
					extAccts: []extsvc.AccountSpec{
						ext("st1", "s1", "c1", "s1/u2"),
					},
					emails: []string{"u2@example.com"},
				},
				{
					user:     types.User{ID: 3, Username: "u3"},
					extAccts: []extsvc.AccountSpec{},
					emails:   []string{},
				},
			},
		},
		// TODO(beyang): add non-verified email cases
		innerCases: []innerCase{
			{
				description: "ext acct exists, user has same username and email",
				op: GetAndSaveUserOp{
					ExternalAccount: ext("st1", "s1", "c1", "s1/u1"),
					UserProps:       userProps("u1", "u1@example.com", true),
				},
				createIfNotExistIrrelevant: true,
				expUserID:                  1,
				expSavedExtAccts: map[int32][]extsvc.AccountSpec{
					1: {ext("st1", "s1", "c1", "s1/u1")},
				},
			},
			{
				description: "ext acct exists, username and email don't exist",
				// Note: for now, we drop the non-matching email; in the future, we may want to
				// save this as a new verified user email
				op: GetAndSaveUserOp{
					ExternalAccount: ext("st1", "s1", "c1", "s1/u1"),
					UserProps:       userProps("doesnotexist", "doesnotexist@example.com", true),
				},
				createIfNotExistIrrelevant: true,
				expUserID:                  1,
				expSavedExtAccts: map[int32][]extsvc.AccountSpec{
					1: {ext("st1", "s1", "c1", "s1/u1")},
				},
			},
			{
				description: "ext acct exists, email belongs to another user",
				// In this case, the external account is already mapped, so we ignore the email
				// inconsistency
				op: GetAndSaveUserOp{
					ExternalAccount: ext("st1", "s1", "c1", "s1/u1"),
					UserProps:       userProps("u1", "u2@example.com", true),
				},
				createIfNotExistIrrelevant: true,
				expUserID:                  1,
				expSavedExtAccts: map[int32][]extsvc.AccountSpec{
					1: {ext("st1", "s1", "c1", "s1/u1")},
				},
			},
			{
				description: "ext acct doesn't exist, user with username and email exists",
				op: GetAndSaveUserOp{
					ExternalAccount: ext("st1", "s-new", "c1", "s-new/u1"),
					UserProps:       userProps("u1", "u1@example.com", true),
				},
				createIfNotExistIrrelevant: true,
				expUserID:                  1,
				expSavedExtAccts: map[int32][]extsvc.AccountSpec{
					1: {ext("st1", "s-new", "c1", "s-new/u1")},
				},
				expCalledGrantPendingPermissions: true,
			},
			{
				description: "ext acct doesn't exist, user with username exists but email doesn't exist",
				// Note: if the email doesn't match, the user effectively doesn't exist from our POV
				op: GetAndSaveUserOp{
					ExternalAccount:  ext("st1", "s-new", "c1", "s-new/u1"),
					UserProps:        userProps("u1", "doesnotmatch@example.com", true),
					CreateIfNotExist: true,
				},
				expSafeErr: "Username \"u1\" already exists, but no verified email matched \"doesnotmatch@example.com\"",
				expErr:     db.MockCannotCreateUserUsernameExistsErr,
			},
			{
				description: "ext acct doesn't exist, user with email exists but username doesn't exist",
				// We treat this as a resolved user and ignore the non-matching username
				op: GetAndSaveUserOp{
					ExternalAccount: ext("st1", "s-new", "c1", "s-new/u1"),
					UserProps:       userProps("doesnotmatch", "u1@example.com", true),
				},
				createIfNotExistIrrelevant: true,
				expUserID:                  1,
				expSavedExtAccts: map[int32][]extsvc.AccountSpec{
					1: {ext("st1", "s-new", "c1", "s-new/u1")},
				},
				expCalledGrantPendingPermissions: true,
			},
			{
				description: "ext acct doesn't exist, username and email don't exist, should create user",
				op: GetAndSaveUserOp{
					ExternalAccount:  ext("st1", "s1", "c1", "s1/u-new"),
					UserProps:        userProps("u-new", "u-new@example.com", true),
					CreateIfNotExist: true,
				},
				expUserID: 10001,
				expSavedExtAccts: map[int32][]extsvc.AccountSpec{
					10001: {ext("st1", "s1", "c1", "s1/u-new")},
				},
				expCreatedUsers: map[int32]db.NewUser{
					10001: userProps("u-new", "u-new@example.com", true),
				},
				expCalledGrantPendingPermissions: true,
			},
			{
				description: "ext acct doesn't exist, username and email don't exist, should NOT create user",
				op: GetAndSaveUserOp{
					ExternalAccount:  ext("st1", "s1", "c1", "s1/u-new"),
					UserProps:        userProps("u-new", "u-new@example.com", true),
					CreateIfNotExist: false,
				},
				expSafeErr: "User account with verified email \"u-new@example.com\" does not exist. Ask a site admin to create your account and then verify your email.",
				expErr:     db.MockUserNotFoundErr,
			},
			{
				description: "ext acct exists, (ignore username and email), authenticated",
				op: GetAndSaveUserOp{
					ExternalAccount: ext("st1", "s1", "c1", "s1/u2"),
					UserProps:       userProps("ignore", "ignore", true),
				},
				createIfNotExistIrrelevant: true,
				actorUID:                   2,
				expUserID:                  2,
				expSavedExtAccts: map[int32][]extsvc.AccountSpec{
					2: {ext("st1", "s1", "c1", "s1/u2")},
				},
				expCalledGrantPendingPermissions: true,
			},
			{
				description: "ext acct doesn't exist, email and username match, authenticated",
				actorUID:    1,
				op: GetAndSaveUserOp{
					ExternalAccount: ext("st1", "s1", "c1", "s1/u1"),
					UserProps:       userProps("u1", "u1@example.com", true),
				},
				createIfNotExistIrrelevant: true,
				expUserID:                  1,
				expSavedExtAccts: map[int32][]extsvc.AccountSpec{
					1: {ext("st1", "s1", "c1", "s1/u1")},
				},
				expCalledGrantPendingPermissions: true,
			},
			{
				description: "ext acct doesn't exist, email matches but username doesn't, authenticated",
				// The non-matching username is ignored
				actorUID: 1,
				op: GetAndSaveUserOp{
					ExternalAccount: ext("st1", "s1", "c1", "s1/u1"),
					UserProps:       userProps("doesnotmatch", "u1@example.com", true),
				},
				createIfNotExistIrrelevant: true,
				expUserID:                  1,
				expSavedExtAccts: map[int32][]extsvc.AccountSpec{
					1: {ext("st1", "s1", "c1", "s1/u1")},
				},
				expCalledGrantPendingPermissions: true,
			},
			{
				description: "ext acct doesn't exist, email doesn't match existing user, authenticated",
				// The non-matching email is ignored. In the future, we may want to save this as
				// a verified user email, but this would be more complicated, because the email
				// might be associated with an existing user (in which case the authentication
				// should fail).
				actorUID: 1,
				op: GetAndSaveUserOp{
					ExternalAccount: ext("st1", "s-new", "c1", "s-new/u1"),
					UserProps:       userProps("u1", "doesnotmatch@example.com", true),
				},
				createIfNotExistIrrelevant: true,
				expUserID:                  1,
				expSavedExtAccts: map[int32][]extsvc.AccountSpec{
					1: {ext("st1", "s-new", "c1", "s-new/u1")},
				},
				expCalledGrantPendingPermissions: true,
			},
			{
				description: "ext acct doesn't exist, user has same username, lookupByUsername=true",
				op: GetAndSaveUserOp{
					ExternalAccount:  ext("st1", "s1", "c1", "doesnotexist"),
					UserProps:        userProps("u1", "", true),
					LookUpByUsername: true,
				},
				createIfNotExistIrrelevant: true,
				expUserID:                  1,
				expSavedExtAccts: map[int32][]extsvc.AccountSpec{
					1: {ext("st1", "s1", "c1", "doesnotexist")},
				},
				expCalledGrantPendingPermissions: true,
			},
		},
	}
	errorCases := []outerCase{
		{
			description: "lookupUserAndSaveErr",
			mock:        mockParams{lookupUserAndSaveErr: unexpectedErr, userInfos: oneUser},
			innerCases: []innerCase{{
				op:                         getOneUserOp,
				createIfNotExistIrrelevant: true,
				expSafeErr:                 "Unexpected error looking up the Sourcegraph user account associated with the external account. Ask a site admin for help.",
				expErr:                     unexpectedErr,
			}},
		},
		{
			description: "createUserAndSaveErr",
			mock:        mockParams{createUserAndSaveErr: unexpectedErr, userInfos: oneUser},
			innerCases: []innerCase{{
				op:         getNonExistentUserCreateIfNotExistOp,
				expSafeErr: "Unable to create a new user account due to a unexpected error. Ask a site admin for help.",
				expErr:     unexpectedErr,
			}},
		},
		{
			description: "associateUserAndSaveErr",
			mock:        mockParams{associateUserAndSaveErr: unexpectedErr, userInfos: oneUser},
			innerCases: []innerCase{{
				op: GetAndSaveUserOp{
					ExternalAccount: ext("st1", "s1", "c1", "nonexistent"),
					UserProps:       userProps("u1", "u1@example.com", true),
				},
				expSafeErr: "Unexpected error associating the external account with your Sourcegraph user. The most likely cause for this problem is that another Sourcegraph user is already linked with this external account. A site admin or the other user can unlink the account to fix this problem.",
				expErr:     unexpectedErr,
			}},
		},
		{
			description: "getByVerifiedEmailErr",
			mock:        mockParams{getByVerifiedEmailErr: unexpectedErr, userInfos: oneUser},
			innerCases: []innerCase{{
				op: GetAndSaveUserOp{
					ExternalAccount: ext("st1", "s1", "c1", "nonexistent"),
					UserProps:       userProps("u1", "u1@example.com", true),
				},
				createIfNotExistIrrelevant: true,
				expSafeErr:                 "Unexpected error looking up the Sourcegraph user by verified email. Ask a site admin for help.",
				expErr:                     unexpectedErr,
			}},
		},
		{
			description: "getByIDErr",
			mock:        mockParams{getByIDErr: unexpectedErr, userInfos: oneUser},
			innerCases: []innerCase{{
				op: GetAndSaveUserOp{
					ExternalAccount: ext("st1", "s1", "c1", "nonexistent"),
					UserProps:       userProps("u1", "u1@example.com", true),
				},
				createIfNotExistIrrelevant: true,
				expSafeErr:                 "Unexpected error getting the Sourcegraph user account. Ask a site admin for help.",
				expErr:                     unexpectedErr,
			}},
		},
		{
			description: "updateErr",
			mock:        mockParams{updateErr: unexpectedErr, userInfos: oneUser},
			innerCases: []innerCase{{
				op: GetAndSaveUserOp{
					ExternalAccount: ext("st1", "s1", "c1", "nonexistent"),
					UserProps: db.NewUser{
						Email:           "u1@example.com",
						EmailIsVerified: true,
						Username:        "u1",
						DisplayName:     "New Name",
					},
				},
				createIfNotExistIrrelevant: true,
				expSafeErr:                 "Unexpected error updating the Sourcegraph user account with new user profile information from the external account. Ask a site admin for help.",
				expErr:                     unexpectedErr,
			}},
		},
	}

	allCases := append(append([]outerCase{}, mainCase), errorCases...)
	for _, oc := range allCases {
		t.Run(oc.description, func(t *testing.T) {
			for _, c := range oc.innerCases {
				if c.expSavedExtAccts == nil {
					c.expSavedExtAccts = map[int32][]extsvc.AccountSpec{}
				}
				if c.expUpdatedUsers == nil {
					c.expUpdatedUsers = map[int32][]db.UserUpdate{}
				}
				if c.expCreatedUsers == nil {
					c.expCreatedUsers = map[int32]db.NewUser{}
				}

				createIfNotExistVals := []bool{c.op.CreateIfNotExist}
				if c.createIfNotExistIrrelevant {
					createIfNotExistVals = []bool{false, true}
				}
				for _, createIfNotExist := range createIfNotExistVals {
					description := c.description
					if len(createIfNotExistVals) == 2 {
						description = fmt.Sprintf("%s, createIfNotExist=%v", description, createIfNotExist)
					}
					t.Run("", func(t *testing.T) {
						t.Logf("Description: %q", description)
						m := newMocks(t, oc.mock)
						m.apply()
						defer m.reset()

						ctx := context.Background()
						if c.actorUID != 0 {
							ctx = actor.WithActor(context.Background(), &actor.Actor{UID: c.actorUID})
						}
						op := c.op
						op.CreateIfNotExist = createIfNotExist
						userID, safeErr, err := GetAndSaveUser(ctx, op)
						for _, v := range []struct {
							label string
							got   interface{}
							want  interface{}
						}{
							{"userID", userID, c.expUserID},
							{"safeErr", safeErr, c.expSafeErr},
							{"err", err, c.expErr},
							{"savedExtAccts (side-effect)", m.savedExtAccts, c.expSavedExtAccts},
							{"updatedUsers (side-effect)", m.updatedUsers, c.expUpdatedUsers},
							{"createdUsers (side-effect)", m.createdUsers, c.expCreatedUsers},
						} {
							if label, got, want := v.label, v.got, v.want; !reflect.DeepEqual(got, want) {
								dmp := diffmatchpatch.New()
								t.Errorf("%s: got != want\n%#v != %#v\ndiff(got, want):\n%s",
									label, got, want, dmp.DiffPrettyText(dmp.DiffMain(spew.Sdump(want), spew.Sdump(got), false)))
							}
						}

						if c.expCalledGrantPendingPermissions != m.calledGrantPendingPermissions {
							t.Fatalf("calledGrantPendingPermissions: want %v but got %v", c.expCalledGrantPendingPermissions, m.calledGrantPendingPermissions)
						}
					})
				}
			}
		})
	}
}

type userInfo struct {
	user     types.User
	extAccts []extsvc.AccountSpec
	emails   []string
}

func newMocks(t *testing.T, m mockParams) *mocks {
	// validation
	extAcctIDs := make(map[string]struct{})
	userIDs := make(map[int32]struct{})
	usernames := make(map[string]struct{})
	emails := make(map[string]struct{})
	for _, u := range m.userInfos {
		if _, exists := usernames[u.user.Username]; exists {
			t.Fatal("mocks: dup username")
		}
		usernames[u.user.Username] = struct{}{}

		if _, exists := userIDs[u.user.ID]; exists {
			t.Fatal("mocks: dup user ID")
		}
		userIDs[u.user.ID] = struct{}{}

		for _, email := range u.emails {
			if _, exists := emails[email]; exists {
				t.Fatal("mocks: dup email")
			}
			emails[email] = struct{}{}
		}
		for _, extAcct := range u.extAccts {
			if _, exists := extAcctIDs[extAcct.AccountID]; exists {
				t.Fatal("mocks: dup ext account ID")
			}
			extAcctIDs[extAcct.AccountID] = struct{}{}
		}
	}

	return &mocks{
		mockParams:    m,
		t:             t,
		savedExtAccts: make(map[int32][]extsvc.AccountSpec),
		updatedUsers:  make(map[int32][]db.UserUpdate),
		createdUsers:  make(map[int32]db.NewUser),
		nextUserID:    10001,
	}
}

type mockParams struct {
	userInfos               []userInfo
	lookupUserAndSaveErr    error
	createUserAndSaveErr    error
	associateUserAndSaveErr error
	getByVerifiedEmailErr   error
	getByUsernameErr        error //nolint:structcheck
	getByIDErr              error
	updateErr               error
}

func (m *mocks) apply() {
	db.Mocks.ExternalAccounts = db.MockExternalAccounts{
		LookupUserAndSave:    m.LookupUserAndSave,
		AssociateUserAndSave: m.AssociateUserAndSave,
		CreateUserAndSave:    m.CreateUserAndSave,
	}
	db.Mocks.Users = db.MockUsers{
		GetByID:            m.GetByID,
		GetByVerifiedEmail: m.GetByVerifiedEmail,
		GetByUsername:      m.GetByUsername,
		Update:             m.Update,
	}
	db.Mocks.Authz = db.MockAuthz{
		GrantPendingPermissions: m.GrantPendingPermissions,
	}
}

func (m *mocks) reset() {
	db.Mocks.ExternalAccounts = db.MockExternalAccounts{}
	db.Mocks.Users = db.MockUsers{}
	db.Mocks.Authz = db.MockAuthz{}
}

// mocks provide mocking. It should only be used for one call of auth.GetAndSaveUser, because saves
// are recorded in the mock struct but will not be reflected in the return values of the mocked
// methods.
type mocks struct {
	mockParams
	t *testing.T

	// savedExtAccts tracks all ext acct "saves" for a given user ID
	savedExtAccts map[int32][]extsvc.AccountSpec

	// createdUsers tracks user creations by user ID
	createdUsers map[int32]db.NewUser

	// updatedUsers tracks all user updates for a given user ID
	updatedUsers map[int32][]db.UserUpdate

	// nextUserID is the user ID of the next created user.
	nextUserID int32

	// calledGrantPendingPermissions tracks if db.Authz.GrantPendingPermissions method is called.
	calledGrantPendingPermissions bool
}

// LookupUserAndSave mocks db.ExternalAccounts.LookupUserAndSave
func (m *mocks) LookupUserAndSave(spec extsvc.AccountSpec, data extsvc.AccountData) (userID int32, err error) {
	if m.lookupUserAndSaveErr != nil {
		return 0, m.lookupUserAndSaveErr
	}

	for _, u := range m.userInfos {
		for _, a := range u.extAccts {
			if a == spec {
				m.savedExtAccts[u.user.ID] = append(m.savedExtAccts[u.user.ID], spec)
				return u.user.ID, nil
			}
		}
	}
	return 0, &errcode.Mock{IsNotFound: true}
}

// CreateUserAndSave mocks db.ExternalAccounts.CreateUserAndSave
func (m *mocks) CreateUserAndSave(newUser db.NewUser, spec extsvc.AccountSpec, data extsvc.AccountData) (createdUserID int32, err error) {
	if m.createUserAndSaveErr != nil {
		return 0, m.createUserAndSaveErr
	}

	// Check if username already exists
	for _, u := range m.userInfos {
		if u.user.Username == newUser.Username {
			return 0, db.MockCannotCreateUserUsernameExistsErr
		}
	}
	// Check if email already exists
	for _, u := range m.userInfos {
		for _, email := range u.emails {
			if email == newUser.Email {
				return 0, db.MockCannotCreateUserEmailExistsErr
			}
		}
	}

	// Create user
	userID := m.nextUserID
	m.nextUserID++
	if _, ok := m.createdUsers[userID]; ok {
		m.t.Fatalf("user %v should not already exist", userID)
	}
	m.createdUsers[userID] = newUser

	// Save ext acct
	m.savedExtAccts[userID] = append(m.savedExtAccts[userID], spec)

	return userID, nil
}

// AssociateUserAndSave mocks db.ExternalAccounts.AssociateUserAndSave
func (m *mocks) AssociateUserAndSave(userID int32, spec extsvc.AccountSpec, data extsvc.AccountData) (err error) {
	if m.associateUserAndSaveErr != nil {
		return m.associateUserAndSaveErr
	}

	// Check if ext acct is associated with different user
	for _, u := range m.userInfos {
		for _, a := range u.extAccts {
			if a == spec && u.user.ID != userID {
				return fmt.Errorf("unable to change association of external account from user %d to user %d (delete the external account and then try again)", u.user.ID, userID)
			}
		}
	}

	m.savedExtAccts[userID] = append(m.savedExtAccts[userID], spec)
	return nil
}

// GetByVerifiedEmail mocks db.Users.GetByVerifiedEmail
func (m *mocks) GetByVerifiedEmail(ctx context.Context, email string) (*types.User, error) {
	if m.getByVerifiedEmailErr != nil {
		return nil, m.getByVerifiedEmailErr
	}

	for _, u := range m.userInfos {
		for _, e := range u.emails {
			if e == email {
				return &u.user, nil
			}
		}
	}
	return nil, db.MockUserNotFoundErr
}

// GetByUsername mocks db.Users.GetByUsername
func (m *mocks) GetByUsername(ctx context.Context, username string) (*types.User, error) {
	if m.getByUsernameErr != nil {
		return nil, m.getByUsernameErr
	}

	for _, u := range m.userInfos {
		if u.user.Username == username {
			return &u.user, nil
		}
	}
	return nil, db.MockUserNotFoundErr
}

// GetByID mocks db.Users.GetByID
func (m *mocks) GetByID(ctx context.Context, id int32) (*types.User, error) {
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}

	for _, u := range m.userInfos {
		if u.user.ID == id {
			return &u.user, nil
		}
	}
	return nil, db.MockUserNotFoundErr
}

// Update mocks db.Users.Update
func (m *mocks) Update(id int32, update db.UserUpdate) error {
	if m.updateErr != nil {
		return m.updateErr
	}

	_, err := m.GetByID(context.Background(), id)
	if err != nil {
		return err
	}

	// Save user
	m.updatedUsers[id] = append(m.updatedUsers[id], update)
	return nil
}

// GrantPendingPermissions mocks db.Authz.GrantPendingPermissions
func (m *mocks) GrantPendingPermissions(context.Context, *db.GrantPendingPermissionsArgs) error {
	m.calledGrantPendingPermissions = true
	return nil
}

func ext(serviceType, serviceID, clientID, accountID string) extsvc.AccountSpec {
	return extsvc.AccountSpec{
		ServiceType: serviceType,
		ServiceID:   serviceID,
		ClientID:    clientID,
		AccountID:   accountID,
	}
}

func userProps(username, email string, verifiedEmail bool) db.NewUser {
	return db.NewUser{
		Username:        username,
		Email:           email,
		EmailIsVerified: verifiedEmail,
	}
}
