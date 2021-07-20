package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/cockroachdb/errors"

	"github.com/sourcegraph/sourcegraph/internal/extsvc/auth"
	"github.com/sourcegraph/sourcegraph/internal/httptestutil"
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

		if !errors.Is(errA, errors.Cause(errB)) {
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

func TestGetAuthenticatedUserV4(t *testing.T) {
	cli, save := newV4Client(t, "GetAuthenticatedUserV4")
	defer save()

	ctx := context.Background()

	user, err := cli.GetAuthenticatedUser(ctx)
	if err != nil {
		t.Fatal(err)
	}

	testutil.AssertGolden(t,
		"testdata/golden/GetAuthenticatedUserV4",
		update("GetAuthenticatedUserV4"),
		user,
	)
}

func TestLoadPullRequest(t *testing.T) {
	cli, save := newV4Client(t, "LoadPullRequest")
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
			err:  "GitHub pull request not found: 0",
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

func TestCreatePullRequest(t *testing.T) {
	cli, save := newV4Client(t, "CreatePullRequest")
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

func TestClosePullRequest(t *testing.T) {
	cli, save := newV4Client(t, "ClosePullRequest")
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

func TestReopenPullRequest(t *testing.T) {
	cli, save := newV4Client(t, "ReopenPullRequest")
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

func TestMarkPullRequestReadyForReview(t *testing.T) {
	cli, save := newV4Client(t, "MarkPullRequestReadyForReview")
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

func TestCreatePullRequestComment(t *testing.T) {
	cli, save := newV4Client(t, "CreatePullRequestComment")
	defer save()

	pr := &PullRequest{
		// https://github.com/sourcegraph/automation-testing/pull/44
		ID: "MDExOlB1bGxSZXF1ZXN0MzQxMDU5OTY5",
	}

	err := cli.CreatePullRequestComment(context.Background(), pr, "test-comment")
	if err != nil {
		t.Fatal(err)
	}
}

func TestMergePullRequest(t *testing.T) {
	cli, save := newV4Client(t, "TestMergePullRequest")
	defer save()

	t.Run("success", func(t *testing.T) {
		pr := &PullRequest{
			// https://github.com/sourcegraph/automation-testing/pull/424
			ID: "MDExOlB1bGxSZXF1ZXN0NTc3Nzc3NzEy",
		}

		err := cli.MergePullRequest(context.Background(), pr, true)
		if err != nil {
			t.Fatal(err)
		}

		testutil.AssertGolden(t,
			"testdata/golden/MergePullRequest-success",
			update("MergePullRequest"),
			pr,
		)
	})

	t.Run("not mergeable", func(t *testing.T) {
		pr := &PullRequest{
			// https://github.com/sourcegraph/automation-testing/pull/419
			ID: "MDExOlB1bGxSZXF1ZXN0NTY1Mzk1NTc3",
		}

		err := cli.MergePullRequest(context.Background(), pr, true)
		if err == nil {
			t.Fatal("invalid nil error")
		}

		testutil.AssertGolden(t,
			"testdata/golden/MergePullRequest-error",
			update("MergePullRequest"),
			err,
		)
	})
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

func TestV4Client_WithAuthenticator(t *testing.T) {
	uri, err := url.Parse("https://github.com")
	if err != nil {
		t.Fatal(err)
	}

	old := &V4Client{
		apiURL: uri,
		auth:   &auth.OAuthBearerToken{Token: "old_token"},
	}

	newToken := &auth.OAuthBearerToken{Token: "new_token"}
	new := old.WithAuthenticator(newToken)
	if old == new {
		t.Fatal("both clients have the same address")
	}

	if new.auth != newToken {
		t.Fatalf("token: want %q but got %q", newToken, new.auth)
	}
}

func newV4Client(t testing.TB, name string) (*V4Client, func()) {
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

	cli := NewV4Client(uri, &auth.OAuthBearerToken{
		Token: os.Getenv("GITHUB_TOKEN"),
	}, doer)

	return cli, save
}
