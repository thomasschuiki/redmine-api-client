package spec

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
)

// sensitiveModelFields are ActiveRecord persistence columns that leaked into the
// spec through the old DB-column introspection and must never appear in the
// API contract.
var sensitiveModelFields = []string{
	"hashed_password",
	"salt",
	"lft",
	"rgt",
	"lock_version",
	"type",
	"password_digest",
	"auth_source_id",
}

// requiredSchemas are the contract component schemas every emitted spec must
// carry. They guard against a future refactor silently dropping a resource.
var requiredSchemas = []string{
	"Issue", "IssueSingle", "IssueList", "IssueRequest", "IssueCreate", "IssueUpdate", "IssueUpdateRequest",
	"Project", "ProjectSingle", "ProjectList", "ProjectRequest", "ProjectCreate",
	"User", "UserSingle", "UserList",
	"TimeEntry", "TimeEntrySingle", "TimeEntryList", "TimeEntryRequest", "TimeEntryCreate",
	"WikiPage", "WikiPageSingle", "WikiPageList",
	"Version", "VersionSingle", "VersionList",
	"Tracker", "TrackerSingle", "TrackerList",
	"IssueStatus", "IssueStatusSingle", "IssueStatusList",
	"IssuePriority", "IssuePrioritySingle", "IssuePriorityList",
	"IssueCategory", "IssueCategorySingle", "IssueCategoryList",
	"Relation", "RelationSingle", "RelationList", "RelationRequest", "RelationCreate",
}

// Validate reads a bundled OpenAPI spec and checks for structural errors:
// broken $ref pointers, invalid schemas, missing required fields, etc.
func Validate(specPath string) (*ValidationResult, error) {
	return validate(specPath, false)
}

// ValidateStrict runs all Validate checks plus the strict gates (e.g. no
// unreferenced component schemas).
func ValidateStrict(specPath string) (*ValidationResult, error) {
	return validate(specPath, true)
}

func validate(specPath string, strict bool) (*ValidationResult, error) {
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
		return result, nil
	}

	result.Valid = true
	result.PathCount = model.Model.Paths.PathItems.Len()
	result.SchemaCount = model.Model.Components.Schemas.Len()
	result.ParameterCount = model.Model.Components.Parameters.Len()
	result.ResponseCount = model.Model.Components.Responses.Len()
	result.SecuritySchemeCount = model.Model.Components.SecuritySchemes.Len()
	result.TagCount = len(model.Model.Tags)
	result.OperationCount = countOperations(&model.Model)

	var root map[string]any
	if err := yaml.Unmarshal(data, &root); err != nil {
		result.Valid = false
		result.ParseErrors = append(result.ParseErrors, fmt.Errorf("decoding spec YAML: %w", err))
		return result, nil
	}

	result.ContentErrors = checkContent(root, strict)
	if len(result.ContentErrors) > 0 {
		result.Valid = false
	}

	return result, nil
}

// ValidationResult holds the outcome of spec validation.
type ValidationResult struct {
	ParseErrors         []error
	ContentErrors       []string
	PathCount           int
	OperationCount      int
	SchemaCount         int
	ParameterCount      int
	ResponseCount       int
	SecuritySchemeCount int
	TagCount            int
	Valid               bool
}

func checkContent(root map[string]any, strict bool) []string {
	var errs []string

	for _, ref := range collectRefs(root) {
		if !resolvePointer(root, ref) {
			errs = append(errs, fmt.Sprintf("unresolved $ref: %s", ref))
		}
	}

	for _, name := range collectPropertyNames(root) {
		for _, leak := range sensitiveModelFields {
			if name == leak {
				errs = append(errs, fmt.Sprintf("leaked DB column in schema properties: %q", name))
			}
		}
	}

	schemas := mapKeys(componentsSchemas(root))
	for _, name := range requiredSchemas {
		if !contains(schemas, name) {
			errs = append(errs, fmt.Sprintf("missing required schema: %s", name))
		}
	}

	errs = append(errs, checkDuplicateOperationIDs(root)...)
	errs = append(errs, checkPayloadsHaveSchemas(root)...)

	if strict {
		for _, name := range checkDeadComponents(root) {
			errs = append(errs, fmt.Sprintf("unreferenced schema (dead component): %s", name))
		}
	}

	return errs
}

// checkDeadComponents reports component schemas never referenced by a $ref in
// the document. A dead schema signals a resource dropped from the contract
// without its schema being removed.
func checkDeadComponents(root map[string]any) []string {
	schemas := componentsSchemas(root)
	if len(schemas) == 0 {
		return nil
	}

	referenced := map[string]bool{}
	const prefix = "#/components/schemas/"
	for _, ref := range collectRefs(root) {
		if strings.HasPrefix(ref, prefix) {
			referenced[strings.TrimPrefix(ref, prefix)] = true
		}
	}

	var dead []string
	for name := range schemas {
		if !referenced[name] {
			dead = append(dead, name)
		}
	}
	sort.Strings(dead)
	return dead
}

