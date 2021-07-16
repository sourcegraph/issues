package dbstore

import (
	"fmt"

	"github.com/sourcegraph/sourcegraph/internal/metrics"
	"github.com/sourcegraph/sourcegraph/internal/observation"
)

type operations struct {
	addUploadPart                          *observation.Operation
	calculateVisibleUploads                *observation.Operation
	commitGraphMetadata                    *observation.Operation
	definitionDumps                        *observation.Operation
	deleteIndexByID                        *observation.Operation
	deleteIndexesWithoutRepository         *observation.Operation
	deleteOldIndexes                       *observation.Operation
	deleteOverlappingDumps                 *observation.Operation
	deleteUploadByID                       *observation.Operation
	deleteUploadsStuckUploading            *observation.Operation
	deleteUploadsWithoutRepository         *observation.Operation
	dequeue                                *observation.Operation
	dequeueIndex                           *observation.Operation
	dirtyRepositories                      *observation.Operation
	findClosestDumps                       *observation.Operation
	findClosestDumpsFromGraphFragment      *observation.Operation
	getAutoindexDisabledRepositories       *observation.Operation
	getDumpsByIDs                          *observation.Operation
	getIndexByID                           *observation.Operation
	getIndexConfigurationByRepositoryID    *observation.Operation
	getIndexes                             *observation.Operation
	getIndexesByIDs                        *observation.Operation
	getOldestCommitDate                    *observation.Operation
	getRepositoriesWithIndexConfiguration  *observation.Operation
	getUploadByID                          *observation.Operation
	getUploads                             *observation.Operation
	getUploadsByIDs                        *observation.Operation
	hardDeleteUploadByID                   *observation.Operation
	hasCommit                              *observation.Operation
	hasRepository                          *observation.Operation
	indexQueueSize                         *observation.Operation
	insertDependencyIndexingJob            *observation.Operation
	insertDependencyRepoAddingJob          *observation.Operation
	insertIndex                            *observation.Operation
	insertUpload                           *observation.Operation
	isQueued                               *observation.Operation
	markComplete                           *observation.Operation
	markErrored                            *observation.Operation
	markFailed                             *observation.Operation
	markIndexComplete                      *observation.Operation
	markIndexErrored                       *observation.Operation
	markQueued                             *observation.Operation
	markRepositoryAsDirty                  *observation.Operation
	queueSize                              *observation.Operation
	referenceIDsAndFilters                 *observation.Operation
	referencesForUpload                    *observation.Operation
	refreshCommitResolvability             *observation.Operation
	repoName                               *observation.Operation
	requeue                                *observation.Operation
	requeueIndex                           *observation.Operation
	softDeleteOldUploads                   *observation.Operation
	staleSourcedCommits                    *observation.Operation
	updateCommitedAt                       *observation.Operation
	updateIndexConfigurationByRepositoryID *observation.Operation
	updatePackageReferences                *observation.Operation
	updatePackages                         *observation.Operation

	writeVisibleUploads        *observation.Operation
	persistNearestUploads      *observation.Operation
	persistNearestUploadsLinks *observation.Operation
	persistUploadsVisibleAtTip *observation.Operation
}

