package client

import (
	"context"

	"github.com/tom-redmine/go-redmine-cli/internal/models"
)

// Query represents a Redmine saved query.
type Query = models.Query

// QueryListResponse is the response from listing queries.
type QueryListResponse = models.QueryList

// ListQueries returns saved queries visible to the current user.
func (c *Client) ListQueries(ctx context.Context, opts ListOpts) (*QueryListResponse, error) {
	var result QueryListResponse
	if err := c.get(ctx, "/queries.json", opts.Params(), &result); err != nil {
		return nil, err
	}
	return &result, nil
}
