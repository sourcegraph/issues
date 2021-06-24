package httpapi

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	store "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/dbstore"
	uploadstoremocks "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/uploadstore/mocks"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtesting"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

func TestMain(m *testing.M) {
	flag.Parse()
	if !testing.Verbose() {
		log15.Root().SetHandler(log15.DiscardHandler())
	}
	os.Exit(m.Run())
}

const testCommit = "deadbeef01deadbeef02deadbeef03deadbeef04"

func TestHandleEnqueueSinglePayload(t *testing.T) {
	setupRepoMocks(t)

	mockDBStore := NewMockDBStore()
	mockUploadStore := uploadstoremocks.NewMockStore()

	mockDBStore.TransactFunc.SetDefaultReturn(mockDBStore, nil)
	mockDBStore.HandleFunc.SetDefaultReturn(basestore.NewHandleWithDB(&dbtesting.MockDB{}, sql.TxOptions{}))
	mockDBStore.DoneFunc.SetDefaultHook(func(err error) error { return err })
	mockDBStore.InsertUploadFunc.SetDefaultReturn(42, nil)

	testURL, err := url.Parse("http://test.com/upload")
	if err != nil {
		t.Fatalf("unexpected error constructing url: %s", err)
	}
	testURL.RawQuery = (url.Values{
		"commit":      []string{testCommit},
		"root":        []string{"proj/"},
		"repository":  []string{"github.com/test/test"},
		"indexerName": []string{"lsif-go"},
	}).Encode()

	var expectedContents []byte
	for i := 0; i < 20000; i++ {
		expectedContents = append(expectedContents, byte(i))
	}

	w := httptest.NewRecorder()
	r, err := http.NewRequest("POST", testURL.String(), bytes.NewReader(expectedContents))
	if err != nil {
		t.Fatalf("unexpected error constructing request: %s", err)
	}

	h := &UploadHandler{
		dbStore:     mockDBStore,
		uploadStore: mockUploadStore,
	}
	h.handleEnqueue(w, r)

	if w.Code != http.StatusAccepted {
		t.Errorf("unexpected status code. want=%d have=%d", http.StatusAccepted, w.Code)
	}
	if diff := cmp.Diff([]byte(`{"id":"42"}`), w.Body.Bytes()); diff != "" {
		t.Errorf("unexpected response payload (-want +got):\n%s", diff)
	}

	if len(mockDBStore.InsertUploadFunc.History()) != 1 {
		t.Errorf("unexpected number of InsertUpload calls. want=%d have=%d", 1, len(mockDBStore.InsertUploadFunc.History()))
	} else {
		call := mockDBStore.InsertUploadFunc.History()[0]
		if call.Arg1.Commit != testCommit {
			t.Errorf("unexpected commit. want=%q have=%q", testCommit, call.Arg1.Commit)
		}
		if call.Arg1.Root != "proj/" {
			t.Errorf("unexpected root. want=%q have=%q", "proj/", call.Arg1.Root)
		}
		if call.Arg1.RepositoryID != 50 {
			t.Errorf("unexpected repository id. want=%d have=%d", 50, call.Arg1.RepositoryID)
		}
		if call.Arg1.Indexer != "lsif-go" {
			t.Errorf("unexpected indexer name. want=%q have=%q", "lsif-go", call.Arg1.Indexer)
		}
	}

	if len(mockUploadStore.UploadFunc.History()) != 1 {
		t.Errorf("unexpected number of SendUpload calls. want=%d have=%d", 1, len(mockUploadStore.UploadFunc.History()))
	} else {
		call := mockUploadStore.UploadFunc.History()[0]
		if call.Arg1 != "upload-42.lsif.gz" {
			t.Errorf("unexpected bundle id. want=%s have=%s", "upload-42.lsif.gz", call.Arg1)
		}

		contents, err := io.ReadAll(call.Arg2)
		if err != nil {
			t.Fatalf("unexpected error reading payload: %s", err)
		}

		if diff := cmp.Diff(expectedContents, contents); diff != "" {
			t.Errorf("unexpected file contents (-want +got):\n%s", diff)
		}
	}
}

