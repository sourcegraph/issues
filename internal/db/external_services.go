package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/keegancsmith/sqlf"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
	"github.com/xeipuuv/gojsonschema"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/jsonc"
	"github.com/sourcegraph/sourcegraph/internal/secret"
	"github.com/sourcegraph/sourcegraph/schema"
)

// An ExternalServicesStore stores external services and their configuration.
// Before updating or creating a new external service, validation is performed.
// The enterprise code registers additional validators at run-time and sets the
// global instance in stores.go
type ExternalServicesStore struct {
	GitHubValidators          []func(*schema.GitHubConnection) error
	GitLabValidators          []func(*schema.GitLabConnection, []schema.AuthProviders) error
	BitbucketServerValidators []func(*schema.BitbucketServerConnection) error

	// PreCreateExternalService (if set) is invoked as a hook prior to creating a
	// new external service in the database.
	PreCreateExternalService func(context.Context) error
}

// ExternalServiceKinds contains a map of all supported kinds of
// external services.
var ExternalServiceKinds = map[string]ExternalServiceKind{
	extsvc.KindAWSCodeCommit:   {CodeHost: true, JSONSchema: schema.AWSCodeCommitSchemaJSON},
	extsvc.KindBitbucketCloud:  {CodeHost: true, JSONSchema: schema.BitbucketCloudSchemaJSON},
	extsvc.KindBitbucketServer: {CodeHost: true, JSONSchema: schema.BitbucketServerSchemaJSON},
	extsvc.KindGitHub:          {CodeHost: true, JSONSchema: schema.GitHubSchemaJSON},
	extsvc.KindGitLab:          {CodeHost: true, JSONSchema: schema.GitLabSchemaJSON},
	extsvc.KindGitolite:        {CodeHost: true, JSONSchema: schema.GitoliteSchemaJSON},
	extsvc.KindPhabricator:     {CodeHost: true, JSONSchema: schema.PhabricatorSchemaJSON},
	extsvc.KindOther:           {CodeHost: true, JSONSchema: schema.OtherExternalServiceSchemaJSON},
}

// ExternalServiceKind describes a kind of external service.
type ExternalServiceKind struct {
	// True if the external service can host repositories.
	CodeHost bool

	JSONSchema string // JSON Schema for the external service's configuration
}

// ExternalServicesListOptions contains options for listing external services.
type ExternalServicesListOptions struct {
	// When true, only include external services not under any namespace (i.e. owned by all site admins),
	// and value of NamespaceUserID is ignored.
	NoNamespace bool
	// When specified, only include external services under given user namespace.
	NamespaceUserID int32
	// When specified, only include external services with given list of kinds.
	Kinds []string
	// When specified, only include external services with ID below this number
	// (because we're sorting results by ID in descending order).
	AfterID int64
	*LimitOffset
}

func (o ExternalServicesListOptions) sqlConditions() []*sqlf.Query {
	conds := []*sqlf.Query{sqlf.Sprintf("deleted_at IS NULL")}
	if o.NoNamespace {
		conds = append(conds, sqlf.Sprintf(`namespace_user_id IS NULL`))
	} else if o.NamespaceUserID > 0 {
		conds = append(conds, sqlf.Sprintf(`namespace_user_id = %d`, o.NamespaceUserID))
	}
	if len(o.Kinds) > 0 {
		kinds := make([]*sqlf.Query, 0, len(o.Kinds))
		for _, kind := range o.Kinds {
			kinds = append(kinds, sqlf.Sprintf("%s", kind))
		}
		conds = append(conds, sqlf.Sprintf("kind IN (%s)", sqlf.Join(kinds, ",")))
	}
	if o.AfterID > 0 {
		conds = append(conds, sqlf.Sprintf(`id < %d`, o.AfterID))
	}
	return conds
}

type ValidateExternalServiceConfigOptions struct {
	// The ID of the external service, 0 is a valid value for not-yet-created external service.
	ID int64
	// The kind of external service.
	Kind string
	// The actual config of the external service.
	Config string
	// The list of authN providers configured on the instance.
	AuthProviders []schema.AuthProviders
	// When true, indicates this is a user-added the external service.
	HasNamespace bool
}

