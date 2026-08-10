package redminecmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/tom-redmine/go-redmine-cli/internal/client"
	"github.com/tom-redmine/go-redmine-cli/internal/output"
	"github.com/urfave/cli/v3"
)

func newsCommand() *cli.Command {
	return &cli.Command{
		Name:  "news",
		Usage: "Manage news items",
		Description: `List and create news items in Redmine.

News items belong to a project and are visible to project members.`,
		Commands: []*cli.Command{
			{
				Name:      "list",
				Usage:     "List news items",
				ArgsUsage: " ",
				Description: `List news, optionally scoped to a project with --project.

Supports pagination with --limit and --offset flags.`,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "project", Usage: "Project identifier"},
					&cli.IntFlag{Name: "limit", Usage: "Max results", Value: 25},
					&cli.IntFlag{Name: "offset", Usage: "Result offset", Value: 0},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					cl, err := newClient(c)
					if err != nil {
						return err
					}
					resp, err := cl.ListNews(ctx, c.String("project"),
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
				Name:      "get",
				Usage:     "Get a news item",
				ArgsUsage: "<id>",
				Description: `Show a single news item by its numeric ID.

Use --include to expand associated data (attachments, comments).`,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "include", Usage: "Associated data to include (attachments, comments)"},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					id, err := strconv.Atoi(c.Args().First())
					if err != nil {
						return fmt.Errorf("invalid news ID: %w", err)
					}
					cl, err := newClient(c)
					if err != nil {
						return err
					}
					news, err := cl.GetNews(ctx, id, c.String("include"))
					if err != nil {
						return err
					}
					if err := output.Print(news, c.String("output")); err != nil {
						return err
					}
					return nil
				},
			},
			{
				Name:      "create",
				Usage:     "Create a news item",
				ArgsUsage: " ",
				Description: `Create a news item in a project.

--title is required. Pass the project with --project (identifier or ID);
the news is created in the default project when omitted.`,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "title", Usage: "News title", Required: true},
					&cli.StringFlag{Name: "project", Usage: "Project identifier"},
					&cli.StringFlag{Name: "summary", Usage: "News summary"},
					&cli.StringFlag{Name: "description", Usage: "News description"},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					var req client.NewsCreateRequest
					req.News.Title = c.String("title")
					summary := c.String("summary")
					if c.IsSet("summary") {
						req.News.Summary = &summary
					}
					description := c.String("description")
					if c.IsSet("description") {
						req.News.Description = &description
					}

					cl, err := newClient(c)
					if err != nil {
						return err
					}
					if c.String("project") != "" {
						project, err := cl.ResolveProject(ctx, c.String("project"))
						if err != nil {
							return err
						}
						req.News.ProjectID = &project.ID
					}
					news, err := cl.CreateNews(ctx, req)
					if err != nil {
						return err
					}
					if err := output.Print(news, c.String("output")); err != nil {
						return err
					}
					return nil
				},
			},
		},
	}
}
