package gitlab

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/cockroachdb/errors"
)

// GetMergeRequestResourceStateEvents retrieves the events for the given merge request. As the
// events are paginated, a function is returned that may be invoked to return the
// next page of results. An empty slice and a nil error indicates that all pages
// have been returned.
func (c *Client) GetMergeRequestResourceStateEvents(ctx context.Context, project *Project, iid ID) func() ([]*ResourceStateEvent, error) {
	if MockGetMergeRequestResourceStateEvents != nil {
		return MockGetMergeRequestResourceStateEvents(c, ctx, project, iid)
	}

	baseURL := fmt.Sprintf("projects/%d/merge_requests/%d/resource_state_events", project.ID, iid)
	currentPage := "1"
	return func() ([]*ResourceStateEvent, error) {
		page := []*ResourceStateEvent{}

		// If there aren't any further pages, we'll return the empty slice we
		// just created.
		if currentPage == "" {
			return page, nil
		}

		time.Sleep(c.rateLimitMonitor.RecommendedWaitForBackgroundOp(1))

		url, err := url.Parse(baseURL)
		if err != nil {
			return nil, err
		}
		q := url.Query()
		q.Add("page", currentPage)
		url.RawQuery = q.Encode()

		req, err := http.NewRequest("GET", url.String(), nil)
		if err != nil {
			return nil, errors.Wrap(err, "creating rse request")
		}

		header, _, err := c.do(ctx, req, &page)
		if err != nil {
			// If this endpoint is not found, the GitLab instance doesn't support these events yet.
			// This is okay and we can't do anything about it, but as GitLab <13.2 ages, we should
			// remove this stopgap.
			var e HTTPError
			if errors.As(err, &e) && e.Code() == http.StatusNotFound {
				return []*ResourceStateEvent{}, nil
			}
			return nil, errors.Wrap(err, "requesting rse page")
		}

		// If there's another page, this will be a page number. If there's not, then
		// this will be an empty string, and we can detect that next iteration
		// to short circuit.
		currentPage = header.Get("X-Next-Page")

		return page, nil
	}
}

// ResourceStateEventState is a type of all known resource state event states.
type ResourceStateEventState string

const (
	ResourceStateEventStateClosed   ResourceStateEventState = "closed"
	ResourceStateEventStateReopened ResourceStateEventState = "reopened"
	ResourceStateEventStateMerged   ResourceStateEventState = "merged"
)

type ResourceStateEvent struct {
	ID           ID                      `json:"id"`
	User         User                    `json:"user"`
	CreatedAt    Time                    `json:"created_at"`
	ResourceType string                  `json:"resource_type"`
	ResourceID   ID                      `json:"resource_id"`
	State        ResourceStateEventState `json:"state"`
}

type MergeRequestClosedEvent struct{ *ResourceStateEvent }

func (e *MergeRequestClosedEvent) Key() string {
	return fmt.Sprintf("closed:%s", e.CreatedAt.Time.Truncate(time.Second))
}

type MergeRequestReopenedEvent struct{ *ResourceStateEvent }

func (e *MergeRequestReopenedEvent) Key() string {
	return fmt.Sprintf("reopened:%s", e.CreatedAt.Time.Truncate(time.Second))
}

type MergeRequestMergedEvent struct{ *ResourceStateEvent }

func (e *MergeRequestMergedEvent) Key() string {
	return fmt.Sprintf("merged:%s", e.CreatedAt.Time.Truncate(time.Second))
}

// ToEvent returns a pointer to a more specific struct, or
// nil if the ResourceStateEvent is not of a known kind.
func (rse *ResourceStateEvent) ToEvent() interface{} {
	switch rse.State {
	case ResourceStateEventStateClosed:
		return &MergeRequestClosedEvent{rse}
	case ResourceStateEventStateReopened:
		return &MergeRequestReopenedEvent{rse}
	case ResourceStateEventStateMerged:
		return &MergeRequestMergedEvent{rse}
	}
	return nil
}
