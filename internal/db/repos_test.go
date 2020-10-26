package db

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/keegancsmith/sqlf"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
)

func TestParseIncludePattern(t *testing.T) {
	tests := map[string]struct {
		exact  []string
		like   []string
		regexp string

		pattern []*sqlf.Query // only tested if non-nil
	}{
		`^$`:              {exact: []string{""}},
		`(^$)`:            {exact: []string{""}},
		`((^$))`:          {exact: []string{""}},
		`^((^$))$`:        {exact: []string{""}},
		`^()$`:            {exact: []string{""}},
		`^(())$`:          {exact: []string{""}},
		`^a$`:             {exact: []string{"a"}},
		`(^a$)`:           {exact: []string{"a"}},
		`((^a$))`:         {exact: []string{"a"}},
		`^((^a$))$`:       {exact: []string{"a"}},
		`^(a)$`:           {exact: []string{"a"}},
		`^((a))$`:         {exact: []string{"a"}},
		`^a|b$`:           {like: []string{"a%", "%b"}}, // "|" has higher precedence than "^" or "$"
		`^(a)|(b)$`:       {like: []string{"a%", "%b"}}, // "|" has higher precedence than "^" or "$"
		`^(a|b)$`:         {exact: []string{"a", "b"}},
		`(^a$)|(^b$)`:     {exact: []string{"a", "b"}},
		`((^a$)|(^b$))`:   {exact: []string{"a", "b"}},
		`^((^a$)|(^b$))$`: {exact: []string{"a", "b"}},
		`^((a)|(b))$`:     {exact: []string{"a", "b"}},
		`abc`:             {like: []string{"%abc%"}},
		`a|b`:             {like: []string{"%a%", "%b%"}},
		`^a(b|c)$`:        {exact: []string{"ab", "ac"}},

		`^github\.com/foo/bar`: {like: []string{"github.com/foo/bar%"}},

		`github.com`:  {regexp: `github.com`},
		`github\.com`: {like: []string{`%github.com%`}},

		// https://github.com/sourcegraph/sourcegraph/issues/9146
		`github.com/.*/ini$`:      {regexp: `github.com/.*/ini$`},
		`github\.com/.*/ini$`:     {regexp: `github\.com/.*/ini$`},
		`github\.com/go-ini/ini$`: {like: []string{`%github.com/go-ini/ini`}},

		// https://github.com/sourcegraph/sourcegraph/issues/4166
		`golang/oauth.*`:                    {like: []string{"%golang/oauth%"}},
		`^golang/oauth.*`:                   {like: []string{"golang/oauth%"}},
		`golang/(oauth.*|bla)`:              {like: []string{"%golang/oauth%", "%golang/bla%"}},
		`golang/(oauth|bla)`:                {like: []string{"%golang/oauth%", "%golang/bla%"}},
		`^github.com/(golang|go-.*)/oauth$`: {regexp: `^github.com/(golang|go-.*)/oauth$`},
		`^github.com/(go.*lang|go)/oauth$`:  {regexp: `^github.com/(go.*lang|go)/oauth$`},

		`(^github\.com/Microsoft/vscode$)|(^github\.com/sourcegraph/go-langserver$)`: {exact: []string{"github.com/Microsoft/vscode", "github.com/sourcegraph/go-langserver"}},

		// Avoid DoS when there are too many possible matches to enumerate.
		`^(a|b)(c|d)(e|f)(g|h)(i|j)(k|l)(m|n)$`: {regexp: `^(a|b)(c|d)(e|f)(g|h)(i|j)(k|l)(m|n)$`},
		`^[0-a]$`:                               {regexp: `^[0-a]$`},
		`sourcegraph|^github\.com/foo/bar$`: {
			like:    []string{`%sourcegraph%`},
			exact:   []string{"github.com/foo/bar"},
			pattern: []*sqlf.Query{sqlf.Sprintf(`(name IN (%s) OR lower(name) LIKE %s)`, "github.com/foo/bar", "%sourcegraph%")},
		},
	}
	for pattern, want := range tests {
		exact, like, regexp, err := parseIncludePattern(pattern)
		if err != nil {
			t.Fatal(pattern, err)
		}
		if !reflect.DeepEqual(exact, want.exact) {
			t.Errorf("got exact %q, want %q for %s", exact, want.exact, pattern)
		}
		if !reflect.DeepEqual(like, want.like) {
			t.Errorf("got like %q, want %q for %s", like, want.like, pattern)
		}
		if regexp != want.regexp {
			t.Errorf("got regexp %q, want %q for %s", regexp, want.regexp, pattern)
		}
		if qs, err := parsePattern(pattern); err != nil {
			t.Fatal(pattern, err)
		} else {
			if testing.Verbose() {
				q := sqlf.Join(qs, "AND")
				t.Log(pattern, q.Query(sqlf.PostgresBindVar), q.Args())
			}

			if want.pattern != nil {
				want := queriesToString(want.pattern)
				q := queriesToString(qs)
				if want != q {
					t.Errorf("got pattern %q, want %q for %s", q, want, pattern)
				}
			}
		}
	}
}

func queriesToString(qs []*sqlf.Query) string {
	q := sqlf.Join(qs, "AND")
	return fmt.Sprintf("%s %v", q.Query(sqlf.PostgresBindVar), q.Args())
}

