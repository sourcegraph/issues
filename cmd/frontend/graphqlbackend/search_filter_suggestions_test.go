package graphqlbackend

import (
	"context"
	"reflect"
	"sort"
	"testing"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/schema"
)

func TestSearchFilterSuggestions(t *testing.T) {
	mockResolveRepoGroups = func() (map[string][]RepoGroupValue, error) {
		return map[string][]RepoGroupValue{
			"repogroup1": {},
			"repogroup2": {},
		}, nil
	}
	defer func() { mockResolveRepoGroups = nil }()

	db.Mocks.Repos.List = func(_ context.Context, _ db.ReposListOptions) ([]*types.Repo, error) {
		return []*types.Repo{
			{Name: "github.com/foo/repo"},
			{Name: "bar-repo"},
		}, nil
	}
	defer func() { db.Mocks.Repos.List = nil }()

	tests := []struct {
		want     *searchFilterSuggestions
		globbing bool
	}{
		{want: &searchFilterSuggestions{
			repogroups: []string{"repogroup1", "repogroup2"},
			repos:      []string{"^bar-repo$", `^github\.com/foo/repo$`}},
			globbing: false,
		},
		{want: &searchFilterSuggestions{
			repogroups: []string{"repogroup1", "repogroup2"},
			repos:      []string{"bar-repo", `github.com/foo/repo`}},
			globbing: true,
		},
	}

	mockDecodedViewerFinalSettings = &schema.Settings{}
	defer func() { mockDecodedViewerFinalSettings = nil }()

	for _, tt := range tests {
		mockDecodedViewerFinalSettings.SearchGlobbing = &tt.globbing

		r, err := (&schemaResolver{}).SearchFilterSuggestions(context.Background())
		if err != nil {
			t.Fatal("SearchFilterSuggestions:", err)
		}

		sort.Strings(r.repogroups)
		sort.Strings(r.repos)
		if !reflect.DeepEqual(r, tt.want) {
			t.Errorf("got != want\ngot:  %v\nwant: %v", r, tt.want)
		}
	}
}
