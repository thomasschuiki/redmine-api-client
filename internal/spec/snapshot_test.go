package spec

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// snapshotExpectations pins the exact contract surface per supported Redmine
// version. Regenerating snapshots (scripts/gen-spec-snapshots.sh) must not
// silently change any of these numbers; bump them only when the version's
// contract genuinely changes.
var snapshotExpectations = []struct {
	name    string
	version string
	paths   int
	ops     int
	schemas int
}{
	{"redmine-5.1.12.yaml", "5.1.12", 30, 68, 58},
	{"redmine-6.0.10.yaml", "6.0.10", 30, 68, 58},
	{"redmine-6.1.3.yaml", "6.1.3", 30, 68, 58},
}

// repoRoot resolves the module root from the test source location so the
// versioned snapshots under the repo-level testdata/ can be read regardless
// of the current working directory.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test source path")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(file)))
}

func TestVersionedSnapshotsAreStrictValid(t *testing.T) {
	root := repoRoot(t)
	for _, snap := range snapshotExpectations {
		snap := snap
		t.Run(snap.name, func(t *testing.T) {
			path := filepath.Join(root, "testdata", "openapi", snap.name)
			res, err := ValidateStrict(path)
			if err != nil {
				t.Fatalf("ValidateStrict error: %v", err)
			}
			if !res.Valid {
				t.Fatalf("snapshot not strictly valid: %v", res.ContentErrors)
			}
			if res.PathCount != snap.paths {
				t.Errorf("paths = %d, want %d", res.PathCount, snap.paths)
			}
			if res.OperationCount != snap.ops {
				t.Errorf("operations = %d, want %d", res.OperationCount, snap.ops)
			}
			if res.SchemaCount != snap.schemas {
				t.Errorf("schemas = %d, want %d", res.SchemaCount, snap.schemas)
			}
		})
	}
}

func TestSnapshotInfoMatchesVersion(t *testing.T) {
	root := repoRoot(t)
	for _, snap := range snapshotExpectations {
		snap := snap
		t.Run(snap.name, func(t *testing.T) {
			path := filepath.Join(root, "testdata", "openapi", snap.name)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read snapshot: %v", err)
			}
			var doc struct {
				Info struct {
					Version string `yaml:"version"`
				} `yaml:"info"`
			}
			if err := yaml.Unmarshal(data, &doc); err != nil {
				t.Fatalf("parse snapshot: %v", err)
			}
			if want := "redmine." + snap.version; !strings.Contains(doc.Info.Version, want) {
				t.Errorf("info.version = %q, want it to contain %q", doc.Info.Version, want)
			}
		})
	}
}

func TestListOperationsCarryQueryFilters(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "testdata", "openapi", "redmine-6.1.3.yaml")
	ops := loadOperations(t, path)

	wantByOp := map[string][]string{
		"getIssues":           {"status_id", "author_id", "assigned_to_id", "project_id", "subproject_id", "sort", "include"},
		"getProjects":         {"name", "include"},
		"getUsers":            {"name", "group_id", "status", "include"},
		"getTimeEntries":      {"project_id", "issue_id", "user_id", "from", "to", "spent_on", "sort", "include"},
		"getProjectVersions":  {"status"},
		"getProjectWikiPages": {"include"},
	}
	for opID, wants := range wantByOp {
		got := queryParams(t, ops, opID)
		for _, want := range wants {
			if !containsString(got, want) {
				t.Errorf("%s: missing query filter %q; have %v", opID, want, got)
			}
		}
	}

	// Nested routes must not repeat a parent scope as a query filter; it is
	// already expressed as a path parameter.
	for _, opID := range []string{"getProjectIssues", "getProjectTimeEntries"} {
		if got := queryParams(t, ops, opID); containsString(got, "project_id") {
			t.Errorf("%s: repeats project_id as a query filter; it is already a path parameter", opID)
		}
	}
}

