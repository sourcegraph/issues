package worker

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/precise-code-intel-worker/internal/metrics"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bloomfilter"
	bundlemocks "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/client/mocks"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence"
	persistencemocks "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence/mocks"
	bundletypes "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/types"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/store"
	storemocks "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/store/mocks"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/vcs"
)

func TestHandle(t *testing.T) {
	setupRepoMocks(t)

	upload := store.Upload{
		ID:           42,
		Root:         "root/",
		Commit:       "deadbeef",
		RepositoryID: 50,
		Indexer:      "lsif-go",
	}

	mockStore := storemocks.NewMockStore()
	mockPersistenceStore := persistencemocks.NewMockStore()
	bundleManagerClient := bundlemocks.NewMockBundleManagerClient()
	gitserverClient := NewMockGitserverClient()

	// Set default transaction behavior
	mockStore.TransactFunc.SetDefaultReturn(mockStore, nil)
	mockStore.DoneFunc.SetDefaultHook(func(err error) error { return err })

	// Set default transaction behavior
	mockPersistenceStore.TransactFunc.SetDefaultReturn(mockPersistenceStore, nil)
	mockStore.DoneFunc.SetDefaultHook(func(err error) error { return err })

	// Give correlation package a valid input dump
	bundleManagerClient.GetUploadFunc.SetDefaultHook(copyTestDump)

	// Allowlist all files in dump
	gitserverClient.DirectoryChildrenFunc.SetDefaultReturn(map[string][]string{
		"": {"foo.go", "bar.go"},
	}, nil)

	handler := &handler{
		bundleManagerClient: bundleManagerClient,
		gitserverClient:     gitserverClient,
		metrics:             metrics.NewWorkerMetrics(&observation.TestContext),
		createStore:         func(id int) persistence.Store { return mockPersistenceStore },
	}

	requeued, err := handler.handle(context.Background(), mockStore, upload)
	if err != nil {
		t.Fatalf("unexpected error handling upload: %s", err)
	} else if requeued {
		t.Errorf("unexpected requeue")
	}

	expectedPackages := []bundletypes.Package{
		{
			DumpID:  42,
			Scheme:  "scheme B",
			Name:    "pkg B",
			Version: "v1.2.3",
		},
	}
	if len(mockStore.UpdatePackagesFunc.History()) != 1 {
		t.Errorf("unexpected number of UpdatePackages calls. want=%d have=%d", 1, len(mockStore.UpdatePackagesFunc.History()))
	} else if diff := cmp.Diff(expectedPackages, mockStore.UpdatePackagesFunc.History()[0].Arg1); diff != "" {
		t.Errorf("unexpected UpdatePackagesFunc args (-want +got):\n%s", diff)
	}

	filter, err := bloomfilter.CreateFilter([]string{"ident A"})
	if err != nil {
		t.Fatalf("unexpected error creating filter: %s", err)
	}
	expectedPackageReferences := []bundletypes.PackageReference{
		{
			DumpID:  42,
			Scheme:  "scheme A",
			Name:    "pkg A",
			Version: "v0.1.0",
			Filter:  filter,
		},
	}
	if len(mockStore.UpdatePackageReferencesFunc.History()) != 1 {
		t.Errorf("unexpected number of UpdatePackageReferences calls. want=%d have=%d", 1, len(mockStore.UpdatePackageReferencesFunc.History()))
	} else if diff := cmp.Diff(expectedPackageReferences, mockStore.UpdatePackageReferencesFunc.History()[0].Arg1); diff != "" {
		t.Errorf("unexpected UpdatePackageReferencesFunc args (-want +got):\n%s", diff)
	}

	if len(mockStore.DeleteOverlappingDumpsFunc.History()) != 1 {
		t.Errorf("unexpected number of DeleteOverlappingDumps calls. want=%d have=%d", 1, len(mockStore.DeleteOverlappingDumpsFunc.History()))
	} else if mockStore.DeleteOverlappingDumpsFunc.History()[0].Arg1 != 50 {
		t.Errorf("unexpected value for repository id. want=%d have=%d", 50, mockStore.DeleteOverlappingDumpsFunc.History()[0].Arg1)
	} else if mockStore.DeleteOverlappingDumpsFunc.History()[0].Arg2 != "deadbeef" {
		t.Errorf("unexpected value for commit. want=%s have=%s", "deadbeef", mockStore.DeleteOverlappingDumpsFunc.History()[0].Arg2)
	} else if mockStore.DeleteOverlappingDumpsFunc.History()[0].Arg3 != "root/" {
		t.Errorf("unexpected value for root. want=%s have=%s", "root/", mockStore.DeleteOverlappingDumpsFunc.History()[0].Arg3)
	} else if mockStore.DeleteOverlappingDumpsFunc.History()[0].Arg4 != "lsif-go" {
		t.Errorf("unexpected value for indexer. want=%s have=%s", "lsif-go", mockStore.DeleteOverlappingDumpsFunc.History()[0].Arg4)
	}

	if len(mockStore.MarkRepositoryAsDirtyFunc.History()) != 1 {
		t.Errorf("unexpected number of MarkRepositoryAsDirtyFunc calls. want=%d have=%d", 1, len(mockStore.MarkRepositoryAsDirtyFunc.History()))
	} else if mockStore.MarkRepositoryAsDirtyFunc.History()[0].Arg1 != 50 {
		t.Errorf("unexpected value for repository id. want=%d have=%d", 50, mockStore.MarkRepositoryAsDirtyFunc.History()[0].Arg1)
	}

	if len(bundleManagerClient.DeleteUploadFunc.History()) != 1 {
		t.Errorf("unexpected number of DeleteUpload calls. want=%d have=%d", 1, len(bundleManagerClient.DeleteUploadFunc.History()))
	}
}

