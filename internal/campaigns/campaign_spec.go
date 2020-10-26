package campaigns

import (
	"time"

	"github.com/sourcegraph/campaignutils/overridable"
	"github.com/sourcegraph/campaignutils/yaml"
	"github.com/sourcegraph/sourcegraph/schema"
)

func NewCampaignSpecFromRaw(rawSpec string) (*CampaignSpec, error) {
	c := &CampaignSpec{RawSpec: rawSpec}

	return c, c.UnmarshalValidate()
}

type CampaignSpec struct {
	ID     int64
	RandID string

	RawSpec string
	Spec    CampaignSpecFields

	NamespaceUserID int32
	NamespaceOrgID  int32

	UserID int32

	CreatedAt time.Time
	UpdatedAt time.Time
}

// Clone returns a clone of a CampaignSpec.
func (cs *CampaignSpec) Clone() *CampaignSpec {
	cc := *cs
	return &cc
}

// UnmarshalValidate unmarshals the RawSpec into Spec and validates it against
// the CampaignSpec schema and does additional semantic validation.
func (cs *CampaignSpec) UnmarshalValidate() error {
	return yaml.UnmarshalValidate(schema.CampaignSpecSchemaJSON, []byte(cs.RawSpec), &cs.Spec)
}

// CampaignSpecTTL specifies the TTL of CampaignSpecs that haven't been applied
// yet. It's set to 1 week.
const CampaignSpecTTL = 7 * 24 * time.Hour

// ExpiresAt returns the time when the CampaignSpec will be deleted if not
// applied.
func (cs *CampaignSpec) ExpiresAt() time.Time {
	return cs.CreatedAt.Add(CampaignSpecTTL)
}

type CampaignSpecFields struct {
	Name              string                    `json:"name"`
	Description       string                    `json:"description"`
	On                []CampaignSpecOn          `json:"on"`
	Steps             []CampaignSpecStep        `json:"steps"`
	ImportChangeset   []CampaignImportChangeset `json:"importChangesets,omitempty"`
	ChangesetTemplate ChangesetTemplate         `json:"changesetTemplate"`
}

type CampaignSpecOn struct {
	RepositoriesMatchingQuery string `json:"repositoriesMatchingQuery,omitempty"`
	Repository                string `json:"repository,omitempty"`
}

type CampaignSpecStep struct {
	Run       string            `json:"run"`
	Container string            `json:"container"`
	Env       map[string]string `json:"env"`
}

type CampaignImportChangeset struct {
	Repository  string        `json:"repository"`
	ExternalIDs []interface{} `json:"externalIDs"`
}

type ChangesetTemplate struct {
	Title     string                   `json:"title"`
	Body      string                   `json:"body"`
	Branch    string                   `json:"branch"`
	Commit    CommitTemplate           `json:"commit"`
	Published overridable.BoolOrString `json:"published"`
}

type CommitTemplate struct {
	Message string `json:"message"`
}
