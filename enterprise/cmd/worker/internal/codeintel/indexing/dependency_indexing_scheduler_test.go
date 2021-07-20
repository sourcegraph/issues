package indexing

import (
	"context"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/dbstore"
	lsifstore "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/lsifstore"
	"github.com/sourcegraph/sourcegraph/lib/codeintel/semantic"
)

func TestDependencyIndexingSchedulerHandler(t *testing.T) {
	dependencyIndexingRepositoryIDs = []int{50}

	mockDBStore := NewMockDBStore()
	mockScanner := NewMockPackageReferenceScanner()
	mockDBStore.WithFunc.SetDefaultReturn(mockDBStore)
	mockDBStore.GetUploadByIDFunc.SetDefaultReturn(dbstore.Upload{ID: 42, RepositoryID: 50}, true, nil)
	mockDBStore.ReferencesForUploadFunc.SetDefaultReturn(mockScanner, nil)

	mockScanner.NextFunc.PushReturn(lsifstore.PackageReference{Package: lsifstore.Package{DumpID: 42, Scheme: "test", Name: "name1", Version: "v2.2.0"}}, true, nil)
	mockScanner.NextFunc.PushReturn(lsifstore.PackageReference{Package: lsifstore.Package{DumpID: 42, Scheme: "test", Name: "name1", Version: "v3.2.0"}}, true, nil)
	mockScanner.NextFunc.PushReturn(lsifstore.PackageReference{Package: lsifstore.Package{DumpID: 42, Scheme: "test", Name: "name2", Version: "v3.2.2"}}, true, nil)
	mockScanner.NextFunc.PushReturn(lsifstore.PackageReference{Package: lsifstore.Package{DumpID: 42, Scheme: "test", Name: "name2", Version: "v2.2.1"}}, true, nil)
	mockScanner.NextFunc.PushReturn(lsifstore.PackageReference{Package: lsifstore.Package{DumpID: 42, Scheme: "test", Name: "name2", Version: "v4.2.3"}}, true, nil)
	mockScanner.NextFunc.PushReturn(lsifstore.PackageReference{Package: lsifstore.Package{DumpID: 42, Scheme: "test", Name: "name1", Version: "v1.2.0"}}, true, nil)
	mockScanner.NextFunc.SetDefaultReturn(lsifstore.PackageReference{}, false, nil)

	indexEnqueuer := NewMockIndexEnqueuer()

	handler := &dependencyIndexingSchedulerHandler{
		dbStore:       mockDBStore,
		indexEnqueuer: indexEnqueuer,
	}

	job := dbstore.DependencyIndexingJob{
		UploadID: 42,
	}
	if err := handler.Handle(context.Background(), job); err != nil {
		t.Fatalf("unexpected error performing update: %s", err)
	}

	if len(indexEnqueuer.QueueIndexesForPackageFunc.History()) != 6 {
		t.Errorf("unexpected number of calls to QueueIndexesForPackage. want=%d have=%d", 6, len(indexEnqueuer.QueueIndexesForRepositoryFunc.History()))
	} else {
		var packages []semantic.Package
		for _, call := range indexEnqueuer.QueueIndexesForPackageFunc.History() {
			packages = append(packages, call.Arg1)
		}
		sort.Slice(packages, func(i, j int) bool {
			for _, pair := range [][2]string{
				{packages[i].Scheme, packages[j].Scheme},
				{packages[i].Name, packages[j].Name},
				{packages[i].Version, packages[j].Version},
			} {
				if pair[0] < pair[1] {
					return true
				}
				if pair[1] < pair[0] {
					break
				}
			}

			return false
		})

		expectedPackages := []semantic.Package{
			{Scheme: "test", Name: "name1", Version: "v1.2.0"},
			{Scheme: "test", Name: "name1", Version: "v2.2.0"},
			{Scheme: "test", Name: "name1", Version: "v3.2.0"},
			{Scheme: "test", Name: "name2", Version: "v2.2.1"},
			{Scheme: "test", Name: "name2", Version: "v3.2.2"},
			{Scheme: "test", Name: "name2", Version: "v4.2.3"},
		}
		if diff := cmp.Diff(expectedPackages, packages); diff != "" {
			t.Errorf("unexpected packages (-want +got):\n%s", diff)
		}
	}
}

func TestDependencyIndexingSchedulerHandlerShouldSkipRepository(t *testing.T) {
	dependencyIndexingRepositoryIDs = []int{50}

	mockDBStore := NewMockDBStore()
	mockScanner := NewMockPackageReferenceScanner()
	mockDBStore.WithFunc.SetDefaultReturn(mockDBStore)
	mockDBStore.GetUploadByIDFunc.SetDefaultReturn(dbstore.Upload{ID: 42, RepositoryID: 51}, true, nil)
	mockDBStore.ReferencesForUploadFunc.SetDefaultReturn(mockScanner, nil)

	indexEnqueuer := NewMockIndexEnqueuer()

	handler := &dependencyIndexingSchedulerHandler{
		dbStore:       mockDBStore,
		indexEnqueuer: indexEnqueuer,
	}

	job := dbstore.DependencyIndexingJob{
		UploadID: 42,
	}
	if err := handler.Handle(context.Background(), job); err != nil {
		t.Fatalf("unexpected error performing update: %s", err)
	}

	if len(indexEnqueuer.QueueIndexesForPackageFunc.History()) != 0 {
		t.Errorf("unexpected number of calls to QueueIndexesForPackage. want=%d have=%d", 0, len(indexEnqueuer.QueueIndexesForRepositoryFunc.History()))
	}
}
