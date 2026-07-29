package client

import (
	"fmt"
	"net/url"
)

// Tracker represents a Redmine issue tracker.
type Tracker struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// TrackerListResponse is the response from listing trackers.
type TrackerListResponse struct {
	Trackers []Tracker `json:"trackers"`
}

// ListTrackers returns all trackers.
func (c *Client) ListTrackers() (*TrackerListResponse, error) {
	var result TrackerListResponse
	if err := c.get("/trackers.json", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// IssueStatus represents a Redmine issue status.
type IssueStatus struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	IsClosed  bool   `json:"is_closed"`
}

// IssueStatusListResponse is the response from listing issue statuses.
type IssueStatusListResponse struct {
	IssueStatuses []IssueStatus `json:"issue_statuses"`
}

// ListIssueStatuses returns all issue statuses.
func (c *Client) ListIssueStatuses() (*IssueStatusListResponse, error) {
	var result IssueStatusListResponse
	if err := c.get("/issue_statuses.json", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// IssuePriority represents a Redmine issue priority.
type IssuePriority struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default"`
	Active    bool   `json:"active"`
}

// IssuePriorityListResponse is the response from listing priorities.
type IssuePriorityListResponse struct {
	IssuePriorities []IssuePriority `json:"issue_priorities"`
}

// ListIssuePriorities returns all issue priorities.
func (c *Client) ListIssuePriorities() (*IssuePriorityListResponse, error) {
	var result IssuePriorityListResponse
	if err := c.get("/enumerations/issue_priorities.json", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// IssueCategory represents a Redmine issue category.
type IssueCategory struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Project     *IDName `json:"project,omitempty"`
	AssignedTo  *IDName `json:"assigned_to,omitempty"`
}

// IssueCategoryListResponse is the response from listing categories.
type IssueCategoryListResponse struct {
	IssueCategories []IssueCategory `json:"issue_categories"`
}

// ListIssueCategories returns categories for a project.
func (c *Client) ListIssueCategories(projectID string) (*IssueCategoryListResponse, error) {
	path := fmt.Sprintf("/projects/%s/issue_categories.json", url.PathEscape(projectID))
	var result IssueCategoryListResponse
	if err := c.get(path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
