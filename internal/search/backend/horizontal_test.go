package backend

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/cockroachdb/errors"
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
		Dial: func(endpoint string) zoekt.Streamer {
			var rle zoekt.RepoListEntry
			rle.Repository.Name = endpoint
			repoID, _ := strconv.Atoi(endpoint)
			client := &mockSearcher{
				searchResult: &zoekt.SearchResult{
					Files: []zoekt.FileMatch{{
						Repository: endpoint,
					}},
				},
				listResult: &zoekt.RepoList{
					Repos:   []*zoekt.RepoListEntry{&rle},
					Crashes: 0,
					Minimal: map[uint32]*zoekt.MinimalRepoListEntry{uint32(repoID): {}},
				},
			}
			// Return metered searcher to test that codepath
			return NewMeteredSearcher(endpoint, &StreamSearchAdapter{client})
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

		// Our list results should be one per server
		rle, err := searcher.List(context.Background(), nil, nil)
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

		got = []string{}
		for r := range rle.Minimal {
			got = append(got, strconv.Itoa(int(r)))
		}
		sort.Strings(got)
		if !cmp.Equal(want, got, cmpopts.EquateEmpty()) {
			t.Errorf("list mismatch (-want +got):\n%s", cmp.Diff(want, got))
		}
	}

	searcher.Close()
}

func TestDoStreamSearch(t *testing.T) {
	var endpoints atomicMap
	endpoints.Store(prefixMap{"1"})

	searcher := &HorizontalSearcher{
		Map: &endpoints,
		Dial: func(endpoint string) zoekt.Streamer {
			client := &mockSearcher{
				searchResult: nil,
				searchError:  errors.Errorf("test error"),
			}
			// Return metered searcher to test that codepath
			return NewMeteredSearcher(endpoint, &StreamSearchAdapter{client})
		},
	}
	defer searcher.Close()

	c := make(chan *zoekt.SearchResult)
	defer close(c)
	err := searcher.StreamSearch(
		context.Background(),
		nil,
		nil,
		ZoektStreamFunc(func(event *zoekt.SearchResult) { c <- event }),
	)
	if err == nil {
		t.Fatalf("received non-nil error, but expected an error")
	}
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
		Dial: func(endpoint string) zoekt.Streamer {
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
			"zoekt-0 ",
		},
		want: "",
	}, {
		name: "one",
		matches: []string{
			"zoekt-0 r1:a r1:a r1:b r2:a",
		},
		want: "r1:a r1:a r1:b r2:a",
	}, {
		name: "some dups",
		matches: []string{
			"zoekt-0 r1:a r1:a r1:b r2:a",
			"zoekt-1 r1:c r1:c r3:a",
		},
		want: "r1:a r1:a r1:b r2:a r3:a",
	}, {
		name: "no dups",
		matches: []string{
			"zoekt-0 r1:a r1:a r1:b r2:a",
			"zoekt-1 r4:c r4:c r5:a",
		},
		want: "r1:a r1:a r1:b r2:a r4:c r4:c r5:a",
	}, {
		name: "shuffled",
		matches: []string{
			"zoekt-0 r1:a r2:a r1:a r1:b",
			"zoekt-1 r1:c r3:a r1:c",
		},
		want: "r1:a r2:a r1:a r1:b r3:a",
	}, {
		name: "some dups multi event",
		matches: []string{
			"zoekt-0 r1:a r1:a",
			"zoekt-1 r1:c r1:c r3:a",
			"zoekt-0 r1:b r2:a",
		},
		want: "r1:a r1:a r3:a r1:b r2:a",
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := dedupper{}
			var got []zoekt.FileMatch
			for _, s := range tc.matches {
				parts := strings.SplitN(s, " ", 2)
				endpoint := parts[0]
				fms := parse(parts[1])
				got = append(got, d.Dedup(endpoint, fms)...)
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
		for clientID, shard := range shards {
			_ = d.Dedup(strconv.Itoa(clientID), shard)
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
			_, err = searcher.List(context.Background(), nil, nil)
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

func (s *mockSearcher) StreamSearch(ctx context.Context, q query.Q, opts *zoekt.SearchOptions, streamer zoekt.Sender) error {
	return (&StreamSearchAdapter{s}).StreamSearch(ctx, q, opts, streamer)
}

func (s *mockSearcher) List(context.Context, query.Q, *zoekt.ListOptions) (*zoekt.RepoList, error) {
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
