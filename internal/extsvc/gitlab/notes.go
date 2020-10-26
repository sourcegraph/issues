package gitlab

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
)

// GetMergeRequestNotes retrieves the notes for the given merge request. As the
// notes are paginated, a function is returned that may be invoked to return the
// next page of results. An empty slice and a nil error indicates that all pages
// have been returned.
func (c *Client) GetMergeRequestNotes(ctx context.Context, project *Project, iid ID) func() ([]*Note, error) {
	if MockGetMergeRequestNotes != nil {
		return MockGetMergeRequestNotes(c, ctx, project, iid)
	}

	url := fmt.Sprintf("projects/%d/merge_requests/%d/notes", project.ID, iid)
	return func() ([]*Note, error) {
		page := []*Note{}

		// If there aren't any further pages, we'll return the empty slice we
		// just created.
		if url == "" {
			return page, nil
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, errors.Wrap(err, "creating notes request")
		}

		header, _, err := c.do(ctx, req, &page)
		if err != nil {
			return nil, errors.Wrap(err, "requesting notes page")
		}

		// If there's another page, this will be a URL. If there's not, then
		// this will be an empty string, and we can detect that next iteration
		// to short circuit.
		url = header.Get("X-Next-Page")

		return page, nil
	}
}

// SystemNoteBody is a type of all known system message bodies.
type SystemNoteBody string

const (
	SystemNoteBodyReviewApproved               SystemNoteBody = "approved this merge request"
	SystemNoteBodyReviewUnapproved             SystemNoteBody = "unapproved this merge request"
	SystemNoteBodyReviewUnmarkedWorkInProgress SystemNoteBody = "unmarked as a **Work In Progress**"
	SystemNoteBodyReviewMarkedWorkInProgress   SystemNoteBody = "marked as a **Work In Progress**"
)

type Note struct {
	ID        ID             `json:"id"`
	Body      SystemNoteBody `json:"body"`
	Author    User           `json:"author"`
	CreatedAt Time           `json:"created_at"`
	System    bool           `json:"system"`
}

func (n *Note) Key() string {
	return fmt.Sprintf("Note:%d", n.ID)
}

// Notes are not strongly typed, but also provide the only real method we have
// of getting historical approval events. We'll define a couple of fake types to
// better match what other external services provide, and a function to convert
// a Note into one of those types if the Note is a system approval comment.

type ReviewApprovedEvent struct{ *Note }
type ReviewUnapprovedEvent struct{ *Note }
type MarkWorkInProgressEvent struct{ *Note }
type UnmarkWorkInProgressEvent struct{ *Note }

// ToEvent returns a pointer to a more specific struct, or
// nil if the Note is not of a known kind.
func (n *Note) ToEvent() interface{} {
	if n.System {
		switch n.Body {
		case SystemNoteBodyReviewApproved:
			return &ReviewApprovedEvent{n}
		case SystemNoteBodyReviewUnapproved:
			return &ReviewUnapprovedEvent{n}
		case SystemNoteBodyReviewUnmarkedWorkInProgress:
			return &UnmarkWorkInProgressEvent{n}
		case SystemNoteBodyReviewMarkedWorkInProgress:
			return &MarkWorkInProgressEvent{n}
		}
	}

	return nil
}
