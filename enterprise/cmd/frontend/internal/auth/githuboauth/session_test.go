package githuboauth

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	githublogin "github.com/dghubble/gologin/github"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/auth"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/auth/providers"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtesting"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	githubsvc "github.com/sourcegraph/sourcegraph/internal/extsvc/github"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

func init() {
	spew.Config.DisablePointerAddresses = true
	spew.Config.SortKeys = true
	spew.Config.SpewKeys = true
}

func TestSessionIssuerHelper_GetOrCreateUser(t *testing.T) {
	ghURL, _ := url.Parse("https://github.com")
	codeHost := extsvc.NewCodeHost(ghURL, extsvc.TypeGitHub)
	clientID := "client-id"

	// Top-level mock data
	//
	// authSaveableUsers that will be accepted by auth.GetAndSaveUser
	authSaveableUsers := map[string]int32{
		"alice": 1,
	}

	type input struct {
		description     string
		ghUser          *github.User
		ghUserEmails    []*githubsvc.UserEmail
		ghUserOrgs      []*githubsvc.Org
		ghUserEmailsErr error
		ghUserOrgsErr   error
		allowSignup     bool
		allowOrgs       []string
	}
	cases := []struct {
		inputs        []input
		expActor      *actor.Actor
		expErr        bool
		expAuthUserOp *auth.GetAndSaveUserOp
	}{
		{
			inputs: []input{{
				description: "ghUser, verified email -> session created",
				ghUser:      &github.User{ID: github.Int64(101), Login: github.String("alice")},
				ghUserEmails: []*githubsvc.UserEmail{{
					Email:    "alice@example.com",
					Primary:  true,
					Verified: true,
				}},
			}},
			expActor: &actor.Actor{UID: 1},
			expAuthUserOp: &auth.GetAndSaveUserOp{
				UserProps:       u("alice", "alice@example.com", true),
				ExternalAccount: acct(extsvc.TypeGitHub, "https://github.com/", clientID, "101"),
			},
		},
		{
			inputs: []input{{
				description: "ghUser, primary email not verified but another is -> no session created",
				ghUser:      &github.User{ID: github.Int64(101), Login: github.String("alice")},
				ghUserEmails: []*githubsvc.UserEmail{{
					Email:    "alice@example1.com",
					Primary:  true,
					Verified: false,
				}, {
					Email:    "alice@example2.com",
					Primary:  false,
					Verified: false,
				}, {
					Email:    "alice@example3.com",
					Primary:  false,
					Verified: true,
				}},
			}},
			expActor: &actor.Actor{UID: 1},
			expAuthUserOp: &auth.GetAndSaveUserOp{
				UserProps:       u("alice", "alice@example3.com", true),
				ExternalAccount: acct(extsvc.TypeGitHub, "https://github.com/", clientID, "101"),
			},
		},
		{
			inputs: []input{{
				description: "ghUser, no emails -> no session created",
				ghUser:      &github.User{ID: github.Int64(101), Login: github.String("alice")},
			}, {
				description:     "ghUser, email fetching err -> no session created",
				ghUser:          &github.User{ID: github.Int64(101), Login: github.String("alice")},
				ghUserEmailsErr: errors.New("x"),
			}, {
				description: "ghUser, plenty of emails but none verified -> no session created",
				ghUser:      &github.User{ID: github.Int64(101), Login: github.String("alice")},
				ghUserEmails: []*githubsvc.UserEmail{{
					Email:    "alice@example1.com",
					Primary:  true,
					Verified: false,
				}, {
					Email:    "alice@example2.com",
					Primary:  false,
					Verified: false,
				}, {
					Email:    "alice@example3.com",
					Primary:  false,
					Verified: false,
				}},
			}, {
				description: "no ghUser -> no session created",
			}, {
				description: "ghUser, verified email, unsaveable -> no session created",
				ghUser:      &github.User{ID: github.Int64(102), Login: github.String("bob")},
			}},
			expErr: true,
		},
		{
			inputs: []input{{
				description: "ghUser, verified email, not in allowed orgs -> no session created",
				allowOrgs:   []string{"sourcegraph"},
				ghUser: &github.User{
					ID:    github.Int64(101),
					Login: github.String("alice"),
				},
				ghUserEmails: []*githubsvc.UserEmail{{
					Email:    "alice@example.com",
					Primary:  true,
					Verified: true,
				}},
			}},
			expErr: true,
		},
		{
			inputs: []input{{
				description: "ghUser, verified email, error getting user orgs -> no session created",
				allowOrgs:   []string{"sourcegraph"},
				ghUser: &github.User{
					ID:    github.Int64(101),
					Login: github.String("alice"),
				},
				ghUserEmails: []*githubsvc.UserEmail{{
					Email:    "alice@example.com",
					Primary:  true,
					Verified: true,
				}},
				ghUserOrgs: []*githubsvc.Org{
					{Login: "sourcegraph"},
					{Login: "example"},
				},
				ghUserOrgsErr: errors.New("boom"),
			}},
			expErr: true,
		},
		{
			inputs: []input{{
				description: "ghUser, verified email, allowed orgs -> session created",
				allowOrgs:   []string{"sourcegraph"},
				ghUser: &github.User{
					ID:    github.Int64(101),
					Login: github.String("alice"),
				},
				ghUserEmails: []*githubsvc.UserEmail{{
					Email:    "alice@example.com",
					Primary:  true,
					Verified: true,
				}},
				ghUserOrgs: []*githubsvc.Org{
					{Login: "sourcegraph"},
					{Login: "example"},
				},
			}},
			expActor: &actor.Actor{UID: 1},
			expAuthUserOp: &auth.GetAndSaveUserOp{
				UserProps:       u("alice", "alice@example.com", true),
				ExternalAccount: acct(extsvc.TypeGitHub, "https://github.com/", clientID, "101"),
			},
		},
	}
	for _, c := range cases {
		for _, ci := range c.inputs {
			c, ci := c, ci
			t.Run(ci.description, func(t *testing.T) {
				githubsvc.MockGetAuthenticatedUserEmails = func(ctx context.Context) ([]*githubsvc.UserEmail, error) {
					return ci.ghUserEmails, ci.ghUserEmailsErr
				}
				githubsvc.MockGetAuthenticatedUserOrgs = func(ctx context.Context) ([]*githubsvc.Org, error) {
					return ci.ghUserOrgs, ci.ghUserOrgsErr
				}
				var gotAuthUserOp *auth.GetAndSaveUserOp
				auth.MockGetAndSaveUser = func(ctx context.Context, op auth.GetAndSaveUserOp) (userID int32, safeErrMsg string, err error) {
					if gotAuthUserOp != nil {
						t.Fatal("GetAndSaveUser called more than once")
					}
					op.ExternalAccountData = extsvc.AccountData{} // ignore AccountData value
					gotAuthUserOp = &op

					if uid, ok := authSaveableUsers[op.UserProps.Username]; ok {
						return uid, "", nil
					}
					return 0, "safeErr", errors.New("auth.GetAndSaveUser error")
				}
				defer func() {
					auth.MockGetAndSaveUser = nil
					githubsvc.MockGetAuthenticatedUserEmails = nil
					githubsvc.MockGetAuthenticatedUserOrgs = nil
				}()

				ctx := githublogin.WithUser(context.Background(), ci.ghUser)
				s := &sessionIssuerHelper{
					CodeHost:    codeHost,
					clientID:    clientID,
					allowSignup: ci.allowSignup,
					allowOrgs:   ci.allowOrgs,
				}
				tok := &oauth2.Token{AccessToken: "dummy-value-that-isnt-relevant-to-unit-correctness"}
				actr, _, err := s.GetOrCreateUser(ctx, tok, "", "")
				if got, exp := actr, c.expActor; !reflect.DeepEqual(got, exp) {
					t.Errorf("expected actor %v, got %v", exp, got)
				}
				if c.expErr && err == nil {
					t.Errorf("expected err %v, but was nil", c.expErr)
				} else if !c.expErr && err != nil {
					t.Errorf("expected no error, but was %v", err)
				}
				if got, exp := gotAuthUserOp, c.expAuthUserOp; !reflect.DeepEqual(got, exp) {
					t.Error(cmp.Diff(got, exp))
				}
			})
		}
	}
}

