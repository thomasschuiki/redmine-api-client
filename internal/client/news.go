package client

import (
	"context"
	"fmt"
	"net/url"

	"github.com/tom-redmine/go-redmine-cli/internal/models"
)

// News represents a Redmine news item.
type News = models.News

// NewsListResponse is the response from listing news.
type NewsListResponse = models.NewsList

// NewsCreateRequest is the request body for creating news.
type NewsCreateRequest = models.NewsCreate

// NewsUpdateRequest is the request body for updating news.
type NewsUpdateRequest = models.NewsUpdateRequest

// ListNews returns news items, optionally scoped to a project.
func (c *Client) ListNews(ctx context.Context, projectID string, opts ListOpts) (*NewsListResponse, error) {
	path := "/news.json"
	if projectID != "" {
		path = fmt.Sprintf("/projects/%s/news.json", url.PathEscape(projectID))
	}
	params := opts.Params()
	var result NewsListResponse
	if err := c.get(ctx, path, params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetNews returns a single news item by ID.
func (c *Client) GetNews(ctx context.Context, id int, include string) (*News, error) {
	path := fmt.Sprintf("/news/%d.json", id)
	params := url.Values{}
	if include != "" {
		params.Set("include", include)
	}
	var result struct {
		News News `json:"news"`
	}
	if err := c.get(ctx, path, params, &result); err != nil {
		return nil, err
	}
	return &result.News, nil
}

// CreateNews creates a news item. News#create render_api_ok: the response is
// 204 No Content.
func (c *Client) CreateNews(ctx context.Context, req NewsCreateRequest) error {
	return c.post(ctx, "/news.json", req, nil)
}

// UpdateNews updates a news item. News#update render_api_ok: 204 No Content.
func (c *Client) UpdateNews(ctx context.Context, id int, req NewsUpdateRequest) error {
	path := fmt.Sprintf("/news/%d.json", id)
	return c.put(ctx, path, req, nil)
}
