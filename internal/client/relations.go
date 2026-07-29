package client

import (
	"fmt"
	"net/url"
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

type Relation struct {
	ID           int          `json:"id"`
	IssueID      int          `json:"issue_id"`
	IssueToID    int          `json:"issue_to_id"`
	RelationType RelationType `json:"relation_type"`
	Delay        *int         `json:"delay,omitempty"`
}

type RelationCreateRequest struct {
	Relation struct {
		IssueID      int          `json:"issue_id"`
		IssueToID    int          `json:"issue_to_id"`
		RelationType RelationType `json:"relation_type"`
		Delay        *int         `json:"delay,omitempty"`
	} `json:"relation"`
}

type RelationListResponse struct {
	Relations []Relation `json:"relations"`
}

func (c *Client) CreateRelation(req RelationCreateRequest) (*Relation, error) {
	var result struct {
		Relation Relation `json:"relation"`
	}
	if err := c.post("/relations.json", req, &result); err != nil {
		return nil, err
	}
	return &result.Relation, nil
}

func (c *Client) ListRelations(issueID int) (*RelationListResponse, error) {
	var result RelationListResponse
	path := fmt.Sprintf("/issues/%d/relations.json", issueID)
	if err := c.get(path, url.Values{}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) DeleteRelation(id int) error {
	path := fmt.Sprintf("/relations/%d.json", id)
	return c.delete(path)
}

func (c *Client) GetRelation(id int) (*Relation, error) {
	var result struct {
		Relation Relation `json:"relation"`
	}
	path := fmt.Sprintf("/relations/%d.json", id)
	if err := c.get(path, url.Values{}, &result); err != nil {
		return nil, err
	}
	return &result.Relation, nil
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
