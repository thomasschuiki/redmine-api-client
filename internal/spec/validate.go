package spec

import (
	"fmt"
	"os"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
)

// Validate reads a bundled OpenAPI spec and checks for structural errors:
// broken $ref pointers, invalid schemas, missing required fields, etc.
func Validate(specPath string) (*ValidationResult, error) {
	data, err := os.ReadFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("reading spec: %w", err)
	}

	config := datamodel.NewDocumentConfiguration()
	config.SpecFilePath = specPath

	doc, err := libopenapi.NewDocumentWithConfiguration(data, config)
	if err != nil {
		return nil, fmt.Errorf("parsing spec: %w", err)
	}

	model, err := doc.BuildV3Model()

	result := &ValidationResult{}

	if err != nil {
		result.Valid = false
		result.ParseErrors = []error{err}
	} else {
		result.Valid = true
		result.PathCount = model.Model.Paths.PathItems.Len()
		result.SchemaCount = model.Model.Components.Schemas.Len()
		result.ParameterCount = model.Model.Components.Parameters.Len()
		result.ResponseCount = model.Model.Components.Responses.Len()
		result.SecuritySchemeCount = model.Model.Components.SecuritySchemes.Len()
		result.TagCount = len(model.Model.Tags)
		result.OperationCount = countOperations(&model.Model)
	}

	return result, nil
}

// ValidationResult holds the outcome of spec validation.
type ValidationResult struct {
	ParseErrors         []error
	PathCount           int
	OperationCount      int
	SchemaCount         int
	ParameterCount      int
	ResponseCount       int
	SecuritySchemeCount int
	TagCount            int
	Valid               bool
}

func countOperations(model *v3high.Document) int {
	count := 0
	if model.Paths == nil {
		return 0
	}
	for _, pathItem := range model.Paths.PathItems.FromOldest() {
		p := pathItem
		if p == nil {
			continue
		}
		ops := []*v3high.Operation{
			p.Get, p.Post, p.Put, p.Patch, p.Delete, p.Head, p.Options, p.Trace,
		}
		for _, op := range ops {
			if op != nil {
				count++
			}
		}
	}
	return count
}
