package handlerutil

import "github.com/sourcegraph/sourcegraph/internal/api"

// URLMovedError should be returned when a requested resource has moved to a new
// address.
type URLMovedError struct {
	NewRepo api.RepoName `json:"RedirectTo"`
}

func (e *URLMovedError) Error() string {
	return "URL moved to " + string(e.NewRepo)
}
