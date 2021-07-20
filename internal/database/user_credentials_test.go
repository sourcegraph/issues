package database

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"math/big"
	"net/http"
	"reflect"
	"testing"

	"github.com/cockroachdb/errors"
	"github.com/gomodule/oauth1/oauth"
	"github.com/google/go-cmp/cmp"
	"github.com/keegancsmith/sqlf"

	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
	"github.com/sourcegraph/sourcegraph/internal/encryption"
	et "github.com/sourcegraph/sourcegraph/internal/encryption/testing"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/auth"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/bitbucketserver"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/gitlab"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

func TestUserCredential_Authenticator(t *testing.T) {
	ctx := context.Background()

	t.Run("errors", func(t *testing.T) {
		for name, credential := range map[string]*UserCredential{
			"no credential": {
				EncryptionKeyID: "test key",
				key:             &et.TestKey{},
			},
			"bad decrypter": {
				EncryptionKeyID:     "it's the bad guy... uh, key",
				EncryptedCredential: []byte("foo"),
				key:                 &et.BadKey{Err: errors.New("bad key bad key what you gonna do")},
			},
			"invalid secret": {
				EncryptionKeyID:     "transparent key",
				EncryptedCredential: []byte("foo"),
				key:                 et.NewTransparentKey(t),
			},
		} {
			t.Run(name, func(t *testing.T) {
				if _, err := credential.Authenticator(ctx); err == nil {
					t.Error("unexpected nil error")
				}
			})
		}
	})

	t.Run("plaintext credential", func(t *testing.T) {
		a := &auth.BasicAuth{}

		enc, err := EncryptAuthenticator(ctx, nil, a)
		if err != nil {
			t.Fatal(err)
		}

		for _, keyID := range []string{"", UserCredentialUnmigratedEncryptionKeyID} {
			t.Run(keyID, func(t *testing.T) {
				uc := &UserCredential{
					EncryptionKeyID:     keyID,
					EncryptedCredential: enc,
					key:                 et.TestKey{},
				}

				have, err := uc.Authenticator(ctx)
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				} else if diff := cmp.Diff(have, a); diff != "" {
					t.Errorf("unexpected authenticator (-have +want):\n%s", diff)
				}
			})
		}
	})

	t.Run("encrypted credential", func(t *testing.T) {
		key := et.TestKey{}
		a := &auth.BasicAuth{Username: "foo", Password: "bar"}

		enc, err := EncryptAuthenticator(ctx, key, a)
		if err != nil {
			t.Fatal(err)
		}
		uc := &UserCredential{
			EncryptionKeyID:     "test key",
			EncryptedCredential: enc,
			key:                 key,
		}

		have, err := uc.Authenticator(ctx)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		} else if diff := cmp.Diff(have, a); diff != "" {
			t.Errorf("unexpected authenticator (-have +want):\n%s", diff)
		}
	})

	t.Run("nil key", func(t *testing.T) {
		a := &auth.BasicAuth{Username: "foo", Password: "bar"}

		enc, err := EncryptAuthenticator(ctx, nil, a)
		if err != nil {
			t.Fatal(err)
		}
		uc := &UserCredential{EncryptedCredential: enc, key: nil}

		have, err := uc.Authenticator(ctx)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		} else if diff := cmp.Diff(have, a); diff != "" {
			t.Errorf("unexpected authenticator (-have +want):\n%s", diff)
		}
	})
}

func TestUserCredential_SetAuthenticator(t *testing.T) {
	ctx := context.Background()
	a := &auth.BasicAuth{Username: "foo", Password: "bar"}

	t.Run("error", func(t *testing.T) {
		uc := &UserCredential{
			EncryptionKeyID: "bad key",
			key:             &et.BadKey{Err: errors.New("error")},
		}

		if err := uc.SetAuthenticator(ctx, a); err == nil {
			t.Error("unexpected nil error")
		}
	})

	t.Run("success", func(t *testing.T) {
		for name, key := range map[string]encryption.Key{
			"":         nil,
			"test key": et.TestKey{},
		} {
			t.Run(name, func(t *testing.T) {
				uc := &UserCredential{
					key: key,
				}

				if err := uc.SetAuthenticator(ctx, a); err != nil {
					t.Errorf("unexpected error: %v", err)
				} else if key == nil && uc.EncryptionKeyID != "" {
					t.Errorf("unexpected non-empty key ID: %q", uc.EncryptionKeyID)
				} else if key != nil && uc.EncryptionKeyID == "" {
					t.Error("unexpected empty key ID")
				}
			})
		}
	})
}

