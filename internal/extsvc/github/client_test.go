package github

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/internal/httptestutil"
	"github.com/sourcegraph/sourcegraph/internal/rcache"
	"github.com/sourcegraph/sourcegraph/internal/testutil"
)

func TestUnmarshal(t *testing.T) {
	type result struct {
		FieldA string
		FieldB string
	}
	cases := map[string]string{
		// Valid
		`[]`:                                  "",
		`[{"FieldA": "hi"}]`:                  "",
		`[{"FieldA": "hi", "FieldB": "bye"}]`: "",

		// Error
		`[[]]`:            `graphql: cannot unmarshal at offset 2: before "[["; after "]]": json: cannot unmarshal array into Go value of type github.result`,
		`[{"FieldA": 1}]`: `graphql: cannot unmarshal at offset 13: before "[{\"FieldA\": 1"; after "}]": json: cannot unmarshal number`,
	}
	// Large body
	repeated := strings.Repeat(`{"FieldA": "hi", "FieldB": "bye"},`, 100)
	cases[fmt.Sprintf(`[%s {"FieldA": 1}, %s]`, repeated, repeated[:len(repeated)-1])] = `graphql: cannot unmarshal at offset 3414: before ", \"FieldB\": \"bye\"},{\"FieldA\": \"hi\", \"FieldB\": \"bye\"},{\"FieldA\": \"hi\", \"FieldB\": \"bye\"}, {\"FieldA\": 1"; after "}, {\"FieldA\": \"hi\", \"FieldB\": \"bye\"},{\"FieldA\": \"hi\", \"FieldB\": \"bye\"},{\"FieldA\": \"hi\", \"FieldB\": \"b": json: cannot unmarshal number`

	for data, errStr := range cases {
		var a []result
		var b []result
		errA := json.Unmarshal([]byte(data), &a)
		errB := unmarshal([]byte(data), &b)

		if len(data) > 50 {
			data = data[:50] + "..."
		}

		if !reflect.DeepEqual(a, b) {
			t.Errorf("Expected the same result unmarshalling %v\na: %v\nb: %v", data, a, b)
		}
		if !reflect.DeepEqual(errA, errors.Cause(errB)) {
			t.Errorf("Expected the same underlying error unmarshalling %v\na: %v\nb: %v", data, errA, errB)
		}
		got := ""
		if errB != nil {
			got = errB.Error()
		}
		if !strings.HasPrefix(got, errStr) {
			t.Errorf("Unexpected error message %v\ngot:  %s\nwant: %s", data, got, errStr)
		}
	}
}

