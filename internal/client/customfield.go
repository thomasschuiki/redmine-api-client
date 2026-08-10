package client

import (
	"context"

	"github.com/tom-redmine/go-redmine-cli/internal/models"
)

// CustomFieldDefinition represents a Redmine custom field definition.
type CustomFieldDefinition = models.CustomFieldDefinition

// CustomFieldDefinitionListResponse is the response from listing custom fields.
type CustomFieldDefinitionListResponse = models.CustomFieldDefinitionList

// ListCustomFieldDefinitions returns all custom field definitions.
func (c *Client) ListCustomFieldDefinitions(ctx context.Context) (*CustomFieldDefinitionListResponse, error) {
	var result CustomFieldDefinitionListResponse
	if err := c.get(ctx, "/custom_fields.json", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
