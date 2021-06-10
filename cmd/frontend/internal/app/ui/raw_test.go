package ui

import (
	"io"
	"io/fs"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtesting"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
	"github.com/sourcegraph/sourcegraph/internal/vcs/util"
)

// initHTTPTestGitServer instantiates an httptest.Server to make it return an HTTP response as set
// by httpStatusCode and a body as set by resp. It also ensures that the server is closed during
// test cleanup, thus ensuring that the caller does not have to remember to close the server.
//
// Finally, initHTTPTestGitServer patches the gitserver.DefaultClient.Addrs to the URL of the test
// HTTP server, so that API calls to the gitserver are received by the test HTTP server.
//
// TL;DR: This function helps us to mock the gitserver without having to define mock functions for
// each of the gitserver client methods.
func initHTTPTestGitServer(t *testing.T, httpStatusCode int, resp string) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Trailer", "X-Exec-Error")
		w.Header().Add("Trailer", "X-Exec-Exit-Status")
		w.Header().Add("Trailer", "X-Exec-Stderr")
		w.Header().Set("X-Exec-Error", "")
		w.Header().Set("X-Exec-Exit-Status", "0")
		w.Header().Set("X-Exec-Stderr", "")
		w.WriteHeader(httpStatusCode)
		_, err := w.Write([]byte(resp))
		if err != nil {
			t.Fatalf("Failed to write to httptest server: %v", err)
		}
	}))

	t.Cleanup(func() { s.Close() })

	// Strip the protocol from the URI while patching the gitserver client's
	// addresses, since the gitserver implementation does not want the protocol in
	// the address.
	gitserver.DefaultClient.Addrs = func() []string {
		return []string{strings.TrimPrefix(s.URL, "http://")}
	}
}

