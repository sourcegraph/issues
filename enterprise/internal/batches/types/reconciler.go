package types

// ReconcilerOperation is an enum to distinguish between different reconciler operations.
type ReconcilerOperation string

const (
	ReconcilerOperationPush         ReconcilerOperation = "PUSH"
	ReconcilerOperationUpdate       ReconcilerOperation = "UPDATE"
	ReconcilerOperationUndraft      ReconcilerOperation = "UNDRAFT"
	ReconcilerOperationRedraft      ReconcilerOperation = "REDRAFT"
	ReconcilerOperationPublish      ReconcilerOperation = "PUBLISH"
	ReconcilerOperationPublishDraft ReconcilerOperation = "PUBLISH_DRAFT"
	ReconcilerOperationSync         ReconcilerOperation = "SYNC"
	ReconcilerOperationImport       ReconcilerOperation = "IMPORT"
	ReconcilerOperationClose        ReconcilerOperation = "CLOSE"
	ReconcilerOperationReopen       ReconcilerOperation = "REOPEN"
	ReconcilerOperationSleep        ReconcilerOperation = "SLEEP"
	ReconcilerOperationDetach       ReconcilerOperation = "DETACH"
	ReconcilerOperationArchive      ReconcilerOperation = "ARCHIVE"
)

// Valid returns true if the given ReconcilerOperation is valid.
func (r ReconcilerOperation) Valid() bool {
	switch r {
	case ReconcilerOperationPush,
		ReconcilerOperationUpdate,
		ReconcilerOperationUndraft,
		ReconcilerOperationRedraft,
		ReconcilerOperationPublish,
		ReconcilerOperationPublishDraft,
		ReconcilerOperationSync,
		ReconcilerOperationImport,
		ReconcilerOperationClose,
		ReconcilerOperationReopen,
		ReconcilerOperationSleep,
		ReconcilerOperationDetach,
		ReconcilerOperationArchive:
		return true
	default:
		return false
	}
}
