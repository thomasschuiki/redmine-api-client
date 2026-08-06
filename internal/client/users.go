package client

import (
	"context"
	"fmt"

	"github.com/tom-redmine/go-redmine-cli/internal/models"
)

// User represents a Redmine user.
type User = models.User

// UserListResponse is the response from listing users.
type UserListResponse = models.UserList

// ListUsers returns all users.
func (c *Client) ListUsers(ctx context.Context, opts ListOpts) (*UserListResponse, error) {
	params := opts.Params()
	var result UserListResponse
	if err := c.get(ctx, "/users.json", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetUser returns a single user by ID.
func (c *Client) GetUser(ctx context.Context, id int) (*User, error) {
	var result struct {
		User User `json:"user"`
	}
	path := fmt.Sprintf("/users/%d.json", id)
	if err := c.get(ctx, path, nil, &result); err != nil {
		return nil, err
	}
	return &result.User, nil
}

// GetCurrentUser returns the currently authenticated user.
func (c *Client) GetCurrentUser(ctx context.Context) (*User, error) {
	var result struct {
		User User `json:"user"`
	}
	if err := c.get(ctx, "/users/current.json", nil, &result); err != nil {
		return nil, err
	}
	return &result.User, nil
}
