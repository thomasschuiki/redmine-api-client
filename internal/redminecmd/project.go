package redminecmd

import (
	"context"
	"fmt"
	"os"

	"github.com/tom-redmine/go-redmine-cli/internal/client"
	"github.com/tom-redmine/go-redmine-cli/internal/output"
	"github.com/urfave/cli/v3"
)

func projectCommand() *cli.Command {
	return &cli.Command{
		Name:  "project",
		Usage: "Manage projects",
		Description: `Create, list, view, and delete Redmine projects.
Projects are the top-level container for issues, wiki pages,
time entries, and versions.`,
		Commands: []*cli.Command{
			{
				Name:      "list",
				Usage:     "List projects",
				ArgsUsage: " ",
				Description: `List all visible projects.

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
				resp, err := cl.ListProjects(ctx,
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
				Usage:     "Get a project by identifier",
				ArgsUsage: "<identifier>",
				Description: `Retrieve full details for a project by its URL identifier (slug).
The identifier is the short name used in URLs, not the numeric ID.

If the exact identifier is not found, close matches are suggested.`,
				Action: func(ctx context.Context, c *cli.Command) error {
					identifier := c.Args().First()
					if identifier == "" {
						return fmt.Errorf("project identifier is required")
					}
				cl, err := newClient(c)
				if err != nil {
					return err
				}
				project, err := cl.ResolveProject(ctx, identifier)
				if err != nil {
					return err
				}
				if err := output.Print(project, c.String("output")); err != nil {
					return err
				}
					return nil
				},
			},
			{
				Name:      "create",
				Usage:     "Create a new project",
				ArgsUsage: " ",
				Description: `Create a new project in Redmine.

Both --name and --identifier are required. The identifier must be unique
and is used as the URL slug for the project.`,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "name", Usage: "Project name", Required: true},
					&cli.StringFlag{Name: "identifier", Usage: "Project identifier (URL slug)", Required: true},
					&cli.StringFlag{Name: "description", Usage: "Project description"},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					var req client.ProjectCreateRequest
					req.Project.Name = c.String("name")
					req.Project.Identifier = c.String("identifier")
					req.Project.Description = c.String("description")

				cl, err := newClient(c)
				if err != nil {
					return err
				}
				project, err := cl.CreateProject(ctx, req)
				if err != nil {
					return err
				}
				if err := output.Print(project, c.String("output")); err != nil {
					return err
				}
					return nil
				},
			},
			{
				Name:      "delete",
				Usage:     "Delete a project",
				ArgsUsage: "<identifier>",
				Description: `Permanently delete a project by its identifier.

This will remove the project and all associated data (issues, wiki pages,
time entries, etc.). This action cannot be undone.`,
				Action: func(ctx context.Context, c *cli.Command) error {
					identifier := c.Args().First()
					if identifier == "" {
						return fmt.Errorf("project identifier is required")
					}
				cl, err := newClient(c)
				if err != nil {
					return err
				}
				if err := cl.DeleteProject(ctx, identifier); err != nil {
					return err
				}
					fmt.Fprintf(os.Stdout, "Project %s deleted\n", identifier)
					return nil
				},
			},
		},
	}
}
