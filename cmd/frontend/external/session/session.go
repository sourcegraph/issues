// Package session exports symbols from frontend/internal/session. See the
// parent package godoc for more information.
package session

import "github.com/sourcegraph/sourcegraph/cmd/frontend/internal/session"

var (
	ResetMockSessionStore        = session.ResetMockSessionStore
	SetActor                     = session.SetActor
	SetData                      = session.SetData
	GetData                      = session.GetData
	InvalidateSessionsByID       = session.InvalidateSessionsByID
	InvalidateSessionCurrentUser = session.InvalidateSessionCurrentUser
)
