package webhooks

import (
	"errors"
	"testing"

	"github.com/sourcegraph/sourcegraph/internal/extsvc/gitlab"
)

func TestUnmarshalEvent(t *testing.T) {
	t.Run("invalid JSON", func(t *testing.T) {
		event, err := UnmarshalEvent([]byte(`{`))
		if event != nil {
			t.Errorf("unexpected non-nil event: %+v", event)
		}
		if err == nil {
			t.Error("unexpected nil error")
		}
	})

	t.Run("missing object kind", func(t *testing.T) {
		event, err := UnmarshalEvent([]byte(`{}`))
		if event != nil {
			t.Errorf("unexpected non-nil event: %+v", event)
		}
		if err == nil {
			t.Error("unexpected nil error")
		} else if !errors.Is(err, ErrObjectKindUnknown) {
			t.Errorf("unexpected error chain: %+v", err)
		}
	})

	t.Run("unknown object kind", func(t *testing.T) {
		event, err := UnmarshalEvent([]byte(`{"object_kind":"github_merger"}`))
		if event != nil {
			t.Errorf("unexpected non-nil event: %+v", event)
		}
		if err == nil {
			t.Error("unexpected nil error")
		} else if !errors.Is(err, ErrObjectKindUnknown) {
			t.Errorf("unexpected error chain: %+v", err)
		}
	})

	t.Run("lying object kind", func(t *testing.T) {
		event, err := UnmarshalEvent([]byte(`
			{
				"object_kind": "merge_request",
				"object_attributes":{
					"iid": ["this", "is", "not", "a", "valid", "id"]
				}
			}
		`))
		if event != nil {
			t.Errorf("unexpected non-nil event: %+v", event)
		}
		if err == nil {
			t.Error("unexpected nil error")
		} else if errors.Is(err, ErrObjectKindUnknown) {
			t.Errorf("unexpected error chain: %+v", err)
		}
	})

	t.Run("valid merge request", func(t *testing.T) {
		event, err := UnmarshalEvent([]byte(`
			{
				"object_kind": "merge_request",
				"event_type": "merge_request",
				"object_attributes":{
					"iid": 42,
					"action": "approved"
				}
			}
		`))
		if event == nil {
			t.Error("unexpected nil event")
		}
		if err != nil {
			t.Errorf("unexpected error: %+v", err)
		}

		mre := event.(*MergeRequestApprovedEvent)
		if want := gitlab.ID(42); mre.MergeRequest.IID != want {
			t.Errorf("unexpected IID: have %d; want %d", mre.MergeRequest.IID, want)
		}
		if want := "merge_request"; mre.EventType != want {
			t.Errorf("unexpected event_type: have %s; want %s", mre.EventType, want)
		}
	})

	t.Run("valid pipeline", func(t *testing.T) {
		event, err := UnmarshalEvent([]byte(`
			{
				"object_kind": "pipeline",
				"object_attributes":{
					"id": 42
				}
			}
		`))
		if event == nil {
			t.Error("unexpected nil event")
		}
		if err != nil {
			t.Errorf("unexpected error: %+v", err)
		}

		pe := event.(*PipelineEvent)
		if want := gitlab.ID(42); pe.Pipeline.ID != want {
			t.Errorf("unexpected IID: have %d; want %d", pe.Pipeline.ID, want)
		}
	})
}