func Test_serveRawWithHTTPRequestMethodHEAD(t *testing.T) {
	// mockNewCommon ensures that we do not need the repo-updater running for this unit test.
	mockNewCommon = func(w http.ResponseWriter, r *http.Request, title string, serveError serveErrorHandler) (*Common, error) {
		return &Common{
			Repo: &types.Repo{
				Name: "test",
			},
			CommitID: api.CommitID("12345"),
		}, nil
	}
	defer func() {
		mockNewCommon = nil
	}()

	t.Run("success response for HEAD request", func(t *testing.T) {
		// httptest server will return a 200 OK, so gitserver.DefaultClient.RepoInfo will not return
		// an error.
		initHTTPTestGitServer(t, http.StatusOK, "{}")

		req := httptest.NewRequest("HEAD", "/github.com/sourcegraph/sourcegraph/-/raw", nil)
		w := httptest.NewRecorder()

		err := serveRaw(&dbtesting.MockDB{})(w, req)
		if err != nil {
			t.Fatalf("Failed to invoke serveRaw: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Fatalf("Want %d but got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("failure response for HEAD request", func(t *testing.T) {
		// httptest server will return a 404 Not Found, so gitserver.DefaultClient.RepoInfo will
		// return an error.
		initHTTPTestGitServer(t, http.StatusNotFound, "{}")

		req := httptest.NewRequest("HEAD", "/github.com/sourcegraph/sourcegraph/-/raw", nil)
		w := httptest.NewRecorder()

		err := serveRaw(&dbtesting.MockDB{})(w, req)
		if err == nil {
			t.Fatal("Want error but got nil")
		}

		if w.Code != http.StatusNotFound {
			t.Fatalf("Want %d but got %d", http.StatusNotFound, w.Code)
		}
	})
}

func Test_serveRawWithContentArchive(t *testing.T) {
	mockNewCommon = func(w http.ResponseWriter, r *http.Request, title string, serveError serveErrorHandler) (*Common, error) {
		return &Common{
			Repo: &types.Repo{
				Name: "test",
			},
			CommitID: api.CommitID("12345"),
		}, nil
	}
	defer func() {
		mockNewCommon = nil
	}()

	mockGitServerResponse := "this is a gitserver archive response"

	t.Run("success response for format=zip", func(t *testing.T) {
		// httptest server will return a 200 OK, so gitserver.DefaultClient.RepoInfo will not return an error.

		initHTTPTestGitServer(t, http.StatusOK, mockGitServerResponse)

		req := httptest.NewRequest("GET", "/github.com/sourcegraph/sourcegraph/-/raw?format=zip", nil)
		w := httptest.NewRecorder()

		err := serveRaw(&dbtesting.MockDB{})(w, req)
		if err != nil {
			t.Fatalf("Failed to invoke serveRaw: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Fatalf("Want %d but got %d", http.StatusOK, w.Code)
		}

		expectedHeaders := map[string]string{
			"X-Content-Type-Options": "nosniff",
			"Content-Type":           "application/zip",
			"Content-Disposition":    mime.FormatMediaType("Attachment", map[string]string{"filename": "test.zip"}),
		}

		if len(w.Header()) != len(expectedHeaders) {
			t.Errorf("Want %d headers but got %d headers", len(w.Header()), len(expectedHeaders))
		}

		for k, v := range expectedHeaders {
			if h := w.Header().Get(k); h != v {
				t.Errorf("Expected header %q to have value %q but got %q", k, v, h)
			}
		}

		body := string(w.Body.Bytes())
		if body != mockGitServerResponse {
			t.Errorf("Want %q in body, but got %q", mockGitServerResponse, body)
		}
	})

	t.Run("success response for format=tar", func(t *testing.T) {
		// httptest server will return a 200 OK, so gitserver.DefaultClient.RepoInfo will not return an error.

		initHTTPTestGitServer(t, http.StatusOK, mockGitServerResponse)

		req := httptest.NewRequest("GET", "/github.com/sourcegraph/sourcegraph/-/raw?format=tar", nil)
		w := httptest.NewRecorder()

		err := serveRaw(&dbtesting.MockDB{})(w, req)
		if err != nil {
			t.Fatalf("Failed to invoke serveRaw: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Fatalf("Want %d but got %d", http.StatusOK, w.Code)
		}

		expectedHeaders := map[string]string{
			"X-Content-Type-Options": "nosniff",
			"Content-Type":           "application/x-tar",
			"Content-Disposition":    mime.FormatMediaType("Attachment", map[string]string{"filename": "test.tar"}),
		}

		if len(w.Header()) != len(expectedHeaders) {
			t.Errorf("Want %d headers but got %d headers", len(w.Header()), len(expectedHeaders))
		}

		for k, v := range expectedHeaders {
			if h := w.Header().Get(k); h != v {
				t.Errorf("Expected header %q to have value %q but got %q", k, v, h)
			}
		}

		body := string(w.Body.Bytes())
		if body != mockGitServerResponse {
			t.Errorf("Want %q in body, but got %q", mockGitServerResponse, body)
		}
	})

}

func Test_serveRawWithContentTypePlain(t *testing.T) {
	mockNewCommon = func(w http.ResponseWriter, r *http.Request, title string, serveError serveErrorHandler) (*Common, error) {
		return &Common{
			Repo: &types.Repo{
				Name: "test",
			},
			CommitID: api.CommitID("12345"),
		}, nil
	}
	defer func() {
		mockNewCommon = nil
	}()

	assertHeaders := func(w http.ResponseWriter) {
		t.Helper()

		expectedHeaders := map[string]string{
			"X-Content-Type-Options": "nosniff",
			"Content-Type":           "text/plain; charset=utf-8",
		}

		if len(w.Header()) != len(expectedHeaders) {
			t.Errorf("Want %d headers but got %d headers", len(w.Header()), len(expectedHeaders))
		}

		for k, v := range expectedHeaders {
			if h := w.Header().Get(k); h != v {
				t.Errorf("Want header %q to have value %q but got %q", k, v, h)
			}
		}
	}

	t.Run("404 Not Found for non existent directory", func(t *testing.T) {
		// httptest server will return a 200 OK, so gitserver.DefaultClient.RepoInfo will not return an error.
		initHTTPTestGitServer(t, http.StatusOK, "{}")

		git.Mocks.Stat = func(commit api.CommitID, name string) (fs.FileInfo, error) {
			return &util.FileInfo{}, os.ErrNotExist
		}
		defer git.ResetMocks()

		req := httptest.NewRequest("GET", "/github.com/sourcegraph/sourcegraph/-/raw", nil)
		w := httptest.NewRecorder()

		err := serveRaw(&dbtesting.MockDB{})(w, req)
		if err != nil {
			t.Fatalf("Failed to invoke serveRaw: %v", err)
		}

		if w.Code != http.StatusNotFound {
			t.Fatalf("Want %d but got %d", http.StatusOK, w.Code)
		}

		assertHeaders(w)
	})

	t.Run("success response for existing directory", func(t *testing.T) {
		// httptest server will return a 200 OK, so gitserver.DefaultClient.RepoInfo will not return an error.
		initHTTPTestGitServer(t, http.StatusOK, "{}")

		git.Mocks.Stat = func(commit api.CommitID, name string) (fs.FileInfo, error) {
			return &util.FileInfo{Mode_: os.ModeDir}, nil
		}

		git.Mocks.ReadDir = func(commit api.CommitID, name string, recurse bool) ([]fs.FileInfo, error) {
			return []fs.FileInfo{
				&util.FileInfo{Name_: "test/a", Mode_: os.ModeDir},
				&util.FileInfo{Name_: "test/b", Mode_: os.ModeDir},
				&util.FileInfo{Name_: "c.go", Mode_: 0},
			}, nil
		}

		defer git.ResetMocks()

		req := httptest.NewRequest("GET", "/github.com/sourcegraph/sourcegraph/-/raw", nil)
		w := httptest.NewRecorder()

		err := serveRaw(&dbtesting.MockDB{})(w, req)
		if err != nil {
			t.Fatalf("Failed to invoke serveRaw: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Fatalf("Want %d but got %d", http.StatusOK, w.Code)
		}

		assertHeaders(w)

		want := `a/
b/
c.go`
		body := string(w.Body.Bytes())
		if body != want {
			t.Errorf("Want %q in body, but got %q", want, body)
		}
	})

	t.Run("success response for existing file", func(t *testing.T) {
		// httptest server will return a 200 OK, so gitserver.DefaultClient.RepoInfo will not return an error.
		initHTTPTestGitServer(t, http.StatusOK, "{}")

		git.Mocks.Stat = func(commit api.CommitID, name string) (fs.FileInfo, error) {
			return &util.FileInfo{Mode_: 0}, nil
		}

		git.Mocks.NewFileReader = func(commit api.CommitID, name string) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader("this is a test file")), nil
		}

		defer git.ResetMocks()

		req := httptest.NewRequest("GET", "/github.com/sourcegraph/sourcegraph/-/raw", nil)
		w := httptest.NewRecorder()

		err := serveRaw(&dbtesting.MockDB{})(w, req)
		if err != nil {
			t.Fatalf("Failed to invoke serveRaw: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Fatalf("Want %d but got %d", http.StatusOK, w.Code)
		}

		assertHeaders(w)

		want := "this is a test file"

		body := string(w.Body.Bytes())
		if body != want {
			t.Errorf("Want %q in body, but got %q", want, body)
		}
	})

	// Ensure that anything apart from tar/zip/text is still handled with a text/plain content type.
	t.Run("success response for existing file with format=exe", func(t *testing.T) {
		// httptest server will return a 200 OK, so gitserver.DefaultClient.RepoInfo will not return an error.
		initHTTPTestGitServer(t, http.StatusOK, "{}")

		git.Mocks.Stat = func(commit api.CommitID, name string) (fs.FileInfo, error) {
			return &util.FileInfo{Mode_: 0}, nil
		}

		git.Mocks.NewFileReader = func(commit api.CommitID, name string) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader("this is a test file")), nil
		}

		defer git.ResetMocks()

		req := httptest.NewRequest("GET", "/github.com/sourcegraph/sourcegraph/-/raw?format=exe", nil)
		w := httptest.NewRecorder()

		err := serveRaw(&dbtesting.MockDB{})(w, req)
		if err != nil {
			t.Fatalf("Failed to invoke serveRaw: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Fatalf("Want %d but got %d", http.StatusOK, w.Code)
		}

		assertHeaders(w)

		want := "this is a test file"

		body := string(w.Body.Bytes())
		if body != want {
			t.Errorf("Want %q in body, but got %q", want, body)
		}
	})
}
