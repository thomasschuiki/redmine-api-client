package redminecmd

import (
	"context"

	"github.com/tom-redmine/go-redmine-cli/internal/client"
	"github.com/tom-redmine/go-redmine-cli/internal/output"
	"github.com/urfave/cli/v3"
)

func queryCommand() *cli.Command {
	return &cli.Command{
		Name:  "query",
		Usage: "List saved queries",
		Description: `List saved issue queries visible to the current user.

Plain users see their own and public queries; administrators see all.
Use the returned IDs with 'redmine issue list --query <ID>'.`,
		Commands: []*cli.Command{
			{
				Name:      "list",
				Usage:     "List saved queries",
				ArgsUsage: " ",
				Description: `List queries with their IDs and names.

Supports pagination with --limit and --offset flags.`,
				Flags: []cli.Flag{
					&cli.IntFlag{Name: "limit", Usage: "Max results", Value: 25},
					&cli.IntFlag{Name: "offset", Usage: "Result offset", Value: 0},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					cl, err := newClient(c)
					if err != nil {
						return err
					}
					resp, err := cl.ListQueries(ctx, client.ListOpts{Offset: c.Int("offset"), Limit: c.Int("limit")})
					if err != nil {
						return err
					}
					if err := output.Print(resp, c.String("output")); err != nil {
						return err
					}
					return nil
				},
			},
		},
	}
}
