package backend

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/zoekt"
	"github.com/sourcegraph/sourcegraph/schema"
)

func TestGetIndexOptions(t *testing.T) {
	vc := parseVersionContext
	vcConf := func(contexts ...*schema.VersionContext) schema.SiteConfiguration {
		return schema.SiteConfiguration{
			ExperimentalFeatures: &schema.ExperimentalFeatures{
				VersionContexts: contexts,
			},
		}
	}
	withBranches := func(c schema.SiteConfiguration, b map[string][]string) schema.SiteConfiguration {
		if c.ExperimentalFeatures == nil {
			c.ExperimentalFeatures = &schema.ExperimentalFeatures{}
		}
		c.ExperimentalFeatures.SearchIndexBranches = b
		return c
	}

	type caseT struct {
		name string
		conf schema.SiteConfiguration
		repo string
		want zoektIndexOptions
	}

	cases := []caseT{{
		name: "default",
		conf: schema.SiteConfiguration{},
		repo: "repo",
		want: zoektIndexOptions{
			RepoID:  1,
			Symbols: true,
			Branches: []zoekt.RepositoryBranch{
				{Name: "HEAD", Version: "!HEAD"},
			},
		},
	}, {
		name: "nosymbols",
		conf: schema.SiteConfiguration{
			SearchIndexSymbolsEnabled: boolPtr(false)},
		repo: "repo",
		want: zoektIndexOptions{
			RepoID: 1,
			Branches: []zoekt.RepositoryBranch{
				{Name: "HEAD", Version: "!HEAD"},
			},
		},
	}, {
		name: "largefiles",
		conf: schema.SiteConfiguration{
			SearchLargeFiles: []string{"**/*.jar", "*.bin"},
		},
		repo: "repo",
		want: zoektIndexOptions{
			RepoID:     1,
			Symbols:    true,
			LargeFiles: []string{"**/*.jar", "*.bin"},
			Branches: []zoekt.RepositoryBranch{
				{Name: "HEAD", Version: "!HEAD"},
			},
		},
	}, {
		name: "implicit HEAD",
		conf: vcConf(vc("foo", "repo@b", "repo@a"), vc("bar", "repo@c", "repo@a", "other@d")),
		repo: "repo",
		want: zoektIndexOptions{
			RepoID:  1,
			Symbols: true,
			Branches: []zoekt.RepositoryBranch{
				{Name: "HEAD", Version: "!HEAD"},
				{Name: "a", Version: "!a"},
				{Name: "b", Version: "!b"},
				{Name: "c", Version: "!c"},
			},
		},
	}, {
		name: "implicit HEAD not in vc",
		conf: vcConf(vc("foo", "repo@a")),
		repo: "not_in_version_context",
		want: zoektIndexOptions{
			RepoID:  3,
			Symbols: true,
			Branches: []zoekt.RepositoryBranch{{
				Name:    "HEAD",
				Version: "!HEAD",
			}},
		},
	}, {
		name: "explicit HEAD",
		conf: vcConf(vc("foo", "repo@HEAD", "repo@a")),
		repo: "repo",
		want: zoektIndexOptions{
			RepoID:  1,
			Symbols: true,
			Branches: []zoekt.RepositoryBranch{
				{Name: "HEAD", Version: "!HEAD"},
				{Name: "a", Version: "!a"},
			},
		},
	}, {
		// a revision can be the empty string, treat as HEAD
		name: "explicit HEAD empty",
		conf: vcConf(vc("foo", "repo", "repo@a")),
		repo: "repo",
		want: zoektIndexOptions{
			RepoID:  1,
			Symbols: true,
			Branches: []zoekt.RepositoryBranch{
				{Name: "HEAD", Version: "!HEAD"},
				{Name: "a", Version: "!a"},
			},
		},
	}, {
		name: "conf index branches",
		conf: withBranches(schema.SiteConfiguration{}, map[string][]string{"repo": {"a"}}),
		repo: "repo",
		want: zoektIndexOptions{
			RepoID:  1,
			Symbols: true,
			Branches: []zoekt.RepositoryBranch{
				{Name: "HEAD", Version: "!HEAD"},
				{Name: "a", Version: "!a"},
			},
		},
	}, {
		name: "conf index branches and vc",
		conf: withBranches(
			vcConf(vc("foo", "repo", "repo@a", "repo@b")),
			map[string][]string{"repo": {"b", "c"}}),
		repo: "repo",
		want: zoektIndexOptions{
			RepoID:  1,
			Symbols: true,
			Branches: []zoekt.RepositoryBranch{
				{Name: "HEAD", Version: "!HEAD"},
				{Name: "a", Version: "!a"},
				{Name: "b", Version: "!b"},
				{Name: "c", Version: "!c"},
			},
		},
	}}

	{
		// Generate case for no more than than 64 branches
		var branches []string
		for i := 0; i < 100; i++ {
			branches = append(branches, fmt.Sprintf("%.2d", i))
		}
		want := []zoekt.RepositoryBranch{{Name: "HEAD", Version: "!HEAD"}}
		for i := 0; i < 63; i++ {
			want = append(want, zoekt.RepositoryBranch{
				Name:    fmt.Sprintf("%.2d", i),
				Version: fmt.Sprintf("!%.2d", i),
			})
		}
		cases = append(cases, caseT{
			name: "limit branches",
			conf: withBranches(schema.SiteConfiguration{}, map[string][]string{"repo": branches}),
			repo: "repo",
			want: zoektIndexOptions{
				RepoID:   1,
				Symbols:  true,
				Branches: want,
			},
		})
	}

	getRepoIndexOptions := func(repo string) (*RepoIndexOptions, error) {
		repoID := int32(1)
		for _, r := range []string{"repo", "foo", "not_in_version_context"} {
			if r == repo {
				break
			}
			repoID++
		}
		return &RepoIndexOptions{
			RepoID: repoID,
			GetVersion: func(branch string) (string, error) {
				return "!" + branch, nil
			},
		}, nil
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := GetIndexOptions(&tc.conf, getRepoIndexOptions, tc.repo)

			var got zoektIndexOptions
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatal("mismatch (-want, +got):\n", diff)
			}
		})
	}
}

