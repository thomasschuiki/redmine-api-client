package redminecmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/tom-redmine/go-redmine-cli/internal/client"
	"github.com/tom-redmine/go-redmine-cli/internal/config"
	"github.com/urfave/cli/v3"
)

var redmineClient *client.Client

func newClient() *client.Client {
	if redmineClient == nil {
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		redmineClient = client.New(cfg)
	}
	return redmineClient
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

// Run creates and runs the CLI application.
func Run() *cli.Command {
	return &cli.Command{
		Name:  "redmine",
		Usage: "Redmine CLI client",
		Commands: []*cli.Command{
			issueCommand(),
			projectCommand(),
			userCommand(),
			timeEntryCommand(),
			wikiCommand(),
			versionCommand(),
		},
	}
}

func issueCommand() *cli.Command {
	return &cli.Command{
		Name:  "issue",
		Usage: "Manage issues",
		Commands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List issues",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "project", Usage: "Project identifier"},
					&cli.IntFlag{Name: "limit", Usage: "Max results", Value: 25},
					&cli.IntFlag{Name: "offset", Usage: "Result offset", Value: 0},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					resp, err := newClient().ListIssues(
						c.String("project"),
						client.ListOpts{Offset: c.Int("offset"), Limit: c.Int("limit")},
					)
					if err != nil {
						return err
					}
					printJSON(resp)
					return nil
				},
			},
			{
				Name:  "get",
				Usage: "Get an issue by ID",
				ArgsUsage: "<id>",
				Action: func(ctx context.Context, c *cli.Command) error {
					id, err := strconv.Atoi(c.Args().First())
					if err != nil {
						return fmt.Errorf("invalid issue ID: %w", err)
					}
					issue, err := newClient().GetIssue(id)
					if err != nil {
						return err
					}
					printJSON(issue)
					return nil
				},
			},
			{
				Name:  "create",
				Usage: "Create a new issue",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "project", Usage: "Project identifier", Required: true},
					&cli.StringFlag{Name: "subject", Usage: "Issue subject", Required: true},
					&cli.StringFlag{Name: "description", Usage: "Issue description"},
					&cli.IntFlag{Name: "tracker", Usage: "Tracker ID"},
					&cli.IntFlag{Name: "status", Usage: "Status ID"},
					&cli.IntFlag{Name: "priority", Usage: "Priority ID"},
					&cli.IntFlag{Name: "assignee", Usage: "Assignee user ID"},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					var req client.IssueCreateRequest
					// We need the project ID, but the API accepts identifier.
					// For now, use a lookup or accept numeric ID.
					// The Redmine API accepts project identifier in the JSON body.
					req.Issue.ProjectID = 0 // will be set below
					req.Issue.Subject = c.String("subject")
					req.Issue.Description = c.String("description")

					if c.IsSet("tracker") {
						v := c.Int("tracker")
						req.Issue.TrackerID = &v
					}
					if c.IsSet("status") {
						v := c.Int("status")
						req.Issue.StatusID = &v
					}
					if c.IsSet("priority") {
						v := c.Int("priority")
						req.Issue.PriorityID = &v
					}
					if c.IsSet("assignee") {
						v := c.Int("assignee")
						req.Issue.AssignedToID = &v
					}

					issue, err := newClient().CreateIssue(req)
					if err != nil {
						return err
					}
					printJSON(issue)
					return nil
				},
			},
			{
				Name:  "update",
				Usage: "Update an issue",
				ArgsUsage: "<id>",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "subject", Usage: "New subject"},
					&cli.StringFlag{Name: "description", Usage: "New description"},
					&cli.IntFlag{Name: "status", Usage: "New status ID"},
					&cli.IntFlag{Name: "assignee", Usage: "New assignee user ID"},
					&cli.IntFlag{Name: "done-ratio", Usage: "Completion percentage (0-100)"},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					id, err := strconv.Atoi(c.Args().First())
					if err != nil {
						return fmt.Errorf("invalid issue ID: %w", err)
					}

					var req client.IssueUpdateRequest
					if c.IsSet("subject") {
						v := c.String("subject")
						req.Issue.Subject = &v
					}
					if c.IsSet("description") {
						v := c.String("description")
						req.Issue.Description = &v
					}
					if c.IsSet("status") {
						v := c.Int("status")
						req.Issue.StatusID = &v
					}
					if c.IsSet("assignee") {
						v := c.Int("assignee")
						req.Issue.AssignedToID = &v
					}
					if c.IsSet("done-ratio") {
						v := c.Int("done-ratio")
						req.Issue.DoneRatio = &v
					}

					if err := newClient().UpdateIssue(id, req); err != nil {
						return err
					}
					fmt.Fprintf(os.Stdout, "Issue %d updated\n", id)
					return nil
				},
			},
			{
				Name:  "delete",
				Usage: "Delete an issue",
				ArgsUsage: "<id>",
				Action: func(ctx context.Context, c *cli.Command) error {
					id, err := strconv.Atoi(c.Args().First())
					if err != nil {
						return fmt.Errorf("invalid issue ID: %w", err)
					}
					if err := newClient().DeleteIssue(id); err != nil {
						return err
					}
					fmt.Fprintf(os.Stdout, "Issue %d deleted\n", id)
					return nil
				},
			},
		},
	}
}

