package gitlab

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
)

type ID int64

type MergeRequestState string

const (
	MergeRequestStateOpened MergeRequestState = "opened"
	MergeRequestStateClosed MergeRequestState = "closed"
	MergeRequestStateLocked MergeRequestState = "locked"
	MergeRequestStateMerged MergeRequestState = "merged"
)

type MergeRequest struct {
	ID             ID                `json:"id"`
	IID            ID                `json:"iid"`
	ProjectID      ID                `json:"project_id"`
	Title          string            `json:"title"`
	Description    string            `json:"description"`
	State          MergeRequestState `json:"state"`
	CreatedAt      Time              `json:"created_at"`
	UpdatedAt      Time              `json:"updated_at"`
	MergedAt       *Time             `json:"merged_at"`
	ClosedAt       *Time             `json:"closed_at"`
	HeadPipeline   *Pipeline         `json:"head_pipeline"`
	Labels         []string          `json:"labels"`
	SourceBranch   string            `json:"source_branch"`
	TargetBranch   string            `json:"target_branch"`
	WebURL         string            `json:"web_url"`
	WorkInProgress bool              `json:"work_in_progress"`
	Author         User              `json:"author"`

	DiffRefs DiffRefs `json:"diff_refs"`

	// The fields below are computed from other REST API requests when getting a
	// Merge Request. Once our minimum version is GitLab 12.0, we can use the
	// GraphQL API to retrieve all of this data at once, but until then, we have
	// to do it the old fashioned way with lots of REST requests.
	Notes               []*Note
	Pipelines           []*Pipeline
	ResourceStateEvents []*ResourceStateEvent
}

// IsWIP returns true if the given title would result in GitLab rendering the MR as 'work in progress'.
func IsWIP(title string) bool {
	return strings.HasPrefix(title, "Draft:") || strings.HasPrefix(title, "WIP:")
}

// SetWIP ensures a "WIP:" prefix on the given title. If a "Draft:" prefix is found, that one is retained instead.
func SetWIP(title string) string {
	if IsWIP(title) {
		return title
	}
	return "WIP: " + title
}

// UnsetWIP removes "WIP:" and "Draft:" prefixes from the given title.
// Depending on the GitLab version, either of them are used so we need to strip them both.
func UnsetWIP(title string) string {
	return strings.TrimPrefix(strings.TrimPrefix(title, "WIP: "), "Draft: ")
}

type DiffRefs struct {
	BaseSHA  string `json:"base_sha"`
	HeadSHA  string `json:"head_sha"`
	StartSHA string `json:"start_sha"`
}

var (
	ErrMergeRequestAlreadyExists = errors.New("merge request already exists")
	ErrTooManyMergeRequests      = errors.New("retrieved too many merge requests")
)

type CreateMergeRequestOpts struct {
	SourceBranch string `json:"source_branch"`
	TargetBranch string `json:"target_branch"`
	Title        string `json:"title"`
	Description  string `json:"description,omitempty"`
	// TODO: other fields at
	// https://docs.gitlab.com/ee/api/merge_requests.html#create-mr as needed.
}

func (c *Client) CreateMergeRequest(ctx context.Context, project *Project, opts CreateMergeRequestOpts) (*MergeRequest, error) {
	if MockCreateMergeRequest != nil {
		return MockCreateMergeRequest(c, ctx, project, opts)
	}

	data, err := json.Marshal(opts)
	if err != nil {
		return nil, errors.Wrap(err, "marshalling options")
	}

	time.Sleep(c.rateLimitMonitor.RecommendedWaitForBackgroundOp(1))

	req, err := http.NewRequest("POST", fmt.Sprintf("projects/%d/merge_requests", project.ID), bytes.NewBuffer(data))
	if err != nil {
		return nil, errors.Wrap(err, "creating request to create a merge request")
	}

	resp := &MergeRequest{}
	if _, code, err := c.do(ctx, req, resp); err != nil {
		if code == http.StatusConflict {
			return nil, ErrMergeRequestAlreadyExists
		}

		return nil, errors.Wrap(err, "sending request to create a merge request")
	}

	return resp, nil
}

func (c *Client) GetMergeRequest(ctx context.Context, project *Project, iid ID) (*MergeRequest, error) {
	if MockGetMergeRequest != nil {
		return MockGetMergeRequest(c, ctx, project, iid)
	}

	time.Sleep(c.rateLimitMonitor.RecommendedWaitForBackgroundOp(1))

	req, err := http.NewRequest("GET", fmt.Sprintf("projects/%d/merge_requests/%d", project.ID, iid), nil)
	if err != nil {
		return nil, errors.Wrap(err, "creating request to get a merge request")
	}

	resp := &MergeRequest{}
	if _, _, err := c.do(ctx, req, resp); err != nil {
		var e HTTPError
		if errors.As(err, &e) && e.Code() == http.StatusNotFound {
			if strings.Contains(e.Message(), "Project Not Found") {
				err = ErrProjectNotFound
			} else {
				err = ErrMergeRequestNotFound
			}
		}
		return nil, errors.Wrap(err, "sending request to get a merge request")
	}

	return resp, nil
}

