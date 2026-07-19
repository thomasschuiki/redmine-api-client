package coverage

import (
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Snapshot represents a captured API surface from a Redmine instance.
type Snapshot struct {
	RedmineVersion string              `json:"redmine_version" yaml:"redmine_version"`
	GeneratedAt    string              `json:"generated_at" yaml:"generated_at"`
	APIRoutes      []Route             `json:"api_routes" yaml:"api_routes"`
	APIControllers map[string][]string `json:"api_controllers" yaml:"api_controllers"`
}

// Route represents a single API endpoint extracted from Redmine.
type Route struct {
	Verb       string `json:"verb" yaml:"verb"`
	Path       string `json:"path" yaml:"path"`
	Controller string `json:"controller" yaml:"controller"`
	Action     string `json:"action" yaml:"action"`
}

// LoadSnapshot reads a snapshot from a JSON or YAML file.
func LoadSnapshot(path string) (*Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading snapshot: %w", err)
	}

	var snap Snapshot

	// Try YAML first (which also handles JSON)
	if err := yaml.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("parsing snapshot: %w", err)
	}

	// Validate it's not empty JSON/YAML
	if len(snap.APIRoutes) == 0 {
		return nil, fmt.Errorf("snapshot contains no routes")
	}

	return &snap, nil
}

// LoadSnapshotJSON reads a snapshot from a JSON file (used by the Docker scanner).
func LoadSnapshotJSON(path string) (*Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading snapshot: %w", err)
	}

	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("parsing snapshot: %w", err)
	}

	if len(snap.APIRoutes) == 0 {
		return nil, fmt.Errorf("snapshot contains no routes")
	}

	return &snap, nil
}

// SaveSnapshot writes a snapshot to a YAML file.
func SaveSnapshot(snap *Snapshot, path string) error {
	data, err := yaml.Marshal(snap)
	if err != nil {
		return fmt.Errorf("marshaling snapshot: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing snapshot: %w", err)
	}

	return nil
}
