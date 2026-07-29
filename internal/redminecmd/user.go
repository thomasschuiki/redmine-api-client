package redminecmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/tom-redmine/go-redmine-cli/internal/client"
	"github.com/tom-redmine/go-redmine-cli/internal/output"
	"github.com/urfave/cli/v3"
)

func userCommand() *cli.Command {
	return &cli.Command{
		Name:  "user",
		Usage: "Manage users",
		Description: `List and retrieve user information from Redmine.
Use 'current' to view the authenticated user's details.`,
		Commands: []*cli.Command{
			{
				Name:      "list",
				Usage:     "List users",
				ArgsUsage: " ",
				Description: `List all visible users.

Supports pagination with --limit and --offset flags.
By default, returns up to 25 results.`,
				Flags: []cli.Flag{
					&cli.IntFlag{Name: "limit", Usage: "Max results", Value: 25},
					&cli.IntFlag{Name: "offset", Usage: "Result offset", Value: 0},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
				cl, err := newClient(c)
				if err != nil {
					return err
				}
				resp, err := cl.ListUsers(ctx,
					client.ListOpts{Offset: c.Int("offset"), Limit: c.Int("limit")},
				)
				if err != nil {
					return err
				}
				if err := output.Print(resp, c.String("output")); err != nil {
					return err
				}
					return nil
				},
			},
			{
				Name:        "current",
				Usage:       "Get the currently authenticated user",
				ArgsUsage:   " ",
				Description: `Display the profile of the currently authenticated user.
Useful for verifying your API key and checking your user ID.`,
				Action: func(ctx context.Context, c *cli.Command) error {
				cl, err := newClient(c)
				if err != nil {
					return err
				}
				user, err := cl.GetCurrentUser(ctx)
				if err != nil {
					return err
				}
				if err := output.Print(user, c.String("output")); err != nil {
					return err
				}
					return nil
				},
			},
			{
				Name:      "get",
				Usage:     "Get a user by ID",
				ArgsUsage: "<id>",
				Description: `Retrieve full details for a single user by their numeric ID.`,
				Action: func(ctx context.Context, c *cli.Command) error {
					id, err := strconv.Atoi(c.Args().First())
					if err != nil {
						return fmt.Errorf("invalid user ID: %w", err)
					}
				cl, err := newClient(c)
				if err != nil {
					return err
				}
				user, err := cl.GetUser(ctx, id)
				if err != nil {
					return err
				}
				if err := output.Print(user, c.String("output")); err != nil {
					return err
				}
					return nil
				},
			},
		},
	}
}
