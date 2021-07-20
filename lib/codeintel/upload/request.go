package upload

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/cockroachdb/errors"
)

type uploadRequestOptions struct {
	UploadOptions

	Payload   io.Reader // Request payload
	Target    *int      // Pointer to upload id decoded from resp
	MultiPart bool      // Whether the request is a multipart init
	NumParts  int       // The number of upload parts
	UploadID  int       // The multipart upload ID
	Index     int       // The index part being uploaded
	Done      bool      // Whether the request is a multipart finalize
}

// ErrUnauthorized occurs when the upload endpoint returns a 401 response.
var ErrUnauthorized = errors.New("unauthorized upload")

// performUploadRequest performs an HTTP POST to the upload endpoint. The query string of the request
// is constructed from the given request options and the body of the request is the unmodified reader.
// If target is a non-nil pointer, it will be assigned the value of the upload identifier present
// in the response body. This function returns an error as well as a boolean flag indicating if the
// function can be retried.
func performUploadRequest(httpClient Client, opts uploadRequestOptions) (bool, error) {
	req, err := makeUploadRequest(opts)
	if err != nil {
		return false, err
	}

	resp, body, err := performRequest(req, httpClient, opts.OutputOptions.Logger)
	if err != nil {
		return false, err
	}

	return decodeUploadPayload(resp, body, opts.Target)
}

// makeUploadRequest creates an HTTP request to the upload endpoint described by the given arguments.
func makeUploadRequest(opts uploadRequestOptions) (*http.Request, error) {
	url, err := makeUploadURL(opts)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url.String(), opts.Payload)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-ndjson+lsif")
	if opts.SourcegraphInstanceOptions.AccessToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", opts.SourcegraphInstanceOptions.AccessToken))
	}

	for k, v := range opts.SourcegraphInstanceOptions.AdditionalHeaders {
		req.Header.Set(k, v)
	}

	return req, nil
}

// performRequest performs an HTTP request and returns the HTTP response as well as the entire
// body as a byte slice. If a logger is supplied, the request, response, and response body will
// be logged.
func performRequest(req *http.Request, httpClient Client, logger RequestLogger) (*http.Response, []byte, error) {
	started := time.Now()
	if logger != nil {
		logger.LogRequest(req)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if logger != nil {
		logger.LogResponse(req, resp, body, time.Since(started))
	}
	if err != nil {
		return nil, nil, err
	}

	return resp, body, nil
}

// decodeUploadPayload reads the given response to an upload request. If target is a non-nil pointer,
// it will be assigned the value of the upload identifier present in the response body. This function
// returns a boolean flag indicating if the function can be retried on failure (error-dependent).
func decodeUploadPayload(resp *http.Response, body []byte, target *int) (bool, error) {
	if resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusUnauthorized {
			return false, ErrUnauthorized
		}

		// Do not retry client errors
		return resp.StatusCode >= 500, errors.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	if target == nil {
		// No target expected, skip decoding body
		return false, nil
	}

	var respPayload struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &respPayload); err != nil {
		return false, errors.Errorf("unexpected response (%s)", err)
	}

	id, err := strconv.Atoi(respPayload.ID)
	if err != nil {
		return false, errors.Errorf("unexpected response (%s)", err)
	}

	*target = id
	return false, nil
}

// makeUploadURL creates a URL pointing to the configured Sourcegraph upload
// endpoint with the query string described by the given request options.
func makeUploadURL(opts uploadRequestOptions) (*url.URL, error) {
	qs := url.Values{}

	if opts.SourcegraphInstanceOptions.GitHubToken != "" {
		qs.Add("github_token", opts.SourcegraphInstanceOptions.GitHubToken)
	}
	if opts.UploadRecordOptions.Repo != "" {
		qs.Add("repository", opts.UploadRecordOptions.Repo)
	}
	if opts.UploadRecordOptions.Commit != "" {
		qs.Add("commit", opts.UploadRecordOptions.Commit)
	}
	if opts.UploadRecordOptions.Root != "" {
		qs.Add("root", opts.UploadRecordOptions.Root)
	}
	if opts.UploadRecordOptions.Indexer != "" {
		qs.Add("indexerName", opts.UploadRecordOptions.Indexer)
	}
	if opts.UploadRecordOptions.AssociatedIndexID != nil {
		qs.Add("associatedIndexId", formatInt(*opts.UploadRecordOptions.AssociatedIndexID))
	}
	if opts.MultiPart {
		qs.Add("multiPart", "true")
	}
	if opts.NumParts != 0 {
		qs.Add("numParts", formatInt(opts.NumParts))
	}
	if opts.UploadID != 0 {
		qs.Add("uploadId", formatInt(opts.UploadID))
	}
	if opts.UploadID != 0 && !opts.MultiPart && !opts.Done {
		// Do not set an index of zero unless we're uploading a part
		qs.Add("index", formatInt(opts.Index))
	}
	if opts.Done {
		qs.Add("done", "true")
	}

	path := opts.SourcegraphInstanceOptions.Path
	if path == "" {
		path = "/.api/lsif/upload"
	}

	url, err := url.Parse(opts.SourcegraphInstanceOptions.SourcegraphURL + path)
	if err != nil {
		return nil, err
	}

	url.RawQuery = qs.Encode()
	return url, nil
}

func formatInt(v int) string {
	return strconv.FormatInt(int64(v), 10)
}