func (c *Client) GetOpenMergeRequestByRefs(ctx context.Context, project *Project, source, target string) (*MergeRequest, error) {
	if MockGetOpenMergeRequestByRefs != nil {
		return MockGetOpenMergeRequestByRefs(c, ctx, project, source, target)
	}

	values := make(url.Values)
	// Since GitLab only allows one merge request per source/target branch pair,
	// we don't need to enumerate the full list of merge requests if more than
	// one matches: just the existence of a second merge request is sufficient
	// for us to return an error from this function.
	values.Add("per_page", "2")
	values.Add("source_branch", source)
	values.Add("target_branch", target)
	// The list endpoint doesn't return the full set of fields that we get from
	// the create and get single endpoints, and we need some of those fields
	// (specifically, diff_refs), so we'll just get the minimal set of fields
	// necessary from the list endpoint and then call the get endpoint to flesh
	// out the response.
	values.Add("view", "simple")
	u := &url.URL{
		Path: fmt.Sprintf("projects/%d/merge_requests", project.ID), RawQuery: values.Encode(),
	}

	time.Sleep(c.rateLimitMonitor.RecommendedWaitForBackgroundOp(1))

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "creating request to get merge request by refs")
	}

	resp := []*MergeRequest{}
	if _, _, err := c.do(ctx, req, &resp); err != nil {
		return nil, errors.Wrap(err, "sending request to get merge request by refs")
	}

	if len(resp) > 1 {
		return nil, ErrTooManyMergeRequests
	} else if len(resp) == 0 {
		return nil, ErrMergeRequestNotFound
	}

	return c.GetMergeRequest(ctx, project, resp[0].IID)
}

type UpdateMergeRequestOpts struct {
	TargetBranch string                       `json:"target_branch"`
	Title        string                       `json:"title"`
	Description  string                       `json:"description,omitempty"`
	StateEvent   UpdateMergeRequestStateEvent `json:"state_event,omitempty"`
}

type UpdateMergeRequestStateEvent string

const (
	UpdateMergeRequestStateEventClose  UpdateMergeRequestStateEvent = "close"
	UpdateMergeRequestStateEventReopen UpdateMergeRequestStateEvent = "reopen"

	// GitLab's update MR API is also used to perform state transitions on MRs:
	// they can be closed or reopened by setting a specific field exposed via
	// UpdateMergeRequestOpts above. To update a merge request _without_
	// changing the state, you omit that field, which is done via the
	// combination of this empty string constant and the omitempty JSON option
	// above on the relevant field.
	UpdateMergeRequestStateEventUnchanged UpdateMergeRequestStateEvent = ""
)

func (c *Client) UpdateMergeRequest(ctx context.Context, project *Project, mr *MergeRequest, opts UpdateMergeRequestOpts) (*MergeRequest, error) {
	if MockUpdateMergeRequest != nil {
		return MockUpdateMergeRequest(c, ctx, project, mr, opts)
	}

	data, err := json.Marshal(opts)
	if err != nil {
		return nil, errors.Wrap(err, "marshalling options")
	}

	time.Sleep(c.rateLimitMonitor.RecommendedWaitForBackgroundOp(1))

	req, err := http.NewRequest("PUT", fmt.Sprintf("projects/%d/merge_requests/%d", project.ID, mr.IID), bytes.NewBuffer(data))
	if err != nil {
		return nil, errors.Wrap(err, "creating request to update a merge request")
	}

	resp := &MergeRequest{}
	if _, _, err := c.do(ctx, req, resp); err != nil {
		return nil, errors.Wrap(err, "sending request to update a merge request")
	}

	return resp, nil
}

// ErrNotMergeable is returned by MergeMergeRequest when the merge request cannot
// be merged, because a precondition isn't met.
var ErrNotMergeable = errors.New("merge request is not in a mergeable state")

func (c *Client) MergeMergeRequest(ctx context.Context, project *Project, mr *MergeRequest, squash bool) (*MergeRequest, error) {
	if MockMergeMergeRequest != nil {
		return MockMergeMergeRequest(c, ctx, project, mr, squash)
	}

	payload := struct {
		Squash              bool   `json:"squash,omitempty"`
		SquashCommitMessage string `json:"squash_commit_message,omitempty"`
	}{
		Squash: squash,
	}
	if squash {
		payload.SquashCommitMessage = mr.Title + "\n\n" + mr.Description
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrap(err, "marshalling options")
	}

	time.Sleep(c.rateLimitMonitor.RecommendedWaitForBackgroundOp(1))

	req, err := http.NewRequest("PUT", fmt.Sprintf("projects/%d/merge_requests/%d/merge", project.ID, mr.IID), bytes.NewBuffer(data))
	if err != nil {
		return nil, errors.Wrap(err, "creating request to merge a merge request")
	}

	resp := &MergeRequest{}
	if _, _, err := c.do(ctx, req, resp); err != nil {
		var e HTTPError
		if errors.As(err, &e) && e.Code() == http.StatusMethodNotAllowed {
			return nil, errors.Wrap(ErrNotMergeable, err.Error())
		}
		return nil, errors.Wrap(err, "sending request to merge a merge request")
	}

	return resp, nil
}

func (c *Client) CreateMergeRequestNote(ctx context.Context, project *Project, mr *MergeRequest, body string) error {
	if MockCreateMergeRequestNote != nil {
		return MockCreateMergeRequestNote(c, ctx, project, mr, body)
	}

	var payload = struct {
		Body string `json:"body"`
	}{
		Body: body,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrap(err, "marshalling payload")
	}

	time.Sleep(c.rateLimitMonitor.RecommendedWaitForBackgroundOp(1))

	req, err := http.NewRequest("POST", fmt.Sprintf("projects/%d/merge_requests/%d/notes", project.ID, mr.IID), bytes.NewBuffer(data))
	if err != nil {
		return errors.Wrap(err, "creating request to comment on a merge request")
	}

	var resp struct {
		ID int32 `json:"id"`
	}
	if _, _, err := c.do(ctx, req, &resp); err != nil {
		return errors.Wrap(err, "sending request to comment on a merge request")
	}

	return nil
}
