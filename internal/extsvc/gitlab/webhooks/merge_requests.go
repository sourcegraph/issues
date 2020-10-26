package webhooks

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/gitlab"
)

// There's a bit going on in this file in terms of types, so here's a high
// level overview of what happens.
//
// When we get a webhook event of kind "merge_request" from GitLab, we want to
// eventually unmarshal it into one of the specific, exported types below, such
// as MergeRequestApprovedEvent or MergeRequestCloseEvent. To do so, we need to
// look at the "action" field embedded within the merge request in the event.
//
// We don't really want to have to unmarshal the JSON an extra time or copy the
// fairly sizable MergeRequest and User structs again, so what we do instead is
// unmarshal it initially into mergeRequestEvent. This unmarshals all of the
// fields that we need to construct the eventual typed event, but only exists
// for as long as it takes to go from the initial unmarshal into
// mergeRequestEvent until its downcast() method is called. That method looks
// at the "action" and then constructs the final struct, moving the pointer
// fields across from mergeRequestEvent into the MergeRequestEventCommon struct
// they all embed.

// MergeRequestEventCommon is the common type that underpins the types defined
// for specific merge request actions.
type MergeRequestEventCommon struct {
	EventCommon

	MergeRequest *gitlab.MergeRequest `json:"merge_request"`
	User         *gitlab.User         `json:"user"`
	Labels       *[]gitlab.Label      `json:"labels"`
}

// MergeRequestEventContainer is a common interface for types that embed
// MergeRequestEvent to provide a method that can return the embedded
// MergeRequestEvent.
type MergeRequestEventContainer interface {
	ToEvent() *MergeRequestEventCommon
}

type MergeRequestApprovedEvent struct{ MergeRequestEventCommon }
type MergeRequestCloseEvent struct{ MergeRequestEventCommon }
type MergeRequestMergeEvent struct{ MergeRequestEventCommon }
type MergeRequestReopenEvent struct{ MergeRequestEventCommon }
type MergeRequestUnapprovedEvent struct{ MergeRequestEventCommon }
type MergeRequestUpdateEvent struct{ MergeRequestEventCommon }

func (e *MergeRequestApprovedEvent) ToEvent() *MergeRequestEventCommon {
	return &e.MergeRequestEventCommon
}
func (e *MergeRequestCloseEvent) ToEvent() *MergeRequestEventCommon {
	return &e.MergeRequestEventCommon
}
func (e *MergeRequestMergeEvent) ToEvent() *MergeRequestEventCommon {
	return &e.MergeRequestEventCommon
}
func (e *MergeRequestReopenEvent) ToEvent() *MergeRequestEventCommon {
	return &e.MergeRequestEventCommon
}
func (e *MergeRequestUnapprovedEvent) ToEvent() *MergeRequestEventCommon {
	return &e.MergeRequestEventCommon
}
func (e *MergeRequestUpdateEvent) ToEvent() *MergeRequestEventCommon {
	return &e.MergeRequestEventCommon
}

// We don't define Key() methods on MergeRequestApprovedEvent and
// MergeRequestUnapprovedEvent because we don't need them when handling
// webhooks: those events don't include enough information (specifically, the
// system note ID) for us to create changeset events from them, so those types
// don't need to implement keyer.

func (e *MergeRequestCloseEvent) Key() string  { return e.key("Close") }
func (e *MergeRequestMergeEvent) Key() string  { return e.key("Merge") }
func (e *MergeRequestReopenEvent) Key() string { return e.key("Reopen") }

func (e *MergeRequestEventCommon) key(prefix string) string {
	// We can't key solely off the merge request ID because it may be reopened
	// and closed multiple times. Instead, we'll use the UpdatedAt field.
	return fmt.Sprintf("MergeRequest:%s:%d:%s", prefix, e.MergeRequest.IID, e.MergeRequest.UpdatedAt.Format(time.RFC3339))
}

// mergeRequestEvent is an internal type used for initially unmarshalling the
// typed event before it is downcast into a more specific type later based on
// the "action" field in the JSON.
type mergeRequestEvent struct {
	EventCommon

	User   *gitlab.User    `json:"user"`
	Labels *[]gitlab.Label `json:"labels"`

	ObjectAttributes mergeRequestEventObjectAttributes `json:"object_attributes"`
}

type mergeRequestEventObjectAttributes struct {
	*gitlab.MergeRequest
	Action string `json:"action"`
}

func (mre *mergeRequestEvent) downcast() (interface{}, error) {
	e := MergeRequestEventCommon{
		EventCommon:  mre.EventCommon,
		MergeRequest: mre.ObjectAttributes.MergeRequest,
		User:         mre.User,
		Labels:       mre.Labels,
	}

	// These action values are completely undocumented in GitLab's webhook
	// documentation: indeed, the _existence_ of the action field is only
	// implied by the examples. Nevertheless, we don't really have any other
	// option but to rely on it, since there's no other way to access the
	// information on what has changed when we get a webhook, since the payload
	// is otherwise untyped and the webhook types are far too coarsely grained
	// to be able to infer anything.
	switch mre.ObjectAttributes.Action {
	case "approved":
		return &MergeRequestApprovedEvent{e}, nil

	case "close":
		return &MergeRequestCloseEvent{e}, nil

	case "merge":
		return &MergeRequestMergeEvent{e}, nil

	case "reopen":
		return &MergeRequestReopenEvent{e}, nil

	case "unapproved":
		return &MergeRequestUnapprovedEvent{e}, nil

	case "update":
		return &MergeRequestUpdateEvent{e}, nil
	}

	return nil, errors.Wrapf(ErrObjectKindUnknown, "unknown merge request event action: %s", mre.ObjectAttributes.Action)
}
