package redminecmd

import (
	"context"
	"fmt"

	"github.com/tom-redmine/go-redmine-cli/internal/output"
	"github.com/urfave/cli/v3"
)

func statusCommand() *cli.Command {
	return &cli.Command{
		Name:  "status",
		Usage: "List issue statuses",
		Description: `Display all available issue statuses with their numeric IDs.

Use the ID with --status-id when listing or filtering issues.`,
		Commands: []*cli.Command{
			{
				Name:      "list",
				Usage:     "List issue statuses",
				ArgsUsage: " ",
				Description: `List all issue statuses configured in Redmine.

Output shows ID, name, and whether the status is a closed state.
Use --output json for machine-parseable output.`,
				Action: func(ctx context.Context, c *cli.Command) error {
				cl, err := newClient(c)
				if err != nil {
					return err
				}
				resp, err := cl.ListIssueStatuses(ctx)
				if err != nil {
					return err
				}
				if c.String("output") == "json" {
					if err := output.Print(resp, "json"); err != nil {
						return err
					}
					} else {
						for _, s := range resp.IssueStatuses {
							closed := ""
							if s.IsClosed {
								closed = " (closed)"
							}
							fmt.Printf("% 4d  %s%s\n", s.ID, s.Name, closed)
						}
					}
					return nil
				},
			},
		},
	}
}
