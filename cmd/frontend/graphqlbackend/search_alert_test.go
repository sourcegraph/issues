package graphqlbackend

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/hashicorp/go-multierror"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/search"
	"github.com/sourcegraph/sourcegraph/internal/search/query"
	"github.com/sourcegraph/sourcegraph/schema"
)

func TestSearchPatternForSuggestion(t *testing.T) {
	cases := []struct {
		Name  string
		Alert searchAlert
		Want  string
	}{
		{
			Name: "with_regex_suggestion",
			Alert: searchAlert{
				title:       "An alert for regex",
				description: "An alert for regex",
				proposedQueries: []*searchQueryDescription{
					{
						description: "Some query description",
						query:       "repo:github.com/sourcegraph/sourcegraph",
						patternType: query.SearchTypeRegex,
					},
				},
			},
			Want: "repo:github.com/sourcegraph/sourcegraph patternType:regexp",
		},
		{
			Name: "with_structural_suggestion",
			Alert: searchAlert{
				title:       "An alert for structural",
				description: "An alert for structural",
				proposedQueries: []*searchQueryDescription{
					{
						description: "Some query description",
						query:       "repo:github.com/sourcegraph/sourcegraph",
						patternType: query.SearchTypeStructural,
					},
				},
			},
			Want: "repo:github.com/sourcegraph/sourcegraph patternType:structural",
		},
	}

	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			got := tt.Alert.ProposedQueries()
			if !reflect.DeepEqual((*got)[0].Query(), tt.Want) {
				t.Errorf("got: %s, want: %s", (*got)[0].query, tt.Want)
			}
		})
	}
}