// TestIndexOnlyResourcesAreTrulyReadOnly pins the accept_api_auth filter: the
// trackers/issue_statuses/enumerations controllers expose only :index via the
// API, so their spec surface must be a single list operation.
func TestIndexOnlyResourcesAreTrulyReadOnly(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "testdata", "openapi", "redmine-6.1.3.yaml")
	ops := loadOperations(t, path)

	want := []string{"getTrackers", "getIssueStatuses", "getIssuePriorities", "getEnumerationIssuePriorities"}
	for _, opID := range want {
		if _, ok := ops[opID]; !ok {
			t.Errorf("missing index operation %s", opID)
		}
	}

	for _, opID := range []string{
		"createTracker", "updateTracker", "updateTracker2", "deleteTracker",
		"createIssueStatus", "updateIssueStatus", "updateIssueStatus2", "deleteIssueStatus",
		"createIssuePriority", "updateIssuePriority", "updateIssuePriority2", "deleteIssuePriority",
	} {
		if _, ok := ops[opID]; ok {
			t.Errorf("accept_api_auth filter leak: %s must not be emitted", opID)
		}
	}
}

// TestAttachmentSurface pins the attachments contract: the five REST ops are
// emitted, while the non-REST binary surface (upload, download, thumbnail) and
// the project files controller stay out of the spec.
func TestAttachmentSurface(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "testdata", "openapi", "redmine-6.1.3.yaml")
	ops := loadOperations(t, path)

	for _, opID := range []string{"getAttachment", "getAttachment2", "updateAttachment", "updateAttachment2", "deleteAttachment"} {
		if _, ok := ops[opID]; !ok {
			t.Errorf("missing attachment operation %s", opID)
		}
	}

	for _, opID := range []string{
		"getProjectAttachments", "createProjectAttachment",
		"uploadAttachment", "downloadAttachment", "thumbnailAttachment",
	} {
		if _, ok := ops[opID]; ok {
			t.Errorf("attachment surface leak: %s must not be emitted", opID)
		}
	}
}

// TestMembershipSurface pins the memberships contract: all six operations are
// emitted, keyed to the members controller (renamed Membership).
func TestMembershipSurface(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "testdata", "openapi", "redmine-6.1.3.yaml")
	ops := loadOperations(t, path)

	for _, opID := range []string{
		"getProjectMemberships", "createProjectMembership",
		"getMembership", "updateMembership", "updateMembership2", "deleteMembership",
	} {
		if _, ok := ops[opID]; !ok {
			t.Errorf("missing membership operation %s", opID)
		}
	}

	// Nested routes bind the project scope as a path parameter; the un-nested
	// show binds the membership id.
	pathParams := func(opID string) []string {
		var names []string
		params, _ := ops[opID].(map[string]any)["parameters"].([]any)
		for _, raw := range params {
			p, ok := raw.(map[string]any)
			if !ok || p["in"] != "path" {
				continue
			}
			if name, ok := p["name"].(string); ok {
				names = append(names, name)
			}
		}
		return names
	}
	if got := pathParams("getProjectMemberships"); !containsString(got, "project_id") || containsString(got, "id") {
		t.Errorf("getProjectMemberships path params = %v, want project_id only", got)
	}
	if got := pathParams("getMembership"); !containsString(got, "id") {
		t.Errorf("getMembership path params = %v, want id", got)
	}
}

func loadOperations(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	var doc struct {
		Paths map[string]any `yaml:"paths"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse snapshot: %v", err)
	}
	ops := map[string]any{}
	for _, pathItem := range doc.Paths {
		methods, ok := pathItem.(map[string]any)
		if !ok {
			continue
		}
		for _, rawOp := range methods {
			op, ok := rawOp.(map[string]any)
			if !ok {
				continue
			}
			if id, ok := op["operationId"].(string); ok {
				ops[id] = op
			}
		}
	}
	return ops
}

func queryParams(t *testing.T, ops map[string]any, opID string) []string {
	t.Helper()
	op, ok := ops[opID]
	if !ok {
		t.Fatalf("operation %s not found", opID)
	}
	var names []string
	params, _ := op.(map[string]any)["parameters"].([]any)
	for _, raw := range params {
		p, ok := raw.(map[string]any)
		if !ok || p["in"] != "query" {
			continue
		}
		if name, ok := p["name"].(string); ok {
			names = append(names, name)
		}
	}
	return names
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
