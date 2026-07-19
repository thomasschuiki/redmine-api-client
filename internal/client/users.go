package client

import "fmt"

// User represents a Redmine user.
type User struct {
	ID        int        `json:"id"`
	Login     string     `json:"login"`
	FirstName string     `json:"firstname"`
	LastName  string     `json:"lastname"`
	Email     string     `json:"mail"`
	Status    int        `json:"status"`
	CreatedOn string     `json:"created_on,omitempty"`
	UpdatedOn string     `json:"updated_on,omitempty"`
	LastLoginOn string   `json:"last_login_on,omitempty"`
	CustomFields []CustomField `json:"custom_fields,omitempty"`
}

// UserListResponse is the response from listing users.
type UserListResponse struct {
	Users      []User `json:"users"`
	TotalCount int    `json:"total_count"`
	Offset     int    `json:"offset"`
	Limit      int    `json:"limit"`
}

// ListUsers returns all users.
func (c *Client) ListUsers(opts ListOpts) (*UserListResponse, error) {
	params := opts.Params()
	var result UserListResponse
	if err := c.get("/users.json", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetUser returns a single user by ID.
func (c *Client) GetUser(id int) (*User, error) {
	var result struct {
		User User `json:"user"`
	}
	path := fmt.Sprintf("/users/%d.json", id)
	if err := c.get(path, nil, &result); err != nil {
		return nil, err
	}
	return &result.User, nil
}

// GetCurrentUser returns the currently authenticated user.
func (c *Client) GetCurrentUser() (*User, error) {
	var result struct {
		User User `json:"user"`
	}
	if err := c.get("/users/current.json", nil, &result); err != nil {
		return nil, err
	}
	return &result.User, nil
}
