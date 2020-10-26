package webhooks

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/gitlab"
)

// EventCommon contains fields that are common to all webhook event types.
type EventCommon struct {
	ObjectKind string               `json:"object_kind"`
	EventType  string               `json:"event_type"`
	Project    gitlab.ProjectCommon `json:"project"`
}

// Simple events that are simply unmarshalled and have no methods attached are defined below.

type PipelineEvent struct {
	EventCommon

	User         gitlab.User          `json:"user"`
	Pipeline     gitlab.Pipeline      `json:"object_attributes"`
	MergeRequest *gitlab.MergeRequest `json:"merge_request"`
}

var ErrObjectKindUnknown = errors.New("unknown object kind")

type downcaster interface {
	downcast() (interface{}, error)
}

// UnmarshalEvent unmarshals the given JSON into an event type. Possible return
// types are *MergeRequestEvent.
//
// Errors caused by a valid payload being of an unknown type may be
// distinguished from other errors by checking for ErrObjectKindUnknown in the
// error chain; note that the top level error value will not be equal to
// ErrObjectKindUnknown in this case.
func UnmarshalEvent(data []byte) (interface{}, error) {
	// We need to unmarshal the event twice: once to determine what the eventual
	// return type should be, and then once to actual unmarshal into that type.
	//
	// Since we only care about the object_kind field, we'll start by
	// unmarshalling into a minimal type that only has that field. We use
	// object_kind instead of event_type because not all GitLab webhook types
	// include event_type, whereas object_kind is generally reliable.
	var event struct {
		ObjectKind string `json:"object_kind"`
	}
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, errors.Wrap(err, "determining object kind")
	}

	// Now we can set up the typed event that we'll unmarshal into.
	var typedEvent interface{}
	switch event.ObjectKind {
	case "merge_request":
		typedEvent = &mergeRequestEvent{}
	case "pipeline":
		typedEvent = &PipelineEvent{}
	default:
		return nil, errors.Wrapf(ErrObjectKindUnknown, "kind: %s", event.ObjectKind)
	}

	// Let's perform the real unmarshal.
	if err := json.Unmarshal(data, typedEvent); err != nil {
		return nil, errors.Wrap(err, "unmarshalling typed event")
	}

	// Some event types need to be able to be downcasted to a more specific type
	// than the one we get just from the object_kind attribute, so let's do that
	// here if we need to, otherwise we can return.
	if dc, ok := typedEvent.(downcaster); ok {
		return dc.downcast()
	}
	return typedEvent, nil
}
