// +build gqltest

package main

import (
	"strings"
	"testing"
	"time"

	"github.com/sourcegraph/sourcegraph/internal/gqltestutil"
	"github.com/sourcegraph/sourcegraph/schema"
)

func TestSiteConfig(t *testing.T) {
	t.Run("builtin auth provider: allowSignup", func(t *testing.T) {
		// Sign up a new user is allowed by default.
		const testUsername1 = "gqltest-auth-user-1"
		testClient1, err := gqltestutil.SignUp(*baseURL, testUsername1+"@sourcegraph.com", testUsername1, "mysecurepassword")
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			err := client.DeleteUser(testClient1.AuthenticatedUserID(), true)
			if err != nil {
				t.Fatal(err)
			}
		}()

		// Update site configuration to not allow sign up for builtin auth provider.
		siteConfig, err := client.SiteConfiguration()
		if err != nil {
			t.Fatal(err)
		}
		oldSiteConfig := new(schema.SiteConfiguration)
		*oldSiteConfig = *siteConfig
		defer func() {
			err = client.UpdateSiteConfiguration(oldSiteConfig)
			if err != nil {
				t.Fatal(err)
			}
		}()

		siteConfig.AuthProviders = []schema.AuthProviders{
			{
				Builtin: &schema.BuiltinAuthProvider{
					AllowSignup: false,
					Type:        "builtin",
				},
			},
		}
		err = client.UpdateSiteConfiguration(siteConfig)
		if err != nil {
			t.Fatal(err)
		}

		var lastErr error
		// Retry because the configuration update endpoint is eventually consistent
		err = gqltestutil.Retry(5*time.Second, func() error {
			// Sign up a new user should fail.
			const testUsername2 = "gqltest-auth-user-2"
			testClient2, err := gqltestutil.SignUp(*baseURL, testUsername2+"@sourcegraph.com", testUsername2, "mysecurepassword")
			if err != nil {
				if strings.Contains(err.Error(), "Signup is not enabled") {
					return nil
				}
				t.Fatal(err)
			}
			defer func() {
				err := client.DeleteUser(testClient2.AuthenticatedUserID(), true)
				if err != nil {
					t.Fatal(err)
				}
			}()
			return gqltestutil.ErrContinueRetry
		})
		if err != nil {
			t.Fatal(err, "lastErr:", lastErr)
		}
	})
}
