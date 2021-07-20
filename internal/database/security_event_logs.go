package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/envvar"
	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/version"
)

type SecurityEventName string

const (
	SecurityEventNameSignOutAttempted SecurityEventName = "SignOutAttempted"
	SecurityEventNameSignOutFailed    SecurityEventName = "SignOutFailed"
	SecurityEventNameSignOutSucceeded SecurityEventName = "SignOutSucceeded"

	SecurityEventNameSignInAttempted SecurityEventName = "SignInAttempted"
	SecurityEventNameSignInFailed    SecurityEventName = "SignInFailed"
	SecurityEventNameSignInSucceeded SecurityEventName = "SignInSucceeded"

	SecurityEventNameAccountCreated SecurityEventName = "AccountCreated"
	SecurityEventNameAccountDeleted SecurityEventName = "AccountDeleted"
	SecurityEventNameAccountNuked   SecurityEventName = "AccountNuked"

	SecurityEventNamPasswordResetRequested SecurityEventName = "PasswordResetRequested"
	SecurityEventNamPasswordRandomized     SecurityEventName = "PasswordRandomized"
	SecurityEventNamePasswordChanged       SecurityEventName = "PasswordChanged"

	SecurityEventNameEmailVerified SecurityEventName = "EmailVerified"

	SecurityEventNameRoleChangeDenied  SecurityEventName = "RoleChangeDenied"
	SecurityEventNameRoleChangeGranted SecurityEventName = "RoleChangeGranted"
)

// SecurityEvent contains information needed for logging a security-relevant event.
type SecurityEvent struct {
	Name            SecurityEventName
	URL             string
	UserID          uint32
	AnonymousUserID string
	Argument        json.RawMessage
	Source          string
	Timestamp       time.Time
}

// A SecurityEventLogStore provides persistence for security events.
type SecurityEventLogStore struct {
	*basestore.Store
}

// SecurityEventLogs instantiates and returns a new SecurityEventLogStore with prepared statements.
func SecurityEventLogs(db dbutil.DB) *SecurityEventLogStore {
	return &SecurityEventLogStore{Store: basestore.NewWithDB(db, sql.TxOptions{})}
}

// Insert adds a new security event to the store.
func (s *SecurityEventLogStore) Insert(ctx context.Context, e *SecurityEvent) error {
	argument := e.Argument
	if argument == nil {
		argument = []byte(`{}`)
	}

	_, err := s.Handle().DB().ExecContext(
		ctx,
		"INSERT INTO security_event_logs(name, url, user_id, anonymous_user_id, source, argument, version, timestamp) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
		e.Name,
		e.URL,
		e.UserID,
		e.AnonymousUserID,
		e.Source,
		argument,
		version.Version(),
		e.Timestamp.UTC(),
	)
	if err != nil {
		return errors.Wrap(err, "INSERT")
	}
	return nil
}

// LogEvent will log security events.
//
// Note that it does not return an error and will instead simply log it.
func (s *SecurityEventLogStore) LogEvent(ctx context.Context, e *SecurityEvent) {
	// We don't want to begin logging authentication or authorization events in
	// on-premises installations yet.
	if !envvar.SourcegraphDotComMode() {
		return
	}

	if err := s.Insert(ctx, e); err != nil {
		log15.Error(string(e.Name), "err", err)
	}
}
