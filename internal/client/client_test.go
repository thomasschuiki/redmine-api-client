package client

import (
	"testing"
	"time"
)

func TestParseCustomField(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantID  int
		wantVal string
		wantErr bool
	}{
		{"simple", "1=hello", 1, "hello", false},
		{"with spaces", "42=a b c", 42, "a b c", false},
		{"no equals", "invalid", 0, "", true},
		{"empty id", "=value", 0, "", true},
		{"non-numeric id", "abc=val", 0, "", true},
		{"trailing equals", "5=foo=bar", 5, "foo=bar", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCustomField(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseCustomField(%q) expected error", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseCustomField(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got.ID != tt.wantID {
				t.Errorf("ParseCustomField(%q).ID = %d, want %d", tt.input, got.ID, tt.wantID)
			}
			if got.Value != tt.wantVal {
				t.Errorf("ParseCustomField(%q).Value = %q, want %q", tt.input, got.Value, tt.wantVal)
			}
			if len(got.Values) != 1 || got.Values[0] != tt.wantVal {
				t.Errorf("ParseCustomField(%q).Values = %v, want [%q]", tt.input, got.Values, tt.wantVal)
			}
		})
	}
}

func TestMergeCustomFields(t *testing.T) {
	tests := []struct {
		name   string
		input  []CustomFieldValue
		output []CustomFieldValue
	}{
		{
			name:   "empty",
			input:  nil,
			output: []CustomFieldValue{},
		},
		{
			name: "single field",
			input: []CustomFieldValue{
				{ID: 1, Value: "a"},
			},
			output: []CustomFieldValue{
				{ID: 1, Value: "a"},
			},
		},
		{
			name: "multiple distinct fields",
			input: []CustomFieldValue{
				{ID: 1, Value: "a"},
				{ID: 2, Value: "b"},
			},
			output: []CustomFieldValue{
				{ID: 1, Value: "a"},
				{ID: 2, Value: "b"},
			},
		},
		{
			name: "duplicate ids become multi-value",
			input: []CustomFieldValue{
				{ID: 1, Value: "x"},
				{ID: 1, Value: "y"},
				{ID: 2, Value: "z"},
			},
			output: []CustomFieldValue{
				{ID: 1, Values: []string{"x", "y"}},
				{ID: 2, Value: "z"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeCustomFields(tt.input)
			if len(got) != len(tt.output) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.output))
			}
			for i := range got {
				if got[i].ID != tt.output[i].ID {
					t.Errorf("[%d].ID = %d, want %d", i, got[i].ID, tt.output[i].ID)
				}
				if got[i].Value != tt.output[i].Value {
					t.Errorf("[%d].Value = %q, want %q", i, got[i].Value, tt.output[i].Value)
				}
				if len(got[i].Values) != len(tt.output[i].Values) {
					t.Errorf("[%d].Values len = %d, want %d", i, len(got[i].Values), len(tt.output[i].Values))
				}
			}
		})
	}
}

func TestContainsFilter(t *testing.T) {
	issues := []Issue{
		{Subject: "Fix login bug", Description: "Users cannot log in", Status: &IDName{Name: "New"}, Tracker: &IDName{Name: "Bug"}, Author: &IDName{Name: "Alice"}},
		{Subject: "Add dark mode", Description: "Implement dark theme", Status: &IDName{Name: "In Progress"}, Tracker: &IDName{Name: "Feature"}, Author: &IDName{Name: "Bob"}},
	}

	tests := []struct {
		name           string
		text           string
		caseInsensitive bool
		want           int
	}{
		{"substring match", "login", true, 1},
		{"case insensitive", "LOGIN", true, 1},
		{"case sensitive (search text always lowered)", "LOGIN", false, 1},
		{"no match", "nonexistent", true, 0},
		{"match in description", "dark theme", true, 1},
		{"match in author", "Alice", true, 1},
		{"empty text", "", true, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ContainsFilter(issues, tt.text, tt.caseInsensitive)
			if len(got) != tt.want {
				t.Errorf("ContainsFilter = %d results, want %d", len(got), tt.want)
			}
		})
	}
}

func TestRegexFilter(t *testing.T) {
	issues := []Issue{
		{Subject: "Fix login bug #123", Description: "Critical", Status: &IDName{Name: "New"}, Tracker: &IDName{Name: "Bug"}, Author: &IDName{Name: "Alice"}},
		{Subject: "Release v2.0", Description: "Major release", Status: &IDName{Name: "Done"}, Tracker: &IDName{Name: "Feature"}, Author: &IDName{Name: "Bob"}},
	}

	tests := []struct {
		name           string
		pattern        string
		caseInsensitive bool
		want           int
		wantErr        bool
	}{
		{"regex match", `#\d+`, true, 1, false},
		{"regex no match", `INVALID\d+`, true, 0, false},
		{"case insensitive regex", `FIX`, true, 1, false},
		{"case sensitive regex", `FIX`, false, 0, false},
		{"invalid regex", `[invalid`, false, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RegexFilter(issues, tt.pattern, tt.caseInsensitive)
			if tt.wantErr {
				if err == nil {
					t.Error("RegexFilter expected error")
				}
				return
			}
			if err != nil {
				t.Errorf("RegexFilter unexpected error: %v", err)
				return
			}
			if len(got) != tt.want {
				t.Errorf("RegexFilter = %d results, want %d", len(got), tt.want)
			}
		})
	}
}

