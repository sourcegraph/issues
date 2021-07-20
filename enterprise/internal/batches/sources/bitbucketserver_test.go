package sources

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/cockroachdb/errors"
	"github.com/inconshreveable/log15"

	btypes "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/auth"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/bitbucketserver"
	"github.com/sourcegraph/sourcegraph/internal/testutil"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/schema"
)

func TestBitbucketServerSource_LoadChangeset(t *testing.T) {
	instanceURL := os.Getenv("BITBUCKET_SERVER_URL")
	if instanceURL == "" {
		// The test fixtures and golden files were generated with
		// this config pointed to bitbucket.sgdev.org
		instanceURL = "https://bitbucket.sgdev.org"
	}

	repo := &types.Repo{
		Metadata: &bitbucketserver.Repo{
			Slug:    "vegeta",
			Project: &bitbucketserver.Project{Key: "SOUR"},
		},
	}

	changesets := []*Changeset{
		{Repo: repo, Changeset: &btypes.Changeset{ExternalID: "2"}},
		{Repo: repo, Changeset: &btypes.Changeset{ExternalID: "999"}},
	}

	testCases := []struct {
		name string
		cs   *Changeset
		err  string
	}{
		{
			name: "found",
			cs:   changesets[0],
		},
		{
			name: "not-found",
			cs:   changesets[1],
			err:  `Changeset with external ID 999 not found`,
		},
	}

	for _, tc := range testCases {
		tc := tc
		tc.name = "BitbucketServerSource_LoadChangeset_" + tc.name

		t.Run(tc.name, func(t *testing.T) {
			cf, save := newClientFactory(t, tc.name)
			defer save(t)

			lg := log15.New()
			lg.SetHandler(log15.DiscardHandler())

			svc := &types.ExternalService{
				Kind: extsvc.KindBitbucketServer,
				Config: marshalJSON(t, &schema.BitbucketServerConnection{
					Url:   instanceURL,
					Token: os.Getenv("BITBUCKET_SERVER_TOKEN"),
				}),
			}

			bbsSrc, err := NewBitbucketServerSource(svc, cf)
			if err != nil {
				t.Fatal(err)
			}

			ctx := context.Background()
			if tc.err == "" {
				tc.err = "<nil>"
			}

			tc.err = strings.ReplaceAll(tc.err, "${INSTANCEURL}", instanceURL)

			err = bbsSrc.LoadChangeset(ctx, tc.cs)
			if have, want := fmt.Sprint(err), tc.err; have != want {
				t.Errorf("error:\nhave: %q\nwant: %q", have, want)
			}

			if err != nil {
				return
			}

			testutil.AssertGolden(
				t,
				"testdata/golden/"+tc.name,
				update(tc.name),
				tc.cs.Changeset.Metadata.(*bitbucketserver.PullRequest),
			)
		})
	}
}

