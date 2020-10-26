package api

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	bundles "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/client"
	bundlemocks "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/client/mocks"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/store"
	storemocks "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/store/mocks"
)

func TestFindClosestDumps(t *testing.T) {
	mockStore := storemocks.NewMockStore()
	mockBundleManagerClient := bundlemocks.NewMockBundleManagerClient()
	mockBundleClient1 := bundlemocks.NewMockBundleClient()
	mockBundleClient2 := bundlemocks.NewMockBundleClient()
	mockBundleClient3 := bundlemocks.NewMockBundleClient()
	mockBundleClient4 := bundlemocks.NewMockBundleClient()
	mockGitserverClient := NewMockGitserverClient()

	setMockStoreHasRepository(t, mockStore, 42, true)
	setMockStoreHasCommit(t, mockStore, 42, testCommit, true)
	setMockStoreFindClosestDumps(t, mockStore, 42, testCommit, "s1/main.go", true, "idx", []store.Dump{
		{ID: 50, Root: "s1/"},
		{ID: 51, Root: "s1/"},
		{ID: 52, Root: "s1/"},
		{ID: 53, Root: "s2/"},
	})
	setMockBundleManagerClientBundleClient(t, mockBundleManagerClient, map[int]bundles.BundleClient{
		50: mockBundleClient1,
		51: mockBundleClient2,
		52: mockBundleClient3,
		53: mockBundleClient4,
	})
	setMockBundleClientExists(t, mockBundleClient1, "main.go", true)
	setMockBundleClientExists(t, mockBundleClient2, "main.go", false)
	setMockBundleClientExists(t, mockBundleClient3, "main.go", true)
	setMockBundleClientExists(t, mockBundleClient4, "s1/main.go", false)

	api := testAPI(mockStore, mockBundleManagerClient, mockGitserverClient)
	dumps, err := api.FindClosestDumps(context.Background(), 42, testCommit, "s1/main.go", true, "idx")
	if err != nil {
		t.Fatalf("unexpected error finding closest dumps: %s", err)
	}

	expected := []store.Dump{
		{ID: 50, Root: "s1/"},
		{ID: 52, Root: "s1/"},
	}
	if diff := cmp.Diff(expected, dumps); diff != "" {
		t.Errorf("unexpected dumps (-want +got):\n%s", diff)
	}
}

func TestFindClosestDumpsInfersClosestUploads(t *testing.T) {
	mockStore := storemocks.NewMockStore()
	mockBundleManagerClient := bundlemocks.NewMockBundleManagerClient()
	mockBundleClient1 := bundlemocks.NewMockBundleClient()
	mockBundleClient2 := bundlemocks.NewMockBundleClient()
	mockBundleClient3 := bundlemocks.NewMockBundleClient()
	mockBundleClient4 := bundlemocks.NewMockBundleClient()
	mockGitserverClient := NewMockGitserverClient()

	graph := map[string][]string{
		"a": {"b", "c"},
		"b": {"d"},
		"c": {},
		"d": {},
	}

	setMockStoreHasRepository(t, mockStore, 42, true)
	setMockStoreHasCommit(t, mockStore, 42, testCommit, false)
	setMockGitserverCommitGraph(t, mockGitserverClient, 42, graph)
	setMockStoreFindClosestDumpsFromGraphFragment(t, mockStore, 42, testCommit, "s1/main.go", true, "idx", graph, []store.Dump{
		{ID: 50, Root: "s1/"},
		{ID: 51, Root: "s1/"},
		{ID: 52, Root: "s1/"},
		{ID: 53, Root: "s2/"},
	})
	setMockBundleManagerClientBundleClient(t, mockBundleManagerClient, map[int]bundles.BundleClient{
		50: mockBundleClient1,
		51: mockBundleClient2,
		52: mockBundleClient3,
		53: mockBundleClient4,
	})
	setMockBundleClientExists(t, mockBundleClient1, "main.go", true)
	setMockBundleClientExists(t, mockBundleClient2, "main.go", false)
	setMockBundleClientExists(t, mockBundleClient3, "main.go", true)
	setMockBundleClientExists(t, mockBundleClient4, "s1/main.go", false)

	api := testAPI(mockStore, mockBundleManagerClient, mockGitserverClient)
	dumps, err := api.FindClosestDumps(context.Background(), 42, testCommit, "s1/main.go", true, "idx")
	if err != nil {
		t.Fatalf("unexpected error finding closest dumps: %s", err)
	}

	expected := []store.Dump{
		{ID: 50, Root: "s1/"},
		{ID: 52, Root: "s1/"},
	}
	if diff := cmp.Diff(expected, dumps); diff != "" {
		t.Errorf("unexpected dumps (-want +got):\n%s", diff)
	}

	if value := len(mockStore.MarkRepositoryAsDirtyFunc.History()); value != 1 {
		t.Errorf("expected number of calls to store.MarkRepositoryAsDirty. want=%d have=%d", 1, value)
	}
}

func TestFindClosestDumpsDoesNotInferClosestUploadForUnknownRepository(t *testing.T) {
	mockStore := storemocks.NewMockStore()
	mockBundleManagerClient := bundlemocks.NewMockBundleManagerClient()
	mockGitserverClient := NewMockGitserverClient()

	setMockStoreHasRepository(t, mockStore, 42, false)
	setMockStoreHasCommit(t, mockStore, 42, testCommit, false)

	api := testAPI(mockStore, mockBundleManagerClient, mockGitserverClient)
	dumps, err := api.FindClosestDumps(context.Background(), 42, testCommit, "s1/main.go", true, "idx")
	if err != nil {
		t.Fatalf("unexpected error finding closest dumps: %s", err)
	}
	if len(dumps) != 0 {
		t.Errorf("unexpected number of dumps. want=%d have=%d", 0, len(dumps))
	}

	if value := len(mockStore.MarkRepositoryAsDirtyFunc.History()); value != 0 {
		t.Errorf("expected number of calls to store.MarkRepositoryAsDirty. want=%d have=%d", 0, value)
	}
}