func TestValidateDate(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"2024-01-15", true},
		{"1999-12-31", true},
		{"", false},
		{"not-a-date", false},
		{"2024/01/15", false},
		{"2024-1-1", false},
		{"20240115", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := ValidateDate(tt.input)
			if tt.valid && err != nil {
				t.Errorf("ValidateDate(%q) = %v, want nil", tt.input, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("ValidateDate(%q) = nil, want error", tt.input)
			}
		})
	}
}

func TestValidateEstimatedHours(t *testing.T) {
	tests := []struct {
		input float64
		valid bool
	}{
		{0, true},
		{1.5, true},
		{100, true},
		{-1, false},
		{-0.01, false},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			err := ValidateEstimatedHours(tt.input)
			if tt.valid && err != nil {
				t.Errorf("ValidateEstimatedHours(%v) = %v, want nil", tt.input, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("ValidateEstimatedHours(%v) = nil, want error", tt.input)
			}
		})
	}
}

func TestNormalizeProjectID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"My Project", "my-project"},
		{"  Hello  ", "hello"},
		{"UPPERCASE", "uppercase"},
		{"special chars!", "special-chars!"},
		{"already-normalized", "already-normalized"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeProjectID(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeProjectID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestListOptsParams(t *testing.T) {
	tests := []struct {
		name string
		opts ListOpts
		want string
	}{
		{"empty", ListOpts{}, ""},
		{"offset only", ListOpts{Offset: 10}, "offset=10"},
		{"limit only", ListOpts{Limit: 25}, "limit=25"},
		{"both", ListOpts{Offset: 0, Limit: 100}, "limit=100"},
		{"offset and limit", ListOpts{Offset: 20, Limit: 50}, "limit=50&offset=20"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.opts.Params().Encode()
			if got != tt.want {
				t.Errorf("Params() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseRelationType(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"relates", true},
		{"blocks", true},
		{"blocked", true},
		{"precedes", true},
		{"follows", true},
		{"duplicates", true},
		{"duplicated", true},
		{"copied_to", true},
		{"copied_from", true},
		{"invalid", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := ParseRelationType(tt.input)
			if tt.valid && err != nil {
				t.Errorf("ParseRelationType(%q) = %v, want nil", tt.input, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("ParseRelationType(%q) = nil, want error", tt.input)
			}
		})
	}
}

func TestBackoff(t *testing.T) {
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{1, 1 * time.Second},
		{2, 2 * time.Second},
		{3, 4 * time.Second},
		{4, 8 * time.Second},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := backoff(tt.attempt)
			if got != tt.want {
				t.Errorf("backoff(%d) = %v, want %v", tt.attempt, got, tt.want)
			}
		})
	}
}

func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		input     string
		candidate string
		want      bool
	}{
		{"my-project", "my-project", true},
		{"my-proj", "my-project", true},
		{"my-project", "my-proj", false},
		{"xyz", "abcdef", false},
		{"project", "my-project", true},
		{"cat", "dog", false},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := fuzzyMatch(tt.input, tt.candidate)
			if got != tt.want {
				t.Errorf("fuzzyMatch(%q, %q) = %v, want %v", tt.input, tt.candidate, got, tt.want)
			}
		})
	}
}

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"a", "", 1},
		{"", "a", 1},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"abc", "ab", 1},
		{"kitten", "sitting", 3},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := levenshtein(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestSearchField(t *testing.T) {
	hits := searchField("The quick brown fox jumps over the lazy dog", "description", 1, "Test Subject", "fox")
	if len(hits) != 1 {
		t.Fatalf("searchField = %d hits, want 1", len(hits))
	}
	if hits[0].IssueID != 1 {
		t.Errorf("IssueID = %d, want 1", hits[0].IssueID)
	}
	if hits[0].Where != "description" {
		t.Errorf("Where = %q, want %q", hits[0].Where, "description")
	}
	if hits[0].Subject != "Test Subject" {
		t.Errorf("Subject = %q, want %q", hits[0].Subject, "Test Subject")
	}

	// Test multiple matches
	hits = searchField("foo bar foo", "notes", 2, "Another", "foo")
	if len(hits) != 2 {
		t.Fatalf("searchField = %d hits, want 2", len(hits))
	}

	// Test no match
	hits = searchField("abc def", "notes", 3, "Third", "xyz")
	if len(hits) != 0 {
		t.Errorf("searchField = %d hits, want 0", len(hits))
	}
}

func TestExtractSnippet(t *testing.T) {
	text := "The quick brown fox jumps over the lazy dog"
	snippet := extractSnippet(text, 16, 3) // "fox" at position 16
	if len(snippet) == 0 {
		t.Error("extractSnippet returned empty")
	}
}

func TestIssueListOptsParams(t *testing.T) {
	tests := []struct {
		name string
		opts IssueListOpts
		want string
	}{
		{"empty", IssueListOpts{}, ""},
		{"status id", IssueListOpts{StatusID: "*"}, "status_id=%2A"},
		{"tracker id", IssueListOpts{TrackerID: 5}, "tracker_id=5"},
		{"sort", IssueListOpts{Sort: "created_on:desc"}, "sort=created_on%3Adesc"},
		{"include", IssueListOpts{Include: "journals,relations"}, "include=journals%2Crelations"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.opts.Params().Encode()
			if got != tt.want {
				t.Errorf("Params() = %q, want %q", got, tt.want)
			}
		})
	}
}