func TestUserCredentials_CreateUpdate(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, "")
	ctx, key, user := setUpUserCredentialTest(t, db)

	// Versions of Go before 1.14.x (where 3 < x < 11) cannot diff *big.Int
	// fields, which causes problems when diffing the keys embedded in OAuth
	// clients. We'll just define a little helper here to wrap cmp.Diff and get
	// us over the hump.
	diffAuthenticators := func(a, b auth.Authenticator) string {
		return cmp.Diff(a, b, cmp.Comparer(func(a, b *big.Int) bool {
			return a.Cmp(b) == 0
		}))
	}

	// Instead of two of every animal, we want one of every authenticator. Same,
	// same.
	for name, auth := range createUserCredentialAuths(t) {
		t.Run(name, func(t *testing.T) {
			scope := UserCredentialScope{
				Domain:              name,
				UserID:              user.ID,
				ExternalServiceType: extsvc.TypeGitHub,
				ExternalServiceID:   "https://github.com",
			}

			cred, err := UserCredentials(db, key).Create(ctx, scope, auth)
			if err != nil {
				t.Errorf("unexpected non-nil error: %v", err)
			} else if cred == nil {
				t.Error("unexpected nil credential")
			}

			if cred.ID == 0 {
				t.Error("unexpected zero ID")
			}
			if cred.Domain != scope.Domain {
				t.Errorf("unexpected domain: have=%q want=%q", cred.Domain, scope.Domain)
			}
			if cred.UserID != scope.UserID {
				t.Errorf("unexpected ID: have=%d want=%d", cred.UserID, scope.UserID)
			}
			if cred.ExternalServiceType != scope.ExternalServiceType {
				t.Errorf("unexpected external service type: have=%q want=%q", cred.ExternalServiceType, scope.ExternalServiceType)
			}
			if cred.ExternalServiceID != scope.ExternalServiceID {
				t.Errorf("unexpected external service id: have=%q want=%q", cred.ExternalServiceID, scope.ExternalServiceID)
			}
			if have, err := cred.Authenticator(ctx); err != nil {
				t.Log("cred.key", cred.key)
				t.Log("key", key)
				t.Fatal(err)
			} else if diff := diffAuthenticators(have, auth); diff != "" {
				t.Errorf("unexpected credential:\n%s", diff)
			}
			if cred.CreatedAt.IsZero() {
				t.Errorf("unexpected zero creation time")
			}
			if cred.UpdatedAt.IsZero() {
				t.Errorf("unexpected zero update time")
			}

			// Ensure that trying to insert again fails.
			if cred, err := UserCredentials(db, key).Create(ctx, scope, auth); err == nil {
				t.Error("unexpected nil error")
			} else if cred != nil {
				t.Errorf("unexpected non-nil credential: %v", cred)
			}

			newExternalServiceType := extsvc.TypeGitLab

			cred.ExternalServiceType = newExternalServiceType

			if err := UserCredentials(db, key).Update(ctx, cred); err != nil {
				t.Errorf("unexpected non-nil error updating: %+v", err)
			}

			updatedCred, err := UserCredentials(db, key).GetByID(ctx, cred.ID)
			if err != nil {
				t.Errorf("unexpected non-nil error getting credential: %+v", err)
			}
			if diff := cmp.Diff(cred, updatedCred, cmp.AllowUnexported(UserCredential{})); diff != "" {
				t.Errorf("credential incorrectly updated: %s", diff)
			}
		})
	}
}

func TestUserCredentials_Delete(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, "")
	ctx, key, user := setUpUserCredentialTest(t, db)

	t.Run("nonextant", func(t *testing.T) {
		err := UserCredentials(db, key).Delete(ctx, 1)
		if err == nil {
			t.Error("unexpected nil error")
		}

		var e UserCredentialNotFoundErr
		if !errors.As(err, &e) {
			t.Errorf("error is not a userCredentialNotFoundError; got %T: %v", err, err)
		}
		if len(e.args) != 1 || e.args[0].(int64) != 1 {
			t.Errorf("unexpected args: have=%v want=[1]", e.args)
		}
	})

	t.Run("extant", func(t *testing.T) {
		scope := UserCredentialScope{
			Domain:              UserCredentialDomainBatches,
			UserID:              user.ID,
			ExternalServiceType: "github",
			ExternalServiceID:   "https://github.com",
		}
		token := &auth.OAuthBearerToken{Token: "abcdef"}

		cred, err := UserCredentials(db, key).Create(ctx, scope, token)
		if err != nil {
			t.Fatal(err)
		}

		if err := UserCredentials(db, key).Delete(ctx, cred.ID); err != nil {
			t.Errorf("unexpected non-nil error: %v", err)
		}

		_, err = UserCredentials(db, key).GetByID(ctx, cred.ID)
		if !errors.HasType(err, UserCredentialNotFoundErr{}) {
			t.Errorf("unexpected error retrieving credential after deletion: %v", err)
		}
	})
}

