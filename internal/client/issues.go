package client

import (
	"fmt"
	"net/url"
)

// Issue represents a Redmine issue.
type Issue struct {
	ID          int            `json:"id"`
	Project     *IDName        `json:"project,omitempty"`
	Tracker     *IDName        `json:"tracker,omitempty"`
	Status      *IDName        `json:"status,omitempty"`
	Priority    *IDName        `json:"priority,omitempty"`
	Author      *IDName        `json:"author,omitempty"`
	AssignedTo  *IDName        `json:"assigned_to,omitempty"`
	Subject     string         `json:"subject"`
	Description string         `json:"description,omitempty"`
	StartDate   string         `json:"start_date,omitempty"`
	DueDate     string         `json:"due_date,omitempty"`
	DoneRatio   int            `json:"done_ratio"`
	IsPrivate   bool           `json:"is_private"`
	CreatedOn   string         `json:"created_on,omitempty"`
	UpdatedOn   string         `json:"updated_on,omitempty"`
	ClosedOn    string         `json:"closed_on,omitempty"`
	CustomFields []CustomField `json:"custom_fields,omitempty"`
}

// IDName is a generic reference to a named object.
type IDName struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// CustomField represents a custom field value.
type CustomField struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Value    any    `json:"value,omitempty"`
	Multiple bool   `json:"multiple,omitempty"`
}

// IssueListResponse is the response from listing issues.
type IssueListResponse struct {
	Issues     []Issue `json:"issues"`
	TotalCount int     `json:"total_count"`
	Offset     int     `json:"offset"`
	Limit      int     `json:"limit"`
}

// IssueCreateRequest is the request body for creating an issue.
type IssueCreateRequest struct {
	Issue struct {
		ProjectID    int                `json:"project_id"`
		Subject      string             `json:"subject"`
		Description  string             `json:"description,omitempty"`
		TrackerID    *int               `json:"tracker_id,omitempty"`
		StatusID     *int               `json:"status_id,omitempty"`
		PriorityID   *int               `json:"priority_id,omitempty"`
		AssignedToID *int               `json:"assigned_to_id,omitempty"`
		CustomFields []CustomFieldValue `json:"custom_fields,omitempty"`
	} `json:"issue"`
}

// IssueUpdateRequest is the request body for updating an issue.
type IssueUpdateRequest struct {
	Issue struct {
		Subject      *string            `json:"subject,omitempty"`
		Description  *string            `json:"description,omitempty"`
		TrackerID    *int               `json:"tracker_id,omitempty"`
		StatusID     *int               `json:"status_id,omitempty"`
		PriorityID   *int               `json:"priority_id,omitempty"`
		AssignedToID *int               `json:"assigned_to_id,omitempty"`
		DoneRatio    *int               `json:"done_ratio,omitempty"`
		CustomFields []CustomFieldValue `json:"custom_fields,omitempty"`
	} `json:"issue"`
}

// CustomFieldValue is a custom field to set.
type CustomFieldValue struct {
	ID    int    `json:"id"`
	Value string `json:"value"`
}

// IssueListOpts holds filtering options for listing issues.
type IssueListOpts struct {
	ListOpts
	StatusID       string
	TrackerID      int
	AssignedToID   int
	PriorityID     int
	CategoryID     int
	FixedVersionID int
	ParentID       int
	Subject        string
	Description    string
	Sort           string
	Include        string
}

// Params converts IssueListOpts to url.Values.
func (o IssueListOpts) Params() url.Values {
	v := o.ListOpts.Params()
	if o.StatusID != "" {
		v.Set("status_id", o.StatusID)
	}
	if o.TrackerID > 0 {
		v.Set("tracker_id", fmt.Sprintf("%d", o.TrackerID))
	}
	if o.AssignedToID > 0 {
		v.Set("assigned_to_id", fmt.Sprintf("%d", o.AssignedToID))
	}
	if o.PriorityID > 0 {
		v.Set("priority_id", fmt.Sprintf("%d", o.PriorityID))
	}
	if o.CategoryID > 0 {
		v.Set("category_id", fmt.Sprintf("%d", o.CategoryID))
	}
	if o.FixedVersionID > 0 {
		v.Set("fixed_version_id", fmt.Sprintf("%d", o.FixedVersionID))
	}
	if o.ParentID > 0 {
		v.Set("parent_id", fmt.Sprintf("%d", o.ParentID))
	}
	if o.Subject != "" {
		v.Set("subject", o.Subject)
	}
	if o.Description != "" {
		v.Set("description", o.Description)
	}
	if o.Sort != "" {
		v.Set("sort", o.Sort)
	}
	if o.Include != "" {
		v.Set("include", o.Include)
	}
	return v
}

// ListIssues returns issues, optionally filtered by project.
func (c *Client) ListIssues(projectID string, opts IssueListOpts) (*IssueListResponse, error) {
	var path string
	if projectID != "" {
		path = fmt.Sprintf("/projects/%s/issues.json", url.PathEscape(projectID))
	} else {
		path = "/issues.json"
	}

	params := opts.Params()
	var result IssueListResponse
	if err := c.get(path, params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetIssue returns a single issue by ID.
func (c *Client) GetIssue(id int) (*Issue, error) {
	var result struct {
		Issue Issue `json:"issue"`
	}
	path := fmt.Sprintf("/issues/%d.json", id)
	if err := c.get(path, nil, &result); err != nil {
		return nil, err
	}
	return &result.Issue, nil
}

// CreateIssue creates a new issue.
func (c *Client) CreateIssue(req IssueCreateRequest) (*Issue, error) {
	var result struct {
		Issue Issue `json:"issue"`
	}
	if err := c.post("/issues.json", req, &result); err != nil {
		return nil, err
	}
	return &result.Issue, nil
}

// UpdateIssue updates an existing issue.
func (c *Client) UpdateIssue(id int, req IssueUpdateRequest) error {
	path := fmt.Sprintf("/issues/%d.json", id)
	return c.put(path, req, nil)
}

// DeleteIssue deletes an issue.
func (c *Client) DeleteIssue(id int) error {
	path := fmt.Sprintf("/issues/%d.json", id)
	return c.delete(path)
}
