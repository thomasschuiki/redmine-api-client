package client

import (
	"context"
	"fmt"
	"net/url"

	"github.com/tom-redmine/go-redmine-cli/internal/models"
)

// Membership represents a Redmine project membership.
type Membership = models.Membership

// MembershipListResponse is the response from listing memberships.
type MembershipListResponse = models.MembershipList

// MembershipCreateRequest is the request body for creating a membership.
type MembershipCreateRequest = models.MembershipCreate

// MembershipUpdateRequest is the request body for updating a membership.
type MembershipUpdateRequest = models.MembershipUpdateRequest

// ListMemberships returns memberships for a project.
func (c *Client) ListMemberships(ctx context.Context, projectID string, opts ListOpts) (*MembershipListResponse, error) {
	path := fmt.Sprintf("/projects/%s/memberships.json", url.PathEscape(projectID))
	params := opts.Params()
	var result MembershipListResponse
	if err := c.get(ctx, path, params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetMembership returns a single membership by ID.
func (c *Client) GetMembership(ctx context.Context, id int) (*Membership, error) {
	path := fmt.Sprintf("/memberships/%d.json", id)
	var result struct {
		Membership Membership `json:"membership"`
	}
	if err := c.get(ctx, path, nil, &result); err != nil {
		return nil, err
	}
	return &result.Membership, nil
}

// CreateMembership adds a user or group to a project.
func (c *Client) CreateMembership(ctx context.Context, projectID string, req MembershipCreateRequest) (*Membership, error) {
	path := fmt.Sprintf("/projects/%s/memberships.json", url.PathEscape(projectID))
	var result struct {
		Membership Membership `json:"membership"`
	}
	if err := c.post(ctx, path, req, &result); err != nil {
		return nil, err
	}
	return &result.Membership, nil
}

// UpdateMembership updates the roles of a membership.
func (c *Client) UpdateMembership(ctx context.Context, id int, req MembershipUpdateRequest) (*Membership, error) {
	path := fmt.Sprintf("/memberships/%d.json", id)
	var result struct {
		Membership Membership `json:"membership"`
	}
	if err := c.put(ctx, path, req, &result); err != nil {
		return nil, err
	}
	return &result.Membership, nil
}

// DeleteMembership removes a membership by ID.
func (c *Client) DeleteMembership(ctx context.Context, id int) error {
	return c.delete(ctx, fmt.Sprintf("/memberships/%d.json", id))
}