func TestHandleEnqueueSinglePayloadNoIndexerName(t *testing.T) {
	setupRepoMocks(t)

	mockDBStore := NewMockDBStore()
	mockUploadStore := uploadstoremocks.NewMockStore()

	mockDBStore.TransactFunc.SetDefaultReturn(mockDBStore, nil)
	mockDBStore.HandleFunc.SetDefaultReturn(basestore.NewHandleWithDB(&dbtesting.MockDB{}, sql.TxOptions{}))
	mockDBStore.DoneFunc.SetDefaultHook(func(err error) error { return err })
	mockDBStore.InsertUploadFunc.SetDefaultReturn(42, nil)

	testURL, err := url.Parse("http://test.com/upload")
	if err != nil {
		t.Fatalf("unexpected error constructing url: %s", err)
	}
	testURL.RawQuery = (url.Values{
		"commit":     []string{testCommit},
		"root":       []string{"proj/"},
		"repository": []string{"github.com/test/test"},
	}).Encode()

	var lines []string
	lines = append(lines, `{"label": "metaData", "toolInfo": {"name": "lsif-go"}}`)
	for i := 0; i < 20000; i++ {
		lines = append(lines, `{"id": "a", "type": "edge", "label": "textDocument/references", "outV": "b", "inV": "c"}`)
	}

	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	_, _ = io.Copy(gzipWriter, bytes.NewReader([]byte(strings.Join(lines, "\n"))))
	gzipWriter.Close()
	expectedContents := buf.Bytes()

	w := httptest.NewRecorder()
	r, err := http.NewRequest("POST", testURL.String(), bytes.NewReader(expectedContents))
	if err != nil {
		t.Fatalf("unexpected error constructing request: %s", err)
	}

	h := &UploadHandler{
		dbStore:     mockDBStore,
		uploadStore: mockUploadStore,
	}
	h.handleEnqueue(w, r)

	if w.Code != http.StatusAccepted {
		t.Errorf("unexpected status code. want=%d have=%d", http.StatusAccepted, w.Code)
	}

	if len(mockUploadStore.UploadFunc.History()) != 1 {
		t.Errorf("unexpected number of Upload calls. want=%d have=%d", 1, len(mockUploadStore.UploadFunc.History()))
	} else {
		call := mockUploadStore.UploadFunc.History()[0]
		if call.Arg1 != "upload-42.lsif.gz" {
			t.Errorf("unexpected bundle id. want=%s have=%s", "upload-42.lsif.gz", call.Arg1)
		}

		contents, err := io.ReadAll(call.Arg2)
		if err != nil {
			t.Fatalf("unexpected error reading payload: %s", err)
		}

		if diff := cmp.Diff(expectedContents, contents); diff != "" {
			t.Errorf("unexpected file contents (-want +got):\n%s", diff)
		}
	}
}

func TestHandleEnqueueMultipartSetup(t *testing.T) {
	setupRepoMocks(t)

	mockDBStore := NewMockDBStore()
	mockUploadStore := uploadstoremocks.NewMockStore()

	mockDBStore.TransactFunc.SetDefaultReturn(mockDBStore, nil)
	mockDBStore.HandleFunc.SetDefaultReturn(basestore.NewHandleWithDB(&dbtesting.MockDB{}, sql.TxOptions{}))
	mockDBStore.DoneFunc.SetDefaultHook(func(err error) error { return err })
	mockDBStore.InsertUploadFunc.SetDefaultReturn(42, nil)

	testURL, err := url.Parse("http://test.com/upload")
	if err != nil {
		t.Fatalf("unexpected error constructing url: %s", err)
	}
	testURL.RawQuery = (url.Values{
		"commit":      []string{testCommit},
		"root":        []string{"proj/"},
		"repository":  []string{"github.com/test/test"},
		"indexerName": []string{"lsif-go"},
		"multiPart":   []string{"true"},
		"numParts":    []string{"3"},
	}).Encode()

	w := httptest.NewRecorder()
	r, err := http.NewRequest("POST", testURL.String(), nil)
	if err != nil {
		t.Fatalf("unexpected error constructing request: %s", err)
	}

	h := &UploadHandler{
		dbStore:     mockDBStore,
		uploadStore: mockUploadStore,
	}
	h.handleEnqueue(w, r)

	if w.Code != http.StatusAccepted {
		t.Errorf("unexpected status code. want=%d have=%d", http.StatusAccepted, w.Code)
	}
	if diff := cmp.Diff([]byte(`{"id":"42"}`), w.Body.Bytes()); diff != "" {
		t.Errorf("unexpected response payload (-want +got):\n%s", diff)
	}

	if len(mockDBStore.InsertUploadFunc.History()) != 1 {
		t.Errorf("unexpected number of InsertUpload calls. want=%d have=%d", 1, len(mockDBStore.InsertUploadFunc.History()))
	} else {
		call := mockDBStore.InsertUploadFunc.History()[0]
		if call.Arg1.Commit != testCommit {
			t.Errorf("unexpected commit. want=%q have=%q", testCommit, call.Arg1.Commit)
		}
		if call.Arg1.Root != "proj/" {
			t.Errorf("unexpected root. want=%q have=%q", "proj/", call.Arg1.Root)
		}
		if call.Arg1.RepositoryID != 50 {
			t.Errorf("unexpected repository id. want=%d have=%d", 50, call.Arg1.RepositoryID)
		}
		if call.Arg1.Indexer != "lsif-go" {
			t.Errorf("unexpected indexer name. want=%q have=%q", "lsif-go", call.Arg1.Indexer)
		}
	}
}

