package redminecmd

import (
	"context"

	"github.com/tom-redmine/go-redmine-cli/internal/output"
	"github.com/urfave/cli/v3"
)

func wikiCommand() *cli.Command {
	return &cli.Command{
		Name:  "wiki",
		Usage: "Manage wiki pages",
		Description: `List and retrieve wiki pages for Redmine projects.

Wiki pages are associated with a project and identified by their title.`,
		Commands: []*cli.Command{
			{
				Name:      "list",
				Usage:     "List wiki pages for a project",
				ArgsUsage: " ",
				Description: `List all wiki pages for a given project.

The --project flag is required and must be a project identifier.`,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "project", Usage: "Project identifier", Required: true},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
				cl, err := newClient(c)
				if err != nil {
					return err
				}
				resp, err := cl.ListWikiPages(ctx, c.String("project"))
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
				Name:      "get",
				Usage:     "Get a wiki page",
				ArgsUsage: " ",
				Description: `Retrieve the content of a wiki page.

Both --project and --page are required. The --page value is the
title of the wiki page as it appears in the URL.`,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "project", Usage: "Project identifier", Required: true},
					&cli.StringFlag{Name: "page", Usage: "Page title", Required: true},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
				cl, err := newClient(c)
				if err != nil {
					return err
				}
				page, err := cl.GetWikiPage(ctx, c.String("project"), c.String("page"))
				if err != nil {
					return err
				}
				if err := output.Print(page, c.String("output")); err != nil {
					return err
				}
					return nil
				},
			},
		},
	}
}
