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
	{"redmine-5.1.12.yaml", "5.1.12", 29, 69, 50},
	{"redmine-6.0.10.yaml", "6.0.10", 29, 69, 50},
	{"redmine-6.1.3.yaml", "6.1.3", 29, 69, 50},
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
