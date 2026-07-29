package client

import (
	"context"
	"fmt"
	"net/url"
)

// TimeEntry represents a Redmine time entry.
type TimeEntry struct {
	Project   *IDName `json:"project,omitempty"`
	Issue     *IDName `json:"issue,omitempty"`
	User      *IDName `json:"user,omitempty"`
	Activity  *IDName `json:"activity,omitempty"`
	Comments  string  `json:"comments,omitempty"`
	CreatedOn string  `json:"created_on,omitempty"`
	UpdatedOn string  `json:"updated_on,omitempty"`
	ID        int     `json:"id"`
	Hours     float64 `json:"hours"`
}

// TimeEntryListResponse is the response from listing time entries.
type TimeEntryListResponse struct {
	TimeEntries []TimeEntry `json:"time_entries"`
	TotalCount  int         `json:"total_count"`
	Offset      int         `json:"offset"`
	Limit       int         `json:"limit"`
}

// TimeEntryCreateRequest is the request body for creating a time entry.
type TimeEntryCreateRequest struct {
	TimeEntry struct {
		IssueID    *int    `json:"issue_id,omitempty"`
		ProjectID  *int    `json:"project_id,omitempty"`
		ActivityID *int    `json:"activity_id,omitempty"`
		Comments   string  `json:"comments,omitempty"`
		Hours      float64 `json:"hours"`
	} `json:"time_entry"`
}

// ListTimeEntries returns time entries, optionally filtered by project or issue.
func (c *Client) ListTimeEntries(ctx context.Context, projectID string, issueID int, opts ListOpts) (*TimeEntryListResponse, error) {
	var path string
	if projectID != "" {
		path = fmt.Sprintf("/projects/%s/time_entries.json", url.PathEscape(projectID))
	} else if issueID > 0 {
		path = fmt.Sprintf("/issues/%d/time_entries.json", issueID)
	} else {
		path = "/time_entries.json"
	}

	params := opts.Params()
	var result TimeEntryListResponse
	if err := c.get(ctx, path, params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateTimeEntry creates a new time entry.
func (c *Client) CreateTimeEntry(ctx context.Context, req TimeEntryCreateRequest) (*TimeEntry, error) {
	var result struct {
		TimeEntry TimeEntry `json:"time_entry"`
	}
	if err := c.post(ctx, "/time_entries.json", req, &result); err != nil {
		return nil, err
	}
	return &result.TimeEntry, nil
}

// DeleteTimeEntry deletes a time entry.
func (c *Client) DeleteTimeEntry(ctx context.Context, id int) error {
	path := fmt.Sprintf("/time_entries/%d.json", id)
	return c.delete(ctx, path)
}
