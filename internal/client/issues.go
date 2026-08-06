package client

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/tom-redmine/go-redmine-cli/internal/models"
)

// Issue represents a Redmine issue.
type Issue = models.Issue

// IssueRef is a lightweight reference to another issue.
type IssueRef = models.IssueRef

// Journal represents a journal (comment/change history) entry on an issue.
type Journal = models.Journal

// JournalDetail represents a single field change within a journal entry.
type JournalDetail = models.JournalDetail

// IDName is a generic reference to a named object.
type IDName = models.IDName

// CustomField represents a custom field value.
type CustomField = models.CustomField

// IssueListResponse is the response from listing issues.
type IssueListResponse = models.IssueList

// IssueCreateRequest is the request body for creating an issue.
type IssueCreateRequest = models.IssueCreate

// IssueUpdateRequest is the request body for updating an issue.
type IssueUpdateRequest = models.IssueUpdateRequest

// CustomFieldValue is a custom field to set.
// For single-value fields, use Value.
// For multi-value fields, use Values (the API receives both; Redmine ignores
// the one that doesn't match the field configuration).
type CustomFieldValue = models.CustomFieldValue

// ParseCustomField parses a "id=value" string into a CustomFieldValue.
func ParseCustomField(s string) (CustomFieldValue, error) {
	idStr, val, ok := strings.Cut(s, "=")
	if !ok {
		return CustomFieldValue{}, fmt.Errorf("invalid custom field format %q — expected <id>=<value>", s)
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return CustomFieldValue{}, fmt.Errorf("invalid custom field id %q: %w", idStr, err)
	}
	return CustomFieldValue{ID: id, Value: &val, Values: []string{val}}, nil
}

// MergeCustomFields combines a slice of CustomFieldValue into the final form:
// IDs that appear once use Value only; IDs that appear multiple times use Values.
func MergeCustomFields(fields []CustomFieldValue) []CustomFieldValue {
	groups := make(map[int][]string)
	order := make([]int, 0)
	for _, f := range fields {
		if _, ok := groups[f.ID]; !ok {
			order = append(order, f.ID)
		}
		if f.Value != nil {
			groups[f.ID] = append(groups[f.ID], *f.Value)
		} else {
			groups[f.ID] = append(groups[f.ID], f.Values...)
		}
	}
	result := make([]CustomFieldValue, 0, len(order))
	for _, id := range order {
		vals := groups[id]
		if len(vals) == 1 {
			v := vals[0]
			result = append(result, CustomFieldValue{ID: id, Value: &v})
		} else {
			result = append(result, CustomFieldValue{ID: id, Values: vals})
		}
	}
	return result
}

