package httpapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/inconshreveable/log15"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
)

func isSiteAdmin(ctx context.Context) bool {
	user, err := db.Users.GetByCurrentAuthUser(ctx)
	if err != nil {
		if errcode.IsNotFound(err) || err == db.ErrNoCurrentUser {
			return false
		}

		log15.Error("precise-code-intel proxy: failed to get up current user", "error", err)
		return false
	}

	return user != nil && user.SiteAdmin
}

func enforceAuth(ctx context.Context, w http.ResponseWriter, r *http.Request, repoName string) bool {
	validatorByCodeHost := map[string]func(context.Context, http.ResponseWriter, *http.Request, string) (int, error){
		"github.com": enforceAuthGithub,
	}

	for codeHost, validator := range validatorByCodeHost {
		if strings.HasPrefix(repoName, codeHost) {
			if status, err := validator(ctx, w, r, repoName); err != nil {
				http.Error(w, err.Error(), status)
				return false
			}

			return true
		}
	}

	http.Error(w, "verification not supported for code host - see https://github.com/sourcegraph/sourcegraph/issues/4967", http.StatusUnprocessableEntity)
	return false
}

func makeUploadRequest(host string, q url.Values, body io.Reader) (*http.Request, error) {
	url, err := url.Parse(fmt.Sprintf("%s/upload", host))
	if err != nil {
		return nil, err
	}
	url.RawQuery = q.Encode()

	return http.NewRequest("POST", url.String(), body)
}
