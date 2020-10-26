package gitlab

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
)

// GetMergeRequestPipelines retrieves the pipelines that have been executed as
// part of the given merge request. As the pipelines are paginated, a function
// is returned that may be invoked to return the next page of results. An empty
// slice and a nil error indicates that all pages have been returned.
func (c *Client) GetMergeRequestPipelines(ctx context.Context, project *Project, iid ID) func() ([]*Pipeline, error) {
	if MockGetMergeRequestPipelines != nil {
		return MockGetMergeRequestPipelines(c, ctx, project, iid)
	}

	url := fmt.Sprintf("projects/%d/merge_requests/%d/pipelines", project.ID, iid)
	return func() ([]*Pipeline, error) {
		page := []*Pipeline{}

		// If there aren't any further pages, we'll return the empty slice we
		// just created.
		if url == "" {
			return page, nil
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, errors.Wrap(err, "creating pipeline request")
		}

		header, _, err := c.do(ctx, req, &page)
		if err != nil {
			return nil, errors.Wrap(err, "requesting pipeline page")
		}

		// If there's another page, this will be a URL. If there's not, then
		// this will be an empty string, and we can detect that next iteration
		// to short circuit.
		url = header.Get("X-Next-Page")

		return page, nil
	}
}

type Pipeline struct {
	ID        ID             `json:"id"`
	SHA       string         `json:"sha"`
	Ref       string         `json:"ref"`
	Status    PipelineStatus `json:"status"`
	WebURL    string         `json:"web_url"`
	CreatedAt Time           `json:"created_at"`
	UpdatedAt Time           `json:"updated_at"`
}

type PipelineStatus string

const (
	PipelineStatusRunning  PipelineStatus = "running"
	PipelineStatusPending  PipelineStatus = "pending"
	PipelineStatusSuccess  PipelineStatus = "success"
	PipelineStatusFailed   PipelineStatus = "failed"
	PipelineStatusCanceled PipelineStatus = "canceled"
	PipelineStatusSkipped  PipelineStatus = "skipped"
	PipelineStatusCreated  PipelineStatus = "created"
	PipelineStatusManual   PipelineStatus = "manual"
)

func (p *Pipeline) Key() string {
	return fmt.Sprintf("Pipeline:%d", p.ID)
}