func projectCommand() *cli.Command {
	return &cli.Command{
		Name:  "project",
		Usage: "Manage projects",
		Commands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List projects",
				Flags: []cli.Flag{
					&cli.IntFlag{Name: "limit", Usage: "Max results", Value: 25},
					&cli.IntFlag{Name: "offset", Usage: "Result offset", Value: 0},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					resp, err := newClient().ListProjects(
						client.ListOpts{Offset: c.Int("offset"), Limit: c.Int("limit")},
					)
					if err != nil {
						return err
					}
					printJSON(resp)
					return nil
				},
			},
			{
				Name:      "get",
				Usage:     "Get a project by identifier",
				ArgsUsage: "<identifier>",
				Action: func(ctx context.Context, c *cli.Command) error {
					identifier := c.Args().First()
					if identifier == "" {
						return fmt.Errorf("project identifier is required")
					}
					project, err := newClient().GetProject(identifier)
					if err != nil {
						return err
					}
					printJSON(project)
					return nil
				},
			},
			{
				Name:  "create",
				Usage: "Create a new project",
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

					project, err := newClient().CreateProject(req)
					if err != nil {
						return err
					}
					printJSON(project)
					return nil
				},
			},
			{
				Name:      "delete",
				Usage:     "Delete a project",
				ArgsUsage: "<identifier>",
				Action: func(ctx context.Context, c *cli.Command) error {
					identifier := c.Args().First()
					if identifier == "" {
						return fmt.Errorf("project identifier is required")
					}
					if err := newClient().DeleteProject(identifier); err != nil {
						return err
					}
					fmt.Fprintf(os.Stdout, "Project %s deleted\n", identifier)
					return nil
				},
			},
		},
	}
}

func userCommand() *cli.Command {
	return &cli.Command{
		Name:  "user",
		Usage: "Manage users",
		Commands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List users",
				Flags: []cli.Flag{
					&cli.IntFlag{Name: "limit", Usage: "Max results", Value: 25},
					&cli.IntFlag{Name: "offset", Usage: "Result offset", Value: 0},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					resp, err := newClient().ListUsers(
						client.ListOpts{Offset: c.Int("offset"), Limit: c.Int("limit")},
					)
					if err != nil {
						return err
					}
					printJSON(resp)
					return nil
				},
			},
			{
				Name:  "current",
				Usage: "Get the currently authenticated user",
				Action: func(ctx context.Context, c *cli.Command) error {
					user, err := newClient().GetCurrentUser()
					if err != nil {
						return err
					}
					printJSON(user)
					return nil
				},
			},
			{
				Name:      "get",
				Usage:     "Get a user by ID",
				ArgsUsage: "<id>",
				Action: func(ctx context.Context, c *cli.Command) error {
					id, err := strconv.Atoi(c.Args().First())
					if err != nil {
						return fmt.Errorf("invalid user ID: %w", err)
					}
					user, err := newClient().GetUser(id)
					if err != nil {
						return err
					}
					printJSON(user)
					return nil
				},
			},
		},
	}
}

func timeEntryCommand() *cli.Command {
	return &cli.Command{
		Name:  "time-entry",
		Usage: "Manage time entries",
		Commands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List time entries",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "project", Usage: "Project identifier"},
					&cli.IntFlag{Name: "issue", Usage: "Issue ID"},
					&cli.IntFlag{Name: "limit", Usage: "Max results", Value: 25},
					&cli.IntFlag{Name: "offset", Usage: "Result offset", Value: 0},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					resp, err := newClient().ListTimeEntries(
						c.String("project"),
						c.Int("issue"),
						client.ListOpts{Offset: c.Int("offset"), Limit: c.Int("limit")},
					)
					if err != nil {
						return err
					}
					printJSON(resp)
					return nil
				},
			},
			{
				Name:  "create",
				Usage: "Create a time entry",
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

					entry, err := newClient().CreateTimeEntry(req)
					if err != nil {
						return err
					}
					printJSON(entry)
					return nil
				},
			},
			{
				Name:      "delete",
				Usage:     "Delete a time entry",
				ArgsUsage: "<id>",
				Action: func(ctx context.Context, c *cli.Command) error {
					id, err := strconv.Atoi(c.Args().First())
					if err != nil {
						return fmt.Errorf("invalid time entry ID: %w", err)
					}
					if err := newClient().DeleteTimeEntry(id); err != nil {
						return err
					}
					fmt.Fprintf(os.Stdout, "Time entry %d deleted\n", id)
					return nil
				},
			},
		},
	}
}

func wikiCommand() *cli.Command {
	return &cli.Command{
		Name:  "wiki",
		Usage: "Manage wiki pages",
		Commands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List wiki pages for a project",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "project", Usage: "Project identifier", Required: true},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					resp, err := newClient().ListWikiPages(c.String("project"))
					if err != nil {
						return err
					}
					printJSON(resp)
					return nil
				},
			},
			{
				Name:  "get",
				Usage: "Get a wiki page",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "project", Usage: "Project identifier", Required: true},
					&cli.StringFlag{Name: "page", Usage: "Page title", Required: true},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					page, err := newClient().GetWikiPage(c.String("project"), c.String("page"))
					if err != nil {
						return err
					}
					printJSON(page)
					return nil
				},
			},
		},
	}
}

func versionCommand() *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "Manage versions",
		Commands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List versions for a project",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "project", Usage: "Project identifier", Required: true},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					resp, err := newClient().ListVersions(c.String("project"))
					if err != nil {
						return err
					}
					printJSON(resp)
					return nil
				},
			},
		},
	}
}