func TestRepos_Count(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()
	ctx = actor.WithActor(ctx, &actor.Actor{UID: 1, Internal: true})

	if count, err := Repos.Count(ctx, ReposListOptions{}); err != nil {
		t.Fatal(err)
	} else if want := 0; count != want {
		t.Errorf("got %d, want %d", count, want)
	}

	if err := Repos.Upsert(ctx, InsertRepoOp{Name: "myrepo", Description: "", Fork: false}); err != nil {
		t.Fatal(err)
	}

	if count, err := Repos.Count(ctx, ReposListOptions{}); err != nil {
		t.Fatal(err)
	} else if want := 1; count != want {
		t.Errorf("got %d, want %d", count, want)
	}

	repos, err := Repos.List(ctx, ReposListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := Repos.Delete(ctx, repos[0].ID); err != nil {
		t.Fatal(err)
	}

	if count, err := Repos.Count(ctx, ReposListOptions{}); err != nil {
		t.Fatal(err)
	} else if want := 0; count != want {
		t.Errorf("got %d, want %d", count, want)
	}
}

func TestRepos_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()
	ctx = actor.WithActor(ctx, &actor.Actor{UID: 1, Internal: true})

	if err := Repos.Upsert(ctx, InsertRepoOp{Name: "myrepo", Description: "", Fork: false}); err != nil {
		t.Fatal(err)
	}

	if count, err := Repos.Count(ctx, ReposListOptions{}); err != nil {
		t.Fatal(err)
	} else if want := 1; count != want {
		t.Errorf("got %d, want %d", count, want)
	}

	repos, err := Repos.List(ctx, ReposListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := Repos.Delete(ctx, repos[0].ID); err != nil {
		t.Fatal(err)
	}

	if count, err := Repos.Count(ctx, ReposListOptions{}); err != nil {
		t.Fatal(err)
	} else if want := 0; count != want {
		t.Errorf("got %d, want %d", count, want)
	}
}

func TestRepos_Upsert(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()
	ctx = actor.WithActor(ctx, &actor.Actor{UID: 1, Internal: true})

	if _, err := Repos.GetByName(ctx, "myrepo"); !errcode.IsNotFound(err) {
		if err == nil {
			t.Fatal("myrepo already present")
		} else {
			t.Fatal(err)
		}
	}

	if err := Repos.Upsert(ctx, InsertRepoOp{Name: "myrepo", Description: "", Fork: false}); err != nil {
		t.Fatal(err)
	}

	rp, err := Repos.GetByName(ctx, "myrepo")
	if err != nil {
		t.Fatal(err)
	}

	if rp.Name != "myrepo" {
		t.Fatalf("rp.Name: %s != %s", rp.Name, "myrepo")
	}

	ext := api.ExternalRepoSpec{
		ID:          "ext:id",
		ServiceType: "test",
		ServiceID:   "ext:test",
	}

	if err := Repos.Upsert(ctx, InsertRepoOp{Name: "myrepo", Description: "asdfasdf", Fork: false, ExternalRepo: ext}); err != nil {
		t.Fatal(err)
	}

	rp, err = Repos.GetByName(ctx, "myrepo")
	if err != nil {
		t.Fatal(err)
	}

	if rp.Name != "myrepo" {
		t.Fatalf("rp.Name: %s != %s", rp.Name, "myrepo")
	}
	if rp.Description != "asdfasdf" {
		t.Fatalf("rp.Name: %q != %q", rp.Description, "asdfasdf")
	}
	if !reflect.DeepEqual(rp.ExternalRepo, ext) {
		t.Fatalf("rp.ExternalRepo: %s != %s", rp.ExternalRepo, ext)
	}

	// Rename. Detected by external repo
	if err := Repos.Upsert(ctx, InsertRepoOp{Name: "myrepo/renamed", Description: "asdfasdf", Fork: false, ExternalRepo: ext}); err != nil {
		t.Fatal(err)
	}

	if _, err := Repos.GetByName(ctx, "myrepo"); !errcode.IsNotFound(err) {
		if err == nil {
			t.Fatal("myrepo should be renamed, but still present as myrepo")
		} else {
			t.Fatal(err)
		}
	}

	rp, err = Repos.GetByName(ctx, "myrepo/renamed")
	if err != nil {
		t.Fatal(err)
	}
	if rp.Name != "myrepo/renamed" {
		t.Fatalf("rp.Name: %s != %s", rp.Name, "myrepo/renamed")
	}
	if rp.Description != "asdfasdf" {
		t.Fatalf("rp.Name: %q != %q", rp.Description, "asdfasdf")
	}
	if !reflect.DeepEqual(rp.ExternalRepo, ext) {
		t.Fatalf("rp.ExternalRepo: %s != %s", rp.ExternalRepo, ext)
	}
}

func TestRepos_UpsertForkAndArchivedFields(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()
	ctx = actor.WithActor(ctx, &actor.Actor{UID: 1, Internal: true})

	i := 0
	for _, fork := range []bool{true, false} {
		for _, archived := range []bool{true, false} {
			i++
			name := api.RepoName(fmt.Sprintf("myrepo-%d", i))

			if err := Repos.Upsert(ctx, InsertRepoOp{Name: name, Fork: fork, Archived: archived}); err != nil {
				t.Fatal(err)
			}

			rp, err := Repos.GetByName(ctx, name)
			if err != nil {
				t.Fatal(err)
			}

			if rp.Fork != fork {
				t.Fatalf("rp.Fork: %v != %v", rp.Fork, fork)
			}
			if rp.Archived != archived {
				t.Fatalf("rp.Archived: %v != %v", rp.Archived, archived)
			}
		}
	}
}