// ValidateConfig validates the given external service configuration.
// A non zero id indicates we are updating an existing service, 0 indicates we are adding a new one.
func (e *ExternalServicesStore) ValidateConfig(ctx context.Context, opt ValidateExternalServiceConfigOptions) error {
	// For user-added external services, we need to prevent them from using disallowed fields.
	if opt.HasNamespace {
		// We do not allow users to add external service other than GitHub.com, GitLab.com and Bitbucket.org
		result := gjson.Get(opt.Config, "url")
		baseURL, err := url.Parse(result.String())
		if err != nil {
			return errors.Wrap(err, "parse base URL")
		}
		normalizedURL := extsvc.NormalizeBaseURL(baseURL).String()
		if normalizedURL != "https://github.com/" &&
			normalizedURL != "https://gitlab.com/" &&
			normalizedURL != "https://bitbucket.org/" {
			return errors.New("users are only allowed to add external service for https://github.com/, https://gitlab.com/ and https://bitbucket.org/")
		}

		disallowedFields := []string{"repositoryPathPattern"}
		results := gjson.GetMany(opt.Config, disallowedFields...)
		for i, r := range results {
			if r.Exists() {
				return errors.Errorf("field %q is not allowed in a user-added external service", disallowedFields[i])
			}
		}
	}

	ext, ok := ExternalServiceKinds[opt.Kind]
	if !ok {
		return fmt.Errorf("invalid external service kind: %s", opt.Kind)
	}

	// All configs must be valid JSON.
	// If this requirement is ever changed, you will need to update
	// serveExternalServiceConfigs to handle this case.

	sl := gojsonschema.NewSchemaLoader()
	sc, err := sl.Compile(gojsonschema.NewStringLoader(ext.JSONSchema))
	if err != nil {
		return errors.Wrapf(err, "unable to compile schema for external service of kind %q", opt.Kind)
	}

	normalized, err := jsonc.Parse(opt.Config)
	if err != nil {
		return errors.Wrapf(err, "unable to normalize JSON")
	}

	res, err := sc.Validate(gojsonschema.NewBytesLoader(normalized))
	if err != nil {
		return errors.Wrap(err, "unable to validate config against schema")
	}

	var errs *multierror.Error
	for _, err := range res.Errors() {
		e := err.String()
		// Remove `(root): ` from error formatting since these errors are
		// presented to users.
		e = strings.TrimPrefix(e, "(root): ")
		errs = multierror.Append(errs, errors.New(e))
	}

	// Extra validation not based on JSON Schema.
	switch opt.Kind {
	case extsvc.KindGitHub:
		var c schema.GitHubConnection
		if err = json.Unmarshal(normalized, &c); err != nil {
			return err
		}
		err = e.validateGitHubConnection(ctx, opt.ID, &c)

	case extsvc.KindGitLab:
		var c schema.GitLabConnection
		if err = json.Unmarshal(normalized, &c); err != nil {
			return err
		}
		err = e.validateGitLabConnection(ctx, opt.ID, &c, opt.AuthProviders)

	case extsvc.KindBitbucketServer:
		var c schema.BitbucketServerConnection
		if err = json.Unmarshal(normalized, &c); err != nil {
			return err
		}
		err = e.validateBitbucketServerConnection(ctx, opt.ID, &c)

	case extsvc.KindBitbucketCloud:
		var c schema.BitbucketCloudConnection
		if err = json.Unmarshal(normalized, &c); err != nil {
			return err
		}
		err = e.validateBitbucketCloudConnection(ctx, opt.ID, &c)

	case extsvc.KindOther:
		var c schema.OtherExternalServiceConnection
		if err = json.Unmarshal(normalized, &c); err != nil {
			return err
		}
		err = validateOtherExternalServiceConnection(&c)
	}

	return multierror.Append(errs, err).ErrorOrNil()
}

// Neither our JSON schema library nor the Monaco editor we use supports
// object dependencies well, so we must validate here that repo items
// match the uri-reference format when url is set, instead of uri when
// it isn't.
func validateOtherExternalServiceConnection(c *schema.OtherExternalServiceConnection) error {
	parseRepo := url.Parse
	if c.Url != "" {
		// We ignore the error because this already validated by JSON Schema.
		baseURL, _ := url.Parse(c.Url)
		parseRepo = baseURL.Parse
	}

	for i, repo := range c.Repos {
		cloneURL, err := parseRepo(repo)
		if err != nil {
			return fmt.Errorf(`repos.%d: %s`, i, err)
		}

		switch cloneURL.Scheme {
		case "git", "http", "https", "ssh":
			continue
		default:
			return fmt.Errorf("repos.%d: scheme %q not one of git, http, https or ssh", i, cloneURL.Scheme)
		}
	}

	return nil
}

