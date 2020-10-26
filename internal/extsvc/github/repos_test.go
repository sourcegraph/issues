package github

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/sourcegraph/sourcegraph/internal/httpcli"
	"github.com/sourcegraph/sourcegraph/internal/rcache"
)

func TestSplitRepositoryNameWithOwner(t *testing.T) {
	owner, name, err := SplitRepositoryNameWithOwner("a/b")
	if err != nil {
		t.Fatal(err)
	}
	if want := "a"; owner != want {
		t.Errorf("got owner %q, want %q", owner, want)
	}
	if want := "b"; name != want {
		t.Errorf("got name %q, want %q", name, want)
	}
}

type mockHTTPResponseBody struct {
	count        int
	responseBody string
	status       int
}

func newMockHTTPResponseBody(responseBody string, status int) *mockHTTPResponseBody {
	return &mockHTTPResponseBody{
		responseBody: responseBody,
	}
}

func (s *mockHTTPResponseBody) Do(req *http.Request) (*http.Response, error) {
	s.count++
	status := s.status
	if status == 0 {
		status = http.StatusOK
	}
	return &http.Response{
		Request:    req,
		StatusCode: status,
		Body:       ioutil.NopCloser(strings.NewReader(s.responseBody)),
	}, nil
}

type mockHTTPEmptyResponse struct {
	statusCode int
}

func (s mockHTTPEmptyResponse) Do(req *http.Request) (*http.Response, error) {
	return &http.Response{
		Request:    req,
		StatusCode: s.statusCode,
		Body:       ioutil.NopCloser(bytes.NewReader(nil)),
	}, nil
}

func newTestClient(t *testing.T, cli httpcli.Doer) *Client {
	return newTestClientWithToken(t, "", cli)
}

func newTestClientWithToken(t *testing.T, token string, cli httpcli.Doer) *Client {
	rcache.SetupForTest(t)

	apiURL := &url.URL{Scheme: "https", Host: "example.com", Path: "/"}
	return NewClient(apiURL, token, cli)
}

// TestClient_GetRepository tests the behavior of GetRepository.
func TestClient_GetRepository(t *testing.T) {
	mock := mockHTTPResponseBody{
		responseBody: `
{
	"node_id": "i",
	"full_name": "o/r",
	"description": "d",
	"html_url": "https://github.example.com/o/r",
	"fork": true
}
`,
	}
	c := newTestClient(t, &mock)

	want := Repository{
		ID:            "i",
		NameWithOwner: "o/r",
		Description:   "d",
		URL:           "https://github.example.com/o/r",
		IsFork:        true,
	}

	repo, err := c.GetRepository(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatal(err)
	}
	if repo == nil {
		t.Error("repo == nil")
	}
	if mock.count != 1 {
		t.Errorf("mock.count == %d, expected to miss cache once", mock.count)
	}
	if !reflect.DeepEqual(repo, &want) {
		t.Errorf("got repository %+v, want %+v", repo, &want)
	}

	// Test that repo is cached (and therefore NOT fetched) from client on second request.
	repo, err = c.GetRepository(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatal(err)
	}
	if repo == nil {
		t.Error("repo == nil")
	}
	if mock.count != 1 {
		t.Errorf("mock.count == %d, expected to hit cache", mock.count)
	}
	if !reflect.DeepEqual(repo, &want) {
		t.Errorf("got repository %+v, want %+v", repo, &want)
	}
}