// checkDuplicateOperationIDs reports operationIds emitted more than once.
func checkDuplicateOperationIDs(root map[string]any) []string {
	seen := map[string]bool{}
	var dups []string

	forEachOperation(root, func(path, method string, op map[string]any) {
		id, _ := op["operationId"].(string)
		if id == "" {
			dups = append(dups, fmt.Sprintf("%s %s: missing operationId", method, path))
			return
		}
		if seen[id] {
			dups = append(dups, fmt.Sprintf("%s %s: duplicate operationId %q", method, path, id))
		}
		seen[id] = true
	})

	return dups
}

// checkPayloadsHaveSchemas verifies that every request body and every non-empty
// response body carries a schema, and that 200/201 responses on object endpoints
// are not content-less (the contract always models them).
func checkPayloadsHaveSchemas(root map[string]any) []string {
	var errs []string

	forEachOperation(root, func(path, method string, op map[string]any) {
		if rb, ok := op["requestBody"].(map[string]any); ok {
			if !mediaTypeHasSchema(rb) {
				errs = append(errs, fmt.Sprintf("%s %s: requestBody without schema", method, path))
			}
		}

		for _, code := range []string{"200", "201"} {
			resp, ok := op["responses"].(map[string]any)[code].(map[string]any)
			if !ok {
				continue
			}
			if _, hasContent := resp["content"]; hasContent {
				if !mediaTypeHasSchema(resp) {
					errs = append(errs, fmt.Sprintf("%s %s: %s response has content but no schema", method, path, code))
				}
			} else if method == "delete" {
				// 204 delete responses are expected to be content-less.
			} else {
				hasContentElsewhere := false
				for code2, other := range op["responses"].(map[string]any) {
					if strings.HasPrefix(code2, "2") && code2 != code {
						if _, ok2 := other.(map[string]any)["content"]; ok2 {
							hasContentElsewhere = true
						}
					}
				}
				if !hasContentElsewhere {
					errs = append(errs, fmt.Sprintf("%s %s: %s response missing content schema", method, path, code))
				}
			}
		}
	})

	return errs
}

func mediaTypeHasSchema(container map[string]any) bool {
	content, ok := container["content"].(map[string]any)
	if !ok {
		return false
	}
	for _, mt := range content {
		if _, ok := mt.(map[string]any)["schema"]; ok {
			return true
		}
	}
	return false
}

// collectRefs returns every "$ref" string in the document.
func collectRefs(value any) []string {
	var refs []string
	var walk func(any)
	walk = func(v any) {
		switch n := v.(type) {
		case map[string]any:
			if ref, ok := n["$ref"].(string); ok {
				refs = append(refs, ref)
			}
			for _, child := range n {
				walk(child)
			}
		case []any:
			for _, child := range n {
				walk(child)
			}
		}
	}
	walk(value)
	return refs
}

// collectPropertyNames returns every key under a "properties" map.
func collectPropertyNames(value any) []string {
	var names []string
	var walk func(any)
	walk = func(v any) {
		switch n := v.(type) {
		case map[string]any:
			if props, ok := n["properties"].(map[string]any); ok {
				for name := range props {
					names = append(names, name)
				}
			}
			for _, child := range n {
				walk(child)
			}
		case []any:
			for _, child := range n {
				walk(child)
			}
		}
	}
	walk(value)
	return names
}

// componentsSchemas returns the schemas map (may be nil).
func componentsSchemas(root map[string]any) map[string]any {
	comps, _ := root["components"].(map[string]any)
	schemas, _ := comps["schemas"].(map[string]any)
	return schemas
}

// resolvePointer resolves a JSON pointer like "#/components/schemas/Issue"
// against the document. Returns false for unresolvable pointers.
func resolvePointer(root map[string]any, pointer string) bool {
	if !strings.HasPrefix(pointer, "#/") {
		return false
	}

	current := any(root)
	for _, raw := range strings.Split(strings.TrimPrefix(pointer, "#/"), "/") {
		part := strings.ReplaceAll(strings.ReplaceAll(raw, "~1", "/"), "~0", "~")
		m, ok := current.(map[string]any)
		if !ok {
			return false
		}
		next, ok := m[part]
		if !ok {
			return false
		}
		current = next
	}
	return true
}

// forEachOperation visits every HTTP operation with its path and method.
func forEachOperation(root map[string]any, fn func(path, method string, op map[string]any)) {
	paths, _ := root["paths"].(map[string]any)
	methods := []string{"get", "post", "put", "patch", "delete", "head", "options", "trace"}

	var pathNames []string
	for name := range paths {
		pathNames = append(pathNames, name)
	}
	sort.Strings(pathNames)

	for _, path := range pathNames {
		item, _ := paths[path].(map[string]any)
		for _, method := range methods {
			op, ok := item[method].(map[string]any)
			if ok {
				fn(path, method, op)
			}
		}
	}
}

func mapKeys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func contains(s []string, v string) bool {
	for _, e := range s {
		if e == v {
			return true
		}
	}
	return false
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