func TestUserCredentials_GetByID(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, "")
	ctx, key, user := setUpUserCredentialTest(t, db)

	t.Run("nonextant", func(t *testing.T) {
		cred, err := UserCredentials(db, key).GetByID(ctx, 1)
		if cred != nil {
			t.Errorf("unexpected non-nil credential: %v", cred)
		}
		if err == nil {
			t.Error("unexpected nil error")
		}

		var e UserCredentialNotFoundErr
		if !errors.As(err, &e) {
			t.Errorf("error is not a userCredentialNotFoundError; got %T: %v", err, err)
		}
		if len(e.args) != 1 || e.args[0].(int64) != 1 {
			t.Errorf("unexpected args: have=%v want=[1]", e.args)
		}
	})

	t.Run("extant", func(t *testing.T) {
		scope := UserCredentialScope{
			Domain:              UserCredentialDomainBatches,
			UserID:              user.ID,
			ExternalServiceType: "github",
			ExternalServiceID:   "https://github.com",
		}
		token := &auth.OAuthBearerToken{Token: "abcdef"}

		want, err := UserCredentials(db, key).Create(ctx, scope, token)
		if err != nil {
			t.Fatal(err)
		}

		have, err := UserCredentials(db, key).GetByID(ctx, want.ID)
		if err != nil {
			t.Errorf("unexpected non-nil error: %v", err)
		}
		if diff := cmp.Diff(have, want, cmp.AllowUnexported(UserCredential{})); diff != "" {
			t.Errorf("unexpected credential:\n%s", diff)
		}
	})
}

func TestUserCredentials_GetByScope(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, "")
	ctx, key, user := setUpUserCredentialTest(t, db)

	scope := UserCredentialScope{
		Domain:              UserCredentialDomainBatches,
		UserID:              user.ID,
		ExternalServiceType: "github",
		ExternalServiceID:   "https://github.com",
	}
	token := &auth.OAuthBearerToken{Token: "abcdef"}

	t.Run("nonextant", func(t *testing.T) {
		cred, err := UserCredentials(db, key).GetByScope(ctx, scope)
		if cred != nil {
			t.Errorf("unexpected non-nil credential: %v", cred)
		}
		if err == nil {
			t.Error("unexpected nil error")
		}

		var e UserCredentialNotFoundErr
		if !errors.As(err, &e) {
			t.Errorf("error is not a userCredentialNotFoundError; got %T: %v", err, err)
		}
		if diff := cmp.Diff(e.args, []interface{}{scope}); diff != "" {
			t.Errorf("unexpected args:\n%s", diff)
		}
	})

	t.Run("extant", func(t *testing.T) {
		want, err := UserCredentials(db, key).Create(ctx, scope, token)
		if err != nil {
			t.Fatal(err)
		}

		have, err := UserCredentials(db, key).GetByScope(ctx, scope)
		if err != nil {
			t.Errorf("unexpected non-nil error: %v", err)
		}
		if diff := cmp.Diff(have, want, cmp.AllowUnexported(UserCredential{})); diff != "" {
			t.Errorf("unexpected credential:\n%s", diff)
		}
	})
}