func TestBitbucketServerSource_CreateChangeset(t *testing.T) {
	instanceURL := os.Getenv("BITBUCKET_SERVER_URL")
	if instanceURL == "" {
		// The test fixtures and golden files were generated with
		// this config pointed to bitbucket.sgdev.org
		instanceURL = "https://bitbucket.sgdev.org"
	}

	repo := &types.Repo{
		Metadata: &bitbucketserver.Repo{
			ID:      10070,
			Slug:    "automation-testing",
			Project: &bitbucketserver.Project{Key: "SOUR"},
		},
	}

	testCases := []struct {
		name   string
		cs     *Changeset
		err    string
		exists bool
	}{
		{
			name: "abbreviated refs",
			cs: &Changeset{
				Title:     "This is a test PR",
				Body:      "This is the body of a test PR",
				BaseRef:   "master",
				HeadRef:   "test-pr-bbs-11",
				Repo:      repo,
				Changeset: &btypes.Changeset{},
			},
		},
		{
			name: "success",
			cs: &Changeset{
				Title:     "This is a test PR",
				Body:      "This is the body of a test PR",
				BaseRef:   "refs/heads/master",
				HeadRef:   "refs/heads/test-pr-bbs-12",
				Repo:      repo,
				Changeset: &btypes.Changeset{},
			},
		},
		{
			name: "already exists",
			cs: &Changeset{
				Title:     "This is a test PR",
				Body:      "This is the body of a test PR",
				BaseRef:   "refs/heads/master",
				HeadRef:   "refs/heads/always-open-pr-bbs",
				Repo:      repo,
				Changeset: &btypes.Changeset{},
			},
			// CreateChangeset is idempotent so if the PR already exists
			// it is not an error
			err:    "",
			exists: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		tc.name = "BitbucketServerSource_CreateChangeset_" + tc.name

		t.Run(tc.name, func(t *testing.T) {
			cf, save := newClientFactory(t, tc.name)
			defer save(t)

			lg := log15.New()
			lg.SetHandler(log15.DiscardHandler())

			svc := &types.ExternalService{
				Kind: extsvc.KindBitbucketServer,
				Config: marshalJSON(t, &schema.BitbucketServerConnection{
					Url:   instanceURL,
					Token: os.Getenv("BITBUCKET_SERVER_TOKEN"),
				}),
			}

			bbsSrc, err := NewBitbucketServerSource(svc, cf)
			if err != nil {
				t.Fatal(err)
			}

			ctx := context.Background()
			if tc.err == "" {
				tc.err = "<nil>"
			}

			tc.err = strings.ReplaceAll(tc.err, "${INSTANCEURL}", instanceURL)

			exists, err := bbsSrc.CreateChangeset(ctx, tc.cs)
			if have, want := fmt.Sprint(err), tc.err; have != want {
				t.Errorf("error:\nhave: %q\nwant: %q", have, want)
			}

			if err != nil {
				return
			}

			if have, want := exists, tc.exists; have != want {
				t.Errorf("exists:\nhave: %t\nwant: %t", have, want)
			}

			pr := tc.cs.Changeset.Metadata.(*bitbucketserver.PullRequest)
			testutil.AssertGolden(t, "testdata/golden/"+tc.name, update(tc.name), pr)
		})
	}
}

func TestBitbucketServerSource_CloseChangeset(t *testing.T) {
	instanceURL := os.Getenv("BITBUCKET_SERVER_URL")
	if instanceURL == "" {
		// The test fixtures and golden files were generated with
		// this config pointed to bitbucket.sgdev.org
		instanceURL = "https://bitbucket.sgdev.org"
	}

	pr := &bitbucketserver.PullRequest{ID: 59, Version: 4}
	pr.ToRef.Repository.Slug = "automation-testing"
	pr.ToRef.Repository.Project.Key = "SOUR"

	testCases := []struct {
		name string
		cs   *Changeset
		err  string
	}{
		{
			name: "success",
			cs:   &Changeset{Changeset: &btypes.Changeset{Metadata: pr}},
		},
	}

	for _, tc := range testCases {
		tc := tc
		tc.name = "BitbucketServerSource_CloseChangeset_" + strings.ReplaceAll(tc.name, " ", "_")

		t.Run(tc.name, func(t *testing.T) {
			cf, save := newClientFactory(t, tc.name)
			defer save(t)

			lg := log15.New()
			lg.SetHandler(log15.DiscardHandler())

			svc := &types.ExternalService{
				Kind: extsvc.KindBitbucketServer,
				Config: marshalJSON(t, &schema.BitbucketServerConnection{
					Url:   instanceURL,
					Token: os.Getenv("BITBUCKET_SERVER_TOKEN"),
				}),
			}

			bbsSrc, err := NewBitbucketServerSource(svc, cf)
			if err != nil {
				t.Fatal(err)
			}

			ctx := context.Background()
			if tc.err == "" {
				tc.err = "<nil>"
			}

			tc.err = strings.ReplaceAll(tc.err, "${INSTANCEURL}", instanceURL)

			err = bbsSrc.CloseChangeset(ctx, tc.cs)
			if have, want := fmt.Sprint(err), tc.err; have != want {
				t.Errorf("error:\nhave: %q\nwant: %q", have, want)
			}

			if err != nil {
				return
			}

			pr := tc.cs.Changeset.Metadata.(*bitbucketserver.PullRequest)
			testutil.AssertGolden(t, "testdata/golden/"+tc.name, update(tc.name), pr)
		})
	}
}

func TestBitbucketServerSource_ReopenChangeset(t *testing.T) {
	instanceURL := os.Getenv("BITBUCKET_SERVER_URL")
	if instanceURL == "" {
		// The test fixtures and golden files were generated with
		// this config pointed to bitbucket.sgdev.org
		instanceURL = "https://bitbucket.sgdev.org"
	}

	pr := &bitbucketserver.PullRequest{ID: 95, Version: 1}
	pr.ToRef.Repository.Slug = "automation-testing"
	pr.ToRef.Repository.Project.Key = "SOUR"

	testCases := []struct {
		name string
		cs   *Changeset
		err  string
	}{
		{
			name: "success",
			cs:   &Changeset{Changeset: &btypes.Changeset{Metadata: pr}},
		},
	}

	for _, tc := range testCases {
		tc := tc
		tc.name = "BitbucketServerSource_ReopenChangeset_" + strings.ReplaceAll(tc.name, " ", "_")

		t.Run(tc.name, func(t *testing.T) {
			cf, save := newClientFactory(t, tc.name)
			defer save(t)

			lg := log15.New()
			lg.SetHandler(log15.DiscardHandler())

			svc := &types.ExternalService{
				Kind: extsvc.KindBitbucketServer,
				Config: marshalJSON(t, &schema.BitbucketServerConnection{
					Url:   instanceURL,
					Token: os.Getenv("BITBUCKET_SERVER_TOKEN"),
				}),
			}

			bbsSrc, err := NewBitbucketServerSource(svc, cf)
			if err != nil {
				t.Fatal(err)
			}

			ctx := context.Background()
			if tc.err == "" {
				tc.err = "<nil>"
			}

			tc.err = strings.ReplaceAll(tc.err, "${INSTANCEURL}", instanceURL)

			err = bbsSrc.ReopenChangeset(ctx, tc.cs)
			if have, want := fmt.Sprint(err), tc.err; have != want {
				t.Errorf("error:\nhave: %q\nwant: %q", have, want)
			}

			if err != nil {
				return
			}

			pr := tc.cs.Changeset.Metadata.(*bitbucketserver.PullRequest)
			testutil.AssertGolden(t, "testdata/golden/"+tc.name, update(tc.name), pr)
		})
	}
}

func TestBitbucketServerSource_UpdateChangeset(t *testing.T) {
	instanceURL := os.Getenv("BITBUCKET_SERVER_URL")
	if instanceURL == "" {
		// The test fixtures and golden files were generated with
		// this config pointed to bitbucket.sgdev.org
		instanceURL = "https://bitbucket.sgdev.org"
	}

	pr := &bitbucketserver.PullRequest{ID: 43, Version: 5}
	pr.ToRef.Repository.Slug = "automation-testing"
	pr.ToRef.Repository.Project.Key = "SOUR"

	testCases := []struct {
		name string
		cs   *Changeset
		err  string
	}{
		{
			name: "success",
			cs: &Changeset{
				Title:     "This is a new title",
				Body:      "This is a new body",
				BaseRef:   "refs/heads/master",
				Changeset: &btypes.Changeset{Metadata: pr},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		tc.name = "BitbucketServerSource_UpdateChangeset_" + strings.ReplaceAll(tc.name, " ", "_")

		t.Run(tc.name, func(t *testing.T) {
			cf, save := newClientFactory(t, tc.name)
			defer save(t)

			lg := log15.New()
			lg.SetHandler(log15.DiscardHandler())

			svc := &types.ExternalService{
				Kind: extsvc.KindBitbucketServer,
				Config: marshalJSON(t, &schema.BitbucketServerConnection{
					Url:   instanceURL,
					Token: os.Getenv("BITBUCKET_SERVER_TOKEN"),
				}),
			}

			bbsSrc, err := NewBitbucketServerSource(svc, cf)
			if err != nil {
				t.Fatal(err)
			}

			ctx := context.Background()
			if tc.err == "" {
				tc.err = "<nil>"
			}

			tc.err = strings.ReplaceAll(tc.err, "${INSTANCEURL}", instanceURL)

			err = bbsSrc.UpdateChangeset(ctx, tc.cs)
			if have, want := fmt.Sprint(err), tc.err; have != want {
				t.Errorf("error:\nhave: %q\nwant: %q", have, want)
			}

			if err != nil {
				return
			}

			pr := tc.cs.Changeset.Metadata.(*bitbucketserver.PullRequest)
			testutil.AssertGolden(t, "testdata/golden/"+tc.name, update(tc.name), pr)
		})
	}
}

func TestBitbucketServerSource_CreateComment(t *testing.T) {
	instanceURL := os.Getenv("BITBUCKET_SERVER_URL")
	if instanceURL == "" {
		// The test fixtures and golden files were generated with
		// this config pointed to bitbucket.sgdev.org
		instanceURL = "https://bitbucket.sgdev.org"
	}

	pr := &bitbucketserver.PullRequest{ID: 59, Version: 4}
	pr.ToRef.Repository.Slug = "automation-testing"
	pr.ToRef.Repository.Project.Key = "SOUR"

	testCases := []struct {
		name string
		cs   *Changeset
		err  string
	}{
		{
			name: "success",
			cs:   &Changeset{Changeset: &btypes.Changeset{Metadata: pr}},
		},
	}

	for _, tc := range testCases {
		tc := tc
		tc.name = "BitbucketServerSource_CreateComment_" + strings.ReplaceAll(tc.name, " ", "_")

		t.Run(tc.name, func(t *testing.T) {
			cf, save := newClientFactory(t, tc.name)
			defer save(t)

			lg := log15.New()
			lg.SetHandler(log15.DiscardHandler())

			svc := &types.ExternalService{
				Kind: extsvc.KindBitbucketServer,
				Config: marshalJSON(t, &schema.BitbucketServerConnection{
					Url:   instanceURL,
					Token: os.Getenv("BITBUCKET_SERVER_TOKEN"),
				}),
			}

			bbsSrc, err := NewBitbucketServerSource(svc, cf)
			if err != nil {
				t.Fatal(err)
			}

			ctx := context.Background()
			if tc.err == "" {
				tc.err = "<nil>"
			}

			tc.err = strings.ReplaceAll(tc.err, "${INSTANCEURL}", instanceURL)

			err = bbsSrc.CreateComment(ctx, tc.cs, "test-comment")
			if have, want := fmt.Sprint(err), tc.err; have != want {
				t.Errorf("error:\nhave: %q\nwant: %q", have, want)
			}
		})
	}
}

func TestBitbucketServerSource_WithAuthenticator(t *testing.T) {
	svc := &types.ExternalService{
		Kind: extsvc.KindBitbucketServer,
		Config: marshalJSON(t, &schema.BitbucketServerConnection{
			Url:   "https://bitbucket.sgdev.org",
			Token: os.Getenv("BITBUCKET_SERVER_TOKEN"),
		}),
	}

	bbsSrc, err := NewBitbucketServerSource(svc, nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("supported", func(t *testing.T) {
		for name, tc := range map[string]auth.Authenticator{
			"BasicAuth":           &auth.BasicAuth{},
			"OAuthBearerToken":    &auth.OAuthBearerToken{},
			"SudoableOAuthClient": &bitbucketserver.SudoableOAuthClient{},
		} {
			t.Run(name, func(t *testing.T) {
				src, err := bbsSrc.WithAuthenticator(tc)
				if err != nil {
					t.Errorf("unexpected non-nil error: %v", err)
				}

				if gs, ok := src.(*BitbucketServerSource); !ok {
					t.Error("cannot coerce Source into bbsSource")
				} else if gs == nil {
					t.Error("unexpected nil Source")
				} else if gs.au != tc {
					t.Errorf("incorrect authenticator: have=%v want=%v", gs.au, tc)
				}
			})
		}
	})

	t.Run("unsupported", func(t *testing.T) {
		for name, tc := range map[string]auth.Authenticator{
			"nil":         nil,
			"OAuthClient": &auth.OAuthClient{},
		} {
			t.Run(name, func(t *testing.T) {
				src, err := bbsSrc.WithAuthenticator(tc)
				if err == nil {
					t.Error("unexpected nil error")
				} else if !errors.HasType(err, UnsupportedAuthenticatorError{}) {
					t.Errorf("unexpected error of type %T: %v", err, err)
				}
				if src != nil {
					t.Errorf("expected non-nil Source: %v", src)
				}
			})
		}
	})
}
