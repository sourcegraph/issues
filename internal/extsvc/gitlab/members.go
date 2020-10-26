package gitlab

import (
	"context"
	"net/http"

	"github.com/peterhellberg/link"
)

// Member contains fields for a group or project membership.
type Member struct {
	ID                int32  `json:"id"`
	Username          string `json:"username"`
	Name              string `json:"name"`
	State             string `json:"state"`
	AvatarURL         string `json:"avatar_url"`
	WebURL            string `json:"web_url"`
	ExpiresAt         string `json:"expires_at"`
	AccessLevel       int    `json:"access_level"`
	GroupSAMLIdentity *struct {
		Provider       string `json:"provider"`
		ExternUID      string `json:"extern_uid"`
		SAMLProviderID int    `json:"saml_provider_id"`
	} `json:"group_saml_identity"`
}

// ListMembers returns a list of members parsed from reponse of given URL.
func (c *Client) ListMembers(ctx context.Context, urlStr string) (members []*Member, nextPageURL *string, err error) {
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, nil, err
	}
	respHeader, _, err := c.do(ctx, req, &members)
	if err != nil {
		return nil, nil, err
	}

	// Get URL to next page. See https://docs.gitlab.com/ee/api/README.html#pagination-link-header.
	if l := link.Parse(respHeader.Get("Link"))["next"]; l != nil {
		nextPageURL = &l.URI
	}

	return members, nextPageURL, nil
}
