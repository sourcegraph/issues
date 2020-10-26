package extsvc

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	otlog "github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/jsonc"
	"github.com/sourcegraph/sourcegraph/schema"
	"golang.org/x/time/rate"
)

// Account represents a row in the `user_external_accounts` table. See the GraphQL API's
// corresponding fields in "ExternalAccount" for documentation.
type Account struct {
	ID          int32
	UserID      int32
	AccountSpec // ServiceType, ServiceID, ClientID, AccountID
	AccountData // AuthData, Data
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// AccountSpec specifies a user external account by its external identifier (i.e., by the
// identifier provided by the account's owner service), instead of by our database's serial
// ID. See the GraphQL API's corresponding fields in "ExternalAccount" for documentation.
type AccountSpec struct {
	ServiceType string
	ServiceID   string
	ClientID    string
	AccountID   string
}

// AccountData contains data that can be freely updated in the user external account after it
// has been created. See the GraphQL API's corresponding fields for documentation.
type AccountData struct {
	AuthData *json.RawMessage
	Data     *json.RawMessage
}

// Repository contains necessary information to identify an external repository on the code host.
type Repository struct {
	// URI is the full name for this repository, e.g. "github.com/user/repo".
	URI string
	api.ExternalRepoSpec
}

// Accounts contains a list of accounts that belong to the same external service.
// All fields have a same meaning to AccountSpec. See GraphQL API's corresponding fields
// in "ExternalAccount" for documentation.
type Accounts struct {
	ServiceType string
	ServiceID   string
	AccountIDs  []string
}

// TracingFields returns tracing fields for the opentracing log.
func (s *Accounts) TracingFields() []otlog.Field {
	return []otlog.Field{
		otlog.String("Accounts.ServiceType", s.ServiceType),
		otlog.String("Accounts.Perm", s.ServiceID),
		otlog.Int("Accounts.AccountIDs.Count", len(s.AccountIDs)),
	}
}

const (
	// The constants below represent the different kinds of external service we support and should be used
	// in preference to the Type values below.

	KindAWSCodeCommit   = "AWSCODECOMMIT"
	KindBitbucketServer = "BITBUCKETSERVER"
	KindBitbucketCloud  = "BITBUCKETCLOUD"
	KindGitHub          = "GITHUB"
	KindGitLab          = "GITLAB"
	KindGitolite        = "GITOLITE"
	KindPhabricator     = "PHABRICATOR"
	KindOther           = "OTHER"
)

const (
	// The constants below represent the values used for the external_service_type column of the repo table.

	// TypeAWSCodeCommit is the (api.ExternalRepoSpec).ServiceType value for AWS CodeCommit
	// repositories. The ServiceID value is the ARN (Amazon Resource Name) omitting the repository name
	// suffix (e.g., "arn:aws:codecommit:us-west-1:123456789:").
	TypeAWSCodeCommit = "awscodecommit"

	// TypeBitbucketServer is the (api.ExternalRepoSpec).ServiceType value for Bitbucket Server projects. The
	// ServiceID value is the base URL to the Bitbucket Server instance.
	TypeBitbucketServer = "bitbucketServer"

	// TypeBitbucketCloud is the (api.ExternalRepoSpec).ServiceType value for Bitbucket Cloud projects. The
	// ServiceID value is the base URL to the Bitbucket Cloud.
	TypeBitbucketCloud = "bitbucketCloud"

	// TypeGitHub is the (api.ExternalRepoSpec).ServiceType value for GitHub repositories. The ServiceID value
	// is the base URL to the GitHub instance (https://github.com or the GitHub Enterprise URL).
	TypeGitHub = "github"

	// TypeGitLab is the (api.ExternalRepoSpec).ServiceType value for GitLab projects. The ServiceID
	// value is the base URL to the GitLab instance (https://gitlab.com or self-hosted GitLab URL).
	TypeGitLab = "gitlab"

	// TypeGitolite is the (api.ExternalRepoSpec).ServiceType value for Gitolite projects.
	TypeGitolite = "gitolite"

	// TypePhabricator is the (api.ExternalRepoSpec).ServiceType value for Phabricator projects.
	TypePhabricator = "phabricator"

	// TypeOther is the (api.ExternalRepoSpec).ServiceType value for other projects.
	TypeOther = "other"
)

// KindToType returns a Type constants given a Kind
// It will panic when given an unknown kind
func KindToType(kind string) string {
	switch kind {
	case KindAWSCodeCommit:
		return TypeAWSCodeCommit
	case KindBitbucketServer:
		return TypeBitbucketServer
	case KindBitbucketCloud:
		return TypeBitbucketCloud
	case KindGitHub:
		return TypeGitHub
	case KindGitLab:
		return TypeGitLab
	case KindGitolite:
		return TypeGitolite
	case KindPhabricator:
		return TypePhabricator
	case KindOther:
		return TypeOther
	default:
		panic(fmt.Sprintf("unknown kind: %q", kind))
	}
}

var (
	// Precompute these for use in ParseServiceType below since the constants are mixed case
	bbsLower = strings.ToLower(TypeBitbucketServer)
	bbcLower = strings.ToLower(TypeBitbucketCloud)
)

// ParseServiceType will return a ServiceType constant after doing a case insensitive match on s.
// It returns ("", false) if no match was found.
func ParseServiceType(s string) (string, bool) {
	switch strings.ToLower(s) {
	case TypeAWSCodeCommit:
		return TypeAWSCodeCommit, true
	case bbsLower:
		return TypeBitbucketServer, true
	case bbcLower:
		return TypeBitbucketCloud, true
	case TypeGitHub:
		return TypeGitHub, true
	case TypeGitLab:
		return TypeGitLab, true
	case TypeGitolite:
		return TypeGitolite, true
	case TypePhabricator:
		return TypePhabricator, true
	case TypeOther:
		return TypeOther, true
	default:
		return "", false
	}
}

// AccountID is a descriptive type for the external identifier of an external account on the
// code host. It can be the string representation of an integer (e.g. GitLab), a GraphQL ID
// (e.g. GitHub), or a username (e.g. Bitbucket Server) depends on the code host type.
type AccountID string

// RepoID is a descriptive type for the external identifier of an external repository on the
// code host. It can be the string representation of an integer (e.g. GitLab and Bitbucket
// Server) or a GraphQL ID (e.g. GitHub) depends on the code host type.
type RepoID string

// ParseConfig attempts to unmarshal the given JSON config into a configuration struct defined in the schema package.
func ParseConfig(kind, config string) (cfg interface{}, _ error) {
	switch strings.ToUpper(kind) {
	case KindAWSCodeCommit:
		cfg = &schema.AWSCodeCommitConnection{}
	case KindBitbucketServer:
		cfg = &schema.BitbucketServerConnection{}
	case KindBitbucketCloud:
		cfg = &schema.BitbucketCloudConnection{}
	case KindGitHub:
		cfg = &schema.GitHubConnection{}
	case KindGitLab:
		cfg = &schema.GitLabConnection{}
	case KindGitolite:
		cfg = &schema.GitoliteConnection{}
	case KindPhabricator:
		cfg = &schema.PhabricatorConnection{}
	case KindOther:
		cfg = &schema.OtherExternalServiceConnection{}
	default:
		return nil, fmt.Errorf("unknown external service kind %q", kind)
	}
	return cfg, jsonc.Unmarshal(config, cfg)
}

const IDParam = "externalServiceID"

func WebhookURL(kind string, externalServiceID int64, externalURL string) string {
	var path string
	switch strings.ToUpper(kind) {
	case KindGitHub:
		path = "github-webhooks"
	case KindBitbucketServer:
		path = "bitbucket-server-webhooks"
	case KindGitLab:
		path = "gitlab-webhooks"
	default:
		return ""
	}
	// eg. https://example.com/.api/github-webhooks?externalServiceID=1
	return fmt.Sprintf("%s/.api/%s?%s=%d", externalURL, path, IDParam, externalServiceID)
}

// ExtractRateLimitConfig extracts the rate limit config from the given args. If rate limiting is not
// supported the error returned will be an ErrRateLimitUnsupported.
func ExtractRateLimitConfig(config, kind, displayName string) (RateLimitConfig, error) {
	parsed, err := ParseConfig(kind, config)
	if err != nil {
		return RateLimitConfig{}, errors.Wrap(err, "loading service configuration")
	}

	rlc, err := GetLimitFromConfig(kind, parsed)
	if err != nil {
		return RateLimitConfig{}, err
	}
	rlc.DisplayName = displayName

	return rlc, nil
}

// ExtractBaseURL will extract the normalised base URL from the given config
// based on the vale of kind
func ExtractBaseURL(kind, config string) (*url.URL, error) {
	cfg, err := ParseConfig(kind, config)
	if err != nil {
		return nil, errors.Wrap(err, "parsing config")
	}

	var rawURL string
	switch c := cfg.(type) {
	case *schema.AWSCodeCommitConnection:
		return nil, errors.New("BaseURL unavailable for AWSCodeCommit")
	case *schema.BitbucketServerConnection:
		rawURL = c.Url
	case *schema.GitHubConnection:
		rawURL = c.Url
	case *schema.GitLabConnection:
		rawURL = c.Url
	case *schema.GitoliteConnection:
		rawURL = c.Host
	case *schema.PhabricatorConnection:
		rawURL = c.Url
	case *schema.OtherExternalServiceConnection:
		rawURL = c.Url
	default:
		return nil, fmt.Errorf("unknown external service type %T", config)
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, errors.Wrap(err, "parsing service URL")
	}

	return NormalizeBaseURL(parsed), nil
}

// RateLimitConfig represents the internal rate limit configured for an external service
type RateLimitConfig struct {
	BaseURL     string
	DisplayName string
	Limit       rate.Limit
	IsDefault   bool
}

// GetLimitFromConfig gets RateLimitConfig from an already parsed config schema.
func GetLimitFromConfig(kind string, config interface{}) (rlc RateLimitConfig, err error) {
	// Rate limit config can be in a few states:
	// 1. Not defined: We fall back to default specified in code.
	// 2. Defined and enabled: We use their defined limit.
	// 3. Defined and disabled: We use an infinite limiter.

	rlc.IsDefault = true
	switch c := config.(type) {
	case *schema.GitLabConnection:
		// 10/s is the default enforced by GitLab on their end
		rlc.Limit = rate.Limit(10)
		if c != nil && c.RateLimit != nil {
			rlc.Limit = limitOrInf(c.RateLimit.Enabled, c.RateLimit.RequestsPerHour)
			rlc.IsDefault = false
		}
		rlc.BaseURL = c.Url
	case *schema.GitHubConnection:
		// 5000 per hour is the default enforced by GitHub on their end
		rlc.Limit = rate.Limit(5000.0 / 3600.0)
		if c != nil && c.RateLimit != nil {
			rlc.Limit = limitOrInf(c.RateLimit.Enabled, c.RateLimit.RequestsPerHour)
			rlc.IsDefault = false
		}
		rlc.BaseURL = c.Url
	case *schema.BitbucketServerConnection:
		// 8/s is the default limit we enforce
		rlc.Limit = rate.Limit(8)
		if c != nil && c.RateLimit != nil {
			rlc.Limit = limitOrInf(c.RateLimit.Enabled, c.RateLimit.RequestsPerHour)
			rlc.IsDefault = false
		}
		rlc.BaseURL = c.Url
	case *schema.BitbucketCloudConnection:
		// 2/s is the default limit we enforce
		rlc.Limit = rate.Limit(2)
		if c != nil && c.RateLimit != nil {
			rlc.Limit = limitOrInf(c.RateLimit.Enabled, c.RateLimit.RequestsPerHour)
			rlc.IsDefault = false
		}
		rlc.BaseURL = c.Url
	default:
		return rlc, ErrRateLimitUnsupported{codehostKind: kind}
	}

	u, err := url.Parse(rlc.BaseURL)
	if err != nil {
		return rlc, errors.Wrap(err, "parsing external service URL")
	}

	rlc.BaseURL = NormalizeBaseURL(u).String()

	return rlc, nil
}

func limitOrInf(enabled bool, perHour float64) rate.Limit {
	if enabled {
		return rate.Limit(perHour / 3600)
	}
	return rate.Inf
}

type ErrRateLimitUnsupported struct {
	codehostKind string
}

func (e ErrRateLimitUnsupported) Error() string {
	return fmt.Sprintf("internal rate limiting not supported for %s", e.codehostKind)
}

// URN returns a unique resource identifier of an external service by given kind and ID.
func URN(kind string, id int64) string {
	return "extsvc:" + strings.ToLower(kind) + ":" + strconv.FormatInt(id, 10)
}
