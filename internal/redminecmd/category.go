package redminecmd

import (
	"context"
	"fmt"

	"github.com/tom-redmine/go-redmine-cli/internal/output"
	"github.com/urfave/cli/v3"
)

func categoryCommand() *cli.Command {
	return &cli.Command{
		Name:  "category",
		Usage: "List issue categories",
		Description: `Display issue categories for a project with their numeric IDs.

Use the ID with --category-id when listing or filtering issues.`,
		Commands: []*cli.Command{
			{
				Name:      "list",
				Usage:     "List categories for a project",
				ArgsUsage: " ",
				Description: `List all issue categories for a given project.

The --project flag is required. Output shows ID, name, and assignee.
Use --output json for machine-parseable output.`,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "project", Usage: "Project identifier", Required: true},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
				cl, err := newClient(c)
				if err != nil {
					return err
				}
				resp, err := cl.ListIssueCategories(ctx, c.String("project"))
				if err != nil {
					return err
				}
				if c.String("output") == "json" {
					if err := output.Print(resp, "json"); err != nil {
						return err
					}
					} else {
						for _, cat := range resp.IssueCategories {
							assigned := ""
							if cat.AssignedTo != nil {
								assigned = " → " + cat.AssignedTo.Name
							}
							fmt.Printf("% 4d  %s%s\n", cat.ID, cat.Name, assigned)
						}
					}
					return nil
				},
			},
		},
	}
}
