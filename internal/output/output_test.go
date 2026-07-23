package output

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

type testStruct struct {
	ID   int    `json:"id" yaml:"id"`
	Name string `json:"name" yaml:"name"`
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
		Print(v, "yaml")
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
		Print(v, "json")
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
		Print(v, "anything")
	})

	if !strings.Contains(out, "id: 1") {
		t.Errorf("default should be YAML, got: %s", out)
	}
}
