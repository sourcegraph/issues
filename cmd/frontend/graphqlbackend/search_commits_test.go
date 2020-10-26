package graphqlbackend

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	//"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/search"
	"github.com/sourcegraph/sourcegraph/internal/search/query"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
)

func TestSearchCommitsInRepo(t *testing.T) {
	ctx := context.Background()

	var calledVCSRawLogDiffSearch bool
	gitSignatureWithDate := git.Signature{Date: time.Now().UTC().AddDate(0, 0, -1)}
	git.Mocks.RawLogDiffSearch = func(opt git.RawLogDiffSearchOptions) ([]*git.LogCommitSearchResult, bool, error) {
		calledVCSRawLogDiffSearch = true
		if want := "p"; opt.Query.Pattern != want {
			t.Errorf("got %q, want %q", opt.Query.Pattern, want)
		}
		if want := []string{
			"--no-prefix",
			"--max-count=" + strconv.Itoa(defaultMaxSearchResults+1),
			"--unified=0",
			"--regexp-ignore-case",
			"rev",
		}; !reflect.DeepEqual(opt.Args, want) {
			t.Errorf("got %v, want %v", opt.Args, want)
		}
		return []*git.LogCommitSearchResult{
			{
				Commit: git.Commit{ID: "c1", Author: gitSignatureWithDate},
				Diff:   &git.RawDiff{Raw: "x"},
			},
		}, true, nil
	}
	defer git.ResetMocks()

	query, err := query.ParseAndCheck("p")
	if err != nil {
		t.Fatal(err)
	}
	repoRevs := &search.RepositoryRevisions{
		Repo: &types.Repo{ID: 1, Name: "repo"},
		Revs: []search.RevisionSpecifier{{RevSpec: "rev"}},
	}
	results, limitHit, timedOut, err := searchCommitsInRepo(ctx, search.CommitParameters{
		RepoRevs:    repoRevs,
		PatternInfo: &search.CommitPatternInfo{Pattern: "p", FileMatchLimit: int32(defaultMaxSearchResults)},
		Query:       query,
		Diff:        true,
	})
	if err != nil {
		t.Fatal(err)
	}

	wantCommit := GitCommitResolver{
		repoResolver:    &RepositoryResolver{repo: &types.Repo{ID: 1, Name: "repo"}},
		oid:             "c1",
		author:          *toSignatureResolver(&gitSignatureWithDate, true),
		includeUserInfo: true,
	}
	wantCommit.once.Do(func() {}) // mark as done

	if want := []*CommitSearchResultResolver{
		{
			commit:      &wantCommit,
			diffPreview: &highlightedString{value: "x", highlights: []*highlightedRange{}},
			icon:        "data:image/svg+xml;base64,PD94bWwgdmVyc2lvbj0iMS4wIiBlbmNvZGluZz0iVVRGLTgiPz48IURPQ1RZUEUgc3ZnIFBVQkxJQyAiLS8vVzNDLy9EVEQgU1ZHIDEuMS8vRU4iICJodHRwOi8vd3d3LnczLm9yZy9HcmFwaGljcy9TVkcvMS4xL0RURC9zdmcxMS5kdGQiPjxzdmcgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIiB4bWxuczp4bGluaz0iaHR0cDovL3d3dy53My5vcmcvMTk5OS94bGluayIgdmVyc2lvbj0iMS4xIiB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCI+PHBhdGggZD0iTTE3LDEyQzE3LDE0LjQyIDE1LjI4LDE2LjQ0IDEzLDE2LjlWMjFIMTFWMTYuOUM4LjcyLDE2LjQ0IDcsMTQuNDIgNywxMkM3LDkuNTggOC43Miw3LjU2IDExLDcuMVYzSDEzVjcuMUMxNS4yOCw3LjU2IDE3LDkuNTggMTcsMTJNMTIsOUEzLDMgMCAwLDAgOSwxMkEzLDMgMCAwLDAgMTIsMTVBMywzIDAgMCwwIDE1LDEyQTMsMyAwIDAsMCAxMiw5WiIgLz48L3N2Zz4=",
			label:       "[repo](/repo) › [](/repo/-/commit/c1): [](/repo/-/commit/c1)",
			url:         "/repo/-/commit/c1",
			detail:      "[`c1` one day ago](/repo/-/commit/c1)",
			matches:     []*searchResultMatchResolver{{url: "/repo/-/commit/c1", body: "```diff\nx```", highlights: []*highlightedRange{}}},
		},
	}; !reflect.DeepEqual(results, want) {
		t.Errorf("results\ngot  %v\nwant %v", results, want)
	}
	if limitHit {
		t.Error("limitHit")
	}
	if timedOut {
		t.Error("timedOut")
	}
	if !calledVCSRawLogDiffSearch {
		t.Error("!calledVCSRawLogDiffSearch")
	}
}

