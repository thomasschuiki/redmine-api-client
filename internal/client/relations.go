package client

import (
	"context"
	"fmt"
	"net/url"

	"github.com/tom-redmine/go-redmine-cli/internal/models"
)

type RelationType string

const (
	RelationRelates    RelationType = "relates"
	RelationDuplicates RelationType = "duplicates"
	RelationDuplicated RelationType = "duplicated"
	RelationBlocks     RelationType = "blocks"
	RelationBlocked    RelationType = "blocked"
	RelationPrecedes   RelationType = "precedes"
	RelationFollows    RelationType = "follows"
	RelationCopiedTo   RelationType = "copied_to"
	RelationCopiedFrom RelationType = "copied_from"
)

// Relation represents an issue-to-issue relation.
type Relation = models.Relation

type RelationCreateRequest = models.RelationCreate

// RelationListResponse is the response from listing relations.
type RelationListResponse = models.RelationList

func (c *Client) CreateRelation(ctx context.Context, req RelationCreateRequest) (*Relation, error) {
	var result struct {
		Relation Relation `json:"relation"`
	}
	if err := c.post(ctx, "/relations.json", req, &result); err != nil {
		return nil, err
	}
	return &result.Relation, nil
}

func (c *Client) ListRelations(ctx context.Context, issueID int) (*RelationListResponse, error) {
	var result RelationListResponse
	path := fmt.Sprintf("/issues/%d/relations.json", issueID)
	if err := c.get(ctx, path, url.Values{}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) DeleteRelation(ctx context.Context, id int) error {
	path := fmt.Sprintf("/relations/%d.json", id)
	return c.delete(ctx, path)
}

func ParseRelationType(s string) (RelationType, error) {
	switch RelationType(s) {
	case RelationRelates, RelationDuplicates, RelationDuplicated,
		RelationBlocks, RelationBlocked,
		RelationPrecedes, RelationFollows,
		RelationCopiedTo, RelationCopiedFrom:
		return RelationType(s), nil
	default:
		return "", fmt.Errorf("invalid relation type %q — valid types: relates, duplicates, duplicated, blocks, blocked, precedes, follows, copied_to, copied_from", s)
	}
}
