package client

import (
	"context"
	"fmt"
	"net/url"

	"github.com/tom-redmine/go-redmine-cli/internal/models"
)

// Group represents a Redmine group.
type Group = models.Group

// GroupListResponse is the response from listing groups.
type GroupListResponse = models.GroupList

// GroupCreateRequest is the request body for creating a group.
type GroupCreateRequest = models.GroupCreate

// GroupUpdateRequest is the request body for updating a group.
type GroupUpdateRequest = models.GroupUpdateRequest

// ListGroups returns all groups.
func (c *Client) ListGroups(ctx context.Context, opts ListOpts) (*GroupListResponse, error) {
	var result GroupListResponse
	if err := c.get(ctx, "/groups.json", opts.Params(), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListGroupsBuiltin returns all groups including built-in ones (admins only).
func (c *Client) ListGroupsBuiltin(ctx context.Context, opts ListOpts) (*GroupListResponse, error) {
	params := opts.Params()
	params.Set("builtin", "1")
	var result GroupListResponse
	if err := c.get(ctx, "/groups.json", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetGroup returns a single group by ID.
func (c *Client) GetGroup(ctx context.Context, id int, include string) (*Group, error) {
	path := fmt.Sprintf("/groups/%d.json", id)
	params := url.Values{}
	if include != "" {
		params.Set("include", include)
	}
	var result struct {
		Group Group `json:"group"`
	}
	if err := c.get(ctx, path, params, &result); err != nil {
		return nil, err
	}
	return &result.Group, nil
}

// CreateGroup creates a new group.
func (c *Client) CreateGroup(ctx context.Context, req GroupCreateRequest) (*Group, error) {
	var result struct {
		Group Group `json:"group"`
	}
	if err := c.post(ctx, "/groups.json", req, &result); err != nil {
		return nil, err
	}
	return &result.Group, nil
}

// UpdateGroup updates a group.
func (c *Client) UpdateGroup(ctx context.Context, id int, req GroupUpdateRequest) (*Group, error) {
	path := fmt.Sprintf("/groups/%d.json", id)
	var result struct {
		Group Group `json:"group"`
	}
	if err := c.put(ctx, path, req, &result); err != nil {
		return nil, err
	}
	return &result.Group, nil
}

// DeleteGroup deletes a group.
func (c *Client) DeleteGroup(ctx context.Context, id int) error {
	return c.delete(ctx, fmt.Sprintf("/groups/%d.json", id))
}
