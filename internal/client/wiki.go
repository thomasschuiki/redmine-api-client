package client

import (
	"fmt"
	"net/url"
)

// WikiPage represents a Redmine wiki page.
type WikiPage struct {
	ID         int        `json:"id"`
	Title      string     `json:"title"`
	Author     *IDName    `json:"author,omitempty"`
	CreatedOn  string     `json:"created_on,omitempty"`
	UpdatedOn  string     `json:"updated_on,omitempty"`
	Text       string     `json:"text,omitempty"`
}

// WikiPageListResponse is the response from listing wiki pages.
type WikiPageListResponse struct {
	WikiPages []WikiPage `json:"wiki_pages"`
}

// Version represents a Redmine version.
type Version struct {
	ID          int        `json:"id"`
	Project     *IDName    `json:"project,omitempty"`
	Name        string     `json:"name"`
	Status      string     `json:"status,omitempty"`
	DueDate     string     `json:"due_date,omitempty"`
	Description string     `json:"description,omitempty"`
	CreatedOn   string     `json:"created_on,omitempty"`
	UpdatedOn   string     `json:"updated_on,omitempty"`
}

// VersionListResponse is the response from listing versions.
type VersionListResponse struct {
	Versions []Version `json:"versions"`
}

// ListWikiPages returns wiki pages for a project.
func (c *Client) ListWikiPages(projectID string) (*WikiPageListResponse, error) {
	path := fmt.Sprintf("/projects/%s/wiki/index.json", url.PathEscape(projectID))
	var result WikiPageListResponse
	if err := c.get(path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetWikiPage returns a single wiki page by title.
func (c *Client) GetWikiPage(projectID, title string) (*WikiPage, error) {
	path := fmt.Sprintf("/projects/%s/wiki/%s.json", url.PathEscape(projectID), url.PathEscape(title))
	var result struct {
		WikiPage WikiPage `json:"wiki_page"`
	}
	if err := c.get(path, nil, &result); err != nil {
		return nil, err
	}
	return &result.WikiPage, nil
}

// ListVersions returns versions for a project.
func (c *Client) ListVersions(projectID string) (*VersionListResponse, error) {
	path := fmt.Sprintf("/projects/%s/versions.json", url.PathEscape(projectID))
	var result VersionListResponse
	if err := c.get(path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