func TestGetIndexOptions_getVersion(t *testing.T) {
	conf := schema.SiteConfiguration{
		ExperimentalFeatures: &schema.ExperimentalFeatures{
			VersionContexts: []*schema.VersionContext{
				parseVersionContext("foo", "repo@b1", "repo@b2"),
			},
		},
	}

	boom := errors.New("boom")
	cases := []struct {
		name    string
		f       func(string) (string, error)
		want    []zoekt.RepositoryBranch
		wantErr string
	}{{
		name: "error",
		f: func(_ string) (string, error) {
			return "", boom
		},
		wantErr: "boom",
	}, {
		// no HEAD means we don't index anything. This leads to zoekt having
		// an empty index.
		name: "no HEAD",
		f: func(branch string) (string, error) {
			if branch == "HEAD" {
				return "", nil
			}
			return "!" + branch, nil
		},
		want: nil,
	}, {
		name: "no branch",
		f: func(branch string) (string, error) {
			if branch == "b1" {
				return "", nil
			}
			return "!" + branch, nil
		},
		want: []zoekt.RepositoryBranch{
			{Name: "HEAD", Version: "!HEAD"},
			{Name: "b2", Version: "!b2"},
		},
	}, {
		name: "all",
		f: func(branch string) (string, error) {
			return "!" + branch, nil
		},
		want: []zoekt.RepositoryBranch{
			{Name: "HEAD", Version: "!HEAD"},
			{Name: "b1", Version: "!b1"},
			{Name: "b2", Version: "!b2"},
		},
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			getRepoIndexOptions := func(repo string) (*RepoIndexOptions, error) {
				return &RepoIndexOptions{
					GetVersion: tc.f,
				}, nil
			}

			b := GetIndexOptions(&conf, getRepoIndexOptions, "repo")

			var got zoektIndexOptions
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatal(err)
			}

			if got.Error != tc.wantErr {
				t.Fatalf("expected error %v, got body %s and error %v", tc.wantErr, b, got.Error)
			}
			if tc.wantErr != "" {
				return
			}

			if diff := cmp.Diff(tc.want, got.Branches); diff != "" {
				t.Fatal("mismatch (-want, +got):\n", diff)
			}
		})
	}
}

func TestGetIndexOptions_batch(t *testing.T) {
	var (
		repos []string
		want  []zoektIndexOptions
	)
	for i := 0; i < 100; i++ {
		if i%20 == 0 {
			repos = append(repos, fmt.Sprintf("error-%02d", i))
			want = append(want, zoektIndexOptions{Error: "error"})
		} else {
			repo := fmt.Sprintf("repo-%02d", i)
			repos = append(repos, repo)
			want = append(want, zoektIndexOptions{
				Symbols: true,
				Branches: []zoekt.RepositoryBranch{
					{Name: "HEAD", Version: "!HEAD-" + repo},
				},
			})
		}
	}
	getRepoIndexOptions := func(repo string) (*RepoIndexOptions, error) {
		return &RepoIndexOptions{
			GetVersion: func(branch string) (string, error) {
				if strings.HasPrefix(repo, "error") {
					return "", errors.New("error")
				}
				return fmt.Sprintf("!%s-%s", branch, repo), nil
			},
		}, nil
	}

	b := GetIndexOptions(&schema.SiteConfiguration{}, getRepoIndexOptions, repos...)
	dec := json.NewDecoder(bytes.NewReader(b))
	got := make([]zoektIndexOptions, len(repos))
	for i := range repos {
		if err := dec.Decode(&got[i]); err != nil {
			t.Fatal(err)
		}
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatal("mismatch (-want, +got):\n", diff)
	}
}

func parseVersionContext(name string, repoRevStrs ...string) *schema.VersionContext {
	var repoRevs []*schema.VersionContextRevision
	for _, repo := range repoRevStrs {
		rev := ""
		if idx := strings.LastIndex(repo, "@"); idx > 0 {
			rev = repo[idx+1:]
			repo = repo[:idx]
		}
		repoRevs = append(repoRevs, &schema.VersionContextRevision{
			Repo: repo,
			Rev:  rev,
		})
	}
	return &schema.VersionContext{
		Name:      name,
		Revisions: repoRevs,
	}
}

func boolPtr(b bool) *bool {
	return &b
}
