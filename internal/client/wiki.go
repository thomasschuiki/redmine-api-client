package client

import (
	"context"
	"fmt"
	"net/url"
)

// WikiPage represents a Redmine wiki page.
type WikiPage struct {
	Author    *IDName `json:"author,omitempty"`
	Title     string  `json:"title"`
	CreatedOn string  `json:"created_on,omitempty"`
	UpdatedOn string  `json:"updated_on,omitempty"`
	Text      string  `json:"text,omitempty"`
	ID        int     `json:"id"`
}

// WikiPageListResponse is the response from listing wiki pages.
type WikiPageListResponse struct {
	WikiPages []WikiPage `json:"wiki_pages"`
}

// Version represents a Redmine version.
type Version struct {
	Project     *IDName `json:"project,omitempty"`
	Name        string  `json:"name"`
	Status      string  `json:"status,omitempty"`
	DueDate     string  `json:"due_date,omitempty"`
	Description string  `json:"description,omitempty"`
	CreatedOn   string  `json:"created_on,omitempty"`
	UpdatedOn   string  `json:"updated_on,omitempty"`
	ID          int     `json:"id"`
}

// VersionListResponse is the response from listing versions.
type VersionListResponse struct {
	Versions []Version `json:"versions"`
}

// ListWikiPages returns wiki pages for a project.
func (c *Client) ListWikiPages(ctx context.Context, projectID string) (*WikiPageListResponse, error) {
	path := fmt.Sprintf("/projects/%s/wiki/index.json", url.PathEscape(projectID))
	var result WikiPageListResponse
	if err := c.get(ctx, path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetWikiPage returns a single wiki page by title.
func (c *Client) GetWikiPage(ctx context.Context, projectID, title string) (*WikiPage, error) {
	path := fmt.Sprintf("/projects/%s/wiki/%s.json", url.PathEscape(projectID), url.PathEscape(title))
	var result struct {
		WikiPage WikiPage `json:"wiki_page"`
	}
	if err := c.get(ctx, path, nil, &result); err != nil {
		return nil, err
	}
	return &result.WikiPage, nil
}

// ListVersions returns versions for a project.
func (c *Client) ListVersions(ctx context.Context, projectID string) (*VersionListResponse, error) {
	path := fmt.Sprintf("/projects/%s/versions.json", url.PathEscape(projectID))
	var result VersionListResponse
	if err := c.get(ctx, path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
