package gitlab

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/sourcegraph/sourcegraph/internal/ratelimit"
	"github.com/sourcegraph/sourcegraph/internal/rcache"
)

type mockHTTPResponseBody struct {
	count        int
	header       http.Header
	responseBody string
}

func (s *mockHTTPResponseBody) Do(req *http.Request) (*http.Response, error) {
	s.count++
	return &http.Response{
		Request:    req,
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(strings.NewReader(s.responseBody)),
		Header:     s.header,
	}, nil
}

type mockHTTPEmptyResponse struct {
	statusCode int
}

func (s mockHTTPEmptyResponse) Do(req *http.Request) (*http.Response, error) {
	return &http.Response{
		Request:    req,
		StatusCode: s.statusCode,
		Body:       ioutil.NopCloser(bytes.NewReader(nil)),
	}, nil
}

func newTestClient(t *testing.T) *Client {
	rcache.SetupForTest(t)
	return &Client{
		baseURL:          &url.URL{Scheme: "https", Host: "example.com", Path: "/"},
		httpClient:       &http.Client{},
		RateLimitMonitor: &ratelimit.Monitor{},
		projCache:        rcache.NewWithTTL("__test__gl_proj", 1000),
	}
}
