package gitlab

import "context"

// MockListProjects, if non-nil, will be called instead of every invocation of Client.ListProjects.
var MockListProjects func(c *Client, ctx context.Context, urlStr string) (proj []*Project, nextPageURL *string, err error)

// MockListUsers, if non-nil, will be called instead of Client.ListUsers
var MockListUsers func(c *Client, ctx context.Context, urlStr string) (users []*User, nextPageURL *string, err error)

// MockGetUser, if non-nil, will be called instead of Client.GetUser
var MockGetUser func(c *Client, ctx context.Context, id string) (*User, error)

// MockGetProject, if non-nil, will be called instead of Client.GetProject
var MockGetProject func(c *Client, ctx context.Context, op GetProjectOp) (*Project, error)

// MockListTree, if non-nil, will be called instead of Client.ListTree
var MockListTree func(c *Client, ctx context.Context, op ListTreeOp) ([]*Tree, error)

// MockCreateMergeRequest, if non-nil, will be called instead of
// Client.CreateMergeRequest
var MockCreateMergeRequest func(c *Client, ctx context.Context, project *Project, opts CreateMergeRequestOpts) (*MergeRequest, error)

// MockGetMergeRequest, if non-nil, will be called instead of
// Client.GetMergeRequest
var MockGetMergeRequest func(c *Client, ctx context.Context, project *Project, iid ID) (*MergeRequest, error)

// MockGetMergeRequestNotes, if non-nil, will be called instead of
// Client.GetMergeRequestNotes
var MockGetMergeRequestNotes func(c *Client, ctx context.Context, project *Project, iid ID) func() ([]*Note, error)

// MockGetMergeRequestPipelines, if non-nil, will be called instead of
// Client.GetMergeRequestPipelines
var MockGetMergeRequestPipelines func(c *Client, ctx context.Context, project *Project, iid ID) func() ([]*Pipeline, error)

// MockGetOpenMergeRequestByRefs, if non-nil, will be called instead of
// Client.GetOpenMergeRequestByRefs
var MockGetOpenMergeRequestByRefs func(c *Client, ctx context.Context, project *Project, source, target string) (*MergeRequest, error)

// MockUpdateMergeRequest, if non-nil, will be called instead of
// Client.UpdateMergeRequest
var MockUpdateMergeRequest func(c *Client, ctx context.Context, project *Project, mr *MergeRequest, opts UpdateMergeRequestOpts) (*MergeRequest, error)
