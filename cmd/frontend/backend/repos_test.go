package backend

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/inventory"
	"github.com/sourcegraph/sourcegraph/internal/rcache"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater/protocol"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
	"github.com/sourcegraph/sourcegraph/internal/vcs/util"
)

func TestReposService_Get(t *testing.T) {
	var s Repos
	ctx := testContext()

	wantRepo := &types.Repo{ID: 1, Name: "github.com/u/r"}

	calledGet := database.Mocks.Repos.MockGet_Return(t, wantRepo)

	repo, err := s.Get(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !*calledGet {
		t.Error("!calledGet")
	}
	// Should not be called because mock GitHub has same data as mock DB.
	if !reflect.DeepEqual(repo, wantRepo) {
		t.Errorf("got %+v, want %+v", repo, wantRepo)
	}
}

func TestReposService_List(t *testing.T) {
	var s Repos
	ctx := testContext()

	wantRepos := []*types.Repo{
		{Name: "r1"},
		{Name: "r2"},
	}

	calledList := database.Mocks.Repos.MockList(t, "r1", "r2")

	repos, err := s.List(ctx, database.ReposListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !*calledList {
		t.Error("!calledList")
	}
	if !reflect.DeepEqual(repos, wantRepos) {
		t.Errorf("got %+v, want %+v", repos, wantRepos)
	}
}

func TestRepos_Add(t *testing.T) {
	var s Repos
	ctx := testContext()

	const repoName = "my/repo"
	const newName = "my/repo2"

	calledRepoLookup := false
	repoupdater.MockRepoLookup = func(args protocol.RepoLookupArgs) (*protocol.RepoLookupResult, error) {
		calledRepoLookup = true
		if args.Repo != repoName {
			t.Errorf("got %q, want %q", args.Repo, repoName)
		}
		return &protocol.RepoLookupResult{
			Repo: &protocol.RepoInfo{Name: newName, Description: "d"},
		}, nil
	}
	defer func() { repoupdater.MockRepoLookup = nil }()

	// The repoName could change if it has been renamed on the code host
	addedName, err := s.Add(ctx, repoName)
	if err != nil {
		t.Fatal(err)
	}
	if addedName != newName {
		t.Fatalf("Want %q, got %q", newName, addedName)
	}
	if !calledRepoLookup {
		t.Error("!calledRepoLookup")
	}
}

type gitObjectInfo string

func (oid gitObjectInfo) OID() git.OID {
	var v git.OID
	copy(v[:], []byte(oid))
	return v
}

func TestReposGetInventory(t *testing.T) {
	var s Repos
	ctx := testContext()

	const (
		wantRepo     = "a"
		wantCommitID = "cccccccccccccccccccccccccccccccccccccccc"
		wantRootOID  = "oid-root"
	)
	repoupdater.MockRepoLookup = func(args protocol.RepoLookupArgs) (*protocol.RepoLookupResult, error) {
		if args.Repo != wantRepo {
			t.Errorf("got %q, want %q", args.Repo, wantRepo)
		}
		return &protocol.RepoLookupResult{Repo: &protocol.RepoInfo{Name: wantRepo}}, nil
	}
	defer func() { repoupdater.MockRepoLookup = nil }()
	git.Mocks.Stat = func(commit api.CommitID, path string) (fs.FileInfo, error) {
		if commit != wantCommitID {
			t.Errorf("got commit %q, want %q", commit, wantCommitID)
		}
		return &util.FileInfo{Name_: path, Mode_: os.ModeDir, Sys_: gitObjectInfo(wantRootOID)}, nil
	}
	git.Mocks.ReadDir = func(commit api.CommitID, name string, recurse bool) ([]fs.FileInfo, error) {
		if commit != wantCommitID {
			t.Errorf("got commit %q, want %q", commit, wantCommitID)
		}
		switch name {
		case "":
			return []fs.FileInfo{
				&util.FileInfo{Name_: "a", Mode_: os.ModeDir, Sys_: gitObjectInfo("oid-a")},
				&util.FileInfo{Name_: "b.go", Size_: 12},
			}, nil
		case "a":
			return []fs.FileInfo{&util.FileInfo{Name_: "a/c.m", Size_: 24}}, nil
		default:
			panic("unhandled mock ReadDir " + name)
		}
	}
	git.Mocks.NewFileReader = func(commit api.CommitID, name string) (io.ReadCloser, error) {
		if commit != wantCommitID {
			t.Errorf("got commit %q, want %q", commit, wantCommitID)
		}
		var data []byte
		switch name {
		case "b.go":
			data = []byte("package main")
		case "a/c.m":
			data = []byte("@interface X:NSObject {}")
		default:
			panic("unhandled mock ReadFile " + name)
		}
		return io.NopCloser(bytes.NewReader(data)), nil
	}
	defer git.ResetMocks()

	tests := []struct {
		useEnhancedLanguageDetection bool
		want                         *inventory.Inventory
	}{
		{
			useEnhancedLanguageDetection: false,
			want: &inventory.Inventory{
				Languages: []inventory.Lang{
					{Name: "Limbo", TotalBytes: 24, TotalLines: 0}, // obviously incorrect, but this is how the pre-enhanced lang detection worked
					{Name: "Go", TotalBytes: 12, TotalLines: 0},
				},
			},
		},
		{
			useEnhancedLanguageDetection: true,
			want: &inventory.Inventory{
				Languages: []inventory.Lang{
					{Name: "Objective-C", TotalBytes: 24, TotalLines: 1},
					{Name: "Go", TotalBytes: 12, TotalLines: 1},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("useEnhancedLanguageDetection=%v", test.useEnhancedLanguageDetection), func(t *testing.T) {
			rcache.SetupForTest(t)
			orig := useEnhancedLanguageDetection
			useEnhancedLanguageDetection = test.useEnhancedLanguageDetection
			defer func() { useEnhancedLanguageDetection = orig }() // reset

			inv, err := s.GetInventory(ctx, &types.Repo{Name: wantRepo}, wantCommitID, false)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, inv); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestMain(m *testing.M) {
	flag.Parse()
	if !testing.Verbose() {
		log15.Root().SetHandler(log15.DiscardHandler())
	}
	os.Exit(m.Run())
}