func TestSessionIssuerHelper_CreateCodeHostConnection(t *testing.T) {
	createCodeHostConnectionHelper(t, false)
}

func TestSessionIssuerHelper_CreateCodeHostConnectionHandlesExistingService(t *testing.T) {
	createCodeHostConnectionHelper(t, true)
}

func createCodeHostConnectionHelper(t *testing.T, serviceExists bool) {
	t.Helper()

	ctx := context.Background()
	s := &sessionIssuerHelper{}
	t.Run("Unauthenticated request", func(t *testing.T) {
		_, err := s.CreateCodeHostConnection(ctx, nil, "")
		if err == nil {
			t.Fatal("Want error but got nil")
		}
	})
	now := time.Now()

	mockGitHubCom := newMockProvider(t, "githubcomclient", "githubcomsecret", "https://github.com/")
	providers.MockProviders = []providers.Provider{mockGitHubCom.Provider}
	defer func() { providers.MockProviders = nil }()

	tok := &oauth2.Token{
		AccessToken: "dummy-value-that-isnt-relevant-to-unit-correctness",
	}
	act := &actor.Actor{UID: 1}
	ghUser := &github.User{
		ID:    github.Int64(101),
		Login: github.String("alice"),
	}

	ctx = actor.WithActor(ctx, act)
	ctx = githublogin.WithUser(ctx, ghUser)

	var got *types.ExternalService
	database.Mocks.ExternalServices.Transact = func(ctx context.Context) (*database.ExternalServiceStore, error) {
		return database.ExternalServices(&dbtesting.MockDB{}), nil
	}
	database.Mocks.ExternalServices.Done = func(err error) error {
		return nil
	}
	database.Mocks.ExternalServices.List = func(opt database.ExternalServicesListOptions) ([]*types.ExternalService, error) {
		if serviceExists {
			return nil, nil
		}
		return []*types.ExternalService{
			{
				Kind:        extsvc.KindGitHub,
				DisplayName: fmt.Sprintf("GitHub (%s)", deref(ghUser.Login)),
				Config: fmt.Sprintf(`
{
  "url": "%s",
  "token": "%s",
  "orgs": []
}
`, mockGitHubCom.ServiceID, "a-token-that-should-be-replaced"),
				NamespaceUserID: act.UID,
				CreatedAt:       now,
				UpdatedAt:       now,
			},
		}, nil
	}
	database.Mocks.ExternalServices.Upsert = func(ctx context.Context, services ...*types.ExternalService) error {
		if len(services) != 1 {
			t.Fatalf("Expected 1 service in Upsert, got %d", len(services))
		}
		// Tweak timestamps
		services[0].CreatedAt = now
		services[0].UpdatedAt = now
		got = services[0]
		return nil
	}
	t.Cleanup(func() {
		database.Mocks.ExternalServices = database.MockExternalServices{}
	})

	_, err := s.CreateCodeHostConnection(ctx, tok, mockGitHubCom.ConfigID().ID)
	if err != nil {
		t.Fatal(err)
	}

	want := &types.ExternalService{
		Kind:        extsvc.KindGitHub,
		DisplayName: fmt.Sprintf("GitHub (%s)", deref(ghUser.Login)),
		Config: fmt.Sprintf(`
{
  "url": "%s",
  "token": "%s",
  "orgs": []
}
`, mockGitHubCom.ServiceID, tok.AccessToken),
		NamespaceUserID: act.UID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("Mismatch (-want +got):\n%s", diff)
	}
}

func u(username, email string, emailIsVerified bool) database.NewUser {
	return database.NewUser{
		Username:        username,
		Email:           email,
		EmailIsVerified: emailIsVerified,
	}
}

func acct(serviceType, serviceID, clientID, accountID string) extsvc.AccountSpec {
	return extsvc.AccountSpec{
		ServiceType: serviceType,
		ServiceID:   serviceID,
		ClientID:    clientID,
		AccountID:   accountID,
	}
}