func TestHandleError(t *testing.T) {
	setupRepoMocks(t)

	upload := store.Upload{
		ID:           42,
		Root:         "root/",
		Commit:       "deadbeef",
		RepositoryID: 50,
		Indexer:      "lsif-go",
	}

	mockStore := storemocks.NewMockStore()
	mockPersistenceStore := persistencemocks.NewMockStore()
	bundleManagerClient := bundlemocks.NewMockBundleManagerClient()
	gitserverClient := NewMockGitserverClient()

	// Set default transaction behavior
	mockStore.TransactFunc.SetDefaultReturn(mockStore, nil)
	mockStore.DoneFunc.SetDefaultHook(func(err error) error { return err })

	// Set default transaction behavior
	mockPersistenceStore.TransactFunc.SetDefaultReturn(mockPersistenceStore, nil)
	mockStore.DoneFunc.SetDefaultHook(func(err error) error { return err })

	// Give correlation package a valid input dump
	bundleManagerClient.GetUploadFunc.SetDefaultHook(copyTestDump)

	// Set a different tip commit
	mockStore.MarkRepositoryAsDirtyFunc.SetDefaultReturn(fmt.Errorf("uh-oh!"))

	handler := &handler{
		bundleManagerClient: bundleManagerClient,
		gitserverClient:     gitserverClient,
		metrics:             metrics.NewWorkerMetrics(&observation.TestContext),
		createStore:         func(id int) persistence.Store { return mockPersistenceStore },
	}

	requeued, err := handler.handle(context.Background(), mockStore, upload)
	if err == nil {
		t.Fatalf("unexpected nil error handling upload")
	} else if !strings.Contains(err.Error(), "uh-oh!") {
		t.Fatalf("unexpected error: %s", err)
	} else if requeued {
		t.Errorf("unexpected requeue")
	}

	if len(mockStore.DoneFunc.History()) != 1 {
		t.Errorf("unexpected number of Done calls. want=%d have=%d", 1, len(mockStore.DoneFunc.History()))
	}

	if len(bundleManagerClient.DeleteUploadFunc.History()) != 0 {
		t.Errorf("unexpected number of DeleteUpload calls. want=%d have=%d", 0, len(bundleManagerClient.DeleteUploadFunc.History()))
	}
}

func TestHandleCloneInProgress(t *testing.T) {
	t.Cleanup(func() {
		backend.Mocks.Repos.Get = nil
		backend.Mocks.Repos.ResolveRev = nil
	})

	backend.Mocks.Repos.Get = func(ctx context.Context, repoID api.RepoID) (*types.Repo, error) {
		if repoID != api.RepoID(50) {
			t.Errorf("unexpected repository name. want=%d have=%d", 50, repoID)
		}
		return &types.Repo{ID: repoID}, nil
	}

	backend.Mocks.Repos.ResolveRev = func(ctx context.Context, repo *types.Repo, rev string) (api.CommitID, error) {
		return api.CommitID(""), &vcs.RepoNotExistError{Repo: repo.Name, CloneInProgress: true}
	}

	upload := store.Upload{
		ID:           42,
		Root:         "root/",
		Commit:       "deadbeef",
		RepositoryID: 50,
		Indexer:      "lsif-go",
	}

	mockStore := storemocks.NewMockStore()
	bundleManagerClient := bundlemocks.NewMockBundleManagerClient()
	gitserverClient := NewMockGitserverClient()

	handler := &handler{
		bundleManagerClient: bundleManagerClient,
		gitserverClient:     gitserverClient,
		metrics:             metrics.NewWorkerMetrics(&observation.TestContext),
	}

	requeued, err := handler.handle(context.Background(), mockStore, upload)
	if err != nil {
		t.Fatalf("unexpected error handling upload: %s", err)
	} else if !requeued {
		t.Errorf("expected upload to be requeued")
	}

	if len(mockStore.RequeueFunc.History()) != 1 {
		t.Errorf("unexpected number of RequeueFunc calls. want=%d have=%d", 1, len(mockStore.RequeueFunc.History()))
	}
}

//
//

func copyTestDump(ctx context.Context, uploadID int) (io.ReadCloser, error) {
	return os.Open("../../testdata/dump1.lsif")
}

func setupRepoMocks(t *testing.T) {
	t.Cleanup(func() {
		backend.Mocks.Repos.Get = nil
		backend.Mocks.Repos.ResolveRev = nil
	})

	backend.Mocks.Repos.Get = func(ctx context.Context, repoID api.RepoID) (*types.Repo, error) {
		if repoID != api.RepoID(50) {
			t.Errorf("unexpected repository name. want=%d have=%d", 50, repoID)
		}
		return &types.Repo{ID: repoID}, nil
	}

	backend.Mocks.Repos.ResolveRev = func(ctx context.Context, repo *types.Repo, rev string) (api.CommitID, error) {
		if rev != "deadbeef" {
			t.Errorf("unexpected commit. want=%s have=%s", "deadbeef", rev)
		}
		return "", nil
	}
}
