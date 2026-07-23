package output

import (
	"encoding/json"
	"fmt"
	"os"

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
