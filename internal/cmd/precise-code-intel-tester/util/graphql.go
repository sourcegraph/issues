package util

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/cockroachdb/errors"
	"github.com/hashicorp/go-multierror"
)

type ErrorPayload struct {
	Errors []GraphQLError `json:"errors"`
}

type GraphQLError struct {
	Message string `json:"message"`
}

// QueryGraphQL performs GraphQL query on the frontend.
//
// The queryName is the name of the GraphQL query, which uniquely identifies the source of the
// GraphQL query and helps e.g. a site admin know where such a query may be coming from. Importantly,
// unnamed queries (empty string) are considered to be unknown end-user API requests and as such will
// have the entire GraphQL request logged by the frontend, and cannot be uniquely identified in monitoring.
func QueryGraphQL(ctx context.Context, endpoint, queryName string, token, query string, variables map[string]interface{}, target interface{}) error {
	body, err := json.Marshal(map[string]interface{}{
		"query":     query,
		"variables": variables,
	})
	if err != nil {
		return err
	}

	if queryName != "" {
		queryName = "?" + queryName
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/.api/graphql%s", endpoint, queryName), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

	// Note: We do not use req.Context(ctx) here as it causes the frontend
	// to output long error logs, which is very noisy under high concurrency.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	contents, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var errorPayload ErrorPayload
	if err := json.Unmarshal(contents, &errorPayload); err == nil && len(errorPayload.Errors) > 0 {
		var combined error
		for _, err := range errorPayload.Errors {
			combined = multierror.Append(combined, errors.Errorf("%s", err.Message))
		}

		return combined
	}

	return json.Unmarshal(contents, &target)
}
