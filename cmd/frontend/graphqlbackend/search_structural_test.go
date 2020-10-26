package graphqlbackend

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/zoekt"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/endpoint"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/search"
	searchbackend "github.com/sourcegraph/sourcegraph/internal/search/backend"
	"github.com/sourcegraph/sourcegraph/internal/search/query"
	"github.com/sourcegraph/sourcegraph/schema"
)

// Tests that indexed repos are filtered in structural search
func TestStructuralSearchRepoFilter(t *testing.T) {
	repoName := "indexed/one"
	indexedFileName := "indexed.go"

	indexedRepo := &types.Repo{Name: api.RepoName(repoName)}

	unindexedRepo := &types.Repo{Name: api.RepoName("unindexed/one")}

	db.Mocks.Repos.List = func(_ context.Context, op db.ReposListOptions) ([]*types.Repo, error) {
		return []*types.Repo{indexedRepo, unindexedRepo}, nil
	}
	defer func() { db.Mocks = db.MockStores{} }()

	mockSearchFilesInRepo = func(
		ctx context.Context,
		repo *types.Repo,
		gitserverRepo gitserver.Repo,
		rev string,
		info *search.TextPatternInfo,
		fetchTimeout time.Duration,
	) (
		matches []*FileMatchResolver,
		limitHit bool,
		err error,
	) {
		repoName := repo.Name
		switch repoName {
		case "indexed/one":
			return []*FileMatchResolver{mkFileMatch(nil, indexedFileName)}, false, nil
		case "unindexed/one":
			return []*FileMatchResolver{mkFileMatch(nil, "unindexed.go")}, false, nil
		default:
			return nil, false, errors.New("Unexpected repo")
		}
	}
	db.Mocks.Repos.Count = mockCount
	defer func() { mockSearchFilesInRepo = nil }()

	zoektRepo := &zoekt.RepoListEntry{
		Repository: zoekt.Repository{
			Name:     string(indexedRepo.Name),
			Branches: []zoekt.RepositoryBranch{{Name: "HEAD", Version: "deadbeef"}},
		},
	}

	zoektFileMatches := []zoekt.FileMatch{{
		FileName:   indexedFileName,
		Repository: string(indexedRepo.Name),
		LineMatches: []zoekt.LineMatch{
			{
				Line: nil,
			},
		},
	}}

	z := &searchbackend.Zoekt{
		Client: &fakeSearcher{
			repos:  []*zoekt.RepoListEntry{zoektRepo},
			result: &zoekt.SearchResult{Files: zoektFileMatches},
		},
		DisableCache: true,
	}

	ctx := context.Background()

	q, err := query.ParseAndCheck(`patterntype:structural index:only foo`)
	if err != nil {
		t.Fatal(err)
	}
	resolver := &searchResolver{
		query:        q,
		patternType:  query.SearchTypeStructural,
		zoekt:        z,
		searcherURLs: endpoint.Static("test"),
		userSettings: &schema.Settings{},
	}
	results, err := resolver.Results(ctx)
	if err != nil {
		t.Fatal("Results:", err)
	}

	fm, _ := results.Results()[0].ToFileMatch()
	if got, want := fm.JPath, indexedFileName; got != want {
		t.Fatalf("wrong indexed filename. want=%s, have=%s\n", want, got)
	}
}

