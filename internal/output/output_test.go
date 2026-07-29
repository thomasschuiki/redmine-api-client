package output

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

type testStruct struct {
	Name string `json:"name" yaml:"name"`
	ID   int    `json:"id" yaml:"id"`
}

func captureOutput(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestPrint_YAML(t *testing.T) {
	v := testStruct{ID: 42, Name: "hello"}
	out := captureOutput(func() {
		_ = Print(v, "yaml")
	})

	if !strings.Contains(out, "id: 42") {
		t.Errorf("expected YAML id: 42, got: %s", out)
	}
	if !strings.Contains(out, "name: hello") {
		t.Errorf("expected YAML name: hello, got: %s", out)
	}
	if strings.Contains(out, "\"id\"") {
		t.Errorf("YAML should not have quoted keys, got: %s", out)
	}
}

func TestPrint_JSON(t *testing.T) {
	v := testStruct{ID: 42, Name: "hello"}
	out := captureOutput(func() {
		_ = Print(v, "json")
	})

	if !strings.Contains(out, `"id": 42`) {
		t.Errorf("expected JSON id: 42, got: %s", out)
	}
	if !strings.Contains(out, `"name": "hello"`) {
		t.Errorf("expected JSON name: hello, got: %s", out)
	}
}

func TestPrint_DefaultsToYAML(t *testing.T) {
	v := testStruct{ID: 1, Name: "test"}
	out := captureOutput(func() {
		_ = Print(v, "anything")
	})

	if !strings.Contains(out, "id: 1") {
		t.Errorf("default should be YAML, got: %s", out)
	}
}

func TestPrint_Error(t *testing.T) {
	err := Print(make(chan int), "json")
	if err == nil {
		t.Error("Print with unencodable value should return error")
	}
}

func TestFilterFields_Object(t *testing.T) {
	v := map[string]any{"id": 1, "name": "test", "hidden": "secret"}
	got := FilterFields(v, []string{"id", "name"})
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("FilterFields returned %T, want map[string]any", got)
	}
	if _, exists := m["hidden"]; exists {
		t.Error("FilterFields kept 'hidden' field")
	}
	if m["id"] != float64(1) {
		t.Errorf("FilterFields id = %v, want 1", m["id"])
	}
}

func TestFilterFields_Slice(t *testing.T) {
	v := []map[string]any{
		{"id": 1, "name": "a", "extra": "x"},
		{"id": 2, "name": "b", "extra": "y"},
	}
	got := FilterFields(v, []string{"id"})
	arr, ok := got.([]map[string]any)
	if !ok {
		t.Fatalf("FilterFields returned %T, want []map[string]any", got)
	}
	if len(arr) != 2 {
		t.Fatalf("len = %d, want 2", len(arr))
	}
	if _, exists := arr[0]["name"]; exists {
		t.Error("FilterFields kept 'name' field")
	}
	if arr[1]["id"] != float64(2) {
		t.Errorf("arr[1].id = %v, want 2", arr[1]["id"])
	}
}

func TestFilterFields_UnknownFields(t *testing.T) {
	v := map[string]any{"a": 1}
	got := FilterFields(v, []string{"nonexistent"})
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("FilterFields returned %T", got)
	}
	if len(m) != 0 {
		t.Errorf("FilterFields = %v, want empty map", m)
	}
}

func TestFilterFields_Passthrough(t *testing.T) {
	v := 42
	got := FilterFields(v, []string{"id"})
	if got != v {
		t.Errorf("FilterFields = %v, want %v", got, v)
	}
}