func TestClient_GetRepositoriesByNodeFromAPI(t *testing.T) {
	tests := []struct {
		responseBody string
		want         map[string]*Repository
		nodeIDs      []string
	}{
		{
			responseBody: `
{
  "data": {
    "nodes": [
      {
		"id": "i0",
		"nameWithOwner": "o/r0",
		"description": "d0",
		"url": "https://github.example.com/o/r0",
		"isFork": false
      },
      {
		"id": "i1",
		"nameWithOwner": "o/r1",
		"description": "d1",
		"url": "https://github.example.com/o/r1",
		"isFork": false
      },
      {
		"id": "i2",
		"nameWithOwner": "o/r2",
		"description": "d2",
		"url": "https://github.example.com/o/r2",
		"isFork": false
      }
    ]
  }
}
`,
			want: map[string]*Repository{
				"i0": {
					ID:            "i0",
					NameWithOwner: "o/r0",
					Description:   "d0",
					URL:           "https://github.example.com/o/r0",
				},
				"i1": {
					ID:            "i1",
					NameWithOwner: "o/r1",
					Description:   "d1",
					URL:           "https://github.example.com/o/r1",
				},
				"i2": {
					ID:            "i2",
					NameWithOwner: "o/r2",
					Description:   "d2",
					URL:           "https://github.example.com/o/r2",
				},
			},
			nodeIDs: []string{"i0", "i1", "i2"},
		},
		{
			responseBody: `
{
  "data": {
    "nodes": [
      {
		"id": "i0",
		"nameWithOwner": "o/r0",
		"description": "d0",
		"url": "https://github.example.com/o/r0",
		"isFork": false
      },
      {
		"id": "i1",
		"nameWithOwner": "o/r1",
		"description": "d1",
		"url": "https://github.example.com/o/r1",
		"isFork": false
      },
      null
    ]
  },
  "errors": [
    {
      "type": "NOT_FOUND",
      "path": [
        "nodes",
        2
      ],
      "locations": [
        {
          "line": 2,
          "column": 3
        }
      ],
      "message": "Could not resolve to a node with the global id of 'asdf'"
    }
  ]
}
`,
			want: map[string]*Repository{
				"i0": {
					ID:            "i0",
					NameWithOwner: "o/r0",
					Description:   "d0",
					URL:           "https://github.example.com/o/r0",
				},
				"i1": {
					ID:            "i1",
					NameWithOwner: "o/r1",
					Description:   "d1",
					URL:           "https://github.example.com/o/r1",
				},
			},
			nodeIDs: []string{"i0", "i1", "asdf"},
		},
	}

	for _, test := range tests {
		mock := mockHTTPResponseBody{
			responseBody: test.responseBody,
		}
		c := newTestClient(t, &mock)
		gotRepos, err := c.GetRepositoriesByNodeIDFromAPI(context.Background(), test.nodeIDs)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(gotRepos, test.want) {
			dmp := diffmatchpatch.New()
			t.Error("gotRepos != test.want", dmp.DiffPrettyText(dmp.DiffMain(spew.Sdump(test.want), spew.Sdump(gotRepos), false)))
		}
	}
}

// TestClient_GetRepository_nonexistent tests the behavior of GetRepository when called
// on a repository that does not exist.
func TestClient_GetRepository_nonexistent(t *testing.T) {
	mock := mockHTTPEmptyResponse{http.StatusNotFound}
	c := newTestClient(t, &mock)

	repo, err := c.GetRepository(context.Background(), "owner", "repo")
	if !IsNotFound(err) {
		t.Errorf("got err == %v, want IsNotFound(err) == true", err)
	}
	if err != ErrNotFound {
		t.Errorf("got err == %q, want ErrNotFound", err)
	}
	if repo != nil {
		t.Error("repo != nil")
	}
}

// TestClient_GetRepositoryByNodeID tests the behavior of GetRepositoryByNodeID.
func TestClient_GetRepositoryByNodeID(t *testing.T) {
	mock := mockHTTPResponseBody{
		responseBody: `
{
	"data": {
		"node": {
			"id": "i",
			"nameWithOwner": "o/r",
			"description": "d",
			"url": "https://github.example.com/o/r",
			"isFork": true
		}
	}
}
`,
	}
	c := newTestClient(t, &mock)

	want := Repository{
		ID:            "i",
		NameWithOwner: "o/r",
		Description:   "d",
		URL:           "https://github.example.com/o/r",
		IsFork:        true,
	}

	repo, err := c.GetRepositoryByNodeID(context.Background(), "i")
	if err != nil {
		t.Fatal(err)
	}
	if repo == nil {
		t.Error("repo == nil")
	}
	if mock.count != 1 {
		t.Errorf("mock.count == %d, expected to miss cache once", mock.count)
	}
	if !reflect.DeepEqual(repo, &want) {
		t.Errorf("got repository %+v, want %+v", repo, &want)
	}

	// Test that repo is cached (and therefore NOT fetched) from client on second request.
	repo, err = c.GetRepositoryByNodeID(context.Background(), "i")
	if err != nil {
		t.Fatal(err)
	}
	if repo == nil {
		t.Error("repo == nil")
	}
	if mock.count != 1 {
		t.Errorf("mock.count == %d, expected to hit cache", mock.count)
	}
	if !reflect.DeepEqual(repo, &want) {
		t.Errorf("got repository %+v, want %+v", repo, &want)
	}
}

