package backend

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/zoekt"
	"github.com/google/zoekt/query"
)

func TestHorizontalSearcher(t *testing.T) {
	var endpoints atomicMap
	endpoints.Store(prefixMap{})

	searcher := &HorizontalSearcher{
		Map: &endpoints,
		Dial: func(endpoint string) zoekt.Searcher {
			var rle zoekt.RepoListEntry
			rle.Repository.Name = endpoint
			client := &mockSearcher{
				searchResult: &zoekt.SearchResult{
					Files: []zoekt.FileMatch{{
						Repository: endpoint,
					}},
					RepoURLs: map[string]string{endpoint: endpoint},
				},
				listResult: &zoekt.RepoList{Repos: []*zoekt.RepoListEntry{&rle}},
			}
			// Return metered searcher to test that codepath
			return NewMeteredSearcher(endpoint, client)
		},
	}
	defer searcher.Close()

	// Start up background goroutines which continuously hit the searcher
	// methods to ensure we are safe under concurrency.
	for i := 0; i < 5; i++ {
		cleanup := backgroundSearch(searcher)
		defer cleanup(t)
	}

	// each map is the set of servers at a point in time. This is to mainly
	// stress the management code.
	maps := []prefixMap{
		// Start with a normal config of two replicas
		{"1", "2"},

		// Add two
		{"1", "2", "3", "4"},

		// Lose two
		{"2", "4"},

		// Lose and add
		{"1", "2"},

		// Lose all
		{},

		// Lots
		{"1", "2", "3", "4", "5", "6", "7", "8", "9"},
	}

	for _, m := range maps {
		t.Log("current", searcher.String(), "next", m)
		endpoints.Store(m)

		// Our search results should be one per server
		sr, err := searcher.Search(context.Background(), nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, fm := range sr.Files {
			got = append(got, fm.Repository)
		}
		sort.Strings(got)
		want := []string(m)
		if !cmp.Equal(want, got, cmpopts.EquateEmpty()) {
			t.Errorf("search mismatch (-want +got):\n%s", cmp.Diff(want, got))
		}

		// repohasfile depends on RepoURLs aggregating
		got = got[:0]
		for repo := range sr.RepoURLs {
			got = append(got, repo)
		}
		sort.Strings(got)
		if !cmp.Equal(want, got, cmpopts.EquateEmpty()) {
			t.Errorf("search mismatch (-want +got):\n%s", cmp.Diff(want, got))
		}

		// Our list results should be one per server
		rle, err := searcher.List(context.Background(), nil)
		if err != nil {
			t.Fatal(err)
		}
		got = []string{}
		for _, r := range rle.Repos {
			got = append(got, r.Repository.Name)
		}
		sort.Strings(got)
		if !cmp.Equal(want, got, cmpopts.EquateEmpty()) {
			t.Errorf("list mismatch (-want +got):\n%s", cmp.Diff(want, got))
		}
	}

	searcher.Close()
}

func TestSyncSearchers(t *testing.T) {
	// This test exists to ensure we test the slow path for
	// HorizontalSearcher.searchers. The slow-path is
	// syncSearchers. TestHorizontalSearcher tests the same code paths, but
	// isn't guaranteed to trigger the all the parts of syncSearchers.
	var endpoints atomicMap
	endpoints.Store(prefixMap{"a"})

	type mock struct {
		mockSearcher
		dialNum int
	}

	dialNumCounter := 0
	searcher := &HorizontalSearcher{
		Map: &endpoints,
		Dial: func(endpoint string) zoekt.Searcher {
			dialNumCounter++
			return &mock{
				dialNum: dialNumCounter,
			}
		},
	}
	defer searcher.Close()

	// First call initializes the list, second should use the fast-path so
	// should have the same dialNum.
	for i := 0; i < 2; i++ {
		t.Log("gen", i)
		m, err := searcher.syncSearchers()
		if err != nil {
			t.Fatal(err)
		}
		if len(m) != 1 {
			t.Fatal(err)
		}
		if got, want := m["a"].(*mock).dialNum, 1; got != want {
			t.Fatalf("expected immutable dail num %d, got %d", want, got)
		}
	}
}

func TestDedupper(t *testing.T) {
	parse := func(s string) []zoekt.FileMatch {
		t.Helper()
		var fms []zoekt.FileMatch
		for _, t := range strings.Split(s, " ") {
			if t == "" {
				continue
			}
			parts := strings.Split(t, ":")
			fms = append(fms, zoekt.FileMatch{
				Repository: parts[0],
				FileName:   parts[1],
			})
		}
		return fms
	}
	cases := []struct {
		name    string
		matches []string
		want    string
	}{{
		name: "empty",
		matches: []string{
			"",
		},
		want: "",
	}, {
		name: "one",
		matches: []string{
			"r1:a r1:a r1:b r2:a",
		},
		want: "r1:a r1:a r1:b r2:a",
	}, {
		name: "some dups",
		matches: []string{
			"r1:a r1:a r1:b r2:a",
			"r1:c r1:c r3:a",
		},
		want: "r1:a r1:a r1:b r2:a r3:a",
	}, {
		name: "no dups",
		matches: []string{
			"r1:a r1:a r1:b r2:a",
			"r4:c r4:c r5:a",
		},
		want: "r1:a r1:a r1:b r2:a r4:c r4:c r5:a",
	}, {
		name: "shuffled",
		matches: []string{
			"r1:a r2:a r1:a r1:b",
			"r1:c r3:a r1:c",
		},
		want: "r1:a r2:a r1:a r1:b r3:a",
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := dedupper{}
			var got []zoekt.FileMatch
			for _, s := range tc.matches {
				fms := parse(s)
				got = append(got, d.Dedup(fms)...)
			}

			want := parse(tc.want)
			if !cmp.Equal(want, got, cmpopts.EquateEmpty()) {
				t.Errorf("mismatch (-want +got):\n%s", cmp.Diff(want, got))
			}
		})
	}
}

