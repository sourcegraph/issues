package assetsutil

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/database"
)

func GetDebugAssetsLocation(r *http.Request) (*url.URL, error) {
	const debugAssetsQueryParam = "debugAssetsUrl"

	q := r.URL.Query()
	debugAssetsValue := q.Get(debugAssetsQueryParam)

	if debugAssetsValue == "" {
		return nil, errors.New("debugAssets query param not present")
	}

	a := actor.FromContext(r.Context())
	if !a.IsAuthenticated() {
		return nil, errors.New("user is not authenticated")
	}

	// The user may have a tag that opts them in
	ok, _ := database.GlobalUsers.HasTag(r.Context(), a.UID, database.TagAllowDebugAssets)
	if !ok {
		return nil, errors.New(("user does not have AllowDebugAssets tag"))
	}

	debugAssetsURL, err := url.Parse(debugAssetsValue)
	if err != nil {
		return nil, err
	}

	return debugAssetsURL, nil
}