// TestClient_GetRepositoryByNodeID_nonexistent tests the behavior of GetRepositoryByNodeID when called
// on a repository that does not exist.
func TestClient_GetRepositoryByNodeID_nonexistent(t *testing.T) {
	mock := mockHTTPResponseBody{
		responseBody: `
{
	"data": {
		"node": null
	}
}
`,
	}
	c := newTestClient(t, &mock)

	repo, err := c.GetRepositoryByNodeID(context.Background(), "i")
	if !IsNotFound(err) {
		t.Errorf("got err == %v, want IsNotFound(err) == true", err)
	}
	if repo != nil {
		t.Error("repo != nil")
	}
}

func TestClient_ListOrgRepositories(t *testing.T) {
	mock := mockHTTPResponseBody{
		responseBody: `[
  {
    "node_id": "i",
    "full_name": "o/r",
    "description": "d",
    "html_url": "https://github.example.com/o/r",
    "fork": true
  },
  {
    "node_id": "j",
    "full_name": "o/b",
    "description": "c",
    "html_url": "https://github.example.com/o/b",
    "fork": false
  }
]
`}

	c := newTestClient(t, &mock)
	wantRepos := []*Repository{
		{
			ID:            "i",
			NameWithOwner: "o/r",
			Description:   "d",
			URL:           "https://github.example.com/o/r",
			IsFork:        true,
		},
		{
			ID:            "j",
			NameWithOwner: "o/b",
			Description:   "c",
			URL:           "https://github.example.com/o/b",
			IsFork:        false,
		},
	}

	repos, hasNextPage, _, err := c.ListOrgRepositories(context.Background(), "o", 1)
	if err != nil {
		t.Fatal(err)
	}
	if !repoListsAreEqual(repos, wantRepos) {
		t.Errorf("got repositories:\n%s\nwant:\n%s", stringForRepoList(repos), stringForRepoList(wantRepos))
	}
	if !hasNextPage {
		t.Errorf("got hasNextPage: false want: true")
	}
}

func stringForRepoList(repos []*Repository) string {
	repoStrings := []string{}
	for _, repo := range repos {
		repoStrings = append(repoStrings, fmt.Sprintf("%#v", repo))
	}
	return "{\n" + strings.Join(repoStrings, ",\n") + "}\n"
}

func repoListsAreEqual(a []*Repository, b []*Repository) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if *a[i] != *b[i] {
			return false
		}
	}
	return true
}

func TestClient_ListRepositoriesForSearch(t *testing.T) {
	mock := mockHTTPResponseBody{
		responseBody: `
{
  "total_count": 2,
  "incomplete_results": false,
  "items": [
    {
      "node_id": "i",
      "full_name": "o/r",
      "description": "d",
      "html_url": "https://github.example.com/o/r",
      "fork": true
    },
    {
      "node_id": "j",
      "full_name": "a/b",
      "description": "c",
      "html_url": "https://github.example.com/a/b",
      "fork": false
    }
  ]
}
`,
	}
	c := newTestClient(t, &mock)

	wantRepos := []*Repository{
		{
			ID:            "i",
			NameWithOwner: "o/r",
			Description:   "d",
			URL:           "https://github.example.com/o/r",
			IsFork:        true,
		},
		{
			ID:            "j",
			NameWithOwner: "a/b",
			Description:   "c",
			URL:           "https://github.example.com/a/b",
			IsFork:        false,
		},
	}

	reposPage, err := c.ListRepositoriesForSearch(context.Background(), "org:sourcegraph", 1)
	if err != nil {
		t.Fatal(err)
	}
	if !repoListsAreEqual(reposPage.Repos, wantRepos) {
		t.Errorf("got repositories:\n%s\nwant:\n%s", stringForRepoList(reposPage.Repos), stringForRepoList(wantRepos))
	}
}

