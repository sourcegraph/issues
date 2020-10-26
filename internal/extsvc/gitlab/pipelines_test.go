package gitlab

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGetMergeRequestPipelines(t *testing.T) {
	ctx := context.Background()
	project := &Project{}

	assertNextPage := func(t *testing.T, it func() ([]*Pipeline, error), want []*Pipeline) {
		pipelines, err := it()
		if diff := cmp.Diff(pipelines, want); diff != "" {
			t.Errorf("unexpected pipelines: %s", diff)
		}
		if err != nil {
			t.Errorf("unexpected error: %+v", err)
		}
	}

	t.Run("error status code", func(t *testing.T) {
		client := newTestClient(t)
		client.httpClient = &mockHTTPEmptyResponse{http.StatusNotFound}

		it := client.GetMergeRequestPipelines(ctx, project, 42)
		if it == nil {
			t.Error("unexpected nil iterator")
		}

		pipelines, err := it()
		if pipelines != nil {
			t.Errorf("unexpected non-nil pipelines: %+v", pipelines)
		}
		if err == nil {
			t.Error("unexpected nil error")
		}
	})

	t.Run("malformed response", func(t *testing.T) {
		client := newTestClient(t)
		client.httpClient = &mockHTTPResponseBody{
			responseBody: `this is not valid JSON`,
		}

		it := client.GetMergeRequestPipelines(ctx, project, 42)
		if it == nil {
			t.Error("unexpected nil iterator")
		}

		pipelines, err := it()
		if pipelines != nil {
			t.Errorf("unexpected non-nil pipelines: %+v", pipelines)
		}
		if err == nil {
			t.Error("unexpected nil error")
		}
	})

	t.Run("invalid response", func(t *testing.T) {
		client := newTestClient(t)
		client.httpClient = &mockHTTPResponseBody{
			responseBody: `[{"id":"the id cannot be a string"}]`,
		}

		it := client.GetMergeRequestPipelines(ctx, project, 42)
		if it == nil {
			t.Error("unexpected nil iterator")
		}

		pipelines, err := it()
		if pipelines != nil {
			t.Errorf("unexpected non-nil pipelines: %+v", pipelines)
		}
		if err == nil {
			t.Error("unexpected nil error")
		}
	})

	t.Run("zero pipelines", func(t *testing.T) {
		client := newTestClient(t)
		client.httpClient = &mockHTTPResponseBody{
			responseBody: `[]`,
		}

		it := client.GetMergeRequestPipelines(ctx, project, 42)
		if it == nil {
			t.Error("unexpected nil iterator")
		}

		assertNextPage(t, it, []*Pipeline{})

		// Calls after iteration should continue to return empty pages.
		assertNextPage(t, it, []*Pipeline{})
	})

	t.Run("one page", func(t *testing.T) {
		client := newTestClient(t)
		client.httpClient = &mockHTTPResponseBody{
			responseBody: `[{"id":42}]`,
		}

		it := client.GetMergeRequestPipelines(ctx, project, 42)
		if it == nil {
			t.Error("unexpected nil iterator")
		}

		assertNextPage(t, it, []*Pipeline{{ID: 42}})

		// Calls after iteration should continue to return empty pages.
		assertNextPage(t, it, []*Pipeline{})
	})

	t.Run("multiple pages", func(t *testing.T) {
		header := make(http.Header)
		header.Add("X-Next-Page", "/foo")

		client := newTestClient(t)
		client.httpClient = &mockHTTPResponseBody{
			header:       header,
			responseBody: `[{"id":1},{"id":2}]`,
		}

		it := client.GetMergeRequestPipelines(ctx, project, 42)
		if it == nil {
			t.Error("unexpected nil iterator")
		}

		assertNextPage(t, it, []*Pipeline{{ID: 1}, {ID: 2}})

		client.httpClient = &mockHTTPResponseBody{
			responseBody: `[{"id":42}]`,
		}
		assertNextPage(t, it, []*Pipeline{{ID: 42}})

		// Calls after iteration should continue to return empty pages.
		assertNextPage(t, it, []*Pipeline{})
	})
}

func TestPipelineKey(t *testing.T) {
	pipeline := &Pipeline{ID: 42}
	if have, want := pipeline.Key(), "Pipeline:42"; have != want {
		t.Errorf("incorrect pipeline key: have %s; want %s", have, want)
	}
}
