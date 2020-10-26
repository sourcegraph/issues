package janitor

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/precise-code-intel-bundle-manager/internal/paths"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/lsifstore"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/store"
	"github.com/sourcegraph/sourcegraph/internal/goroutine"
)

type Janitor struct {
	store            store.Store
	lsifStore        lsifstore.Store
	bundleDir        string
	maxUploadAge     time.Duration
	maxUploadPartAge time.Duration
	maxDataAge       time.Duration
	metrics          JanitorMetrics
}

var _ goroutine.Handler = &Janitor{}

func New(
	store store.Store,
	lsifStore lsifstore.Store,
	bundleDir string,
	janitorInterval time.Duration,
	maxUploadAge time.Duration,
	maxUploadPartAge time.Duration,
	maxDataAge time.Duration,
	metrics JanitorMetrics,
) goroutine.BackgroundRoutine {
	return goroutine.NewPeriodicGoroutine(context.Background(), janitorInterval, &Janitor{
		store:            store,
		lsifStore:        lsifStore,
		bundleDir:        bundleDir,
		maxUploadAge:     maxUploadAge,
		maxUploadPartAge: maxUploadPartAge,
		maxDataAge:       maxDataAge,
		metrics:          metrics,
	})
}

// Handle performs a best-effort cleanup process.
func (j *Janitor) Handle(ctx context.Context) error {
	fns := []func(ctx context.Context){
		j.removeOldUploadFiles,
		j.removeOldUploadPartFiles,
		j.removeOldUploadingRecords,
		j.removeRecordsForDeletedRepositories,
		j.removeExpiredData,
		j.hardDeleteDeletedRecords,
		j.removeOrphanedData,
	}

	var wg sync.WaitGroup
	wg.Add(len(fns))

	for _, f := range fns {
		go func(f func(ctx context.Context)) {
			defer wg.Done()
			f(ctx)
		}(f)
	}

	wg.Wait()
	return nil
}

// removeOldUploadFiles removes all upload files that are older than the configured
// max unconverted upload age.
func (j *Janitor) removeOldUploadFiles(ctx context.Context) {
	count := j.removeOldFiles(paths.UploadsDir(j.bundleDir), j.maxUploadAge)
	if count > 0 {
		log15.Debug("Removed old upload files", "count", count)
		j.metrics.UploadFilesRemoved.Add(float64(count))
	}
}

// removeOldUploadPartFiles removes all upload part files that are older than the configured
// max upload part age.
func (j *Janitor) removeOldUploadPartFiles(ctx context.Context) {
	count := j.removeOldFiles(paths.UploadPartsDir(j.bundleDir), j.maxUploadPartAge)
	if count > 0 {
		log15.Debug("Removed old upload part files", "count", count)
		j.metrics.PartFilesRemoved.Add(float64(count))
	}
}

func (j *Janitor) removeOldFiles(root string, maxAge time.Duration) (count int) {
	fileInfos, err := ioutil.ReadDir(root)
	if err != nil {
		j.error("Failed to read directory", "path", root, "error", err)
		return 0
	}

	for _, fileInfo := range fileInfos {
		if time.Since(fileInfo.ModTime()) <= maxAge {
			continue
		}

		path := filepath.Join(root, fileInfo.Name())

		if err := os.RemoveAll(path); err != nil {
			j.error("Failed to remove path", "path", path, "error", err)
			continue
		}

		count++
	}

	return count
}

// removeOldUploadingRecords removes all upload records in the uploading state that
// are older than the max upload part age.
func (j *Janitor) removeOldUploadingRecords(ctx context.Context) {
	count, err := j.store.DeleteUploadsStuckUploading(ctx, time.Now().UTC().Add(-j.maxUploadPartAge))
	if err != nil {
		j.error("Failed to get upload records with a stuck 'uploading' state", "error", err)
		return
	}

	log15.Debug("Removed upload records stuck in the 'uploading' state", "count", count)
	j.metrics.UploadRecordsRemoved.Inc()
}

