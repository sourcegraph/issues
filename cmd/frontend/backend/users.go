package backend

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/randstring"
)

func MakeRandomHardToGuessPassword() string {
	return randstring.NewLen(36)
}

var MockMakePasswordResetURL func(ctx context.Context, userID int32) (*url.URL, error)

func MakePasswordResetURL(ctx context.Context, userID int32) (*url.URL, error) {
	if MockMakePasswordResetURL != nil {
		return MockMakePasswordResetURL(ctx, userID)
	}
	resetCode, err := db.Users.RenewPasswordResetCode(ctx, userID)
	if err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Set("userID", strconv.Itoa(int(userID)))
	query.Set("code", resetCode)
	return &url.URL{Path: "/password-reset", RawQuery: query.Encode()}, nil
}

// CheckActorHasTag reports whether the context actor has the given tag. If not, or if an error
// occurs, a non-nil error is returned.
func CheckActorHasTag(ctx context.Context, tag string) error {
	actor := actor.FromContext(ctx)
	if !actor.IsAuthenticated() {
		return ErrNotAuthenticated
	}
	user, err := db.Users.GetByID(ctx, actor.UID)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrNotAuthenticated
	}
	for _, t := range user.Tags {
		if t == tag {
			return nil
		}
	}
	return fmt.Errorf("actor lacks required tag %q", tag)
}

const (
	TagAllowUserExternalServicePublic = "AllowUserExternalServicePublic"
)