func TestClient_ListRepositoriesForSearch_incomplete(t *testing.T) {
	mock := mockHTTPResponseBody{
		responseBody: `
{
  "total_count": 2,
  "incomplete_results": true,
  "items": [
    {
      "node_id": "i",
      "full_name": "o/r",
      "description": "d",
      "html_url": "https://github.example.com/o/r",
      "fork": true
    },
    {
      "node_id": "j",
      "full_name": "a/b",
      "description": "c",
      "html_url": "https://github.example.com/a/b",
      "fork": false
    }
  ]
}
`,
	}
	c := newTestClient(t, &mock)

	// If we have incomplete results we want to fail. Our syncer requires all
	// repositories to be returned, otherwise it will delete the missing
	// repositories.
	_, err := c.ListRepositoriesForSearch(context.Background(), "org:sourcegraph", 1)

	if have, want := err, ErrIncompleteResults; want != have {
		t.Errorf("\nhave: %s\nwant: %s", have, want)
	}
}

// 🚨 SECURITY: test that cache entries are keyed by auth token
func TestClient_GetRepositoryByNodeID_security(t *testing.T) {
	c0 := newTestClient(t, nil)
	c1 := newTestClientWithToken(t, "tok1", nil)
	c2 := newTestClientWithToken(t, "tok2", nil)

	// Get "id0" and cache the result for c1
	c1.httpClient = newMockHTTPResponseBody(`{ "data": { "node": { "id": "id0-tok1" } } }`, http.StatusOK)
	got, err := c1.GetRepositoryByNodeID(context.Background(), "id0")
	if err != nil {
		t.Fatal(err)
	}
	expRepo := &Repository{ID: "id0-tok1"}
	if diff := cmp.Diff(expRepo, got); diff != "" {
		t.Fatal(diff)
	}

	// Verify c1 gets the "id0" from cache in subsequent calls
	c1.httpClient = newMockHTTPResponseBody(`{ "data": { "node": { "id": "SHOULD NOT BE SEEN" } } }`, http.StatusOK)
	got, err = c1.GetRepositoryByNodeID(context.Background(), "id0")
	if err != nil {
		t.Fatal(err)
	}
	expRepo = &Repository{ID: "id0-tok1"}
	if diff := cmp.Diff(expRepo, got); diff != "" {
		t.Fatal(diff)
	}

	// c2 should not get "id0" from cache
	c2.httpClient = newMockHTTPResponseBody(`{ "data": { "node": { "id": "id0-tok2" } } }`, http.StatusOK)
	got, err = c2.GetRepositoryByNodeID(context.Background(), "id0")
	if err != nil {
		t.Fatal(err)
	}
	expRepo = &Repository{ID: "id0-tok2"}
	if diff := cmp.Diff(expRepo, got); diff != "" {
		t.Fatal(diff)
	}

	// Let c1 cache "id1" as not found
	c1.httpClient = newMockHTTPResponseBody(`{}`, http.StatusNotFound)
	_, err = c1.GetRepositoryByNodeID(context.Background(), "id1")
	if err != ErrNotFound {
		t.Fatalf("want err %v, but got %v", ErrNotFound, err)
	}

	// Verify c1 sees "id1" as "not found" from cache in subsequent calls
	c1.httpClient = newMockHTTPResponseBody(`{ "data": { "node": { "id": "id1-tok1" } } }`, http.StatusOK)
	_, err = c1.GetRepositoryByNodeID(context.Background(), "id1")
	if err != ErrNotFound {
		t.Fatalf("want err %v, but got %v", ErrNotFound, err)
	}

	// c2 should get "id1" as usual
	c2.httpClient = newMockHTTPResponseBody(`{ "data": { "node": { "id": "id1-tok2" } } }`, http.StatusOK)
	got, err = c2.GetRepositoryByNodeID(context.Background(), "id1")
	if err != nil {
		t.Fatal(err)
	}
	expRepo = &Repository{ID: "id1-tok2"}
	if diff := cmp.Diff(expRepo, got); diff != "" {
		t.Fatal(diff)
	}

	// For sanity, c0 (unauthenticated) should get "id1" as usual
	c0.httpClient = newMockHTTPResponseBody(`{ "data": { "node": { "id": "id1" } } }`, http.StatusOK)
	got, err = c0.GetRepositoryByNodeID(context.Background(), "id1")
	if err != nil {
		t.Fatal(err)
	}
	expRepo = &Repository{ID: "id1"}
	if diff := cmp.Diff(expRepo, got); diff != "" {
		t.Fatal(diff)
	}
}

