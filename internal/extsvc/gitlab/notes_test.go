package gitlab

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGetMergeRequestNotes(t *testing.T) {
	ctx := context.Background()
	project := &Project{}

	assertNextPage := func(t *testing.T, it func() ([]*Note, error), want []*Note) {
		notes, err := it()
		if diff := cmp.Diff(notes, want); diff != "" {
			t.Errorf("unexpected notes: %s", diff)
		}
		if err != nil {
			t.Errorf("unexpected error: %+v", err)
		}
	}

	t.Run("error status code", func(t *testing.T) {
		client := newTestClient(t)
		client.httpClient = &mockHTTPEmptyResponse{http.StatusNotFound}

		it := client.GetMergeRequestNotes(ctx, project, 42)
		if it == nil {
			t.Error("unexpected nil iterator")
		}

		notes, err := it()
		if notes != nil {
			t.Errorf("unexpected non-nil notes: %+v", notes)
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

		it := client.GetMergeRequestNotes(ctx, project, 42)
		if it == nil {
			t.Error("unexpected nil iterator")
		}

		notes, err := it()
		if notes != nil {
			t.Errorf("unexpected non-nil notes: %+v", notes)
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

		it := client.GetMergeRequestNotes(ctx, project, 42)
		if it == nil {
			t.Error("unexpected nil iterator")
		}

		notes, err := it()
		if notes != nil {
			t.Errorf("unexpected non-nil notes: %+v", notes)
		}
		if err == nil {
			t.Error("unexpected nil error")
		}
	})

	t.Run("zero notes", func(t *testing.T) {
		client := newTestClient(t)
		client.httpClient = &mockHTTPResponseBody{
			responseBody: `[]`,
		}

		it := client.GetMergeRequestNotes(ctx, project, 42)
		if it == nil {
			t.Error("unexpected nil iterator")
		}

		assertNextPage(t, it, []*Note{})

		// Calls after iteration should continue to return empty pages.
		assertNextPage(t, it, []*Note{})
	})

	t.Run("one page", func(t *testing.T) {
		client := newTestClient(t)
		client.httpClient = &mockHTTPResponseBody{
			responseBody: `[{"id":42}]`,
		}

		it := client.GetMergeRequestNotes(ctx, project, 42)
		if it == nil {
			t.Error("unexpected nil iterator")
		}

		assertNextPage(t, it, []*Note{{ID: 42}})

		// Calls after iteration should continue to return empty pages.
		assertNextPage(t, it, []*Note{})
	})

	t.Run("multiple pages", func(t *testing.T) {
		header := make(http.Header)
		header.Add("X-Next-Page", "/foo")

		client := newTestClient(t)
		client.httpClient = &mockHTTPResponseBody{
			header:       header,
			responseBody: `[{"id":1},{"id":2}]`,
		}

		it := client.GetMergeRequestNotes(ctx, project, 42)
		if it == nil {
			t.Error("unexpected nil iterator")
		}

		assertNextPage(t, it, []*Note{{ID: 1}, {ID: 2}})

		client.httpClient = &mockHTTPResponseBody{
			responseBody: `[{"id":42}]`,
		}
		assertNextPage(t, it, []*Note{{ID: 42}})

		// Calls after iteration should continue to return empty pages.
		assertNextPage(t, it, []*Note{})
	})
}

func TestNoteKey(t *testing.T) {
	note := &Note{ID: 42}
	if have, want := note.Key(), "Note:42"; have != want {
		t.Errorf("incorrect note key: have %s; want %s", have, want)
	}
}

func TestNoteToEvent(t *testing.T) {
	t.Run("non-system note", func(t *testing.T) {
		note := &Note{System: false}
		if v := note.ToEvent(); v != nil {
			t.Errorf("unexpected non-nil ToEvent value: %+v", v)
		}
	})

	t.Run("system, non approval note", func(t *testing.T) {
		note := &Note{System: true, Body: ""}
		if v := note.ToEvent(); v != nil {
			t.Errorf("unexpected non-nil ToEvent value: %+v", v)
		}
	})

	t.Run("system, approval note", func(t *testing.T) {
		note := &Note{System: true, Body: "approved this merge request"}
		if v, ok := note.ToEvent().(*ReviewApprovedEvent); v == nil || !ok {
			t.Errorf("unexpected ToEvent value: %+v", v)
		}
	})

	t.Run("system, unapproval note", func(t *testing.T) {
		note := &Note{System: true, Body: "unapproved this merge request"}
		if v, ok := note.ToEvent().(*ReviewUnapprovedEvent); v == nil || !ok {
			t.Errorf("unexpected ToEvent value: %+v", v)
		}
	})

	t.Run("system, wip note", func(t *testing.T) {
		note := &Note{System: true, Body: "marked as a **Work In Progress**"}
		if v, ok := note.ToEvent().(*MarkWorkInProgressEvent); v == nil || !ok {
			t.Errorf("unexpected ToEvent value: %+v", v)
		}
	})

	t.Run("system, un wip note", func(t *testing.T) {
		note := &Note{System: true, Body: "unmarked as a **Work In Progress**"}
		if v, ok := note.ToEvent().(*UnmarkWorkInProgressEvent); v == nil || !ok {
			t.Errorf("unexpected ToEvent value: %+v", v)
		}
	})
}
