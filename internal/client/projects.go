package client

import (
	"fmt"
	"net/url"
	"strings"
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

// ResolveProject normalizes the identifier and attempts to fetch the project.
// On 404, it lists all projects and suggests close matches.
func (c *Client) ResolveProject(identifier string) (*Project, error) {
	normalized := NormalizeProjectID(identifier)
	project, err := c.GetProject(normalized)
	if err == nil {
		return project, nil
	}

	apiErr, ok := err.(*APIError)
	if !ok || apiErr.StatusCode != 404 {
		return nil, err
	}

	// 404 — try to suggest close matches
	suggestions, listErr := c.suggestProjects(normalized)
	if listErr != nil {
		return nil, fmt.Errorf("project %q not found (and could not list projects for suggestions: %v)", normalized, listErr)
	}
	if len(suggestions) > 0 {
		return nil, fmt.Errorf("project %q not found. Did you mean: %s", normalized, strings.Join(suggestions, ", "))
	}
	return nil, fmt.Errorf("project %q not found", normalized)
}

// NormalizeProjectID lowercases, trims, and replaces spaces with hyphens.
func NormalizeProjectID(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	return s
}

func (c *Client) suggestProjects(input string) ([]string, error) {
	resp, err := c.ListProjects(ListOpts{Limit: 100})
	if err != nil {
		return nil, err
	}
	var matches []string
	for _, p := range resp.Projects {
		if fuzzyMatch(input, p.Identifier) || fuzzyMatch(input, p.Name) {
			matches = append(matches, p.Identifier)
		}
		if len(matches) >= 5 {
			break
		}
	}
	return matches, nil
}

func fuzzyMatch(input, candidate string) bool {
	candidate = strings.ToLower(candidate)
	if strings.Contains(candidate, input) {
		return true
	}
	// simple edit-distance ≤ 2 check for short strings
	if len(input) <= 20 && levenshtein(input, candidate) <= 2 {
		return true
	}
	return false
}

func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr := make([]int, lb+1)
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(curr[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev = curr
	}
	return prev[lb]
}

func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}