func TestHandleEnqueueMultipartUpload(t *testing.T) {
	mockDBStore := NewMockDBStore()
	mockUploadStore := uploadstoremocks.NewMockStore()

	upload := store.Upload{
		ID:            42,
		NumParts:      5,
		UploadedParts: []int{0, 1, 2, 3, 4},
	}

	mockDBStore.TransactFunc.SetDefaultReturn(mockDBStore, nil)
	mockDBStore.HandleFunc.SetDefaultReturn(basestore.NewHandleWithDB(&dbtesting.MockDB{}, sql.TxOptions{}))
	mockDBStore.DoneFunc.SetDefaultHook(func(err error) error { return err })
	mockDBStore.GetUploadByIDFunc.SetDefaultReturn(upload, true, nil)

	testURL, err := url.Parse("http://test.com/upload")
	if err != nil {
		t.Fatalf("unexpected error constructing url: %s", err)
	}
	testURL.RawQuery = (url.Values{
		"uploadId": []string{"42"},
		"index":    []string{"3"},
	}).Encode()

	var expectedContents []byte
	for i := 0; i < 20000; i++ {
		expectedContents = append(expectedContents, byte(i))
	}

	w := httptest.NewRecorder()
	r, err := http.NewRequest("POST", testURL.String(), bytes.NewReader(expectedContents))
	if err != nil {
		t.Fatalf("unexpected error constructing request: %s", err)
	}

	h := &UploadHandler{
		dbStore:     mockDBStore,
		uploadStore: mockUploadStore,
	}
	h.handleEnqueue(w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("unexpected status code. want=%d have=%d", http.StatusNoContent, w.Code)
	}

	if len(mockDBStore.AddUploadPartFunc.History()) != 1 {
		t.Errorf("unexpected number of AddUploadPart calls. want=%d have=%d", 1, len(mockDBStore.AddUploadPartFunc.History()))
	} else {
		call := mockDBStore.AddUploadPartFunc.History()[0]
		if call.Arg1 != 42 {
			t.Errorf("unexpected commit. want=%q have=%q", 42, call.Arg1)
		}
		if call.Arg2 != 3 {
			t.Errorf("unexpected root. want=%q have=%q", 3, call.Arg2)
		}
	}

	if len(mockUploadStore.UploadFunc.History()) != 1 {
		t.Errorf("unexpected number of Upload calls. want=%d have=%d", 1, len(mockUploadStore.UploadFunc.History()))
	} else {
		call := mockUploadStore.UploadFunc.History()[0]
		if call.Arg1 != "upload-42.3.lsif.gz" {
			t.Errorf("unexpected bundle id. want=%s have=%s", "upload-42.3.lsif.gz", call.Arg1)
		}

		contents, err := io.ReadAll(call.Arg2)
		if err != nil {
			t.Fatalf("unexpected error reading payload: %s", err)
		}

		if diff := cmp.Diff(expectedContents, contents); diff != "" {
			t.Errorf("unexpected file contents (-want +got):\n%s", diff)
		}
	}
}