func (e *ExternalServicesStore) validateGitHubConnection(ctx context.Context, id int64, c *schema.GitHubConnection) error {
	err := new(multierror.Error)
	for _, validate := range e.GitHubValidators {
		err = multierror.Append(err, validate(c))
	}

	if c.Repos == nil && c.RepositoryQuery == nil && c.Orgs == nil {
		err = multierror.Append(err, errors.New("at least one of repositoryQuery, repos or orgs must be set"))
	}

	err = multierror.Append(err, e.validateDuplicateRateLimits(ctx, id, extsvc.KindGitHub, c))

	return err.ErrorOrNil()
}

func (e *ExternalServicesStore) validateGitLabConnection(ctx context.Context, id int64, c *schema.GitLabConnection, ps []schema.AuthProviders) error {
	err := new(multierror.Error)
	for _, validate := range e.GitLabValidators {
		err = multierror.Append(err, validate(c, ps))
	}

	err = multierror.Append(err, e.validateDuplicateRateLimits(ctx, id, extsvc.KindGitLab, c))

	return err.ErrorOrNil()
}

func (e *ExternalServicesStore) validateBitbucketServerConnection(ctx context.Context, id int64, c *schema.BitbucketServerConnection) error {
	err := new(multierror.Error)
	for _, validate := range e.BitbucketServerValidators {
		err = multierror.Append(err, validate(c))
	}

	if c.Repos == nil && c.RepositoryQuery == nil {
		err = multierror.Append(err, errors.New("at least one of repositoryQuery or repos must be set"))
	}

	err = multierror.Append(err, e.validateDuplicateRateLimits(ctx, id, extsvc.KindBitbucketServer, c))

	return err.ErrorOrNil()
}

func (e *ExternalServicesStore) validateBitbucketCloudConnection(ctx context.Context, id int64, c *schema.BitbucketCloudConnection) error {
	return e.validateDuplicateRateLimits(ctx, id, extsvc.KindBitbucketCloud, c)
}

// validateDuplicateRateLimits returns an error if given config has duplicated non-default rate limit
// with another external service for the same code host.
func (e *ExternalServicesStore) validateDuplicateRateLimits(ctx context.Context, id int64, kind string, parsedConfig interface{}) error {
	// Check if rate limit is already defined for this code host on another external service
	rlc, err := extsvc.GetLimitFromConfig(kind, parsedConfig)
	if err != nil {
		return errors.Wrap(err, "getting rate limit config")
	}

	// Default implies that no overriding rate limit has been set so it can't conflict with anything
	if rlc.IsDefault {
		return nil
	}

	baseURL := rlc.BaseURL
	opt := ExternalServicesListOptions{
		Kinds: []string{kind},
		LimitOffset: &LimitOffset{
			Limit: 500, // The number is randomly chosen
		},
	}
	for {
		svcs, err := e.List(ctx, opt)
		if err != nil {
			return errors.Wrap(err, "list")
		}
		if len(svcs) == 0 {
			break // No more results, exiting
		}
		opt.AfterID = svcs[len(svcs)-1].ID // Advance the cursor

		for _, svc := range svcs {
			rlc, err := extsvc.ExtractRateLimitConfig(svc.Config, svc.Kind, svc.DisplayName)
			if err != nil {
				return errors.Wrap(err, "extracting rate limit config")
			}
			if rlc.BaseURL == baseURL && svc.ID != id && !rlc.IsDefault {
				return fmt.Errorf("existing external service, %q, already has a rate limit set", rlc.DisplayName)
			}
		}

		if len(svcs) < opt.Limit {
			break // Less results than limit means we've reached end
		}
	}
	return nil
}