func newOperations(observationContext *observation.Context) *operations {
	metrics := metrics.NewOperationMetrics(
		observationContext.Registerer,
		"codeintel_dbstore",
		metrics.WithLabels("op"),
		metrics.WithCountHelp("Total number of method invocations."),
	)

	op := func(name string) *observation.Operation {
		return observationContext.Operation(observation.Op{
			Name:         fmt.Sprintf("codeintel.dbstore.%s", name),
			MetricLabels: []string{name},
			Metrics:      metrics,
		})
	}

	// suboperations do not have their own metrics but do have their
	// own opentracing spans. This allows us to more granularly track
	// the latency for parts of a request without noising up Prometheus.
	subOp := func(name string) *observation.Operation {
		return observationContext.Operation(observation.Op{
			Name: fmt.Sprintf("codeintel.dbstore.%s", name),
		})
	}

	return &operations{
		addUploadPart:                          op("AddUploadPart"),
		calculateVisibleUploads:                op("CalculateVisibleUploads"),
		commitGraphMetadata:                    op("CommitGraphMetadata"),
		definitionDumps:                        op("DefinitionDumps"),
		deleteIndexByID:                        op("DeleteIndexByID"),
		deleteIndexesWithoutRepository:         op("DeleteIndexesWithoutRepository"),
		deleteOldIndexes:                       op("DeleteOldIndexes"),
		deleteOverlappingDumps:                 op("DeleteOverlappingDumps"),
		deleteUploadByID:                       op("DeleteUploadByID"),
		deleteUploadsStuckUploading:            op("DeleteUploadsStuckUploading"),
		deleteUploadsWithoutRepository:         op("DeleteUploadsWithoutRepository"),
		dequeue:                                op("Dequeue"),
		dequeueIndex:                           op("DequeueIndex"),
		dirtyRepositories:                      op("DirtyRepositories"),
		findClosestDumps:                       op("FindClosestDumps"),
		findClosestDumpsFromGraphFragment:      op("FindClosestDumpsFromGraphFragment"),
		getAutoindexDisabledRepositories:       op("getAutoindexDisabledRepositories"),
		getDumpsByIDs:                          op("GetDumpsByIDs"),
		getIndexByID:                           op("GetIndexByID"),
		getIndexConfigurationByRepositoryID:    op("GetIndexConfigurationByRepositoryID"),
		getIndexes:                             op("GetIndexes"),
		getIndexesByIDs:                        op("GetIndexesByIDs"),
		getOldestCommitDate:                    op("GetOldestCommitDate"),
		getRepositoriesWithIndexConfiguration:  op("GetRepositoriesWithIndexConfiguration"),
		getUploadByID:                          op("GetUploadByID"),
		getUploads:                             op("GetUploads"),
		getUploadsByIDs:                        op("GetUploadsByIDs"),
		hardDeleteUploadByID:                   op("HardDeleteUploadByID"),
		hasCommit:                              op("HasCommit"),
		hasRepository:                          op("HasRepository"),
		indexQueueSize:                         op("IndexQueueSize"),
		insertDependencyIndexingJob:            op("InsertDependencyIndexingJob"),
		insertDependencyRepoAddingJob:          op("InsertDependencyRepoAddingJob"),
		insertIndex:                            op("InsertIndex"),
		insertUpload:                           op("InsertUpload"),
		isQueued:                               op("IsQueued"),
		markComplete:                           op("MarkComplete"),
		markErrored:                            op("MarkErrored"),
		markFailed:                             op("MarkFailed"),
		markIndexComplete:                      op("MarkIndexComplete"),
		markIndexErrored:                       op("MarkIndexErrored"),
		markQueued:                             op("MarkQueued"),
		markRepositoryAsDirty:                  op("MarkRepositoryAsDirty"),
		queueSize:                              op("QueueSize"),
		referenceIDsAndFilters:                 op("ReferenceIDsAndFilters"),
		referencesForUpload:                    op("ReferencesForUpload"),
		refreshCommitResolvability:             op("RefreshCommitResolvability"),
		repoName:                               op("RepoName"),
		requeue:                                op("Requeue"),
		requeueIndex:                           op("RequeueIndex"),
		softDeleteOldUploads:                   op("SoftDeleteOldUploads"),
		staleSourcedCommits:                    op("StaleSourcedCommits"),
		updateCommitedAt:                       op("UpdateCommitedAt"),
		updateIndexConfigurationByRepositoryID: op("UpdateIndexConfigurationByRepositoryID"),
		updatePackageReferences:                op("UpdatePackageReferences"),
		updatePackages:                         op("UpdatePackages"),

		writeVisibleUploads:        subOp("writeVisibleUploads"),
		persistNearestUploads:      subOp("persistNearestUploads"),
		persistNearestUploadsLinks: subOp("persistNearestUploadsLinks"),
		persistUploadsVisibleAtTip: subOp("persistUploadsVisibleAtTip"),
	}
}
