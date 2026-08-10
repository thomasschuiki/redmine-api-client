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
	{"redmine-5.1.12.yaml", "5.1.12", 37, 84, 76},
	{"redmine-6.0.10.yaml", "6.0.10", 37, 84, 76},
	{"redmine-6.1.3.yaml", "6.1.3", 37, 84, 76},
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

// TestNewsSurface pins the news contract: index, nested index, create,
// auto-disambiguated show (getNews2), update pair, and delete are all emitted.
func TestNewsSurface(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "testdata", "openapi", "redmine-6.1.3.yaml")
	ops := loadOperations(t, path)

	for _, opID := range []string{
		"getNews", "getNews2", "createNews", "getProjectNews", "createProjectNews",
		"updateNews", "updateNews2", "deleteNews",
	} {
		if _, ok := ops[opID]; !ok {
			t.Errorf("missing news operation %s", opID)
		}
	}

	// The show op (getNews2) must carry the include expansion.
	if got := queryParams(t, ops, "getNews2"); !containsString(got, "include") {
		t.Errorf("getNews2: missing include query param; have %v", got)
	}
}

// TestGroupSurface pins the groups contract: all six operations plus the
// builtin index param and include show param.
func TestGroupSurface(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "testdata", "openapi", "redmine-6.1.3.yaml")
	ops := loadOperations(t, path)

	for _, opID := range []string{"getGroups", "createGroup", "getGroup", "updateGroup", "updateGroup2", "deleteGroup"} {
		if _, ok := ops[opID]; !ok {
			t.Errorf("missing group operation %s", opID)
		}
	}

	if got := queryParams(t, ops, "getGroups"); !containsString(got, "builtin") {
		t.Errorf("getGroups: missing builtin query param; have %v", got)
	}
	if got := queryParams(t, ops, "getGroup"); !containsString(got, "include") {
		t.Errorf("getGroup: missing include query param; have %v", got)
	}
}

// TestQueryAndCustomFieldSurface pins the read-only queries and custom fields
// contracts: the accept_api_auth filter restricts both controllers to their
// index op, so no create/update/delete surface may leak.
func TestQueryAndCustomFieldSurface(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "testdata", "openapi", "redmine-6.1.3.yaml")
	ops := loadOperations(t, path)

	for _, opID := range []string{"getQueries", "getCustomFieldDefinitions"} {
		if _, ok := ops[opID]; !ok {
			t.Errorf("missing index operation %s", opID)
		}
	}

	for _, opID := range []string{
		"createQuery", "updateQuery", "updateQuery2", "deleteQuery",
		"createCustomField", "updateCustomField", "updateCustomField2", "deleteCustomField",
	} {
		if _, ok := ops[opID]; ok {
			t.Errorf("accept_api_auth filter leak: %s must not be emitted", opID)
		}
	}
}

// TestRenderOkWritesPinEmptyResponses pins the render_api_ok contract: news
// create/update, membership update, group update, and attachment update reply
// 204 No Content with no response body.
func TestRenderOkWritesPinEmptyResponses(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "testdata", "openapi", "redmine-6.1.3.yaml")
	ops := loadOperations(t, path)

	for _, opID := range []string{
		"createNews", "createProjectNews",
		"updateNews", "updateNews2",
		"updateMembership",
		"updateGroup",
		"updateAttachment",
	} {
		op, ok := ops[opID].(map[string]any)
		if !ok {
			t.Fatalf("operation %s not found", opID)
		}
		responses, ok := op["responses"].(map[string]any)
		if !ok {
			t.Fatalf("operation %s has no responses", opID)
		}
		resp204, ok := responses["204"].(map[string]any)
		if !ok {
			t.Errorf("%s: missing 204 response; have %v", opID, keysOf(responses))
			continue
		}
		if _, ok := resp204["content"]; ok {
			t.Errorf("%s: 204 must not carry a response body", opID)
		}
	}

	// GetNews2 (show) is a normal 200 with a body.
	show := ops["getNews2"].(map[string]any)["responses"].(map[string]any)
	if _, ok := show["200"]; !ok {
		t.Errorf("getNews2: missing 200 response")
	}
}

func keysOf(m map[string]any) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	return keys
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
