package client

import (
	"context"
	"fmt"
	"net/url"

	"github.com/tom-redmine/go-redmine-cli/internal/models"
)

// Tracker represents a Redmine issue tracker.
type Tracker = models.Tracker

// TrackerListResponse is the response from listing trackers.
type TrackerListResponse = models.TrackerList

// ListTrackers returns all trackers.
func (c *Client) ListTrackers(ctx context.Context) (*TrackerListResponse, error) {
	var result TrackerListResponse
	if err := c.get(ctx, "/trackers.json", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// IssueStatus represents a Redmine issue status.
type IssueStatus = models.IssueStatus

// IssueStatusListResponse is the response from listing issue statuses.
type IssueStatusListResponse = models.IssueStatusList

// ListIssueStatuses returns all issue statuses.
func (c *Client) ListIssueStatuses(ctx context.Context) (*IssueStatusListResponse, error) {
	var result IssueStatusListResponse
	if err := c.get(ctx, "/issue_statuses.json", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// IssuePriority represents a Redmine issue priority.
type IssuePriority = models.IssuePriority

// IssuePriorityListResponse is the response from listing priorities.
type IssuePriorityListResponse = models.IssuePriorityList

// ListIssuePriorities returns all issue priorities.
func (c *Client) ListIssuePriorities(ctx context.Context) (*IssuePriorityListResponse, error) {
	var result IssuePriorityListResponse
	if err := c.get(ctx, "/enumerations/issue_priorities.json", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// IssueCategory represents a Redmine issue category.
type IssueCategory = models.IssueCategory

// IssueCategoryListResponse is the response from listing categories.
type IssueCategoryListResponse = models.IssueCategoryList

// ListIssueCategories returns categories for a project.
func (c *Client) ListIssueCategories(ctx context.Context, projectID string) (*IssueCategoryListResponse, error) {
	path := fmt.Sprintf("/projects/%s/issue_categories.json", url.PathEscape(projectID))
	var result IssueCategoryListResponse
	if err := c.get(ctx, path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
