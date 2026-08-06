package client

import (
	"context"
	"fmt"
	"net/url"

	"github.com/tom-redmine/go-redmine-cli/internal/models"
)

// TimeEntry represents a Redmine time entry.
type TimeEntry = models.TimeEntry

// TimeEntryListResponse is the response from listing time entries.
type TimeEntryListResponse = models.TimeEntryList

// TimeEntryCreateRequest is the request body for creating a time entry.
type TimeEntryCreateRequest = models.TimeEntryCreate

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
