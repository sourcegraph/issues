package backend

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/envvar"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtesting"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/rcache"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
	"github.com/sourcegraph/sourcegraph/internal/vcs/util"
)

func TestCountGoImporters(t *testing.T) {
	ctx := testContext()
	const wantRepoName = "github.com/alice/myrepo"

	rcache.SetupForTest(t)

	orig := envvar.SourcegraphDotComMode()
	envvar.MockSourcegraphDotComMode(true)
	defer envvar.MockSourcegraphDotComMode(orig) // reset

	mockTransport := mockRoundTripper{
		response: `{"results":[{"path":"w/x"},{"path":"y/z"}]}`,
	}
	countGoImportersHTTPClient = &http.Client{Transport: mockTransport}
	defer func() { countGoImportersHTTPClient = nil }()

	Mocks.Repos.GetByName = func(_ context.Context, repoName api.RepoName) (*types.Repo, error) {
		if repoName != wantRepoName {
			t.Errorf("got repo name %q, want %q", repoName, wantRepoName)
		}
		return &types.Repo{
			Name: repoName,
			ExternalRepo: api.ExternalRepoSpec{
				ServiceType: extsvc.TypeGitHub,
			},
		}, nil
	}
	git.Mocks.ResolveRevision = func(spec string, opt git.ResolveRevisionOptions) (api.CommitID, error) {
		return "c", nil
	}
	git.Mocks.ReadDir = func(commit api.CommitID, name string, recurse bool) ([]fs.FileInfo, error) {
		return []fs.FileInfo{
			&util.FileInfo{Name_: "d/a.go", Mode_: os.ModeDir},
			&util.FileInfo{Name_: "d/b.go", Mode_: os.ModeDir},
			&util.FileInfo{Name_: "c.go", Mode_: 0},
		}, nil
	}

	count, err := CountGoImporters(ctx, &dbtesting.MockDB{}, wantRepoName)
	if err != nil {
		t.Fatal(err)
	}
	if want := 4; /* 2 results (w/x, y/z) * 2 Go packages (d and root) */ count != want {
		t.Errorf("got count %d, want %d", count, want)
	}
}

func TestListGoPackagesInRepoImprecise(t *testing.T) {
	t.Run("disabled on non-Sourcegraph.com", func(t *testing.T) {
		if _, err := listGoPackagesInRepoImprecise(context.Background(), &dbtesting.MockDB{}, "a"); err == nil || !strings.Contains(err.Error(), "only supported on Sourcegraph.com") {
			t.Error("want listGoPackagesInRepoImprecise to only be supported on Sourcegraph.com")
		}
	})
}

type mockRoundTripper struct {
	response string
}

func (t mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(t.response)),
		Header:     make(http.Header),
	}, nil
}

func TestIsPossibleExternallyImportableGoPackageDir(t *testing.T) {
	tests := map[string]bool{
		"a":            true,
		"a/b":          true,
		".":            true,
		"a/internal/b": false,
		"a/_b":         false,
		"a/_b/c":       false,
		"vendor/a":     false,
		"a/vendor/b":   false,
		"a/testdata/b": false,
	}
	for dirPath, want := range tests {
		if got := isPossibleExternallyImportableGoPackageDir(dirPath); got != want {
			t.Errorf("%q: got %v, want %v", dirPath, got, want)
		}
	}
}
