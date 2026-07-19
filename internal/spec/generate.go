package spec

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/generator/golang"
)

// GenerateModels reads a bundled OpenAPI spec and generates Go model types
// for all component schemas.
func GenerateModels(specPath, outDir, pkgName string) error {
	data, err := os.ReadFile(specPath)
	if err != nil {
		return fmt.Errorf("reading spec: %w", err)
	}

	config := datamodel.NewDocumentConfiguration()
	config.SpecFilePath = specPath

	doc, err := libopenapi.NewDocumentWithConfiguration(data, config)
	if err != nil {
		return fmt.Errorf("parsing spec: %w", err)
	}

	model, err := doc.BuildV3Model()
	if err != nil {
		return fmt.Errorf("building model: %w", err)
	}

	if model.Model.Components == nil || model.Model.Components.Schemas == nil {
		return fmt.Errorf("no component schemas found in spec")
	}

	gen := golang.NewGenerator(
		golang.WithPackageName(pkgName),
		golang.WithGeneratedComment(true),
		golang.WithEnumConstants(true),
		golang.WithOptionalFieldsAsPointers(true),
		golang.WithOmitEmpty(true),
		golang.WithFormatMapping("date-time", "time.Time", "time"),
	)

	file, err := gen.RenderSchemas(model.Model.Components.Schemas)
	if err != nil {
		return fmt.Errorf("generating models: %w", err)
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	outPath := filepath.Join(outDir, "models.go")
	if err := os.WriteFile(outPath, file.Source, 0o644); err != nil {
		return fmt.Errorf("writing models: %w", err)
	}

	fmt.Printf("Generated %d types to %s\n", len(file.Types), outPath)

	if len(file.Diagnostics) > 0 {
		fmt.Fprintf(os.Stderr, "\nDiagnostics:\n")
		for _, d := range file.Diagnostics {
			fmt.Fprintf(os.Stderr, "  %s: %s\n", d.Code, d.Message)
		}
	}

	return nil
}
