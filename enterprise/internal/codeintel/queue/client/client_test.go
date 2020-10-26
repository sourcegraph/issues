package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDequeue(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("unexpected method. want=%s have=%s", "POST", r.Method)
		}
		if r.URL.Path != "/.internal-code-intel/index-queue/dequeue" {
			t.Errorf("unexpected method. want=%s have=%s", "/.internal-code-intel/index-queue/dequeue", r.URL.Path)
		}
		if _, password, _ := r.BasicAuth(); password != "hunter2" {
			t.Errorf("unexpected password. want=%s have=%s", "hunter2", password)
		}

		comparePayload(t, r.Body, []byte(`{
			"indexerName": "deadbeef"
		}`))
		w.Write([]byte(`{"id": 42}`))
	}))
	defer ts.Close()

	index, dequeued, err := testClient(ts.URL).Dequeue(context.Background())
	if err != nil {
		t.Fatalf("unexpected error dequeueing record: %s", err)
	}
	if !dequeued {
		t.Fatalf("expected record to be dequeued")
	}
	if index.ID != 42 {
		t.Errorf("unexpected id. want=%d have=%d", 42, index.ID)
	}
}

func TestDequeueNoRecord(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	_, dequeued, err := testClient(ts.URL).Dequeue(context.Background())
	if err != nil {
		t.Fatalf("unexpected error dequeueing record: %s", err)
	}
	if dequeued {
		t.Fatalf("unexpected record")
	}
}

func TestDequeueBadResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	if _, _, err := testClient(ts.URL).Dequeue(context.Background()); err == nil {
		t.Fatalf("unexpected nil error dequeueing record")
	}
}

func TestSetLogContents(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("unexpected method. want=%s have=%s", "POST", r.Method)
		}
		if r.URL.Path != "/.internal-code-intel/index-queue/setlog" {
			t.Errorf("unexpected method. want=%s have=%s", "/.internal-code-intel/index-queue/setlog", r.URL.Path)
		}
		if _, password, _ := r.BasicAuth(); password != "hunter2" {
			t.Errorf("unexpected password. want=%s have=%s", "hunter2", password)
		}

		comparePayload(t, r.Body, []byte(`{
			"indexerName": "deadbeef",
			"indexId": 42,
			"payload": "test payload"
		}`))

		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	if err := testClient(ts.URL).SetLogContents(context.Background(), 42, "test payload"); err != nil {
		t.Fatalf("unexpected error setting log contents: %s", err)
	}
}

func TestSetLogContentsBadResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	if err := testClient(ts.URL).SetLogContents(context.Background(), 42, "test payload"); err == nil {
		t.Fatalf("unexpected error setting log contents: %s", err)
	}
}

func TestComplete(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("unexpected method. want=%s have=%s", "POST", r.Method)
		}
		if r.URL.Path != "/.internal-code-intel/index-queue/complete" {
			t.Errorf("unexpected method. want=%s have=%s", "/.internal-code-intel/index-queue/complete", r.URL.Path)
		}
		if _, password, _ := r.BasicAuth(); password != "hunter2" {
			t.Errorf("unexpected password. want=%s have=%s", "hunter2", password)
		}

		comparePayload(t, r.Body, []byte(`{
			"indexerName": "deadbeef",
			"indexId": 42,
			"errorMessage": ""
		}`))

		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	if err := testClient(ts.URL).Complete(context.Background(), 42, nil); err != nil {
		t.Fatalf("unexpected error marking record complete: %s", err)
	}
}

func TestCompleteError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		comparePayload(t, r.Body, []byte(`{
			"indexerName": "deadbeef",
			"indexId": 42,
			"errorMessage": "oops"
		}`))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	if err := testClient(ts.URL).Complete(context.Background(), 42, fmt.Errorf("oops")); err != nil {
		t.Fatalf("unexpected error marking record complete: %s", err)
	}
}

func TestCompleteBadResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	if err := testClient(ts.URL).Complete(context.Background(), 42, fmt.Errorf("oops")); err == nil {
		t.Fatalf("unexpected error marking record complete: %s", err)
	}
}

func TestHeartbeat(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("unexpected method. want=%s have=%s", "POST", r.Method)
		}
		if r.URL.Path != "/.internal-code-intel/index-queue/heartbeat" {
			t.Errorf("unexpected method. want=%s have=%s", "/.internal-code-intel/index-queue/heartbeat", r.URL.Path)
		}
		if _, password, _ := r.BasicAuth(); password != "hunter2" {
			t.Errorf("unexpected password. want=%s have=%s", "hunter2", password)
		}

		comparePayload(t, r.Body, []byte(`{
			"indexerName": "deadbeef",
			"indexIds": [1, 2, 3, 4, 5]
		}`))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	if err := testClient(ts.URL).Heartbeat(context.Background(), []int{1, 2, 3, 4, 5}); err != nil {
		t.Fatalf("unexpected error performing heartbeat: %s", err)
	}
}

func TestHeartbeatBadResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader((http.StatusInternalServerError))
	}))
	defer ts.Close()

	if err := testClient(ts.URL).Heartbeat(context.Background(), []int{1, 2, 3, 4, 5}); err == nil {
		t.Fatalf("unexpected nil error dequeueing record")
	}
}

func testClient(frontendURL string) *client {
	return &client{
		frontendURL: frontendURL,
		indexerName: "deadbeef",
		authToken:   "hunter2",
	}
}

func comparePayload(t *testing.T, raw io.Reader, expected []byte) {
	content, err := ioutil.ReadAll(raw)
	if err != nil {
		t.Fatalf("unexpected error reading payload: %s", err)
	}

	if diff := cmp.Diff(normalize(expected), normalize(content)); diff != "" {
		t.Errorf("unexpected request payload (-want +got):\n%s", diff)
	}
}

func normalize(v []byte) string {
	temp := map[string]interface{}{}
	_ = json.Unmarshal(v, &temp)
	v, _ = json.Marshal(temp)
	return string(v)
}