func (r *CommitSearchResultResolver) String() string {
	return fmt.Sprintf("{commit: %+v diffPreview: %+v messagePreview: %+v}", r.commit, r.diffPreview, r.messagePreview)
}

func TestExpandUsernamesToEmails(t *testing.T) {
	resetMocks()
	db.Mocks.Users.GetByUsername = func(ctx context.Context, username string) (*types.User, error) {
		if want := "alice"; username != want {
			t.Errorf("got %q, want %q", username, want)
		}
		return &types.User{ID: 123}, nil
	}
	db.Mocks.UserEmails.ListByUser = func(_ context.Context, opt db.UserEmailsListOptions) ([]*db.UserEmail, error) {
		if want := int32(123); opt.UserID != want {
			t.Errorf("got %v, want %v", opt.UserID, want)
		}
		t := time.Now()
		return []*db.UserEmail{
			{Email: "alice@example.com", VerifiedAt: &t},
			{Email: "alice@example.org", VerifiedAt: &t},
		}, nil
	}

	x, err := expandUsernamesToEmails(context.Background(), []string{"foo", "@alice"})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"foo", `alice@example\.com`, `alice@example\.org`}; !reflect.DeepEqual(x, want) {
		t.Errorf("got %q, want %q", x, want)
	}
}

func TestHighlightMatches(t *testing.T) {
	type args struct {
		pattern *regexp.Regexp
		data    []byte
	}
	tests := []struct {
		name string
		args args
		want *highlightedString
	}{
		{
			// https://github.com/sourcegraph/sourcegraph/issues/4512
			name: "match at end",
			args: args{
				pattern: regexp.MustCompile(`白`),
				data:    []byte(`加一行空白`),
			},
			want: &highlightedString{
				value: "加一行空白",
				highlights: []*highlightedRange{
					{
						line:      1,
						character: 4,
						length:    1,
					},
				},
			},
		},
		{
			// https://github.com/sourcegraph/sourcegraph/issues/4512
			name: "two character match in middle",
			args: args{
				pattern: regexp.MustCompile(`行空`),
				data:    []byte(`加一行空白`),
			},
			want: &highlightedString{
				value: "加一行空白",
				highlights: []*highlightedRange{
					{
						line:      1,
						character: 2,
						length:    2,
					},
				},
			},
		},
		{
			// https://github.com/sourcegraph/sourcegraph/issues/4512
			name: "match at beginning",
			args: args{
				pattern: regexp.MustCompile(`加`),
				data:    []byte(`加一行空白`),
			},
			want: &highlightedString{
				value: "加一行空白",
				highlights: []*highlightedRange{
					{
						line:      1,
						character: 0,
						length:    1,
					},
				},
			},
		},

		{
			name: "invalid utf-8 ",
			args: args{
				pattern: regexp.MustCompile(`.`),
				data:    []byte("a\xc5z"),
			},
			want: &highlightedString{
				value: "a\xc5z",
				highlights: []*highlightedRange{
					{
						line:      1,
						character: 0,
						length:    1,
					},
					{
						line:      1,
						character: 1,
						length:    1,
					},
					{
						line:      1,
						character: 2,
						length:    1,
					},
				},
			},
		},

		{
			name: "multiline",
			args: args{
				pattern: regexp.MustCompile(`行`),
				data:    []byte("加一行空白\n加一空行白"),
			},
			want: &highlightedString{
				value: "加一行空白\n加一空行白",
				highlights: []*highlightedRange{
					{
						line:      1,
						character: 2,
						length:    1,
					},
					{
						line:      2,
						character: 3,
						length:    1,
					},
				},
			},
		},

		// https://github.com/sourcegraph/sourcegraph/issues/4791
		{
			name: "unicode search that would be broken by tolower",
			args: args{
				pattern: regexp.MustCompile(`İ`),
				data:    []byte(`İi`),
			},
			want: &highlightedString{
				value: "İi",
				highlights: []*highlightedRange{
					{
						line:      1,
						character: 0,
						length:    1,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := highlightMatches(tt.args.pattern, tt.args.data); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("highlightMatches() = %v, want %v", spew.Sdump(got), spew.Sdump(tt.want))
			}
		})
	}
}

func Benchmark_highlightMatches(b *testing.B) {
	as := bytes.Repeat([]byte{'a'}, 5000)
	lines := append(as, byte('\n'))
	lines = append(lines, as...)
	rx := regexp.MustCompile(`a`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = highlightMatches(rx, lines)
	}
}