func TestUserCredentials_List(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, "")
	ctx, key, user := setUpUserCredentialTest(t, db)

	githubScope := UserCredentialScope{
		Domain:              UserCredentialDomainBatches,
		UserID:              user.ID,
		ExternalServiceType: "github",
		ExternalServiceID:   "https://github.com",
	}
	gitlabScope := UserCredentialScope{
		Domain:              UserCredentialDomainBatches,
		UserID:              user.ID,
		ExternalServiceType: "gitlab",
		ExternalServiceID:   "https://gitlab.com",
	}
	token := &auth.OAuthBearerToken{Token: "abcdef"}

	// Unlike the other tests in this file, we'll set up a couple of credentials
	// right now, and then list from there.
	githubCred, err := UserCredentials(db, key).Create(ctx, githubScope, token)
	if err != nil {
		t.Fatal(err)
	}

	gitlabCred, err := UserCredentials(db, key).Create(ctx, gitlabScope, token)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("not found", func(t *testing.T) {
		creds, next, err := UserCredentials(db, key).List(ctx, UserCredentialsListOpts{
			Scope: UserCredentialScope{
				Domain: "this is not a valid domain",
			},
		})
		if err != nil {
			t.Errorf("unexpected non-nil error: %v", err)
		}
		if next != 0 {
			t.Errorf("unexpected next: have=%d want=%d", next, 0)
		}
		if len(creds) != 0 {
			t.Errorf("unexpected non-zero number of credentials: %v", creds)
		}
	})

	for name, tc := range map[string]struct {
		scope UserCredentialScope
		want  *UserCredential
	}{
		"service ID only": {
			scope: UserCredentialScope{
				ExternalServiceID: "https://github.com",
			},
			want: githubCred,
		},
		"service type only": {
			scope: UserCredentialScope{
				ExternalServiceType: "gitlab",
			},
			want: gitlabCred,
		},
		"full scope": {
			scope: githubScope,
			want:  githubCred,
		},
	} {
		t.Run("single match on "+name, func(t *testing.T) {
			creds, next, err := UserCredentials(db, key).List(ctx, UserCredentialsListOpts{
				Scope: tc.scope,
			})
			if err != nil {
				t.Errorf("unexpected non-nil error: %v", err)
			}
			if next != 0 {
				t.Errorf("unexpected next: have=%d want=%d", next, 0)
			}
			if diff := cmp.Diff(creds, []*UserCredential{tc.want}, cmp.AllowUnexported(UserCredential{})); diff != "" {
				t.Errorf("unexpected credentials:\n%s", diff)
			}
		})
	}

	for name, opts := range map[string]UserCredentialsListOpts{
		"no options":   {},
		"domain only":  {Scope: UserCredentialScope{Domain: UserCredentialDomainBatches}},
		"user ID only": {Scope: UserCredentialScope{UserID: user.ID}},
		"domain and user ID": {
			Scope: UserCredentialScope{
				Domain: UserCredentialDomainBatches,
				UserID: user.ID,
			},
		},
	} {
		t.Run("multiple matches on "+name, func(t *testing.T) {
			creds, next, err := UserCredentials(db, key).List(ctx, opts)
			if err != nil {
				t.Errorf("unexpected non-nil error: %v", err)
			}
			if next != 0 {
				t.Errorf("unexpected next: have=%d want=%d", next, 0)
			}
			if diff := cmp.Diff(creds, []*UserCredential{githubCred, gitlabCred}, cmp.AllowUnexported(UserCredential{})); diff != "" {
				t.Errorf("unexpected credentials:\n%s", diff)
			}
		})

		t.Run("pagination for "+name, func(t *testing.T) {
			o := opts
			o.LimitOffset = &LimitOffset{Limit: 1}
			creds, next, err := UserCredentials(db, key).List(ctx, o)
			if err != nil {
				t.Errorf("unexpected non-nil error: %v", err)
			}
			if next != 1 {
				t.Errorf("unexpected next: have=%d want=%d", next, 1)
			}
			if diff := cmp.Diff(creds, []*UserCredential{githubCred}, cmp.AllowUnexported(UserCredential{})); diff != "" {
				t.Errorf("unexpected credentials:\n%s", diff)
			}

			o.LimitOffset = &LimitOffset{Limit: 1, Offset: next}
			creds, next, err = UserCredentials(db, key).List(ctx, o)
			if err != nil {
				t.Errorf("unexpected non-nil error: %v", err)
			}
			if next != 0 {
				t.Errorf("unexpected next: have=%d want=%d", next, 0)
			}
			if diff := cmp.Diff(creds, []*UserCredential{gitlabCred}, cmp.AllowUnexported(UserCredential{})); diff != "" {
				t.Errorf("unexpected credentials:\n%s", diff)
			}
		})
	}
}

