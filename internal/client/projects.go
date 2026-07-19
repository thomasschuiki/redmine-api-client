package client

import (
	"fmt"
	"net/url"
)

// Project represents a Redmine project.
type Project struct {
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	Identifier  string     `json:"identifier"`
	Description string     `json:"description,omitempty"`
	Status      int        `json:"status"`
	IsPublic    bool       `json:"is_public"`
	CreatedOn   string     `json:"created_on,omitempty"`
	UpdatedOn   string     `json:"updated_on,omitempty"`
	CustomFields []CustomField `json:"custom_fields,omitempty"`
}

// ProjectListResponse is the response from listing projects.
type ProjectListResponse struct {
	Projects   []Project `json:"projects"`
	TotalCount int       `json:"total_count"`
	Offset     int       `json:"offset"`
	Limit      int       `json:"limit"`
}

// ProjectCreateRequest is the request body for creating a project.
type ProjectCreateRequest struct {
	Project struct {
		Name        string `json:"name"`
		Identifier  string `json:"identifier"`
		Description string `json:"description,omitempty"`
		IsPublic    *bool  `json:"is_public,omitempty"`
	} `json:"project"`
}

// ProjectUpdateRequest is the request body for updating a project.
type ProjectUpdateRequest struct {
	Project struct {
		Name        *string `json:"name,omitempty"`
		Description *string `json:"description,omitempty"`
		IsPublic    *bool   `json:"is_public,omitempty"`
	} `json:"project"`
}

// ListProjects returns all projects.
func (c *Client) ListProjects(opts ListOpts) (*ProjectListResponse, error) {
	params := opts.Params()
	var result ProjectListResponse
	if err := c.get("/projects.json", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetProject returns a single project by identifier.
func (c *Client) GetProject(identifier string) (*Project, error) {
	var result struct {
		Project Project `json:"project"`
	}
	path := fmt.Sprintf("/projects/%s.json", url.PathEscape(identifier))
	if err := c.get(path, nil, &result); err != nil {
		return nil, err
	}
	return &result.Project, nil
}

// CreateProject creates a new project.
func (c *Client) CreateProject(req ProjectCreateRequest) (*Project, error) {
	var result struct {
		Project Project `json:"project"`
	}
	if err := c.post("/projects.json", req, &result); err != nil {
		return nil, err
	}
	return &result.Project, nil
}

// UpdateProject updates an existing project.
func (c *Client) UpdateProject(identifier string, req ProjectUpdateRequest) error {
	path := fmt.Sprintf("/projects/%s.json", url.PathEscape(identifier))
	return c.put(path, req, nil)
}

// DeleteProject deletes a project.
func (c *Client) DeleteProject(identifier string) error {
	path := fmt.Sprintf("/projects/%s.json", url.PathEscape(identifier))
	return c.delete(path)
}