func TestHandleEnqueueMultipartFinalize(t *testing.T) {
	mockDBStore := NewMockDBStore()
	mockUploadStore := uploadstoremocks.NewMockStore()

	upload := store.Upload{
		ID:            42,
		NumParts:      5,
		UploadedParts: []int{0, 1, 2, 3, 4},
	}
	mockDBStore.TransactFunc.SetDefaultReturn(mockDBStore, nil)
	mockDBStore.HandleFunc.SetDefaultReturn(basestore.NewHandleWithDB(&dbtesting.MockDB{}, sql.TxOptions{}))
	mockDBStore.DoneFunc.SetDefaultHook(func(err error) error { return err })
	mockDBStore.GetUploadByIDFunc.SetDefaultReturn(upload, true, nil)

	testURL, err := url.Parse("http://test.com/upload")
	if err != nil {
		t.Fatalf("unexpected error constructing url: %s", err)
	}
	testURL.RawQuery = (url.Values{
		"uploadId": []string{"42"},
		"done":     []string{"true"},
	}).Encode()

	w := httptest.NewRecorder()
	r, err := http.NewRequest("POST", testURL.String(), nil)
	if err != nil {
		t.Fatalf("unexpected error constructing request: %s", err)
	}

	h := &UploadHandler{
		dbStore:     mockDBStore,
		uploadStore: mockUploadStore,
	}
	h.handleEnqueue(w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("unexpected status code. want=%d have=%d", http.StatusNoContent, w.Code)
	}

	if len(mockDBStore.MarkQueuedFunc.History()) != 1 {
		t.Errorf("unexpected number of MarkQueued calls. want=%d have=%d", 1, len(mockDBStore.MarkQueuedFunc.History()))
	} else if call := mockDBStore.MarkQueuedFunc.History()[0]; call.Arg1 != 42 {
		t.Errorf("unexpected upload id. want=%d have=%d", 42, call.Arg1)
	}

	if len(mockUploadStore.ComposeFunc.History()) != 1 {
		t.Errorf("unexpected number of Compose calls. want=%d have=%d", 1, len(mockUploadStore.ComposeFunc.History()))
	} else {
		call := mockUploadStore.ComposeFunc.History()[0]

		if call.Arg1 != "upload-42.lsif.gz" {
			t.Errorf("unexpected bundle id. want=%s have=%s", "upload-42.lsif.gz", call.Arg1)
		}

		expectedFilenames := []string{
			"upload-42.0.lsif.gz",
			"upload-42.1.lsif.gz",
			"upload-42.2.lsif.gz",
			"upload-42.3.lsif.gz",
			"upload-42.4.lsif.gz",
		}
		if diff := cmp.Diff(expectedFilenames, call.Arg2); diff != "" {
			t.Errorf("unexpected source filenames (-want +got):\n%s", diff)
		}
	}
}

func TestHandleEnqueueMultipartFinalizeIncompleteUpload(t *testing.T) {
	mockDBStore := NewMockDBStore()
	mockUploadStore := uploadstoremocks.NewMockStore()

	upload := store.Upload{
		ID:            42,
		NumParts:      5,
		UploadedParts: []int{0, 1, 3, 4},
	}
	mockDBStore.GetUploadByIDFunc.SetDefaultReturn(upload, true, nil)

	testURL, err := url.Parse("http://test.com/upload")
	if err != nil {
		t.Fatalf("unexpected error constructing url: %s", err)
	}
	testURL.RawQuery = (url.Values{
		"uploadId": []string{"42"},
		"done":     []string{"true"},
	}).Encode()

	w := httptest.NewRecorder()
	r, err := http.NewRequest("POST", testURL.String(), nil)
	if err != nil {
		t.Fatalf("unexpected error constructing request: %s", err)
	}

	h := &UploadHandler{
		dbStore:     mockDBStore,
		uploadStore: mockUploadStore,
	}
	h.handleEnqueue(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("unexpected status code. want=%d have=%d", http.StatusBadRequest, w.Code)
	}
}

func setupRepoMocks(t testing.TB) {
	t.Cleanup(func() {
		backend.Mocks.Repos.GetByName = nil
		backend.Mocks.Repos.ResolveRev = nil
	})

	backend.Mocks.Repos.GetByName = func(ctx context.Context, name api.RepoName) (*types.Repo, error) {
		if name != "github.com/test/test" {
			t.Errorf("unexpected repository name. want=%s have=%s", "github.com/test/test", name)
		}
		return &types.Repo{ID: 50}, nil
	}

	backend.Mocks.Repos.ResolveRev = func(ctx context.Context, repo *types.Repo, rev string) (api.CommitID, error) {
		if rev != testCommit {
			t.Errorf("unexpected commit. want=%s have=%s", testCommit, rev)
		}
		return "", nil
	}
}