func TestNewRepoCache(t *testing.T) {
	cmpOpts := cmp.AllowUnexported(rcache.Cache{})
	t.Run("GitHub.com", func(t *testing.T) {
		url, _ := url.Parse("https://www.github.com")
		token := "asdf"

		// github.com caches should:
		// (1) use githubProxyURL for the prefix hash rather than the given url
		// (2) have a TTL of 10 minutes
		key := sha256.Sum256([]byte(token + ":" + githubProxyURL.String()))
		prefix := "gh_repo:" + base64.URLEncoding.EncodeToString(key[:])
		got := newRepoCache(url, token)
		want := rcache.NewWithTTL(prefix, 600)
		if diff := cmp.Diff(want, got, cmpOpts); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("GitHub Enterprise", func(t *testing.T) {
		url, _ := url.Parse("https://www.sourcegraph.com")
		token := "asdf"

		// GitHub Enterprise caches should:
		// (1) use the given URL for the prefix hash
		// (2) have a TTL of 30 seconds
		key := sha256.Sum256([]byte(token + ":" + url.String()))
		prefix := "gh_repo:" + base64.URLEncoding.EncodeToString(key[:])
		got := newRepoCache(url, token)
		want := rcache.NewWithTTL(prefix, 30)
		if diff := cmp.Diff(want, got, cmpOpts); diff != "" {
			t.Fatal(diff)
		}
	})
}

var updateRegex = flag.String("update", "", "Update testdata of tests matching the given regex")

func update(name string) bool {
	if updateRegex == nil || *updateRegex == "" {
		return false
	}
	return regexp.MustCompile(*updateRegex).MatchString(name)
}

func TestClient_WithToken(t *testing.T) {
	uri, err := url.Parse("https://github.com")
	if err != nil {
		t.Fatal(err)
	}

	old := &Client{
		apiURL: uri,
		token:  "old_token",
	}

	newToken := "new_token"
	new := old.WithToken(newToken)
	if old == new {
		t.Fatal("both clients have the same address")
	}

	if new.token != newToken {
		t.Fatalf("token: want %q but got %q", newToken, new.token)
	}
}

// NOTE: To update VCR for this test, please use the token of "sourcegraph-vcr"
// for GITHUB_TOKEN, which can be found in 1Password.
func TestClient_ListAffiliatedRepositories(t *testing.T) {
	tests := []struct {
		name       string
		visibility Visibility
		wantRepos  []*Repository
	}{
		{
			name:       "list all repositories",
			visibility: VisibilityAll,
			wantRepos: []*Repository{
				{
					ID:               "MDEwOlJlcG9zaXRvcnkyNjMwMzQxNTE=",
					DatabaseID:       263034151,
					NameWithOwner:    "sourcegraph-vcr-repos/private-org-repo-1",
					URL:              "https://github.com/sourcegraph-vcr-repos/private-org-repo-1",
					IsPrivate:        true,
					ViewerPermission: "ADMIN",
				}, {
					ID:               "MDEwOlJlcG9zaXRvcnkyNjMwMzQwNzM=",
					DatabaseID:       263034073,
					NameWithOwner:    "sourcegraph-vcr/private-user-repo-1",
					URL:              "https://github.com/sourcegraph-vcr/private-user-repo-1",
					IsPrivate:        true,
					ViewerPermission: "ADMIN",
				}, {
					ID:               "MDEwOlJlcG9zaXRvcnkyNjMwMzM5NDk=",
					DatabaseID:       263033949,
					NameWithOwner:    "sourcegraph-vcr/public-user-repo-1",
					URL:              "https://github.com/sourcegraph-vcr/public-user-repo-1",
					ViewerPermission: "ADMIN",
				}, {
					ID:               "MDEwOlJlcG9zaXRvcnkyNjMwMzM3NjE=",
					DatabaseID:       263033761,
					NameWithOwner:    "sourcegraph-vcr-repos/public-org-repo-1",
					URL:              "https://github.com/sourcegraph-vcr-repos/public-org-repo-1",
					ViewerPermission: "ADMIN",
				},
			},
		},
		{
			name:       "list public repositories",
			visibility: VisibilityPublic,
			wantRepos: []*Repository{
				{
					ID:               "MDEwOlJlcG9zaXRvcnkyNjMwMzM5NDk=",
					DatabaseID:       263033949,
					NameWithOwner:    "sourcegraph-vcr/public-user-repo-1",
					URL:              "https://github.com/sourcegraph-vcr/public-user-repo-1",
					ViewerPermission: "ADMIN",
				}, {
					ID:               "MDEwOlJlcG9zaXRvcnkyNjMwMzM3NjE=",
					DatabaseID:       263033761,
					NameWithOwner:    "sourcegraph-vcr-repos/public-org-repo-1",
					URL:              "https://github.com/sourcegraph-vcr-repos/public-org-repo-1",
					ViewerPermission: "ADMIN",
				},
			},
		},
		{
			name:       "list private repositories",
			visibility: VisibilityPrivate,
			wantRepos: []*Repository{
				{
					ID:               "MDEwOlJlcG9zaXRvcnkyNjMwMzQxNTE=",
					DatabaseID:       263034151,
					NameWithOwner:    "sourcegraph-vcr-repos/private-org-repo-1",
					URL:              "https://github.com/sourcegraph-vcr-repos/private-org-repo-1",
					IsPrivate:        true,
					ViewerPermission: "ADMIN",
				}, {
					ID:               "MDEwOlJlcG9zaXRvcnkyNjMwMzQwNzM=",
					DatabaseID:       263034073,
					NameWithOwner:    "sourcegraph-vcr/private-user-repo-1",
					URL:              "https://github.com/sourcegraph-vcr/private-user-repo-1",
					IsPrivate:        true,
					ViewerPermission: "ADMIN",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client, save := newClient(t, "ListAffiliatedRepositories_"+test.name)
			defer save()

			repos, _, _, err := client.ListAffiliatedRepositories(context.Background(), test.visibility, 1)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(test.wantRepos, repos); diff != "" {
				t.Fatalf("Repos mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestClient_LoadPullRequest(t *testing.T) {
	cli, save := newClient(t, "LoadPullRequest")
	defer save()

	for i, tc := range []struct {
		name string
		ctx  context.Context
		pr   *PullRequest
		err  string
	}{
		{
			name: "non-existing-repo",
			pr:   &PullRequest{RepoWithOwner: "whoisthis/sourcegraph", Number: 5550},
			err:  "GitHub repository not found",
		},
		{
			name: "non-existing-pr",
			pr:   &PullRequest{RepoWithOwner: "sourcegraph/sourcegraph", Number: 0},
			err:  "GitHub pull requests not found: 0",
		},
		{
			name: "success",
			pr:   &PullRequest{RepoWithOwner: "sourcegraph/sourcegraph", Number: 5550},
		},
		{
			name: "with more than 250 events",
			pr:   &PullRequest{RepoWithOwner: "sourcegraph/sourcegraph", Number: 596},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.ctx == nil {
				tc.ctx = context.Background()
			}

			if tc.err == "" {
				tc.err = "<nil>"
			}

			err := cli.LoadPullRequest(tc.ctx, tc.pr)
			if have, want := fmt.Sprint(err), tc.err; have != want {
				t.Errorf("error:\nhave: %q\nwant: %q", have, want)
			}

			if err != nil {
				return
			}

			testutil.AssertGolden(t,
				"testdata/golden/LoadPullRequest-"+strconv.Itoa(i),
				update("LoadPullRequest"),
				tc.pr,
			)
		})
	}
}

func TestClient_CreatePullRequest(t *testing.T) {
	cli, save := newClient(t, "CreatePullRequest")
	defer save()

	// Repository used: sourcegraph/automation-testing
	// The requests here cannot be easily rerun with `-update` since you can
	// only open a pull request once.
	// In order to update specific tests, comment out the other ones and then
	// run with -update.
	for i, tc := range []struct {
		name  string
		ctx   context.Context
		input *CreatePullRequestInput
		err   string
	}{
		{
			name: "success",
			input: &CreatePullRequestInput{
				RepositoryID: "MDEwOlJlcG9zaXRvcnkyMjExNDc1MTM=",
				BaseRefName:  "master",
				HeadRefName:  "test-pr-3",
				Title:        "This is a test PR, feel free to ignore",
				Body:         "I'm opening this PR to test something. Please ignore.",
			},
		},
		{
			name: "already-existing-pr",
			input: &CreatePullRequestInput{
				RepositoryID: "MDEwOlJlcG9zaXRvcnkyMjExNDc1MTM=",
				BaseRefName:  "master",
				HeadRefName:  "always-open-pr",
				Title:        "This is a test PR that is always open",
				Body:         "Feel free to ignore this. This is a test PR that is always open.",
			},
			err: ErrPullRequestAlreadyExists.Error(),
		},
		{
			name: "invalid-head-ref",
			input: &CreatePullRequestInput{
				RepositoryID: "MDEwOlJlcG9zaXRvcnkyMjExNDc1MTM=",
				BaseRefName:  "master",
				HeadRefName:  "this-head-ref-should-not-exist",
				Title:        "Test",
			},
			err: "error in GraphQL response: Head sha can't be blank, Base sha can't be blank, No commits between master and this-head-ref-should-not-exist, Head ref must be a branch",
		},
		{
			name: "draft-pr",
			input: &CreatePullRequestInput{
				RepositoryID: "MDEwOlJlcG9zaXRvcnkyMjExNDc1MTM=",
				BaseRefName:  "master",
				HeadRefName:  "test-pr-4",
				Title:        "This is a test PR, feel free to ignore",
				Body:         "I'm opening this PR to test something. Please ignore.",
				Draft:        true,
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.ctx == nil {
				tc.ctx = context.Background()
			}

			if tc.err == "" {
				tc.err = "<nil>"
			}

			pr, err := cli.CreatePullRequest(tc.ctx, tc.input)
			if have, want := fmt.Sprint(err), tc.err; have != want {
				t.Errorf("error:\nhave: %q\nwant: %q", have, want)
			}

			if err != nil {
				return
			}

			testutil.AssertGolden(t,
				"testdata/golden/CreatePullRequest-"+strconv.Itoa(i),
				update("CreatePullRequest"),
				pr,
			)
		})
	}
}

func TestClient_ClosePullRequest(t *testing.T) {
	cli, save := newClient(t, "ClosePullRequest")
	defer save()

	// Repository used: sourcegraph/automation-testing
	// The requests here cannot be easily rerun with `-update` since you can
	// only close a pull request once.
	// In order to update specific tests, comment out the other ones and then
	// run with -update.
	for i, tc := range []struct {
		name string
		ctx  context.Context
		pr   *PullRequest
		err  string
	}{
		{
			name: "success",
			// github.com/sourcegraph/automation-testing/pull/44
			pr: &PullRequest{ID: "MDExOlB1bGxSZXF1ZXN0MzQxMDU5OTY5"},
		},
		{
			name: "already closed",
			// github.com/sourcegraph/automation-testing/pull/29
			pr: &PullRequest{ID: "MDExOlB1bGxSZXF1ZXN0MzQxMDU5OTY5"},
			// Doesn't return an error
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.ctx == nil {
				tc.ctx = context.Background()
			}

			if tc.err == "" {
				tc.err = "<nil>"
			}

			err := cli.ClosePullRequest(tc.ctx, tc.pr)
			if have, want := fmt.Sprint(err), tc.err; have != want {
				t.Errorf("error:\nhave: %q\nwant: %q", have, want)
			}

			if err != nil {
				return
			}

			testutil.AssertGolden(t,
				"testdata/golden/ClosePullRequest-"+strconv.Itoa(i),
				update("ClosePullRequest"),
				tc.pr,
			)
		})
	}
}

func TestClient_ReopenPullRequest(t *testing.T) {
	cli, save := newClient(t, "ReopenPullRequest")
	defer save()

	// Repository used: sourcegraph/automation-testing
	// The requests here cannot be easily rerun with `-update` since you can
	// only reopen a pull request once.
	// In order to update specific tests, comment out the other ones and then
	// run with -update.
	for i, tc := range []struct {
		name string
		ctx  context.Context
		pr   *PullRequest
	}{
		{
			name: "success",
			// https://github.com/sourcegraph/automation-testing/pull/356
			pr: &PullRequest{ID: "MDExOlB1bGxSZXF1ZXN0NDg4NjEzODA3"},
		},
		{
			name: "already open",
			// https://github.com/sourcegraph/automation-testing/pull/355
			pr: &PullRequest{ID: "MDExOlB1bGxSZXF1ZXN0NDg4NjA0NTQ5"},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.ctx == nil {
				tc.ctx = context.Background()
			}

			err := cli.ReopenPullRequest(tc.ctx, tc.pr)
			if err != nil {
				t.Fatalf("ReopenPullRequest returned unexpected error: %s", err)
			}

			testutil.AssertGolden(t,
				"testdata/golden/ReopenPullRequest-"+strconv.Itoa(i),
				update("ReopenPullRequest"),
				tc.pr,
			)
		})
	}
}

func TestClient_MarkPullRequestReadyForReview(t *testing.T) {
	cli, save := newClient(t, "MarkPullRequestReadyForReview")
	defer save()

	// Repository used: sourcegraph/automation-testing
	// The requests here cannot be easily rerun with `-update` since you can
	// only get the success response if the changeset is in draft mode.
	for i, tc := range []struct {
		name string
		ctx  context.Context
		pr   *PullRequest
	}{
		{
			name: "success",
			// https://github.com/sourcegraph/automation-testing/pull/361
			pr: &PullRequest{ID: "MDExOlB1bGxSZXF1ZXN0NTA0NDczNDU1"},
		},
		{
			name: "already ready for review",
			// https://github.com/sourcegraph/automation-testing/pull/362
			pr: &PullRequest{ID: "MDExOlB1bGxSZXF1ZXN0NTA2MDg0ODU2"},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.ctx == nil {
				tc.ctx = context.Background()
			}

			err := cli.MarkPullRequestReadyForReview(tc.ctx, tc.pr)
			if err != nil {
				t.Fatalf("MarkPullRequestReadyForReview returned unexpected error: %s", err)
			}

			testutil.AssertGolden(t,
				"testdata/golden/MarkPullRequestReadyForReview-"+strconv.Itoa(i),
				update("MarkPullRequestReadyForReview"),
				tc.pr,
			)
		})
	}
}

func TestClient_GetAuthenticatedUserOrgs(t *testing.T) {
	cli, save := newClient(t, "GetAuthenticatedUserOrgs")
	defer save()

	ctx := context.Background()
	orgs, err := cli.GetAuthenticatedUserOrgs(ctx)
	if err != nil {
		t.Fatal(err)
	}

	testutil.AssertGolden(t,
		"testdata/golden/GetAuthenticatedUserOrgs",
		update("GetAuthenticatedUserOrgs"),
		orgs,
	)
}

func newClient(t testing.TB, name string) (*Client, func()) {
	t.Helper()

	cf, save := httptestutil.NewGitHubRecorderFactory(t, update(name), name)
	uri, err := url.Parse("https://github.com")
	if err != nil {
		t.Fatal(err)
	}

	doer, err := cf.Doer()
	if err != nil {
		t.Fatal(err)
	}

	cli := NewClient(uri, os.Getenv("GITHUB_TOKEN"), doer)

	return cli, save
}

func TestEstimateGraphQLCost(t *testing.T) {
	for _, tc := range []struct {
		name  string
		query string
		want  int
	}{
		{
			name: "Canonical example",
			query: `query {
  viewer {
    login
    repositories(first: 100) {
      edges {
        node {
          id

          issues(first: 50) {
            edges {
              node {
                id
                labels(first: 60) {
                  edges {
                    node {
                      id
                      name
                    }
                  }
                }
              }
            }
          }
        }
      }
    }
  }
}`,
			want: 51,
		},
		{
			name: "simple query",
			query: `
query {
  viewer {
    repositories(first: 50) {
      edges {
        repository:node {
          name
          issues(first: 10) {
            totalCount
            edges {
              node {
                title
                bodyHTML
              }
            }
          }
        }
      }
    }
  }
}
`,
			want: 1,
		},
		{
			name: "complex query",
			query: `query {
  viewer {
    repositories(first: 50) {
      edges {
        repository:node {
          name

          pullRequests(first: 20) {
            edges {
              pullRequest:node {
                title

                comments(first: 10) {
                  edges {
                    comment:node {
                      bodyHTML
                    }
                  }
                }
              }
            }
          }

          issues(first: 20) {
            totalCount
            edges {
              issue:node {
                title
                bodyHTML

                comments(first: 10) {
                  edges {
                    comment:node {
                      bodyHTML
                    }
                  }
                }
              }
            }
          }
        }
      }
    }

    followers(first: 10) {
      edges {
        follower:node {
          login
        }
      }
    }
  }
}`,
			want: 21,
		},
		{
			name: "Multiple top level queries",
			query: `query {
  thing
}
query{
  thing
}
`,
			want: 1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			have, err := estimateGraphQLCost(tc.query)
			if err != nil {
				t.Fatal(err)
			}
			if have != tc.want {
				t.Fatalf("have %d, want %d", have, tc.want)
			}
		})
	}
}
