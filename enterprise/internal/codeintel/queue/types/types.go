package types

// DequeueRequest is sent to the index manager API to lock and retrieve a
// queued index record for processing.
type DequeueRequest struct {
	// IndexerName is a unique name identifying the requesting indexer.
	IndexerName string `json:"indexerName"`
}

// SetLogRequest is sent to the index manager API to set the log contents
// of an index job that is currently being processed.
type SetLogRequest struct {
	// IndexerName is a unique name identifying the requesting indexer.
	IndexerName string `json:"indexerName"`

	// IndexID is the identifier of the index record is being processed.
	IndexID int `json:"indexId"`

	// Payload the content of the index job logs.
	Contents string `json:"payload"`
}

// CompleteRequest is sent to the index manager API once an index request
// has finished. This request is used both on success and failure.
type CompleteRequest struct {
	// IndexerName is a unique name identifying the requesting indexer.
	IndexerName string `json:"indexerName"`

	// IndexID is the identifier of the index record that was processed.
	IndexID int `json:"indexId"`

	// ErrorMessage a description of the job failure, if indexing did not succeed.
	ErrorMessage string `json:"errorMessage"`
}

// HeartbeatRequest is sent to the index manager API periodically to keep
// the transactions held for a particular indexer alive.
type HeartbeatRequest struct {
	// IndexerName is a unique name identifying the requesting indexer.
	IndexerName string `json:"indexerName"`

	// IndexIDs is a list of index identifiers which are currently being processed
	// by the indexer.
	IndexIDs []int `json:"indexIds"`
}
