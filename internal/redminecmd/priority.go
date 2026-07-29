package redminecmd

import (
	"context"
	"fmt"

	"github.com/tom-redmine/go-redmine-cli/internal/output"
	"github.com/urfave/cli/v3"
)

func priorityCommand() *cli.Command {
	return &cli.Command{
		Name:  "priority",
		Usage: "List issue priorities",
		Description: `Display all available issue priorities with their numeric IDs.

Use the ID with --priority-id when creating or filtering issues.`,
		Commands: []*cli.Command{
			{
				Name:      "list",
				Usage:     "List issue priorities",
				ArgsUsage: " ",
				Description: `List all issue priorities configured in Redmine.

Output shows ID and name. Default priority is marked with an asterisk.
Use --output json for machine-parseable output.`,
				Action: func(ctx context.Context, c *cli.Command) error {
				cl, err := newClient(c)
				if err != nil {
					return err
				}
				resp, err := cl.ListIssuePriorities(ctx)
				if err != nil {
					return err
				}
				if c.String("output") == "json" {
					if err := output.Print(resp, "json"); err != nil {
						return err
					}
					} else {
						for _, p := range resp.IssuePriorities {
							def := ""
							if p.IsDefault {
								def = " *"
							}
							fmt.Printf("% 4d  %s%s\n", p.ID, p.Name, def)
						}
					}
					return nil
				},
			},
		},
	}
}
