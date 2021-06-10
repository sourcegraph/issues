package backend

import (
	"context"
	"net/url"
	"strconv"

	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/randstring"
)

func MakeRandomHardToGuessPassword() string {
	return randstring.NewLen(36)
}

var MockMakePasswordResetURL func(ctx context.Context, userID int32) (*url.URL, error)

func MakePasswordResetURL(ctx context.Context, db dbutil.DB, userID int32) (*url.URL, error) {
	if MockMakePasswordResetURL != nil {
		return MockMakePasswordResetURL(ctx, userID)
	}
	resetCode, err := database.Users(db).RenewPasswordResetCode(ctx, userID)
	if err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Set("userID", strconv.Itoa(int(userID)))
	query.Set("code", resetCode)
	return &url.URL{Path: "/password-reset", RawQuery: query.Encode()}, nil
}