// Create creates an external service.
//
// Since this method is used before the configuration server has started
// (search for "EXTSVC_CONFIG_FILE") you must pass the conf.Get function in so
// that an alternative can be used when the configuration server has not
// started, otherwise a panic would occur once pkg/conf's deadlock detector
// determines a deadlock occurred.
//
// 🚨 SECURITY: The caller must ensure that the actor is a site admin.
// Otherwise, `es.NamespaceUserID` must be specified (i.e. non-nil) for
// a user-added external service.
func (e *ExternalServicesStore) Create(ctx context.Context, confGet func() *conf.Unified, es *types.ExternalService) error {
	if Mocks.ExternalServices.Create != nil {
		return Mocks.ExternalServices.Create(ctx, confGet, es)
	}

	ps := confGet().AuthProviders
	if err := e.ValidateConfig(ctx, ValidateExternalServiceConfigOptions{
		Kind:          es.Kind,
		Config:        es.Config,
		AuthProviders: ps,
		HasNamespace:  es.NamespaceUserID != nil,
	}); err != nil {
		return err
	}

	es.CreatedAt = time.Now().UTC().Truncate(time.Microsecond)
	es.UpdatedAt = es.CreatedAt

	// Prior to saving the record, run a validation hook.
	if e.PreCreateExternalService != nil {
		if err := e.PreCreateExternalService(ctx); err != nil {
			return err
		}
	}

	return dbconn.Global.QueryRowContext(
		ctx,
		"INSERT INTO external_services(kind, display_name, config, created_at, updated_at, namespace_user_id) VALUES($1, $2, $3, $4, $5, $6) RETURNING id",
		es.Kind, es.DisplayName, secret.StringValue{S: &es.Config}, es.CreatedAt, es.UpdatedAt, es.NamespaceUserID,
	).Scan(&es.ID)
}

// ExternalServiceUpdate contains optional fields to update.
type ExternalServiceUpdate struct {
	DisplayName *string
	Config      *string
}

// Update updates an external service.
//
// 🚨 SECURITY: The caller must ensure that the actor is a site admin,
// or has the legitimate access to the external service (i.e. the owner).
func (e *ExternalServicesStore) Update(ctx context.Context, ps []schema.AuthProviders, id int64, update *ExternalServiceUpdate) error {
	if Mocks.ExternalServices.Update != nil {
		return Mocks.ExternalServices.Update(ctx, ps, id, update)
	}

	if update.Config != nil {
		// Query to get the kind (which is immutable) so we can validate the new config.
		externalService, err := e.GetByID(ctx, id)
		if err != nil {
			return err
		}

		if err := e.ValidateConfig(ctx, ValidateExternalServiceConfigOptions{
			ID:            id,
			Kind:          externalService.Kind,
			Config:        *update.Config,
			AuthProviders: ps,
			HasNamespace:  externalService.NamespaceUserID != nil,
		}); err != nil {
			return err
		}
	}

	execUpdate := func(ctx context.Context, tx *sql.Tx, update *sqlf.Query) error {
		q := sqlf.Sprintf("UPDATE external_services SET %s, updated_at=now() WHERE id=%d AND deleted_at IS NULL", update, id)
		res, err := tx.ExecContext(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...)
		if err != nil {
			return err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return externalServiceNotFoundError{id: id}
		}
		return nil
	}
	return dbutil.Transaction(ctx, dbconn.Global, func(tx *sql.Tx) error {
		if update.DisplayName != nil {
			if err := execUpdate(ctx, tx, sqlf.Sprintf("display_name=%s", update.DisplayName)); err != nil {
				return err
			}
		}
		if update.Config != nil {
			if err := execUpdate(ctx, tx, sqlf.Sprintf("config=%s, next_sync_at=now()", secret.StringValue{S: update.Config})); err != nil {
				return err
			}
		}
		return nil
	})
}

type externalServiceNotFoundError struct {
	id int64
}

func (e externalServiceNotFoundError) Error() string {
	return fmt.Sprintf("external service not found: %v", e.id)
}

func (e externalServiceNotFoundError) NotFound() bool {
	return true
}

// Delete deletes an external service.
//
// 🚨 SECURITY: The caller must ensure that the actor is a site admin.
func (*ExternalServicesStore) Delete(ctx context.Context, id int64) error {
	if Mocks.ExternalServices.Delete != nil {
		return Mocks.ExternalServices.Delete(ctx, id)
	}

	res, err := dbconn.Global.ExecContext(ctx, "UPDATE external_services SET deleted_at=now() WHERE id=$1 AND deleted_at IS NULL", id)
	if err != nil {
		return err
	}
	nrows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if nrows == 0 {
		return externalServiceNotFoundError{id: id}
	}
	return nil
}

