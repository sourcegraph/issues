package gqltestutil

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"

	"github.com/sourcegraph/sourcegraph/internal/jsonc"
	"github.com/sourcegraph/sourcegraph/schema"
)

// OverwriteSettings overwrites settings for given subject ID with contents.
func (c *Client) OverwriteSettings(subjectID, contents string) error {
	lastID, err := c.lastSettingsID(subjectID)
	if err != nil {
		return errors.Wrap(err, "get last settings ID")
	}

	const query = `
mutation OverwriteSettings($subject: ID!, $lastID: Int, $contents: String!) {
	settingsMutation(input: { subject: $subject, lastID: $lastID }) {
		overwriteSettings(contents: $contents) {
			empty {
				alwaysNil
			}
		}
	}
}
`
	variables := map[string]interface{}{
		"subject":  subjectID,
		"lastID":   lastID,
		"contents": contents,
	}
	err = c.GraphQL("", "", query, variables, nil)
	if err != nil {
		return errors.Wrap(err, "request GraphQL")
	}
	return nil
}

// lastSettingsID returns the ID of last settings of given subject.
// It is required to be used to update corresponding settings.
func (c *Client) lastSettingsID(subjectID string) (int, error) {
	const query = `
query ViewerSettings {
	viewerSettings {
		subjects {
			id
			latestSettings {
				id
			}
		}
	}
}
`
	var resp struct {
		Data struct {
			ViewerSettings struct {
				Subjects []struct {
					ID             string `json:"id"`
					LatestSettings *struct {
						ID int
					} `json:"latestSettings"`
				} `json:"subjects"`
			} `json:"viewerSettings"`
		} `json:"data"`
	}
	err := c.GraphQL("", "", query, nil, &resp)
	if err != nil {
		return 0, errors.Wrap(err, "request GraphQL")
	}

	lastID := 0
	for _, s := range resp.Data.ViewerSettings.Subjects {
		if s.ID != subjectID {
			continue
		}

		// It is nil in the initial state, which effectively makes lastID as 0.
		if s.LatestSettings != nil {
			lastID = s.LatestSettings.ID
		}
		break
	}
	return lastID, nil
}

// ViewerSettings returns the latest cascaded settings of authenticated user.
func (c *Client) ViewerSettings() (string, error) {
	const query = `
query ViewerSettings {
	viewerSettings {
		final
	}
}
`
	var resp struct {
		Data struct {
			ViewerSettings struct {
				Final string `json:"final"`
			} `json:"viewerSettings"`
		} `json:"data"`
	}
	err := c.GraphQL("", "", query, nil, &resp)
	if err != nil {
		return "", errors.Wrap(err, "request GraphQL")
	}
	return resp.Data.ViewerSettings.Final, nil
}

// SiteConfiguration returns current effective site configuration.
//
// This method requires the authenticated user to be a site admin.
func (c *Client) SiteConfiguration() (*schema.SiteConfiguration, error) {
	const query = `
query Site {
	site {
		configuration {
			effectiveContents
		}
	}
}
`

	var resp struct {
		Data struct {
			Site struct {
				Configuration struct {
					EffectiveContents string `json:"effectiveContents"`
				} `json:"configuration"`
			} `json:"site"`
		} `json:"data"`
	}
	err := c.GraphQL("", "", query, nil, &resp)
	if err != nil {
		return nil, errors.Wrap(err, "request GraphQL")
	}

	config := new(schema.SiteConfiguration)
	err = jsonc.Unmarshal(resp.Data.Site.Configuration.EffectiveContents, config)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal configuration")
	}
	return config, nil
}

// UpdateSiteConfiguration updates site configuration.
//
// This method requires the authenticated user to be a site admin.
func (c *Client) UpdateSiteConfiguration(config *schema.SiteConfiguration) error {
	input, err := jsoniter.Marshal(config)
	if err != nil {
		return errors.Wrap(err, "marshal configuration")
	}

	const query = `
mutation UpdateSiteConfiguration($input: String!) {
	updateSiteConfiguration(lastID: 0, input: $input)
}
`
	variables := map[string]interface{}{
		"input": string(input),
	}
	err = c.GraphQL("", "", query, variables, nil)
	if err != nil {
		return errors.Wrap(err, "request GraphQL")
	}
	return nil
}
