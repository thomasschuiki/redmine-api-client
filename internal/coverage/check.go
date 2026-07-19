package coverage

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
)

// SpecRoute represents a route extracted from the OpenAPI spec for comparison.
type SpecRoute struct {
	Method string
	Path   string
}

// CoverageReport shows the diff between a Redmine snapshot and the OpenAPI spec.
type CoverageReport struct {
	MissingInSpec []SpecRoute // routes in Redmine but not in the spec
	StaleInSpec   []SpecRoute // routes in the spec but not in Redmine
	SnapshotCount int
	SpecCount     int
}

// Check compares a Redmine snapshot against an OpenAPI spec and reports differences.
func Check(snapshot *Snapshot, specPath string) (*CoverageReport, error) {
	specRoutes, err := extractSpecRoutes(specPath)
	if err != nil {
		return nil, fmt.Errorf("extracting spec routes: %w", err)
	}

	snapshotRoutes := make(map[SpecRoute]bool)
	for _, r := range snapshot.APIRoutes {
		normalized := NormalizeRedminePath(r.Path)
		snapshotRoutes[SpecRoute{Method: strings.ToUpper(r.Verb), Path: normalized}] = true
	}

	report := &CoverageReport{
		SnapshotCount: len(snapshotRoutes),
		SpecCount:     len(specRoutes),
	}

	specSet := make(map[SpecRoute]bool)
	for _, r := range specRoutes {
		specSet[r] = true
	}

	// Routes in snapshot but not in spec
	for r := range snapshotRoutes {
		if !specSet[r] {
			report.MissingInSpec = append(report.MissingInSpec, r)
		}
	}

	// Routes in spec but not in snapshot
	for r := range specSet {
		if !snapshotRoutes[r] {
			report.StaleInSpec = append(report.StaleInSpec, r)
		}
	}

	sort.Slice(report.MissingInSpec, func(i, j int) bool {
		a, b := report.MissingInSpec[i], report.MissingInSpec[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		return a.Method < b.Method
	})

	sort.Slice(report.StaleInSpec, func(i, j int) bool {
		a, b := report.StaleInSpec[i], report.StaleInSpec[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		return a.Method < b.Method
	})

	return report, nil
}

// extractSpecRoutes reads an OpenAPI spec and extracts all method+path combinations.
func extractSpecRoutes(specPath string) ([]SpecRoute, error) {
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
	if err != nil {
		return nil, fmt.Errorf("building model: %w", err)
	}

	var routes []SpecRoute

	if model.Model.Paths == nil {
		return routes, nil
	}

	for key, pathItem := range model.Model.Paths.PathItems.FromOldest() {
		_ = key
		p := pathItem
		if p == nil {
			continue
		}

		if p.Get != nil {
			routes = append(routes, SpecRoute{Method: "GET", Path: key})
		}
		if p.Post != nil {
			routes = append(routes, SpecRoute{Method: "POST", Path: key})
		}
		if p.Put != nil {
			routes = append(routes, SpecRoute{Method: "PUT", Path: key})
		}
		if p.Patch != nil {
			routes = append(routes, SpecRoute{Method: "PATCH", Path: key})
		}
		if p.Delete != nil {
			routes = append(routes, SpecRoute{Method: "DELETE", Path: key})
		}
		if p.Head != nil {
			routes = append(routes, SpecRoute{Method: "HEAD", Path: key})
		}
		if p.Options != nil {
			routes = append(routes, SpecRoute{Method: "OPTIONS", Path: key})
		}
		if p.Trace != nil {
			routes = append(routes, SpecRoute{Method: "TRACE", Path: key})
		}
	}

	return routes, nil
}

// NormalizeRedminePath converts a Rails-style path to an OpenAPI-style path.
// Examples:
//
//	/issues(.:format)           → /issues.{format}
//	/issues/:id(.:format)       → /issues/{id}.{format}
//	/projects/:id/issues(.:format) → /projects/{id}/issues.{format}
func NormalizeRedminePath(p string) string {
	// Remove trailing constraints like {format: /\d+/}
	re := regexp.MustCompile(`\{(\w+):[^}]+\}`)
	p = re.ReplaceAllString(p, "{$1}")

	// Convert Rails :param to OpenAPI {param}
	re2 := regexp.MustCompile(`:(\w+)`)
	p = re2.ReplaceAllString(p, "{$1}")

	// Handle .(:format) → .{format} and (.:format) → .{format}
	p = strings.ReplaceAll(p, "(:format)", ".{format}")
	p = strings.ReplaceAll(p, "(.:format)", ".{format}")

	// Clean up any double dots from path joining
	p = path.Clean(p)

	// Ensure leading slash
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}

	return p
}

// FormatReport returns a human-readable coverage report.
func FormatReport(report *CoverageReport) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Snapshot routes: %d\n", report.SnapshotCount))
	sb.WriteString(fmt.Sprintf("Spec routes:     %d\n", report.SpecCount))
	sb.WriteString(fmt.Sprintf("Missing in spec: %d\n", len(report.MissingInSpec)))
	sb.WriteString(fmt.Sprintf("Stale in spec:   %d\n", len(report.StaleInSpec)))

	if len(report.MissingInSpec) > 0 {
		sb.WriteString("\n== Routes in Redmine but NOT in spec ==\n")
		for _, r := range report.MissingInSpec {
			sb.WriteString(fmt.Sprintf("  %s %s\n", r.Method, r.Path))
		}
	}

	if len(report.StaleInSpec) > 0 {
		sb.WriteString("\n== Routes in spec but NOT in Redmine ==\n")
		for _, r := range report.StaleInSpec {
			sb.WriteString(fmt.Sprintf("  %s %s\n", r.Method, r.Path))
		}
	}

	if len(report.MissingInSpec) == 0 && len(report.StaleInSpec) == 0 {
		sb.WriteString("\nOK: spec matches Redmine routes.\n")
	}

	return sb.String()
}
