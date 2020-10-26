package gqltestutil

import (
	"github.com/pkg/errors"
)

// GitBlob returns blob content of the file in given repository at given revision.
func (c *Client) GitBlob(repoName, revision, filePath string) (string, error) {
	const gqlQuery = `
query Blob($repoName: String!, $revision: String!, $filePath: String!) {
	repository(name: $repoName) {
		commit(rev: $revision) {
			file(path: $filePath) {
				content
			}
		}
	}
}
`
	variables := map[string]interface{}{
		"repoName": repoName,
		"revision": revision,
		"filePath": filePath,
	}
	var resp struct {
		Data struct {
			Repository struct {
				Commit struct {
					File struct {
						Content string `json:"content"`
					} `json:"file"`
				} `json:"commit"`
			} `json:"repository"`
		} `json:"data"`
	}
	err := c.GraphQL("", "", gqlQuery, variables, &resp)
	if err != nil {
		return "", errors.Wrap(err, "request GraphQL")
	}

	return resp.Data.Repository.Commit.File.Content, nil
}