func TestClient_buildGetRepositoriesBatchQuery(t *testing.T) {
	repos := []string{
		"sourcegraph/grapher-tutorial",
		"sourcegraph/clojure-grapher",
		"sourcegraph/programming-challenge",
		"sourcegraph/annotate",
		"sourcegraph/sourcegraph-sublime-old",
		"sourcegraph/makex",
		"sourcegraph/pydep",
		"sourcegraph/vcsstore",
		"sourcegraph/contains.dot",
	}

	wantIncluded := `
repo0: repository(owner: "sourcegraph", name: "grapher-tutorial") { ... on Repository { ...RepositoryFields } }
repo1: repository(owner: "sourcegraph", name: "clojure-grapher") { ... on Repository { ...RepositoryFields } }
repo2: repository(owner: "sourcegraph", name: "programming-challenge") { ... on Repository { ...RepositoryFields } }
repo3: repository(owner: "sourcegraph", name: "annotate") { ... on Repository { ...RepositoryFields } }
repo4: repository(owner: "sourcegraph", name: "sourcegraph-sublime-old") { ... on Repository { ...RepositoryFields } }
repo5: repository(owner: "sourcegraph", name: "makex") { ... on Repository { ...RepositoryFields } }
repo6: repository(owner: "sourcegraph", name: "pydep") { ... on Repository { ...RepositoryFields } }
repo7: repository(owner: "sourcegraph", name: "vcsstore") { ... on Repository { ...RepositoryFields } }
repo8: repository(owner: "sourcegraph", name: "contains.dot") { ... on Repository { ...RepositoryFields } }`

	mock := mockHTTPResponseBody{responseBody: ""}
	c := newTestClient(t, &mock)
	query, err := c.buildGetReposBatchQuery(repos)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(query, wantIncluded) {
		t.Fatalf("query does not contain repository query. query=%q, want=%q", query, wantIncluded)
	}
}