func BenchmarkDedup(b *testing.B) {
	nRepos := 100
	nMatchPerRepo := 50
	// primes to avoid the need of dedup most of the time :)
	shardStrides := []int{7, 5, 3, 2, 1}

	shardsOrig := [][]zoekt.FileMatch{}
	for _, stride := range shardStrides {
		shard := []zoekt.FileMatch{}
		for i := stride; i <= nRepos; i += stride {
			repo := fmt.Sprintf("repo-%d", i)
			for j := 0; j < nMatchPerRepo; j++ {
				path := fmt.Sprintf("%d.go", j)
				shard = append(shard, zoekt.FileMatch{
					Repository: repo,
					FileName:   path,
				})
			}
		}
		shardsOrig = append(shardsOrig, shard)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		// Create copy since we mutate the input in Deddup
		b.StopTimer()
		shards := make([][]zoekt.FileMatch, 0, len(shardsOrig))
		for _, shard := range shardsOrig {
			shards = append(shards, append([]zoekt.FileMatch{}, shard...))
		}
		b.StartTimer()

		d := dedupper{}
		for _, shard := range shards {
			_ = d.Dedup(shard)
		}
	}
}

func backgroundSearch(searcher zoekt.Searcher) func(t *testing.T) {
	done := make(chan struct{})
	errC := make(chan error)
	go func() {
		for {
			_, err := searcher.Search(context.Background(), nil, nil)
			if err != nil {
				errC <- err
				return
			}
			_, err = searcher.List(context.Background(), nil)
			if err != nil {
				errC <- err
				return
			}

			select {
			case <-done:
				errC <- err
				return
			default:
			}
		}
	}()

	return func(t *testing.T) {
		t.Helper()
		close(done)
		if err := <-errC; err != nil {
			t.Error("concurrent search failed: ", err)
		}
	}
}

type mockSearcher struct {
	searchResult *zoekt.SearchResult
	searchError  error
	listResult   *zoekt.RepoList
	listError    error
}

func (s *mockSearcher) Search(context.Context, query.Q, *zoekt.SearchOptions) (*zoekt.SearchResult, error) {
	res := s.searchResult
	if s.searchResult != nil {
		// Copy since we mutate the File slice
		sr := *res
		sr.Files = append([]zoekt.FileMatch{}, sr.Files...)
		res = &sr
	}
	return res, s.searchError
}

func (s *mockSearcher) List(context.Context, query.Q) (*zoekt.RepoList, error) {
	return s.listResult, s.listError
}

func (*mockSearcher) Close() {}

func (*mockSearcher) String() string {
	return "mockSearcher"
}

type atomicMap struct {
	atomic.Value
}

func (m *atomicMap) Endpoints() (map[string]struct{}, error) {
	return m.Value.Load().(EndpointMap).Endpoints()
}

func (m *atomicMap) GetMany(keys ...string) ([]string, error) {
	return m.Value.Load().(EndpointMap).GetMany(keys...)
}