func TestAddQueryRegexpField(t *testing.T) {
	tests := []struct {
		query      string
		addField   string
		addPattern string
		want       string
	}{
		{
			query:      "",
			addField:   "repo",
			addPattern: "p",
			want:       "repo:p",
		},
		{
			query:      "foo",
			addField:   "repo",
			addPattern: "p",
			want:       "foo repo:p",
		},
		{
			query:      "foo repo:q",
			addField:   "repo",
			addPattern: "p",
			want:       "foo repo:q repo:p",
		},
		{
			query:      "foo repo:p",
			addField:   "repo",
			addPattern: "p",
			want:       "foo repo:p",
		},
		{
			query:      "foo repo:p",
			addField:   "repo",
			addPattern: "pp",
			want:       "foo repo:pp",
		},
		{
			query:      "foo repo:p",
			addField:   "repo",
			addPattern: "^p",
			want:       "foo repo:^p",
		},
		{
			query:      "foo repo:p",
			addField:   "repo",
			addPattern: "p$",
			want:       "foo repo:p$",
		},
		{
			query:      "foo repo:^p",
			addField:   "repo",
			addPattern: "^pq",
			want:       "foo repo:^pq",
		},
		{
			query:      "foo repo:p$",
			addField:   "repo",
			addPattern: "qp$",
			want:       "foo repo:qp$",
		},
		{
			query:      "foo repo:^p",
			addField:   "repo",
			addPattern: "x$",
			want:       "foo repo:^p repo:x$",
		},
		{
			query:      "foo repo:p|q",
			addField:   "repo",
			addPattern: "pq",
			want:       "foo repo:p|q repo:pq",
		},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("%s, add %s:%s", test.query, test.addField, test.addPattern), func(t *testing.T) {
			parseTree, err := query.Parse(test.query)
			if err != nil {
				t.Fatal(err)
			}
			got := addRegexpField(parseTree, test.addField, test.addPattern)
			if got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestErrorToAlertStructuralSearch(t *testing.T) {
	cases := []struct {
		name           string
		errors         []error
		wantErrors     []error
		wantAlertTitle string
	}{
		{
			name:           "multierr_is_unaffected",
			errors:         []error{errors.New("some error")},
			wantErrors:     []error{errors.New("some error")},
			wantAlertTitle: "",
		},
		{
			name: "surface_friendly_alert_on_oom_err_message",
			errors: []error{
				errors.New("some error"),
				errors.New("Worker_oomed"),
				errors.New("some other error"),
			},
			wantErrors: []error{
				errors.New("some error"),
				errors.New("some other error"),
			},
			wantAlertTitle: "Structural search needs more memory",
		},
	}
	for _, test := range cases {
		multiErr := &multierror.Error{
			Errors:      test.errors,
			ErrorFormat: multierror.ListFormatFunc,
		}
		haveMultiErr, haveAlert := alertForStructuralSearch(multiErr)

		if !reflect.DeepEqual(haveMultiErr.Errors, test.wantErrors) {
			t.Fatalf("test %s, have errors: %q, want: %q", test.name, haveMultiErr.Errors, test.wantErrors)
		}

		if haveAlert != nil && haveAlert.title != test.wantAlertTitle {
			t.Fatalf("test %s, have alert: %q, want: %q", test.name, haveAlert.title, test.wantAlertTitle)
		}

	}
}

func TestAlertForOverRepoLimit(t *testing.T) {

	generateRepoRevs := func(numRepos int) []*search.RepositoryRevisions {
		repoRevs := make([]*search.RepositoryRevisions, numRepos)
		chars := []string{"a", "b", "c"} // create some parent names
		j := 0
		for i := range repoRevs {
			repoRevs[i] = &search.RepositoryRevisions{
				Repo: &types.Repo{
					ID:   api.RepoID(i),
					Name: api.RepoName(chars[j] + "/repoName" + strconv.Itoa(i)),
				},
			}
			if j == 2 {
				j = 0
			} else {
				j++
			}
		}
		return repoRevs
	}

	setMockResolveRepositories := func(numRepos int) {
		mockResolveRepositories = func(effectiveRepoFieldValues []string) (resolved resolvedRepositories, err error) {
			return resolvedRepositories{
				repoRevs:        generateRepoRevs(numRepos),
				missingRepoRevs: make([]*search.RepositoryRevisions, 0),
				overLimit:       true,
			}, nil
		}
	}
	defer func() { mockResolveRepositories = nil }()

	cases := []struct {
		name      string
		globbing  bool
		repoRevs  int
		query     string
		wantAlert *searchAlert

		// simulates a timeout in alertForOverRepoLimit if "true"
		cancelContext bool
	}{
		{
			name:          "should return default alert because of 0 resolved repos",
			cancelContext: false,
			repoRevs:      0,
			query:         "foo",
			wantAlert: &searchAlert{
				prometheusType:  "over_repo_limit",
				title:           "Too many matching repositories",
				proposedQueries: nil,
				description:     "Use a 'repo:' or 'repogroup:' filter to narrow your search and see results.",
			},
		},
		{
			name:          "should return default alert because time limit is reached",
			cancelContext: true,
			repoRevs:      1,
			query:         "foo",
			wantAlert: &searchAlert{
				prometheusType: "over_repo_limit",
				title:          "Too many matching repositories",
				proposedQueries: []*searchQueryDescription{
					{
						"in the repository a/repoName0",
						"foo repo:^a/repoName0$",
						query.SearchType(0),
					},
				},
				description: "Use a 'repo:' or 'repogroup:' filter to narrow your search and see results.",
			},
		},
		{
			name:          "should return default alert because globbing is activated",
			globbing:      true,
			cancelContext: false,
			repoRevs:      1,
			query:         "foo",
			wantAlert: &searchAlert{
				prometheusType:  "over_repo_limit",
				title:           "Too many matching repositories",
				proposedQueries: nil,
				description:     "Use a 'repo:' or 'repogroup:' filter to narrow your search and see results.",
			},
		},
		{
			name:          "should return smart alert",
			cancelContext: false,
			repoRevs:      1,
			query:         "foo",
			wantAlert: &searchAlert{
				prometheusType: "over_repo_limit",
				title:          "Too many matching repositories",
				proposedQueries: []*searchQueryDescription{
					{
						"in repositories under a (further filtering required)",
						"foo repo:^a/",
						query.SearchType(0),
					},
				},
				description: "Use a 'repo:' or 'repogroup:' filter to narrow your search and see results.",
			},
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			setMockResolveRepositories(test.repoRevs)
			q, err := query.ProcessAndOr(test.query, query.ParserOptions{SearchType: query.SearchType(0), Globbing: test.globbing})
			if err != nil {
				t.Fatal(err)
			}
			sr := searchResolver{
				query: q,
				userSettings: &schema.Settings{
					SearchGlobbing: &test.globbing,
				},
			}

			ctx, cancel := context.WithCancel(context.Background())
			if test.cancelContext {
				cancel()
			}
			alert := sr.alertForOverRepoLimit(ctx)

			wantAlert := test.wantAlert
			if !reflect.DeepEqual(alert, wantAlert) {
				t.Fatalf("test %s, have alert %+v, want: %+v", test.name, alert, test.wantAlert)
			}
		})
	}
}
