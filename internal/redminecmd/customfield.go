package redminecmd

import (
	"context"

	"github.com/tom-redmine/go-redmine-cli/internal/output"
	"github.com/urfave/cli/v3"
)

func customFieldCommand() *cli.Command {
	return &cli.Command{
		Name:  "custom-field",
		Usage: "List custom field definitions",
		Description: `List custom field definitions configured in Redmine.

Useful for discovering the ID and format of a field before passing
--custom-field ID=VALUE to redmine issue create/update.`,
		Commands: []*cli.Command{
			{
				Name:      "list",
				Usage:     "List custom fields",
				ArgsUsage: " ",
				Description: `List all custom field definitions with their formats.

Use the IDs with --custom-field ID=VALUE when creating or updating issues.`,
				Action: func(ctx context.Context, c *cli.Command) error {
					cl, err := newClient(c)
					if err != nil {
						return err
					}
					resp, err := cl.ListCustomFieldDefinitions(ctx)
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