// removeRecordsForDeletedRepositories removes all upload records for deleted repositories.
func (j *Janitor) removeRecordsForDeletedRepositories(ctx context.Context) {
	counts, err := j.store.DeleteUploadsWithoutRepository(ctx, time.Now())
	if err != nil {
		j.error("Failed to get uploads without repositories", "error", err)
		return
	}

	totalCount := 0
	for _, count := range counts {
		totalCount += count
	}

	log15.Debug("Removed upload records for a deleted repository", "count", totalCount, "repository_count", len(counts))
	j.metrics.UploadRecordsRemoved.Add(float64(totalCount))
}

// removeExpiredData removes upload records that have exceeded a threshold age and
// is not an index visible from the head of the default branch.
func (j *Janitor) removeExpiredData(ctx context.Context) {
	count, err := j.store.SoftDeleteOldDumps(ctx, j.maxDataAge, time.Now())
	if err != nil {
		j.error("Failed to delete old dumps", "error", err)
		return
	}

	if count > 0 {
		log15.Debug("Removed old records not visible to the tip of the default branch of their repository", "count", count)
		j.metrics.DataRowsRemoved.Add(float64(count))
	}
}

const uploadBatchSize = 100

// hardDeleteDeletedRecords removes upload records in the deleted state.
func (j *Janitor) hardDeleteDeletedRecords(ctx context.Context) {
	count := 0
	for {
		uploads, totalCount, err := j.store.GetUploads(ctx, store.GetUploadsOptions{
			State: "deleted",
			Limit: uploadBatchSize,
		})
		if err != nil {
			j.error("Failed to get deleted upload identifiers", "error", err)
			return
		}

		ids := make([]int, 0, len(uploads))
		for _, upload := range uploads {
			ids = append(ids, upload.ID)
		}

		if j.hardDeleteBatch(ctx, ids) != nil {
			break
		}

		count += len(uploads)

		if len(uploads) >= totalCount {
			break
		}
	}

	if count > 0 {
		log15.Debug("Removed orphaned data rows", "count", count)
		j.metrics.DataRowsRemoved.Add(float64(count))
	}
}

func (j *Janitor) hardDeleteBatch(ctx context.Context, ids []int) (err error) {
	tx, err := j.store.Transact(ctx)
	if err != nil {
		return err
	}
	defer func() {
		err = tx.Done(err)
	}()

	if err := j.lsifStore.Clear(ctx, ids...); err != nil {
		j.error("Failed to remove associated data in the codeintel db", "error", err)
		return err
	}

	if err := tx.HardDeleteUploadByID(ctx, ids...); err != nil {
		j.error("Failed to hard deleted upload records", "error", err)
		return err
	}

	return nil
}

// We will allow you to sit on the council, but we do not grant you the rank
// of master. This block of code is here because the cloud deployment is all
// out-of-whack with our "clean room" ideal of what the disk and codeintel db
// relation should be. This is here just to vacuum up some of the junk that
// we may have dropped in it earlier.
//
// This should not affect (but can run harmlessly) in all deployments.
//
// TODO(efritz) - remove after 3.21 branch cut.

const orphanBatchSize = 100

// removeOrphanedData removes all the data from the codeintel database that does not
// hav ea corresponding upload record in the frontend database.
func (j *Janitor) removeOrphanedData(ctx context.Context) {
	offset := 0
	for {
		dumpIDs, err := j.lsifStore.DumpIDs(ctx, orphanBatchSize, offset)
		if err != nil {
			j.error("Failed to list dump identifiers", "error", err)
			return
		}

		states, err := j.store.GetStates(ctx, dumpIDs)
		if err != nil {
			j.error("Failed to get states for dumps", "error", err)
			return
		}

		count := 0
		for _, dumpID := range dumpIDs {
			if _, ok := states[dumpID]; !ok {
				if err := j.lsifStore.Clear(ctx, dumpID); err != nil {
					j.error("Failed to remove data for dump", "dump_id", dumpID, "error", err)
					continue
				}

				count++
			}
		}

		if count > 0 {
			log15.Debug("Removed orphaned data rows from skunkworks orphan harvester", "count", count)
			j.metrics.DataRowsRemoved.Add(float64(count))
		}

		if len(dumpIDs) < orphanBatchSize {
			break
		}

		offset += orphanBatchSize
	}
}

func (j *Janitor) error(message string, ctx ...interface{}) {
	j.metrics.Errors.Inc()
	log15.Error(message, ctx...)
}
