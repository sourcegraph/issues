// +build gqltest

package main

import (
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/gqltestutil"
)

func TestSearch(t *testing.T) {
	if len(*githubToken) == 0 {
		t.Skip("Environment variable GITHUB_TOKEN is not set")
	}

	// Set up external service
	esID, err := client.AddExternalService(gqltestutil.AddExternalServiceInput{
		Kind:        extsvc.KindGitHub,
		DisplayName: "gqltest-github-search",
		Config: mustMarshalJSONString(struct {
			URL   string   `json:"url"`
			Token string   `json:"token"`
			Repos []string `json:"repos"`
		}{
			URL:   "http://github.com",
			Token: *githubToken,
			Repos: []string{
				"sgtest/java-langserver",
				"sgtest/jsonrpc2",
				"sgtest/go-diff",
				"sgtest/appdash",
				"sgtest/sourcegraph-typescript",
				"sgtest/private",
				"sgtest/mux",      // Fork
				"sgtest/archived", // Archived
			},
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := client.DeleteExternalService(esID)
		if err != nil {
			t.Fatal(err)
		}
	}()

	err = client.WaitForReposToBeCloned(
		"github.com/sgtest/java-langserver",
		"github.com/sgtest/jsonrpc2",
		"github.com/sgtest/go-diff",
		"github.com/sgtest/appdash",
		"github.com/sgtest/sourcegraph-typescript",
		"github.com/sgtest/private",
		"github.com/sgtest/mux",      // Fork
		"github.com/sgtest/archived", // Archived
	)
	if err != nil {
		t.Fatal(err)
	}

	err = client.WaitForReposToBeIndex(
		"github.com/sgtest/java-langserver",
	)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("visibility", func(t *testing.T) {
		tests := []struct {
			query       string
			wantMissing []string
		}{
			{
				query:       "type:repo visibility:private",
				wantMissing: []string{},
			},
			{
				query:       "type:repo visibility:public",
				wantMissing: []string{"github.com/sgtest/private"},
			},
			{
				query:       "type:repo visibility:any",
				wantMissing: []string{},
			},
		}
		for _, test := range tests {
			t.Run(test.query, func(t *testing.T) {
				results, err := client.SearchRepositories(test.query)
				if err != nil {
					t.Fatal(err)
				}
				missing := results.Exists("github.com/sgtest/private")
				if diff := cmp.Diff(test.wantMissing, missing); diff != "" {
					t.Fatalf("Missing mismatch (-want +got):\n%s", diff)
				}
			})
		}
	})

	t.Run("execute search with search parameters", func(t *testing.T) {
		results, err := client.SearchFiles("repo:^github.com/sgtest/go-diff$ type:file file:.go -file:.md")
		if err != nil {
			t.Fatal(err)
		}

		// Make sure only got .go files and no .md files
		for _, r := range results.Results {
			if !strings.HasSuffix(r.File.Name, ".go") {
				t.Fatalf("Found file name does not end with .go: %s", r.File.Name)
			}
		}
	})

	t.Run("multiple revisions per repository", func(t *testing.T) {
		results, err := client.SearchFiles("repo:sgtest/go-diff$@master:print-options:*refs/heads/ func NewHunksReader")
		if err != nil {
			t.Fatal(err)
		}

		wantExprs := map[string]struct{}{
			"master":        {},
			"print-options": {},

			// These next 2 branches are included because of the *refs/heads/ in the query.
			"test-already-exist-pr": {},
			"bug-fix-wip":           {},
		}

		for _, r := range results.Results {
			delete(wantExprs, r.RevSpec.Expr)
		}

		if len(wantExprs) > 0 {
			missing := make([]string, 0, len(wantExprs))
			for expr := range wantExprs {
				missing = append(missing, expr)
			}
			t.Fatalf("Missing exprs: %v", missing)
		}
	})

	t.Run("repository groups", func(t *testing.T) {
		const repoName = "github.com/sgtest/go-diff"
		err := client.OverwriteSettings(client.AuthenticatedUserID(), fmt.Sprintf(`{"search.repositoryGroups":{"gql_test_group": ["%s"]}}`, repoName))
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			err := client.OverwriteSettings(client.AuthenticatedUserID(), `{}`)
			if err != nil {
				t.Fatal(err)
			}
		}()

		results, err := client.SearchFiles("repogroup:gql_test_group diff.")
		if err != nil {
			t.Fatal(err)
		}

		// Make sure there are results and all results are from the same repository
		if len(results.Results) == 0 {
			t.Fatal("Unexpected zero result")
		}
		for _, r := range results.Results {
			if r.Repository.Name != repoName {
				t.Fatalf("Repository: want %q but got %q", repoName, r.Repository.Name)
			}
		}
	})

	t.Run("search statistics", func(t *testing.T) {
		err := client.OverwriteSettings(client.AuthenticatedUserID(), `{"experimentalFeatures":{"searchStats": true}}`)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			err := client.OverwriteSettings(client.AuthenticatedUserID(), `{}`)
			if err != nil {
				t.Fatal(err)
			}
		}()

		var lastResult *gqltestutil.SearchStatsResult
		// Retry because the configuration update endpoint is eventually consistent
		err = gqltestutil.Retry(5*time.Second, func() error {
			// This is a substring that appears in the sgtest/go-diff repository.
			// It is OK if it starts to appear in other repositories, the test just
			// checks that it is found in at least 1 Go file.
			result, err := client.SearchStats("Incomplete-Lines")
			if err != nil {
				t.Fatal(err)
			}
			lastResult = result

			for _, lang := range result.Languages {
				if strings.EqualFold(lang.Name, "Go") {
					return nil
				}
			}

			return gqltestutil.ErrContinueRetry
		})
		if err != nil {
			t.Fatal(err, "lastResult:", lastResult)
		}
	})

	t.Run("repository search", func(t *testing.T) {
		tests := []struct {
			name        string
			query       string
			zeroResult  bool
			wantMissing []string
		}{
			{
				name:       `archived excluded, zero results`,
				query:      `type:repo archived`,
				zeroResult: true,
			},
			{
				name:  `archived included, nonzero result`,
				query: `type:repo archived archived:yes`,
			},
			{
				name:  `archived included if exact without option, nonzero result`,
				query: `repo:^github\.com/sgtest/archived$`,
			},
			{
				name:       `fork excluded, zero results`,
				query:      `type:repo sgtest/mux`,
				zeroResult: true,
			},
			{
				name:  `fork included, nonzero result`,
				query: `type:repo sgtest/mux fork:yes`,
			},
			{
				name:  `fork included if exact without option, nonzero result`,
				query: `repo:^github\.com/sgtest/mux$`,
			},
			{
				name:  `exclude counts for fork and archive`,
				query: `repo:mux|archived|go-diff`,
				wantMissing: []string{
					"github.com/sgtest/archived",
					"github.com/sgtest/mux",
				},
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				results, err := client.SearchRepositories(test.query)
				if err != nil {
					t.Fatal(err)
				}

				if test.zeroResult {
					if len(results) > 0 {
						t.Fatalf("Want zero result but got %d", len(results))
					}
				} else {
					if len(results) == 0 {
						t.Fatal("Want non-zero results but got 0")
					}
				}

				if test.wantMissing != nil {
					missing := results.Exists(test.wantMissing...)
					sort.Strings(missing)
					if diff := cmp.Diff(test.wantMissing, missing); diff != "" {
						t.Fatalf("Missing mismatch (-want +got):\n%s", diff)
					}
				}
			})
		}
	})

	t.Run("global text search", func(t *testing.T) {
		tests := []struct {
			name          string
			query         string
			zeroResult    bool
			minMatchCount int64
		}{
			// Global search
			{
				name:  "error",
				query: "error",
			},
			{
				name:  "error count:1000",
				query: "error count:1000",
			},
			{
				name:          "something with more than 1000 results and use count:1000",
				query:         ". count:1000",
				minMatchCount: 1001,
			},
			{
				name:  "regular expression without indexed search",
				query: "index:no patterntype:regexp ^func.*$",
			},
			{
				name:  "fork:only",
				query: "fork:only router",
			},
			{
				name:  "double-quoted pattern, nonzero result",
				query: `"func main() {\n" patterntype:regexp count:1 stable:yes type:file`,
			},
			{
				name:  "exclude repo, nonzero result",
				query: `"func main() {\n" -repo:go-diff patterntype:regexp count:1 stable:yes type:file`,
			},
			{
				name:       "fork:no",
				query:      "fork:no FORK_SENTINEL",
				zeroResult: true,
			},
			{
				name:       "random characters, zero results",
				query:      "asdfalksd+jflaksjdfklas patterntype:literal -repo:sourcegraph",
				zeroResult: true,
			},
			// Repo search
			{
				name:  "repo search by name, nonzero result",
				query: "repo:go-diff$",
			},
			{
				name:  "repo search by name, case yes, nonzero result",
				query: `repo:^github\.com/sgtest/go-diff$ String case:yes count:1 stable:yes type:file`,
			},
			{
				name:  "true is an alias for yes when fork is set",
				query: `repo:github\.com/sgtest/mux fork:true`,
			},
			{
				name:  "non-master branch, nonzero result",
				query: `repo:^github\.com/sgtest/java-langserver$@v1 void sendPartialResult(Object requestId, JsonPatch jsonPatch); patterntype:literal count:1 stable:yes type:file`,
			},
			{
				name:  "indexed multiline search, nonzero result",
				query: `repo:^github\.com/sgtest/java-langserver$ \nimport index:only patterntype:regexp count:1 stable:yes type:file`,
			},
			{
				name:  "unindexed multiline search, nonzero result",
				query: `repo:^github\.com/sgtest/java-langserver$ \nimport index:no patterntype:regexp count:1 stable:yes type:file`,
			},
			{
				name:       "random characters, zero result",
				query:      `repo:^github\.com/sgtest/java-langserver$ doesnot734734743734743exist`,
				zeroResult: true,
			},
			// Filename search
			{
				name:  "search for a known file",
				query: "file:doc.go",
			},
			{
				name:       "search for a non-existent file",
				query:      "file:asdfasdf.go",
				zeroResult: true,
			},
			// Symbol search
			{
				name:  "search for a known symbol",
				query: "type:symbol count:100 patterntype:regexp ^newroute",
			},
			{
				name:       "search for a non-existent symbol",
				query:      "type:symbol asdfasdf",
				zeroResult: true,
			},
			// Commit search
			{
				name:  "commit search, nonzero result",
				query: `repo:^github\.com/sgtest/go-diff$ type:commit count:1`,
			},
			// Diff search
			{
				name:  "diff search, nonzero result",
				query: `repo:^github\.com/sgtest/go-diff$ type:diff main count:1`,
			},
			// Repohascommitafter
			{
				name:  `Repohascommitafter, nonzero result`,
				query: `repo:^github\.com/sgtest/go-diff$ repohascommitafter:"8 months ago" test patterntype:literal count:1`,
			},
			// Regex text search
			{
				name:  `regex, unindexed, nonzero result`,
				query: `^func.*$ patterntype:regexp index:only count:1 stable:yes type:file`,
			},
			{
				name:  `regex, fork only, nonzero result`,
				query: `fork:only patterntype:regexp FORK_SENTINEL`,
			},
			{
				name:  `regex, filter by language`,
				query: `\bfunc\b lang:go count:1 stable:yes type:file patterntype:regexp`,
			},
			{
				name:       `regex, filename, zero results`,
				query:      `file:asdfasdf.go patterntype:regexp`,
				zeroResult: true,
			},
			{
				name:  `regexp, filename, nonzero result`,
				query: `file:doc.go patterntype:regexp count:1`,
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				results, err := client.SearchFiles(test.query)
				if err != nil {
					t.Fatal(err)
				}

				if results.Alert != nil {
					t.Fatalf("Unexpected alert: %v", results.Alert)
				}

				if test.zeroResult {
					if len(results.Results) > 0 {
						t.Fatalf("Want zero result but got %d", len(results.Results))
					}
				} else {
					if len(results.Results) == 0 {
						t.Fatal("Want non-zero results but got 0")
					}
				}

				if results.MatchCount < test.minMatchCount {
					t.Fatalf("Want at least %d match count but got %d", test.minMatchCount, results.MatchCount)
				}
			})
		}
	})

	t.Run("timeout search options", func(t *testing.T) {
		results, err := client.SearchFiles(`router index:no timeout:1ns`)
		if err != nil {
			t.Fatal(err)
		}

		if results.Alert == nil {
			t.Fatal("Want search alert but got nil")
		}
	})

	t.Run("structural search", func(t *testing.T) {
		tests := []struct {
			name       string
			query      string
			zeroResult bool
			wantAlert  *gqltestutil.SearchAlert
		}{
			{
				name:  "Global search, structural, index only, nonzero result",
				query: `repo:^github\.com/sgtest/go-diff$ make(:[1]) index:only patterntype:structural count:3`,
			},
			{
				name:  `Structural search quotes are interpreted literally`,
				query: `repo:^github\.com/sgtest/sourcegraph-typescript$ file:^README\.md "basic :[_] access :[_]" patterntype:structural`,
			},
			{
				name:       `Alert to activate structural search mode`,
				query:      `repo:^github\.com/sgtest/go-diff$ patterntype:literal i can't :[believe] it's not butter`,
				zeroResult: true,
				wantAlert: &gqltestutil.SearchAlert{
					Title:       "No results",
					Description: "It looks like you may have meant to run a structural search, but it is not toggled.",
					ProposedQueries: []gqltestutil.ProposedQuery{
						{
							Description: "Activate structural search",
							Query:       `repo:^github\.com/sgtest/go-diff$ patterntype:literal i can't :[believe] it's not butter patternType:structural`,
						},
					},
				},
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				results, err := client.SearchFiles(test.query)
				if err != nil {
					t.Fatal(err)
				}

				if diff := cmp.Diff(test.wantAlert, results.Alert); diff != "" {
					t.Fatalf("Alert mismatch (-want +got):\n%s", diff)
				}

				if test.zeroResult {
					if len(results.Results) > 0 {
						t.Fatalf("Want zero result but got %d", len(results.Results))
					}
				} else {
					if len(results.Results) == 0 {
						t.Fatal("Want non-zero results but got 0")
					}
				}
			})
		}
	})

	t.Run("And/Or queries", func(t *testing.T) {
		tests := []struct {
			name       string
			query      string
			zeroResult bool
			wantAlert  *gqltestutil.SearchAlert
		}{
			{
				name:  `And operator, basic`,
				query: `repo:^github\.com/sgtest/go-diff$ func and main count:1 stable:yes type:file`,
			},
			{
				name:  `Or operator, single and double quoted`,
				query: `repo:^github\.com/sgtest/go-diff$ "func PrintMultiFileDiff" or 'func readLine(' stable:yes type:file count:1 patterntype:regexp`,
			},
			{
				name:  `Literals, grouped parens with parens-as-patterns heuristic`,
				query: `repo:^github\.com/sgtest/go-diff$ (() or ()) stable:yes type:file count:1 patterntype:regexp`,
			},
			{
				name:  `Literals, no grouped parens`,
				query: `repo:^github\.com/sgtest/go-diff$ () or () stable:yes type:file count:1 patterntype:regexp`,
			},
			{
				name:  `Literals, escaped parens`,
				query: `repo:^github\.com/sgtest/go-diff$ \(\) or \(\) stable:yes type:file count:1 patterntype:regexp`,
			},
			{
				name:  `Literals, escaped and unescaped parens, no group`,
				query: `repo:^github\.com/sgtest/go-diff$ () or \(\) stable:yes type:file count:1 patterntype:regexp`,
			},
			{
				name:  `Literals, escaped and unescaped parens, grouped`,
				query: `repo:^github\.com/sgtest/go-diff$ (() or \(\)) stable:yes type:file count:1 patterntype:regexp`,
			},
			{
				name:       `Literals, double paren`,
				query:      `repo:^github\.com/sgtest/go-diff$ ()() or ()()`,
				zeroResult: true,
			},
			{
				name:       `Literals, double paren, dangling paren right side`,
				query:      `repo:^github\.com/sgtest/go-diff$ ()() or main()(`,
				zeroResult: true,
			},
			{
				name:       `Literals, double paren, dangling paren left side`,
				query:      `repo:^github\.com/sgtest/go-diff$ ()( or ()()`,
				zeroResult: true,
			},
			{
				name:  `Mixed regexp and literal`,
				query: `repo:^github\.com/sgtest/go-diff$ func(.*) or does_not_exist_3744 count:1 stable:yes type:file`,
			},
			{
				name:  `Mixed regexp and literal heuristic`,
				query: `repo:^github\.com/sgtest/go-diff$ func( or func(.*) count:1 stable:yes type:file`,
			},
			{
				name:       `Mixed regexp and quoted literal`,
				query:      `repo:^github\.com/sgtest/go-diff$ "*" and cert.*Load count:1 stable:yes type:file`,
				zeroResult: true,
				wantAlert: &gqltestutil.SearchAlert{
					Title:       "Too many files to search for and-expression",
					Description: "One and-expression in the query requires a lot of work! Try using the 'repo:' or 'file:' filters to narrow your search. We're working on improving this experience in https://github.com/sourcegraph/sourcegraph/issues/9824",
				},
			},
			{
				name:  `Escape sequences`,
				query: `repo:^github\.com/sgtest/go-diff$ \' and \" and \\ and /`,
			},
			{
				name:  `Escaped whitespace sequences with 'and'`,
				query: `repo:^github\.com/sgtest/go-diff$ \ and /`,
			},
			{
				name:  `Concat converted to spaces for literal search`,
				query: `repo:^github\.com/sgtest/go-diff$ file:^diff/print\.go t := or ts Time patterntype:literal`,
			},
			{
				name:  `Literal parentheses match pattern`,
				query: `repo:^github\.com/sgtest/go-diff file:^diff/print\.go Bytes() and Time() patterntype:literal`,
			},
			{
				name:  `Dangling right parens, heuristic for literal search`,
				query: `repo:^github\.com/sgtest/go-diff$ diffPath) and main patterntype:literal`,
			},
			{
				name:  `Dangling right parens, heuristic for literal search, double parens`,
				query: `repo:^github\.com/sgtest/go-diff$ MarshalTo and OrigName)) patterntype:literal`,
			},
			{
				name:  `Dangling right parens, heuristic for literal search, simple group before right paren`,
				query: `repo:^github\.com/sgtest/go-diff$ MarshalTo and (m.OrigName)) patterntype:literal`,
			},
			{
				name:       `Dangling right parens, heuristic for literal search, cannot succeed, too confusing`,
				query:      `repo:^github\.com/sgtest/go-diff$ (respObj.Size and (data))) patterntype:literal`,
				zeroResult: true,
				wantAlert: &gqltestutil.SearchAlert{
					Title:       "Unable To Process Query",
					Description: "Error parsing regexp: missing closing ): `(respObj.Size`",
				},
			},
			{
				name:       `No result for confusing grouping`,
				query:      `repo:^github\.com/sgtest/go-diff file:^README\.md (bar and (foo or x\) ()) patterntype:literal`,
				zeroResult: true,
			},
			{
				name:       `Successful grouping removes alert`,
				query:      `repo:^github\.com/sgtest/go-diff file:^README\.md (bar and (foo or (x\) ())) patterntype:literal`,
				zeroResult: true,
			},
			{
				name:  `No dangling right paren with complex group for literal search`,
				query: `repo:^github\.com/sgtest/go-diff$ (m *FileDiff and (data)) patterntype:literal`,
			},
			{
				name:  `Concat converted to .* for regexp search`,
				query: `repo:^github\.com/sgtest/go-diff$ file:^diff/print\.go t := or ts Time patterntype:regexp stable:yes type:file`,
			},
			{
				name:  `Structural search uses literal search parser`,
				query: `repo:^github\.com/sgtest/go-diff$ file:^diff/print\.go :[[v]] := ts and printFileHeader(:[_]) patterntype:structural`,
			},
			{
				name:  `Union file matches per file and accurate counts`,
				query: `repo:^github\.com/sgtest/go-diff file:^diff/print\.go func or package`,
			},
			{
				name:  `Intersect file matches per file and accurate counts`,
				query: `repo:^github\.com/sgtest/go-diff file:^diff/print\.go func and package`,
			},
			{
				name:  `Simple combined union and intersect file matches per file and accurate counts`,
				query: `repo:^github\.com/sgtest/go-diff file:^diff/print\.go ((func timePtr and package diff) or return buf.Bytes())`,
			},
			{
				name:  `Complex union of intersect file matches per file and accurate counts`,
				query: `repo:^github\.com/sgtest/go-diff file:^diff/print\.go ((func timePtr and package diff) or (ts == nil and ts.Time()))`,
			},
			{
				name:  `Complex intersect of union file matches per file and accurate counts`,
				query: `repo:^github\.com/sgtest/go-diff file:^diff/print\.go ((func timePtr or package diff) and (ts == nil or ts.Time()))`,
			},
			{
				name:       `Intersect file matches per file against an empty result set`,
				query:      `repo:^github\.com/sgtest/go-diff file:^diff/print\.go func and doesnotexist838338`,
				zeroResult: true,
			},
			{
				name:  `Dedupe union operation`,
				query: `file:diff.go|print.go|parse.go repo:^github\.com/sgtest/go-diff _, :[[x]] := range :[src.] { :[_] } or if :[s1] == :[s2] patterntype:structural`,
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				results, err := client.SearchFiles(test.query)
				if err != nil {
					t.Fatal(err)
				}

				if diff := cmp.Diff(test.wantAlert, results.Alert); diff != "" {
					t.Fatalf("Alert mismatch (-want +got):\n%s", diff)
				}

				if test.zeroResult {
					if len(results.Results) > 0 {
						t.Fatalf("Want zero result but got %d", len(results.Results))
					}
				} else {
					if len(results.Results) == 0 {
						t.Fatal("Want non-zero results but got 0")
					}
				}
			})
		}
	})

	t.Run("And/Or search expression queries", func(t *testing.T) {
		tests := []struct {
			name       string
			query      string
			zeroResult bool
			wantAlert  *gqltestutil.SearchAlert
		}{
			{
				name:  `Or distributive property on content and file`,
				query: `repo:^github\.com/sgtest/sourcegraph-typescript$ (Fetches OR file:language-server.ts)`,
			},
			{
				name:  `Or distributive property on nested file on content`,
				query: `repo:^github\.com/sgtest/sourcegraph-typescript$ ((file:^renovate\.json extends) or file:progress.ts createProgressProvider)`,
			},
			{
				name:  `Or distributive property on commit`,
				query: `repo:^github\.com/sgtest/sourcegraph-typescript$ (type:diff or type:commit) author:felix yarn`,
			},
			{
				name:  `Or distributive property on rev`,
				query: `repo:^github\.com/sgtest/go-diff$ (rev:garo/lsif-indexing-campaign or rev:test-already-exist-pr) file:README.md`,
			},
			{
				name:  `Or distributive property on rev`,
				query: `repo:^github\.com/sgtest/go-diff$ (rev:garo/lsif-indexing-campaign or rev:test-already-exist-pr)`,
			},
			{
				name:  `Or distributive property on repo`,
				query: `(repo:^github\.com/sgtest/go-diff$@garo/lsif-indexing-campaign:test-already-exist-pr or repo:^github\.com/sgtest/sourcegraph-typescript$) file:README.md #`,
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				results, err := client.SearchFiles(test.query)
				if err != nil {
					t.Fatal(err)
				}

				if diff := cmp.Diff(test.wantAlert, results.Alert); diff != "" {
					t.Fatalf("Alert mismatch (-want +got):\n%s", diff)
				}

				if test.zeroResult {
					if len(results.Results) > 0 {
						t.Fatalf("Want zero result but got %d", len(results.Results))
					}
				} else {
					if len(results.Results) == 0 {
						t.Fatal("Want non-zero results but got 0")
					}
				}
			})
		}
	})
}