func TestClient_GetReposByNameWithOwner(t *testing.T) {
	namesWithOwners := []string{
		"sourcegraph/grapher-tutorial",
		"sourcegraph/clojure-grapher",
	}

	grapherTutorialRepo := &Repository{
		ID:               "MDEwOlJlcG9zaXRvcnkxNDYwMTc5OA==",
		DatabaseID:       14601798,
		NameWithOwner:    "sourcegraph/grapher-tutorial",
		Description:      "monkey language",
		URL:              "https://github.com/sourcegraph/grapher-tutorial",
		IsPrivate:        true,
		IsFork:           false,
		IsArchived:       true,
		ViewerPermission: "ADMIN",
	}

	clojureGrapherRepo := &Repository{
		ID:               "MDEwOlJlcG9zaXRvcnkxNTc1NjkwOA==",
		DatabaseID:       15756908,
		NameWithOwner:    "sourcegraph/clojure-grapher",
		Description:      "clojure grapher",
		URL:              "https://github.com/sourcegraph/clojure-grapher",
		IsPrivate:        true,
		IsFork:           false,
		IsArchived:       true,
		ViewerPermission: "ADMIN",
	}

	testCases := []struct {
		name             string
		mockResponseBody string
		wantRepos        []*Repository
		err              string
	}{

		{
			name: "found",
			mockResponseBody: `
{
  "data": {
    "repo_sourcegraph_grapher_tutorial": {
      "id": "MDEwOlJlcG9zaXRvcnkxNDYwMTc5OA==",
      "databaseId": 14601798,
      "nameWithOwner": "sourcegraph/grapher-tutorial",
      "description": "monkey language",
      "url": "https://github.com/sourcegraph/grapher-tutorial",
      "isPrivate": true,
      "isFork": false,
      "isArchived": true,
      "viewerPermission": "ADMIN"
    },
    "repo_sourcegraph_clojure_grapher": {
      "id": "MDEwOlJlcG9zaXRvcnkxNTc1NjkwOA==",
	  "databaseId": 15756908,
      "nameWithOwner": "sourcegraph/clojure-grapher",
      "description": "clojure grapher",
      "url": "https://github.com/sourcegraph/clojure-grapher",
      "isPrivate": true,
      "isFork": false,
      "isArchived": true,
      "viewerPermission": "ADMIN"
    }
  }
}
`,
			wantRepos: []*Repository{grapherTutorialRepo, clojureGrapherRepo},
		},
		{
			name: "not found",
			mockResponseBody: `
{
  "data": {
    "repo_sourcegraph_grapher_tutorial": {
      "id": "MDEwOlJlcG9zaXRvcnkxNDYwMTc5OA==",
      "databaseId": 14601798,
      "nameWithOwner": "sourcegraph/grapher-tutorial",
      "description": "monkey language",
      "url": "https://github.com/sourcegraph/grapher-tutorial",
      "isPrivate": true,
      "isFork": false,
      "isArchived": true,
      "viewerPermission": "ADMIN"
    },
    "repo_sourcegraph_clojure_grapher": null
  },
  "errors": [
    {
      "type": "NOT_FOUND",
      "path": [
        "repo_sourcegraph_clojure_grapher"
      ],
      "locations": [
        {
          "line": 13,
          "column": 3
        }
      ],
      "message": "Could not resolve to a Repository with the name 'clojure-grapher'."
    }
  ]
}
`,
			wantRepos: []*Repository{grapherTutorialRepo},
		},
		{
			name: "error",
			mockResponseBody: `
{
  "errors": [
    {
      "path": [
        "fragment RepositoryFields",
        "foobar"
      ],
      "extensions": {
        "code": "undefinedField",
        "typeName": "Repository",
        "fieldName": "foobar"
      },
      "locations": [
        {
          "line": 10,
          "column": 3
        }
      ],
      "message": "Field 'foobar' doesn't exist on type 'Repository'"
    }
  ]
}
`,
			wantRepos: []*Repository{},
			err:       "error in GraphQL response: Field 'foobar' doesn't exist on type 'Repository'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := mockHTTPResponseBody{responseBody: tc.mockResponseBody}
			c := newTestClient(t, &mock)

			repos, err := c.GetReposByNameWithOwner(context.Background(), namesWithOwners...)
			if have, want := fmt.Sprint(err), fmt.Sprint(tc.err); tc.err != "" && have != want {
				t.Errorf("error:\nhave: %v\nwant: %v", have, want)
			}

			if mock.count != 1 {
				t.Errorf("mock.count == %d", mock.count)
			}
			if want, have := len(tc.wantRepos), len(repos); want != have {
				t.Errorf("wrong number of repos. want=%d, have=%d", want, have)
			}

			newSortFunc := func(s []*Repository) func(int, int) bool {
				return func(i, j int) bool { return s[i].ID < s[j].ID }
			}

			sort.Slice(tc.wantRepos, newSortFunc(tc.wantRepos))
			sort.Slice(repos, newSortFunc(repos))

			if !repoListsAreEqual(repos, tc.wantRepos) {
				t.Errorf("got repositories:\n%s\nwant:\n%s", stringForRepoList(repos), stringForRepoList(tc.wantRepos))
			}
		})
	}
}
