package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func Print(v any, format string) {
	var data []byte
	var err error

	switch format {
	case "json":
		data, err = json.MarshalIndent(v, "", "  ")
	default:
		format = "yaml"
		data, err = yaml.Marshal(v)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error marshalling output as %s: %v\n", format, err)
		os.Exit(1)
	}

	fmt.Println(string(data))
}

// FilterFields returns v with only the listed top-level keys kept.
// For slices, each element is filtered individually.
// Unknown field names are silently ignored.
func FilterFields(v any, fields []string) any {
	set := make(map[string]bool, len(fields))
	for _, f := range fields {
		set[strings.TrimSpace(f)] = true
	}

	raw, err := json.Marshal(v)
	if err != nil {
		return v
	}

	// Try object first
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err == nil {
		return filterMap(obj, set)
	}

	// Try array
	var arr []map[string]any
	if err := json.Unmarshal(raw, &arr); err == nil {
		out := make([]map[string]any, len(arr))
		for i, item := range arr {
			out[i] = filterMap(item, set)
		}
		return out
	}

	return v
}

func filterMap(m map[string]any, keep map[string]bool) map[string]any {
	result := make(map[string]any, len(keep))
	for k, v := range m {
		if keep[k] {
			result[k] = v
		}
	}
	return result
}
