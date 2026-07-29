package redminecmd

import (
	"context"
	"fmt"

	"github.com/tom-redmine/go-redmine-cli/internal/output"
	"github.com/urfave/cli/v3"
)

func trackerCommand() *cli.Command {
	return &cli.Command{
		Name:  "tracker",
		Usage: "List issue trackers",
		Description: `Display all available issue trackers with their numeric IDs.

Use the ID with --tracker-id or --tracker when creating or updating issues.`,
		Commands: []*cli.Command{
			{
				Name:      "list",
				Usage:     "List trackers",
				ArgsUsage: " ",
				Description: `List all trackers configured in Redmine.

Output shows ID and name. Use --output json for machine-parseable output.`,
				Action: func(ctx context.Context, c *cli.Command) error {
				cl, err := newClient(c)
				if err != nil {
					return err
				}
				resp, err := cl.ListTrackers(ctx)
				if err != nil {
					return err
				}
				if c.String("output") == "json" {
					if err := output.Print(resp, "json"); err != nil {
						return err
					}
					} else {
						for _, t := range resp.Trackers {
							fmt.Printf("% 4d  %s\n", t.ID, t.Name)
						}
					}
					return nil
				},
			},
		},
	}
}