// GetByID returns the external service for id.
//
// 🚨 SECURITY: The caller must ensure that the actor is a site admin.
func (e *ExternalServicesStore) GetByID(ctx context.Context, id int64) (*types.ExternalService, error) {
	if Mocks.ExternalServices.GetByID != nil {
		return Mocks.ExternalServices.GetByID(id)
	}

	conds := []*sqlf.Query{
		sqlf.Sprintf("deleted_at IS NULL"),
		sqlf.Sprintf("id=%d", id),
	}
	ess, err := e.list(ctx, conds, nil)
	if err != nil {
		return nil, err
	}
	if len(ess) == 0 {
		return nil, externalServiceNotFoundError{id: id}
	}
	return ess[0], nil
}

// List returns external services under given namespace.
// If no namespace is given, it returns all external services.
//
// 🚨 SECURITY: The caller must ensure one of the following:
// 	- The actor is a site admin
// 	- The opt.NamespaceUserID is same as authenticated user ID (i.e. actor.UID)
func (e *ExternalServicesStore) List(ctx context.Context, opt ExternalServicesListOptions) ([]*types.ExternalService, error) {
	if Mocks.ExternalServices.List != nil {
		return Mocks.ExternalServices.List(opt)
	}
	return e.list(ctx, opt.sqlConditions(), opt.LimitOffset)
}

// DistinctKinds returns the distinct list of external services kinds that are stored in the database.
func (e *ExternalServicesStore) DistinctKinds(ctx context.Context) ([]string, error) {
	q := sqlf.Sprintf(`
SELECT ARRAY_AGG(DISTINCT(kind)::TEXT)
FROM external_services
WHERE deleted_at IS NULL
`)

	var kinds []string
	err := dbconn.Global.QueryRowContext(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...).Scan(pq.Array(&kinds))
	if err != nil {
		if err == sql.ErrNoRows {
			return []string{}, nil
		}
		return nil, err
	}

	return kinds, nil
}

func (*ExternalServicesStore) list(ctx context.Context, conds []*sqlf.Query, limitOffset *LimitOffset) ([]*types.ExternalService, error) {
	q := sqlf.Sprintf(`
		SELECT id, kind, display_name, config, created_at, updated_at, deleted_at, last_sync_at, next_sync_at, namespace_user_id
		FROM external_services
		WHERE (%s)
		ORDER BY id DESC
		%s`,
		sqlf.Join(conds, ") AND ("),
		limitOffset.SQL(),
	)

	rows, err := dbconn.Global.QueryContext(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*types.ExternalService
	for rows.Next() {
		var (
			h              types.ExternalService
			deletedAt      sql.NullTime
			lastSyncAt     sql.NullTime
			nextSyncAt     sql.NullTime
			namepaceUserID sql.NullInt32
		)
		if err := rows.Scan(&h.ID, &h.Kind, &h.DisplayName, &secret.StringValue{S: &h.Config}, &h.CreatedAt, &h.UpdatedAt, &deletedAt, &lastSyncAt, &nextSyncAt, &namepaceUserID); err != nil {
			return nil, err
		}

		if deletedAt.Valid {
			h.DeletedAt = &deletedAt.Time
		}
		if lastSyncAt.Valid {
			h.LastSyncAt = &lastSyncAt.Time
		}
		if nextSyncAt.Valid {
			h.NextSyncAt = &nextSyncAt.Time
		}
		if namepaceUserID.Valid {
			h.NamespaceUserID = &namepaceUserID.Int32
		}
		results = append(results, &h)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// Count counts all external services that satisfy the options (ignoring limit and offset).
//
// 🚨 SECURITY: The caller must ensure that the actor is a site admin.
func (*ExternalServicesStore) Count(ctx context.Context, opt ExternalServicesListOptions) (int, error) {
	if Mocks.ExternalServices.Count != nil {
		return Mocks.ExternalServices.Count(ctx, opt)
	}

	q := sqlf.Sprintf("SELECT COUNT(*) FROM external_services WHERE (%s)", sqlf.Join(opt.sqlConditions(), ") AND ("))
	var count int
	if err := dbconn.Global.QueryRowContext(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// MockExternalServices mocks the external services store.
type MockExternalServices struct {
	Create  func(ctx context.Context, confGet func() *conf.Unified, externalService *types.ExternalService) error
	Delete  func(ctx context.Context, id int64) error
	GetByID func(id int64) (*types.ExternalService, error)
	List    func(opt ExternalServicesListOptions) ([]*types.ExternalService, error)
	Update  func(ctx context.Context, ps []schema.AuthProviders, id int64, update *ExternalServiceUpdate) error
	Count   func(ctx context.Context, opt ExternalServicesListOptions) (int, error)
}
