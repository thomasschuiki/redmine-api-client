package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/tom-redmine/go-redmine-cli/internal/coverage"
	"github.com/tom-redmine/go-redmine-cli/internal/spec"
	"github.com/urfave/cli/v3"
)

// Run creates and runs the CLI application.
func Run() *cli.Command {
	return &cli.Command{
		Name:  "redmine-spec",
		Usage: "OpenAPI spec tooling for the Redmine API",
		Commands: []*cli.Command{
			specCommand(),
			coverageCommand(),
			generateModelsCommand(),
		},
	}
}

func specCommand() *cli.Command {
	defaultOut := "docs/openapi/openapi.yaml"

	return &cli.Command{
		Name:  "spec",
		Usage: "OpenAPI spec operations",
		Commands: []*cli.Command{
			{
				Name:  "validate",
				Usage: "Validate an OpenAPI spec for structural errors",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "spec", Usage: "Spec file path", Value: defaultOut},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					specPath := c.String("spec")
					fmt.Printf("Validating %s\n", specPath)
					result, err := spec.Validate(specPath)
					if err != nil {
						return err
					}

					fmt.Printf("Paths:       %d\n", result.PathCount)
					fmt.Printf("Operations:  %d\n", result.OperationCount)
					fmt.Printf("Schemas:     %d\n", result.SchemaCount)
					fmt.Printf("Parameters:  %d\n", result.ParameterCount)
					fmt.Printf("Responses:   %d\n", result.ResponseCount)
					fmt.Printf("Sec schemes: %d\n", result.SecuritySchemeCount)
					fmt.Printf("Tags:        %d\n", result.TagCount)

					if len(result.ParseErrors) > 0 {
						fmt.Fprintf(os.Stderr, "\nErrors:\n")
						for _, e := range result.ParseErrors {
							fmt.Fprintf(os.Stderr, "  %v\n", e)
						}
						return fmt.Errorf("spec validation failed with %d error(s)", len(result.ParseErrors))
					}

					fmt.Println("\nOK: spec is valid.")
					return nil
				},
			},
		},
	}
}

func coverageCommand() *cli.Command {
	return &cli.Command{
		Name:  "coverage",
		Usage: "Check spec coverage against Redmine routes",
		Commands: []*cli.Command{
			{
				Name:  "check",
				Usage: "Compare a Redmine route snapshot against the OpenAPI spec",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "snapshot", Usage: "Path to the route snapshot file (YAML or JSON)"},
					&cli.StringFlag{Name: "spec", Usage: "Path to the OpenAPI spec", Value: "docs/openapi/openapi.yaml"},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					snapshotPath := c.String("snapshot")
					specPath := c.String("spec")

					snap, err := coverage.LoadSnapshot(snapshotPath)
					if err != nil {
						return err
					}

					fmt.Printf("Loaded snapshot: %s (%d routes)\n", snap.RedmineVersion, len(snap.APIRoutes))

					report, err := coverage.Check(snap, specPath)
					if err != nil {
						return err
					}

					fmt.Print(coverage.FormatReport(report))

					if len(report.MissingInSpec) > 0 {
						return fmt.Errorf("%d route(s) missing from spec", len(report.MissingInSpec))
					}
					return nil
				},
			},
		},
	}
}

func generateModelsCommand() *cli.Command {
	return &cli.Command{
		Name:  "generate-models",
		Usage: "Generate Go model types from the OpenAPI spec",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "spec", Usage: "Path to the OpenAPI spec", Value: "docs/openapi/openapi.yaml"},
			&cli.StringFlag{Name: "out", Usage: "Output directory for generated models", Value: "internal/models"},
			&cli.StringFlag{Name: "package", Usage: "Go package name", Value: "models"},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			specPath := c.String("spec")
			outDir := c.String("out")
			pkgName := c.String("package")

			fmt.Printf("Generating models from %s -> %s\n", specPath, outDir)
			if err := spec.GenerateModels(specPath, outDir, pkgName); err != nil {
				return err
			}
			fmt.Println("Done.")
			return nil
		},
	}
}
