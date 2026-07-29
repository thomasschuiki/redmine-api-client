package redminecmd

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/tom-redmine/go-redmine-cli/internal/client"
	"github.com/tom-redmine/go-redmine-cli/internal/output"
	"github.com/urfave/cli/v3"
)

func timeEntryCommand() *cli.Command {
	return &cli.Command{
		Name:  "time-entry",
		Usage: "Manage time entries",
		Description: `Log and manage time entries in Redmine.

Time entries can be associated with issues or projects directly.
Each entry tracks hours spent and can include a comment and activity type.`,
		Commands: []*cli.Command{
			{
				Name:      "list",
				Usage:     "List time entries",
				ArgsUsage: " ",
				Description: `List time entries with optional filtering.

Filter by project using --project and/or by issue using --issue.
Supports pagination with --limit and --offset flags.`,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "project", Usage: "Project identifier"},
					&cli.IntFlag{Name: "issue", Usage: "Issue ID"},
					&cli.IntFlag{Name: "limit", Usage: "Max results", Value: 25},
					&cli.IntFlag{Name: "offset", Usage: "Result offset", Value: 0},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
				cl, err := newClient(c)
				if err != nil {
					return err
				}
				resp, err := cl.ListTimeEntries(ctx,
					c.String("project"),
					c.Int("issue"),
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
				Name:      "create",
				Usage:     "Create a time entry",
				ArgsUsage: " ",
				Description: `Log a new time entry.

--hours is required. Associate the entry with an issue (--issue) or a
project (--project). Optionally specify an activity type and comment.`,
				Flags: []cli.Flag{
					&cli.IntFlag{Name: "issue", Usage: "Issue ID"},
					&cli.StringFlag{Name: "project", Usage: "Project identifier"},
					&cli.Float64Flag{Name: "hours", Usage: "Hours spent", Required: true},
					&cli.IntFlag{Name: "activity", Usage: "Activity ID"},
					&cli.StringFlag{Name: "comment", Usage: "Comment"},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					var req client.TimeEntryCreateRequest
					req.TimeEntry.Hours = c.Float64("hours")
					req.TimeEntry.Comments = c.String("comment")

					if c.IsSet("issue") {
						v := c.Int("issue")
						req.TimeEntry.IssueID = &v
					}
					if c.IsSet("activity") {
						v := c.Int("activity")
						req.TimeEntry.ActivityID = &v
					}

				cl, err := newClient(c)
				if err != nil {
					return err
				}
				entry, err := cl.CreateTimeEntry(ctx, req)
				if err != nil {
					return err
				}
				if err := output.Print(entry, c.String("output")); err != nil {
					return err
				}
					return nil
				},
			},
			{
				Name:      "delete",
				Usage:     "Delete a time entry",
				ArgsUsage: "<id>",
				Description: `Permanently delete a time entry by its numeric ID.
This action cannot be undone.`,
				Action: func(ctx context.Context, c *cli.Command) error {
					id, err := strconv.Atoi(c.Args().First())
					if err != nil {
						return fmt.Errorf("invalid time entry ID: %w", err)
					}
				cl, err := newClient(c)
				if err != nil {
					return err
				}
				if err := cl.DeleteTimeEntry(ctx, id); err != nil {
					return err
				}
					fmt.Fprintf(os.Stdout, "Time entry %d deleted\n", id)
					return nil
				},
			},
		},
	}
}