func TestStructuralPatToRegexpQuery(t *testing.T) {
	cases := []struct {
		Name    string
		Pattern string
		Want    string
	}{
		{
			Name:    "Just a hole",
			Pattern: ":[1]",
			Want:    `(.|\s)*?`,
		},
		{
			Name:    "Adjacent holes",
			Pattern: ":[1]:[2]:[3]",
			Want:    `(.|\s)*?`,
		},
		{
			Name:    "Substring between holes",
			Pattern: ":[1] substring :[2]",
			Want:    `([\s]+substring[\s]+)`,
		},
		{
			Name:    "Substring before and after different hole kinds",
			Pattern: "prefix :[[1]] :[2.] suffix",
			Want:    `(prefix[\s]+)(.|\s)*?([\s]+)(.|\s)*?([\s]+suffix)`,
		},
		{
			Name:    "Substrings covering all hole kinds.",
			Pattern: `1. :[1] 2. :[[2]] 3. :[3.] 4. :[4\n] 5. :[ ] 6. :[ 6] done.`,
			Want:    `(1\.[\s]+)(.|\s)*?([\s]+2\.[\s]+)(.|\s)*?([\s]+3\.[\s]+)(.|\s)*?([\s]+4\.[\s]+)(.|\s)*?([\s]+5\.[\s]+)(.|\s)*?([\s]+6\.[\s]+)(.|\s)*?([\s]+done\.)`,
		},
		{
			Name:    "Allow alphanumeric identifiers in holes",
			Pattern: "sub :[alphanum_ident_123] string",
			Want:    `(sub[\s]+)(.|\s)*?([\s]+string)`,
		},

		{
			Name:    "Whitespace separated holes",
			Pattern: ":[1] :[2]",
			Want:    `([\s]+)`,
		},
		{
			Name:    "Expect newline separated pattern",
			Pattern: "ParseInt(:[stuff], :[x]) if err ",
			Want:    `(ParseInt\()(.|\s)*?(,[\s]+)(.|\s)*?(\)[\s]+if[\s]+err[\s]+)`,
		},
		{
			Name: "Contiguous whitespace is replaced by regex",
			Pattern: `ParseInt(:[stuff],    :[x])
             if err `,
			Want: `(ParseInt\()(.|\s)*?(,[\s]+)(.|\s)*?(\)[\s]+if[\s]+err[\s]+)`,
		},
		{
			Name:    "Regex holes extracts regex",
			Pattern: `:[x~[yo]]`,
			Want:    `([yo])`,
		},
		{
			Name:    "Regex holes with escaped space",
			Pattern: `:[x~\ ]`,
			Want:    `(\ )`,
		},
		{
			Name:    "Shorthand",
			Pattern: ":[[1]]",
			Want:    `(.|\s)*?`,
		},
		{
			Name:    "Array-like preserved",
			Pattern: `[:[x]]`,
			Want:    `(\[)(.|\s)*?(\])`,
		},
		{
			Name:    "Shorthand",
			Pattern: ":[[1]]",
			Want:    `(.|\s)*?`,
		},
		{
			Name:    "Not well-formed is undefined",
			Pattern: ":[[",
			Want:    `(.|\s)*?`,
		},
		{
			Name:    "Complex regex with character class",
			Pattern: `:[chain~[^(){}\[\],]+\n( +\..*\n)+]`,
			Want:    `([^(){}\[\],]+\n( +\..*\n)+)`,
		},
		{
			Name:    "Colon regex",
			Pattern: `:[~:]`,
			Want:    `(:)`,
		},
		{
			Name:    "Colon prefix",
			Pattern: `::[version]bar`,
			Want:    `(:)(.|\s)*?(bar)`,
		},
		{
			Name:    "Colon prefix",
			Pattern: `::::[version]bar`,
			Want:    `(:::)(.|\s)*?(bar)`,
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			got := StructuralPatToRegexpQuery(tt.Pattern, false)
			if diff := cmp.Diff(tt.Want, got); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestBuildQuery(t *testing.T) {
	pattern := ":[x~*]"
	want := "error parsing regexp: missing argument to repetition operator: `*`"
	t.Run("build query", func(t *testing.T) {
		_, err := buildQuery(
			&search.TextParameters{
				PatternInfo: &search.TextPatternInfo{Pattern: pattern},
			},
			nil,
			nil,
			false,
		)
		if diff := cmp.Diff(err.Error(), want); diff != "" {
			t.Error(diff)
		}
	})
}