// IssueListOpts holds filtering options for listing issues.
type IssueListOpts struct {
	StatusID    string
	Subject     string
	Description string
	Sort        string
	Include     string
	ListOpts
	TrackerID      int
	AssignedToID   int
	PriorityID     int
	CategoryID     int
	FixedVersionID int
	ParentID       int
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
func (c *Client) ListIssues(ctx context.Context, projectID string, opts IssueListOpts) (*IssueListResponse, error) {
	var path string
	if projectID != "" {
		path = fmt.Sprintf("/projects/%s/issues.json", url.PathEscape(projectID))
	} else {
		path = "/issues.json"
	}

	params := opts.Params()
	var result IssueListResponse
	if err := c.get(ctx, path, params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

const listPageSize = 100

// ListAllIssues auto-paginates through all matching issues.
func (c *Client) ListAllIssues(ctx context.Context, projectID string, opts IssueListOpts) (*IssueListResponse, error) {
	opts.Limit = listPageSize
	var all []Issue
	var total int

	for offset := 0; ; {
		opts.Offset = offset
		resp, err := c.ListIssues(ctx, projectID, opts)
		if err != nil {
			return nil, err
		}
		total = 0
		if resp.TotalCount != nil {
			total = *resp.TotalCount
		}
		all = append(all, resp.Issues...)
		if len(resp.Issues) < listPageSize || offset+listPageSize >= total {
			break
		}
		offset += listPageSize
	}

	return &IssueListResponse{Issues: all, TotalCount: &total}, nil
}

// ContainsFilter returns issues where any text field contains the given
// substring (case-insensitive by default).
func ContainsFilter(issues []Issue, text string, caseInsensitive bool) []Issue {
	lower := strings.ToLower(text)
	var out []Issue
	for _, issue := range issues {
		if containsIssue(&issue, lower, caseInsensitive) {
			out = append(out, issue)
		}
	}
	return out
}

func containsIssue(issue *Issue, lower string, caseInsensitive bool) bool {
	fields := []string{
		issue.Subject,
		issue.Status.Name,
		issue.Tracker.Name,
		issue.Author.Name,
	}
	if issue.Description != nil {
		fields = append(fields, *issue.Description)
	}
	if issue.AssignedTo != nil {
		fields = append(fields, issue.AssignedTo.Name)
	}
	for _, f := range fields {
		if caseInsensitive {
			if strings.Contains(strings.ToLower(f), lower) {
				return true
			}
		} else {
			if strings.Contains(f, lower) {
				return true
			}
		}
	}
	return false
}

// RegexFilter returns issues where any text field matches the given pattern.
func RegexFilter(issues []Issue, pattern string, caseInsensitive bool) ([]Issue, error) {
	flags := ""
	if caseInsensitive {
		flags = "(?i)"
	}
	re, err := regexp.Compile(flags + pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex: %w", err)
	}
	var out []Issue
	for _, issue := range issues {
		if regexMatchIssue(&issue, re) {
			out = append(out, issue)
		}
	}
	return out, nil
}

func regexMatchIssue(issue *Issue, re *regexp.Regexp) bool {
	fields := []string{
		issue.Subject,
		issue.Status.Name,
		issue.Tracker.Name,
		issue.Author.Name,
	}
	if issue.Description != nil {
		fields = append(fields, *issue.Description)
	}
	if issue.AssignedTo != nil {
		fields = append(fields, issue.AssignedTo.Name)
	}
	for _, f := range fields {
		if re.MatchString(f) {
			return true
		}
	}
	return false
}

// GetIssue returns a single issue by ID.
// The include parameter is a comma-separated list of associated objects to
// include (e.g. "journals,watchers"). Pass empty string for no extras.
func (c *Client) GetIssue(ctx context.Context, id int, include string) (*Issue, error) {
	var result struct {
		Issue Issue `json:"issue"`
	}
	path := fmt.Sprintf("/issues/%d.json", id)
	var params url.Values
	if include != "" {
		params = url.Values{"include": {include}}
	}
	if err := c.get(ctx, path, params, &result); err != nil {
		return nil, err
	}
	return &result.Issue, nil
}

// CreateIssue creates a new issue.
func (c *Client) CreateIssue(ctx context.Context, req IssueCreateRequest) (*Issue, error) {
	var result struct {
		Issue Issue `json:"issue"`
	}
	if err := c.post(ctx, "/issues.json", req, &result); err != nil {
		return nil, err
	}
	return &result.Issue, nil
}

// UpdateIssue updates an existing issue.
func (c *Client) UpdateIssue(ctx context.Context, id int, req IssueUpdateRequest) error {
	path := fmt.Sprintf("/issues/%d.json", id)
	return c.put(ctx, path, req, nil)
}

// DeleteIssue deletes an issue.
func (c *Client) DeleteIssue(ctx context.Context, id int) error {
	path := fmt.Sprintf("/issues/%d.json", id)
	return c.delete(ctx, path)
}

// AddWatcher adds a watcher to an issue.
func (c *Client) AddWatcher(ctx context.Context, issueID, userID int) error {
	path := fmt.Sprintf("/issues/%d/watchers.json", issueID)
	body := map[string]int{"user_id": userID}
	return c.post(ctx, path, body, nil)
}

// RemoveWatcher removes a watcher from an issue.
func (c *Client) RemoveWatcher(ctx context.Context, issueID, userID int) error {
	path := fmt.Sprintf("/issues/%d/watchers/%d.json", issueID, userID)
	return c.delete(ctx, path)
}

// GrepHit represents a single text match within an issue.
type GrepHit struct {
	Subject string `json:"subject"`
	Where   string `json:"where"`
	Snippet string `json:"snippet"`
	IssueID int    `json:"issue_id"`
}

// GrepResult holds all matches from a grep search.
type GrepResult struct {
	Matches []GrepHit `json:"matches"`
	Total   int       `json:"total"`
}

// GrepOpts holds options for a grep search across issues.
type GrepOpts struct {
	Project      string
	Text         string
	In           string
	StatusID     string
	ParentID     int
	TrackerID    int
	AssignedToID int
}

// GrepIssues searches for text in issue descriptions and/or journal notes,
// auto-paginating through all results in the given scope.
func (c *Client) GrepIssues(ctx context.Context, opts GrepOpts) (*GrepResult, error) {
	searchDesc := strings.Contains(opts.In, "description")
	searchNotes := strings.Contains(opts.In, "notes")

	var allMatches []GrepHit
	totalScanned := 0

	for offset := 0; ; {
		listOpts := IssueListOpts{
			ListOpts: ListOpts{Offset: offset, Limit: listPageSize},
			Include:  "journals",
		}
		if opts.ParentID > 0 {
			listOpts.ParentID = opts.ParentID
		}
		if opts.StatusID != "" {
			listOpts.StatusID = opts.StatusID
		}
		if opts.TrackerID > 0 {
			listOpts.TrackerID = opts.TrackerID
		}
		if opts.AssignedToID > 0 {
			listOpts.AssignedToID = opts.AssignedToID
		}

		resp, err := c.ListIssues(ctx, opts.Project, listOpts)
		if err != nil {
			return nil, err
		}

		for i := range resp.Issues {
			issue := &resp.Issues[i]
			totalScanned++

			if searchDesc && issue.Description != nil {
				if hits := searchField(*issue.Description, "description", issue.ID, issue.Subject, opts.Text); len(hits) > 0 {
					allMatches = append(allMatches, hits...)
				}
			}
			if searchNotes {
				for _, j := range issue.Journals {
					if j.Notes != nil && *j.Notes != "" {
						if hits := searchField(*j.Notes, "note", issue.ID, issue.Subject, opts.Text); len(hits) > 0 {
							allMatches = append(allMatches, hits...)
						}
					}
				}
			}
		}

		if len(resp.Issues) < listPageSize || (resp.TotalCount != nil && offset+listPageSize >= *resp.TotalCount) {
			break
		}
		offset += listPageSize
	}

	return &GrepResult{Matches: allMatches, Total: totalScanned}, nil
}

func searchField(text, where string, issueID int, subject, keyword string) []GrepHit {
	lowerText := strings.ToLower(text)
	lowerKeyword := strings.ToLower(keyword)
	idx := 0
	var hits []GrepHit
	for {
		pos := strings.Index(lowerText[idx:], lowerKeyword)
		if pos < 0 {
			break
		}
		absPos := idx + pos
		snippet := extractSnippet(text, absPos, len(keyword))
		hits = append(hits, GrepHit{
			IssueID: issueID,
			Subject: subject,
			Where:   where,
			Snippet: snippet,
		})
		idx = absPos + len(keyword)
	}
	return hits
}

func extractSnippet(text string, pos, matchLen int) string {
	const ctx = 40
	start := pos - ctx
	if start < 0 {
		start = 0
	}
	end := pos + matchLen + ctx
	if end > len(text) {
		end = len(text)
	}
	snippet := text[start:end]
	snippet = strings.ReplaceAll(snippet, "\n", " ")
	snippet = strings.TrimSpace(snippet)
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(text) {
		snippet = snippet + "..."
	}
	return snippet
}

var dateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func ValidateDate(v string) error {
	if !dateRe.MatchString(v) {
		return fmt.Errorf("invalid date %q — expected YYYY-MM-DD", v)
	}
	return nil
}

func ValidateEstimatedHours(v float64) error {
	if v < 0 {
		return fmt.Errorf("estimated hours must be non-negative, got %f", v)
	}
	return nil
}