func TestUserCredentials_Invalid(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, "")
	ctx, key, user := setUpUserCredentialTest(t, db)

	t.Run("marshal", func(t *testing.T) {
		if _, err := UserCredentials(db, key).Create(ctx, UserCredentialScope{}, &invalidAuth{}); err == nil {
			t.Error("unexpected nil error")
		}
	})

	t.Run("unmarshal", func(t *testing.T) {
		// We'll set up some cases here that just shouldn't happen at all, and
		// make sure they bubble up with errors where we expect. Let's define a
		// helper to make that easier.

		insertRawCredential := func(t *testing.T, domain string, raw string) int64 {
			kid, err := keyID(ctx, key)
			if err != nil {
				t.Fatal(err)
			}

			secret, err := key.Encrypt(ctx, []byte(raw))
			if err != nil {
				t.Fatal(err)
			}

			q := sqlf.Sprintf(
				userCredentialsCreateQueryFmtstr,
				domain,
				user.ID,
				"type",
				"id",
				secret,
				kid,
				sqlf.Sprintf("id"),
			)

			var id int64
			row := db.QueryRowContext(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...)
			if err := row.Scan(&id); err != nil {
				t.Fatal(err)
			}

			return id
		}

		for name, id := range map[string]int64{
			"invalid credential type": insertRawCredential(t, "invalid", `{"type":"InvalidType"}`),
			"lying credential type":   insertRawCredential(t, "lying", `{"type":"BasicAuth","username":42}`),
			"malformed JSON":          insertRawCredential(t, "malformed", "this is not valid JSON"),
		} {
			t.Run(name, func(t *testing.T) {
				cred, err := UserCredentials(db, key).GetByID(ctx, id)
				if err != nil {
					t.Error("unexpected error")
				}

				if _, err := cred.Authenticator(ctx); err == nil {
					t.Error("unexpected nil error")
				}
			})
		}
	})
}

func TestUserCredentialNotFoundErr(t *testing.T) {
	err := UserCredentialNotFoundErr{}
	if have := errcode.IsNotFound(err); !have {
		t.Error("UserCredentialNotFoundErr does not say it represents a not found error")
	}
}

func createUserCredentialAuths(t *testing.T) map[string]auth.Authenticator {
	t.Helper()

	createOAuthClient := func(t *testing.T, token, secret string) *oauth.Client {
		t.Helper()

		// Generate a random key so we can test different clients are different.
		// Note that this is wildly insecure.
		key, err := rsa.GenerateKey(rand.Reader, 64)
		if err != nil {
			t.Fatal(err)
		}

		return &oauth.Client{
			Credentials: oauth.Credentials{
				Token:  token,
				Secret: secret,
			},
			PrivateKey: key,
		}
	}

	auths := make(map[string]auth.Authenticator)
	for _, a := range []auth.Authenticator{
		&auth.OAuthClient{Client: createOAuthClient(t, "abc", "def")},
		&auth.BasicAuth{Username: "foo", Password: "bar"},
		&auth.BasicAuthWithSSH{BasicAuth: auth.BasicAuth{Username: "foo", Password: "bar"}, PrivateKey: "private", PublicKey: "public", Passphrase: "pass"},
		&auth.OAuthBearerToken{Token: "abcdef"},
		&auth.OAuthBearerTokenWithSSH{OAuthBearerToken: auth.OAuthBearerToken{Token: "abcdef"}, PrivateKey: "private", PublicKey: "public", Passphrase: "pass"},
		&bitbucketserver.SudoableOAuthClient{
			Client:   auth.OAuthClient{Client: createOAuthClient(t, "ghi", "jkl")},
			Username: "neo",
		},
		&gitlab.SudoableToken{Token: "mnop", Sudo: "qrs"},
	} {
		auths[reflect.TypeOf(a).String()] = a
	}

	return auths
}

func setUpUserCredentialTest(t *testing.T, db *sql.DB) (context.Context, encryption.Key, *types.User) {
	if testing.Short() {
		t.Skip()
	}

	t.Helper()
	ctx := context.Background()

	// Create a user that allows us to link the credential somewhere.
	user, err := Users(db).Create(ctx, NewUser{
		Email:                 "a@example.com",
		Username:              "u2",
		Password:              "pw",
		EmailVerificationCode: "c",
	})
	if err != nil {
		t.Fatal(err)
	}

	return ctx, et.TestKey{}, user
}

type invalidAuth struct{}

var _ auth.Authenticator = &invalidAuth{}

func (*invalidAuth) Authenticate(_ *http.Request) error { panic("should not be called") }
func (*invalidAuth) Hash() string                       { panic("should not be called") }
