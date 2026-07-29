package redminecmd

import (
	"context"

	"github.com/tom-redmine/go-redmine-cli/internal/output"
	"github.com/urfave/cli/v3"
)

func versionCommand() *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "Manage versions",
		Description: `List and manage project versions (milestones).

Versions are used to group issues into milestones or releases
within a project.`,
		Commands: []*cli.Command{
			{
				Name:      "list",
				Usage:     "List versions for a project",
				ArgsUsage: " ",
				Description: `List all versions (milestones) for a given project.

The --project flag is required and must be a project identifier.`,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "project", Usage: "Project identifier", Required: true},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
				cl, err := newClient(c)
				if err != nil {
					return err
				}
				resp, err := cl.ListVersions(ctx, c.String("project"))
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
