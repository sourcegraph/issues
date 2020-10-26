package gqltestutil

import (
	"github.com/pkg/errors"
)

// CreateOrganization creates a new organization with given name and display name.
// It returns GraphQL node ID of newly created organization.
func (c *Client) CreateOrganization(name, displayName string) (string, error) {
	const query = `
mutation CreateOrganization($name: String!, $displayName: String) {
	createOrganization(name: $name, displayName: $displayName) {
		id
	}
}
`
	variables := map[string]interface{}{
		"name":        name,
		"displayName": displayName,
	}
	var resp struct {
		Data struct {
			CreateOrganization struct {
				ID string `json:"id"`
			} `json:"createOrganization"`
		} `json:"data"`
	}
	err := c.GraphQL("", "", query, variables, &resp)
	if err != nil {
		return "", errors.Wrap(err, "request GraphQL")
	}
	return resp.Data.CreateOrganization.ID, nil
}

// DeleteOrganization deletes the organization by given GraphQL node ID.
//
// This method requires the authenticated user to be a site admin.
func (c *Client) DeleteOrganization(id string) error {
	const query = `
mutation DeleteOrganization($organization: ID!) {
	deleteOrganization(organization: $organization) {
		alwaysNil
	}
}
`
	variables := map[string]interface{}{
		"organization": id,
	}
	err := c.GraphQL("", "", query, variables, nil)
	if err != nil {
		return errors.Wrap(err, "request GraphQL")
	}
	return nil
}

// RemoveUserFromOrganization removes user from given organization.
func (c *Client) RemoveUserFromOrganization(userID, orgID string) error {
	const query = `
mutation RemoveUserFromOrganization($user: ID!, $organization: ID!) {
	removeUserFromOrganization(user: $user, organization: $organization) {
		alwaysNil
	}
}
`
	variables := map[string]interface{}{
		"user":         userID,
		"organization": orgID,
	}
	err := c.GraphQL("", "", query, variables, nil)
	if err != nil {
		return errors.Wrap(err, "request GraphQL")
	}
	return nil
}
