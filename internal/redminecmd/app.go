package redminecmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tom-redmine/go-redmine-cli/internal/client"
	"github.com/tom-redmine/go-redmine-cli/internal/config"
	"github.com/tom-redmine/go-redmine-cli/internal/output"
	"github.com/urfave/cli/v3"
)

var redmineClient *client.Client

func newClient(c *cli.Command) *client.Client {
	if redmineClient == nil {
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		connectTimeout, err := time.ParseDuration(c.String("connect-timeout"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid --connect-timeout: %v\n", err)
			os.Exit(1)
		}
		maxTime, err := time.ParseDuration(c.String("max-time"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid --max-time: %v\n", err)
			os.Exit(1)
		}
		redmineClient = client.New(cfg, connectTimeout, maxTime, c.Int("retries"))
		client.SetLogWriter(os.Stderr)
	}
	return redmineClient
}

// Run creates and runs the CLI application.
func Run() *cli.Command {
	return &cli.Command{
		Name:  "redmine",
		Usage: "Redmine CLI client",
		Description: `A command-line interface for interacting with a Redmine instance.

Configure the connection using environment variables or a config file.
See 'redmine help' for a list of available commands.`,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "output", Usage: "Output format (yaml, json)", Value: "yaml"},
			&cli.StringFlag{Name: "connect-timeout", Usage: "TCP connection timeout (e.g. 10s)", Value: "10s"},
			&cli.StringFlag{Name: "max-time", Usage: "Maximum time per request (e.g. 30s)", Value: "30s"},
			&cli.IntFlag{Name: "retries", Usage: "Number of retries on failure", Value: 3},
		},
		Commands: []*cli.Command{
			issueCommand(),
			projectCommand(),
			userCommand(),
			timeEntryCommand(),
			wikiCommand(),
			versionCommand(),
			trackerCommand(),
			statusCommand(),
			priorityCommand(),
			categoryCommand(),
		},
	}
}

func issueCommand() *cli.Command {
	return &cli.Command{
		Name:  "issue",
		Usage: "Manage issues",
		Description: `Create, read, update, and delete issues in Redmine.
Issues can be filtered by project, assigned to users, and tracked
across different statuses and priorities.`,
		Commands: []*cli.Command{
			{
				Name:      "list",
				Usage:     "List issues",
				ArgsUsage: " ",
				Description: `List issues with optional filtering.

By default, returns up to 25 results. Use --limit and --offset for pagination,
or --all to auto-paginate through every matching issue.
Filter by project using the --project flag with a project identifier.

Filter flags map to Redmine API query parameters:
  --status-id     → status_id (use '*' for open, '!*' for closed)
  --tracker-id    → tracker_id
  --assigned-to-id → assigned_to_id
  --priority-id   → priority_id
  --category-id   → category_id
  --fixed-version-id → fixed_version_id
  --parent-id     → parent_id (filter children of a parent issue)
  --subject       → subject (substring match)
  --description   → description (substring match)
  --sort          → sort order (e.g. 'created_on:desc')
  --include       → include associated objects (e.g. 'attachments,relations')

Client-side filters (applied after fetch):
  --contains      → substring match across subject, description, status, etc.
  --regex         → regex match across subject, description, status, etc.
  --case-insensitive → make --contains/--regex case-insensitive (default: sensitive)

Output options:
  --fields        → comma-separated list of fields to include in output
  --top N         → limit results to first N matches`,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "project", Usage: "Project identifier"},
					&cli.IntFlag{Name: "limit", Usage: "Max results", Value: 25},
					&cli.IntFlag{Name: "offset", Usage: "Result offset", Value: 0},
					&cli.BoolFlag{Name: "all", Usage: "Auto-paginate through all results"},
					&cli.IntFlag{Name: "top", Usage: "Limit to first N results"},
					&cli.StringFlag{Name: "status-id", Usage: "Status ID ('*' open, '!*' closed, or numeric)"},
					&cli.IntFlag{Name: "tracker-id", Usage: "Tracker ID"},
					&cli.IntFlag{Name: "assigned-to-id", Usage: "Assigned to user ID"},
					&cli.IntFlag{Name: "priority-id", Usage: "Priority ID"},
					&cli.IntFlag{Name: "category-id", Usage: "Category ID"},
					&cli.IntFlag{Name: "fixed-version-id", Usage: "Fixed version ID"},
					&cli.IntFlag{Name: "parent-id", Usage: "Parent issue ID"},
					&cli.StringFlag{Name: "subject", Usage: "Filter by subject (substring)"},
					&cli.StringFlag{Name: "description", Usage: "Filter by description (substring)"},
					&cli.StringFlag{Name: "sort", Usage: "Sort order (e.g. 'created_on:desc')"},
					&cli.StringFlag{Name: "include", Usage: "Include associated objects (e.g. 'attachments,relations')"},
					&cli.StringFlag{Name: "contains", Usage: "Client-side substring filter across fields"},
					&cli.StringFlag{Name: "regex", Usage: "Client-side regex filter across fields"},
					&cli.BoolFlag{Name: "case-insensitive", Usage: "Make --contains/--regex case-insensitive"},
					&cli.StringFlag{Name: "fields", Usage: "Comma-separated fields to include in output"},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					opts := client.IssueListOpts{
						StatusID:       c.String("status-id"),
						TrackerID:      c.Int("tracker-id"),
						AssignedToID:   c.Int("assigned-to-id"),
						PriorityID:     c.Int("priority-id"),
						CategoryID:     c.Int("category-id"),
						FixedVersionID: c.Int("fixed-version-id"),
						ParentID:       c.Int("parent-id"),
						Subject:        c.String("subject"),
						Description:    c.String("description"),
						Sort:           c.String("sort"),
						Include:        c.String("include"),
					}
					opts.Offset = c.Int("offset")
					opts.Limit = c.Int("limit")

					var resp *client.IssueListResponse
					var err error
					if c.Bool("all") {
						resp, err = newClient(c).ListAllIssues(c.String("project"), opts)
					} else {
						resp, err = newClient(c).ListIssues(c.String("project"), opts)
					}
					if err != nil {
						return err
					}

					issues := resp.Issues

					if v := c.String("contains"); v != "" {
						issues = client.ContainsFilter(issues, v, c.Bool("case-insensitive"))
					}
					if v := c.String("regex"); v != "" {
						issues, err = client.RegexFilter(issues, v, c.Bool("case-insensitive"))
						if err != nil {
							return err
						}
					}
					if v := c.Int("top"); v > 0 && v < len(issues) {
						issues = issues[:v]
					}

					result := &client.IssueListResponse{
						Issues:     issues,
						TotalCount: resp.TotalCount,
						Offset:     resp.Offset,
						Limit:      resp.Limit,
					}

					if fields := c.String("fields"); fields != "" {
						fieldList := strings.Split(fields, ",")
						output.Print(output.FilterFields(result, fieldList), c.String("output"))
					} else {
						output.Print(result, c.String("output"))
					}
					return nil
				},
			},
			{
				Name:      "get",
				Usage:     "Get an issue by ID",
				ArgsUsage: "<id>",
				Description: `Retrieve full details for a single issue by its numeric ID.
Returns the complete issue object including description, status,
assignee, and custom fields.

Use --include to add associated objects like journals, watchers, or relations.

Examples:
  redmine issue get 1234
  redmine issue get 1234 --include journals --output json
  redmine issue get 1234 --include "journals,watchers" -o yaml`,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "include", Usage: "Include associated objects (e.g. 'journals,watchers')"},
					&cli.StringFlag{Name: "fields", Usage: "Comma-separated fields to include in output"},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					id, err := strconv.Atoi(c.Args().First())
					if err != nil {
						return fmt.Errorf("invalid issue ID: %w", err)
					}
					issue, err := newClient(c).GetIssue(id, c.String("include"))
					if err != nil {
						return err
					}
					if fields := c.String("fields"); fields != "" {
						fieldList := strings.Split(fields, ",")
						output.Print(output.FilterFields(issue, fieldList), c.String("output"))
					} else {
						output.Print(issue, c.String("output"))
					}
					return nil
				},
			},
			{
				Name:      "grep",
				Usage:     "Search text across issues",
				ArgsUsage: " ",
				Description: `Search for text in issue descriptions and/or journal notes.

Auto-paginates through all matching results in the given scope.
Use --project, --parent-id, and other flags to narrow the search.

The --in flag controls which fields are searched (comma-separated):
  description  → issue description
  notes        → journal/comment notes

Examples:
  redmine issue grep --text contractenddate --in description,notes
  redmine issue grep --project ework --parent-id 45141 --text contract --in notes`,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "project", Usage: "Project identifier"},
					&cli.IntFlag{Name: "parent-id", Usage: "Parent issue ID"},
					&cli.StringFlag{Name: "text", Usage: "Text to search for", Required: true},
					&cli.StringFlag{Name: "in", Usage: "Fields to search (description, notes)", Value: "description,notes"},
					&cli.StringFlag{Name: "status-id", Usage: "Status ID ('*' open, '!*' closed, or numeric)"},
					&cli.IntFlag{Name: "tracker-id", Usage: "Tracker ID"},
					&cli.IntFlag{Name: "assigned-to-id", Usage: "Assigned to user ID"},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					opts := client.GrepOpts{
						Project:      c.String("project"),
						ParentID:     c.Int("parent-id"),
						Text:         c.String("text"),
						In:           c.String("in"),
						StatusID:     c.String("status-id"),
						TrackerID:    c.Int("tracker-id"),
						AssignedToID: c.Int("assigned-to-id"),
					}
					result, err := newClient(c).GrepIssues(opts)
					if err != nil {
						return err
					}
					format := c.String("output")
					if format == "json" {
						output.Print(result, "json")
					} else {
						for _, m := range result.Matches {
							fmt.Printf("#%d | %s | %s | %s\n", m.IssueID, m.Subject, m.Where, m.Snippet)
						}
						fmt.Fprintf(os.Stderr, "%d matches in %d issues\n", len(result.Matches), result.Total)
					}
					return nil
				},
			},
			{
				Name:      "create",
				Usage:     "Create a new issue",
				ArgsUsage: " ",
				Description: `Create a new issue in a project.

At minimum, --project and --subject are required. Optional fields include
description, parent-id, tracker, status, priority, assignee,
start-date, due-date, estimated-hours, fixed-version-id, category-id,
and custom-field.

Use --parent-id to create a child issue under a parent (epic/feature).
The parent must exist and be in the same project.

Date values must be in YYYY-MM-DD format. Estimated hours must be
non-negative.

Custom fields:
  --custom-field <id>=<value> (repeatable)
  For multi-value fields, repeat the same id:
    --custom-field 1=opt1 --custom-field 1=opt2

Examples:
  redmine issue create --project myproject --subject "Fix bug" --parent-id 123
  redmine issue create --project myproject --subject "Task" \
    --start-date 2026-08-01 --due-date 2026-08-15 --estimated-hours 8 \
    --custom-field 1="value" --custom-field 2="opt1" --custom-field 2="opt2"`,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "project", Usage: "Project identifier", Required: true},
					&cli.StringFlag{Name: "subject", Usage: "Issue subject", Required: true},
					&cli.StringFlag{Name: "description", Usage: "Issue description"},
					&cli.IntFlag{Name: "parent-id", Usage: "Parent issue ID"},
					&cli.IntFlag{Name: "tracker", Usage: "Tracker ID"},
					&cli.IntFlag{Name: "status", Usage: "Status ID"},
					&cli.IntFlag{Name: "priority", Usage: "Priority ID"},
					&cli.IntFlag{Name: "assignee", Usage: "Assignee user ID"},
					&cli.StringFlag{Name: "start-date", Usage: "Start date (YYYY-MM-DD)"},
					&cli.StringFlag{Name: "due-date", Usage: "Due date (YYYY-MM-DD)"},
					&cli.FloatFlag{Name: "estimated-hours", Usage: "Estimated hours (non-negative)"},
					&cli.IntFlag{Name: "fixed-version-id", Usage: "Target version/milestone ID"},
					&cli.IntFlag{Name: "category-id", Usage: "Category ID"},
					&cli.StringSliceFlag{Name: "watcher", Usage: "Watcher user ID (repeatable)"},
					&cli.StringSliceFlag{Name: "custom-field", Usage: "Custom field (<id>=<value>, repeatable)"},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					var req client.IssueCreateRequest
					req.Issue.ProjectID = 0 // will be set below
					req.Issue.Subject = c.String("subject")
					req.Issue.Description = c.String("description")

					if c.IsSet("parent-id") {
						v := c.Int("parent-id")
						req.Issue.ParentID = &v
					}
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
					if c.IsSet("start-date") {
						v := c.String("start-date")
						if err := client.ValidateDate(v); err != nil {
							return err
						}
						req.Issue.StartDate = v
					}
					if c.IsSet("due-date") {
						v := c.String("due-date")
						if err := client.ValidateDate(v); err != nil {
							return err
						}
						req.Issue.DueDate = v
					}
					if c.IsSet("estimated-hours") {
						v := c.Float("estimated-hours")
						if err := client.ValidateEstimatedHours(v); err != nil {
							return err
						}
						req.Issue.EstimatedHours = &v
					}
					if c.IsSet("fixed-version-id") {
						v := c.Int("fixed-version-id")
						req.Issue.FixedVersionID = &v
					}
					if c.IsSet("category-id") {
						v := c.Int("category-id")
						req.Issue.CategoryID = &v
					}
					if c.IsSet("watcher") {
						ids, err := parseInts(c.StringSlice("watcher"), "watcher")
						if err != nil {
							return err
						}
						req.Issue.WatcherUserIDs = ids
					}
					if c.IsSet("custom-field") {
						fields, err := parseCustomFields(c.StringSlice("custom-field"))
						if err != nil {
							return err
						}
						req.Issue.CustomFields = client.MergeCustomFields(fields)
					}

					issue, err := newClient(c).CreateIssue(req)
					if err != nil {
						return err
					}
					output.Print(issue, c.String("output"))
					return nil
				},
			},
			{
				Name:      "update",
				Usage:     "Update an issue",
				ArgsUsage: "<id>",
				Description: `Update fields on an existing issue.

Only the fields you specify will be changed. Use --subject, --description,
--status, --assignee, --done-ratio, --start-date, --due-date,
--estimated-hours, --fixed-version-id, --category-id, --notes,
--watcher, --unwatcher, or --custom-field to update the corresponding
fields.

Use --notes to add a journal entry alongside field changes, keeping
an audit trail of why the change was made.

Use --watcher (repeatable) to add watchers and --unwatcher (repeatable)
to remove them. Redmine ignores duplicate additions.

Date values must be in YYYY-MM-DD format. Estimated hours must be
non-negative.

Custom fields:
  --custom-field <id>=<value> (repeatable)
  For multi-value fields, repeat the same id:
    --custom-field 1=opt1 --custom-field 1=opt2

Examples:
  redmine issue update 123 --status 5
  redmine issue update 123 --status 5 --notes "Resolved, waiting on QA"
  redmine issue update 123 --watcher 42 --watcher 7
  redmine issue update 123 --unwatcher 99
  redmine issue update 123 --start-date 2026-08-01 --due-date 2026-08-15
  redmine issue update 123 --estimated-hours 12.5
  redmine issue update 123 --custom-field 1="new value"`,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "subject", Usage: "New subject"},
					&cli.StringFlag{Name: "description", Usage: "New description"},
					&cli.IntFlag{Name: "status", Usage: "New status ID"},
					&cli.IntFlag{Name: "assignee", Usage: "New assignee user ID"},
					&cli.IntFlag{Name: "done-ratio", Usage: "Completion percentage (0-100)"},
					&cli.StringFlag{Name: "start-date", Usage: "Start date (YYYY-MM-DD)"},
					&cli.StringFlag{Name: "due-date", Usage: "Due date (YYYY-MM-DD)"},
					&cli.FloatFlag{Name: "estimated-hours", Usage: "Estimated hours (non-negative)"},
					&cli.IntFlag{Name: "fixed-version-id", Usage: "Target version/milestone ID"},
					&cli.IntFlag{Name: "category-id", Usage: "Category ID"},
					&cli.StringFlag{Name: "notes", Usage: "Journal note (audit trail)"},
					&cli.StringSliceFlag{Name: "watcher", Usage: "Add watcher user ID (repeatable)"},
					&cli.StringSliceFlag{Name: "unwatcher", Usage: "Remove watcher user ID (repeatable)"},
					&cli.StringSliceFlag{Name: "custom-field", Usage: "Custom field (<id>=<value>, repeatable)"},
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
					if c.IsSet("start-date") {
						v := c.String("start-date")
						if err := client.ValidateDate(v); err != nil {
							return err
						}
						req.Issue.StartDate = &v
					}
					if c.IsSet("due-date") {
						v := c.String("due-date")
						if err := client.ValidateDate(v); err != nil {
							return err
						}
						req.Issue.DueDate = &v
					}
					if c.IsSet("estimated-hours") {
						v := c.Float("estimated-hours")
						if err := client.ValidateEstimatedHours(v); err != nil {
							return err
						}
						req.Issue.EstimatedHours = &v
					}
					if c.IsSet("fixed-version-id") {
						v := c.Int("fixed-version-id")
						req.Issue.FixedVersionID = &v
					}
					if c.IsSet("category-id") {
						v := c.Int("category-id")
						req.Issue.CategoryID = &v
					}
					if c.IsSet("custom-field") {
						fields, err := parseCustomFields(c.StringSlice("custom-field"))
						if err != nil {
							return err
						}
						req.Issue.CustomFields = client.MergeCustomFields(fields)
					}
					if notes := c.String("notes"); notes != "" {
						req.Issue.Notes = notes
					} else if c.IsSet("notes") {
						return fmt.Errorf("--notes requires non-empty text")
					}

					cl := newClient(c)

					if err := cl.UpdateIssue(id, req); err != nil {
						return err
					}

					if c.IsSet("watcher") {
						ids, err := parseInts(c.StringSlice("watcher"), "watcher")
						if err != nil {
							return err
						}
						for _, uid := range ids {
							if err := cl.AddWatcher(id, uid); err != nil {
								return fmt.Errorf("adding watcher %d: %w", uid, err)
							}
						}
					}

					if c.IsSet("unwatcher") {
						ids, err := parseInts(c.StringSlice("unwatcher"), "unwatcher")
						if err != nil {
							return err
						}
						for _, uid := range ids {
							if err := cl.RemoveWatcher(id, uid); err != nil {
								return fmt.Errorf("removing watcher %d: %w", uid, err)
							}
						}
					}

					fmt.Fprintf(os.Stdout, "Issue %d updated\n", id)
					return nil
				},
			},
			{
				Name:      "delete",
				Usage:     "Delete an issue",
				ArgsUsage: "<id>",
				Description: `Permanently delete an issue by its numeric ID.
This action cannot be undone.`,
				Action: func(ctx context.Context, c *cli.Command) error {
					id, err := strconv.Atoi(c.Args().First())
					if err != nil {
						return fmt.Errorf("invalid issue ID: %w", err)
					}
					if err := newClient(c).DeleteIssue(id); err != nil {
						return err
					}
					fmt.Fprintf(os.Stdout, "Issue %d deleted\n", id)
					return nil
				},
			},
			{
				Name:  "relation",
				Usage: "Manage issue relations (dependencies)",
				Description: `Manage relations between issues for dependency tracking.

Relation types:
  relates       — general relationship
  duplicates    — this issue duplicates the target
  duplicated    — this issue is duplicated by the target
  blocks        — this issue blocks the target
  blocked       — this issue is blocked by the target
  precedes      — this issue must be done before the target
  follows       — this issue must be done after the target
  copied_to     — this issue was copied to the target
  copied_from   — this issue was copied from the target

For precedes/follows, use --delay to specify the number of days
between the two issues.`,
				Commands: []*cli.Command{
					{
						Name:      "add",
						Usage:     "Add a relation between two issues",
						ArgsUsage: "<from-id> <to-id>",
						Description: `Create a relation from one issue to another.

The first argument is the source issue ID, the second is the target
issue ID. Use --type to specify the relation type.

For precedes/follows relations, use --delay to set the number of
days of lag between the issues.

Examples:
  redmine issue relation add 123 456 --type blocks
  redmine issue relation add 100 200 --type precedes --delay 3`,
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "type", Usage: "Relation type", Required: true},
							&cli.IntFlag{Name: "delay", Usage: "Delay in days (for precedes/follows)"},
						},
						Action: func(ctx context.Context, c *cli.Command) error {
							args := c.Args().Slice()
							if len(args) < 2 {
								return fmt.Errorf("requires exactly 2 arguments: <from-id> <to-id>")
							}
							fromID, err := strconv.Atoi(args[0])
							if err != nil {
								return fmt.Errorf("invalid from-id: %w", err)
							}
							toID, err := strconv.Atoi(args[1])
							if err != nil {
								return fmt.Errorf("invalid to-id: %w", err)
							}

							rt, err := client.ParseRelationType(c.String("type"))
							if err != nil {
								return err
							}

							var req client.RelationCreateRequest
							req.Relation.IssueID = fromID
							req.Relation.IssueToID = toID
							req.Relation.RelationType = rt
							if c.IsSet("delay") {
								v := c.Int("delay")
								req.Relation.Delay = &v
							}

							rel, err := newClient(c).CreateRelation(req)
							if err != nil {
								return err
							}
							output.Print(rel, c.String("output"))
							return nil
						},
					},
					{
						Name:      "list",
						Usage:     "List relations for an issue",
						ArgsUsage: "<id>",
						Description: `List all relations for a given issue.

Shows relation ID, type, target issue, and delay (if applicable).

Examples:
  redmine issue relation list 123`,
						Action: func(ctx context.Context, c *cli.Command) error {
							id, err := strconv.Atoi(c.Args().First())
							if err != nil {
								return fmt.Errorf("invalid issue ID: %w", err)
							}
							resp, err := newClient(c).ListRelations(id)
							if err != nil {
								return err
							}
							if c.String("output") == "json" {
								output.Print(resp, "json")
							} else {
								for _, r := range resp.Relations {
									delay := ""
									if r.Delay != nil {
										delay = fmt.Sprintf(" (delay %d days)", *r.Delay)
									}
									fmt.Printf("#%d → #%d  %s%s\n", r.IssueID, r.IssueToID, r.RelationType, delay)
								}
							}
							return nil
						},
					},
					{
						Name:      "remove",
						Usage:     "Remove a relation",
						ArgsUsage: "<relation-id>",
						Description: `Delete a relation by its numeric ID.

Use 'redmine issue relation list <issue-id>' to find the relation ID.

Example:
  redmine issue relation remove 42`,
						Action: func(ctx context.Context, c *cli.Command) error {
							id, err := strconv.Atoi(c.Args().First())
							if err != nil {
								return fmt.Errorf("invalid relation ID: %w", err)
							}
							if err := newClient(c).DeleteRelation(id); err != nil {
								return err
							}
							fmt.Fprintf(os.Stdout, "Relation %d removed\n", id)
							return nil
						},
					},
				},
			},
		},
	}
}

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
					resp, err := newClient(c).ListProjects(
						client.ListOpts{Offset: c.Int("offset"), Limit: c.Int("limit")},
					)
					if err != nil {
						return err
					}
					output.Print(resp, c.String("output"))
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
					project, err := newClient(c).ResolveProject(identifier)
					if err != nil {
						return err
					}
					output.Print(project, c.String("output"))
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

					project, err := newClient(c).CreateProject(req)
					if err != nil {
						return err
					}
					output.Print(project, c.String("output"))
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
					if err := newClient(c).DeleteProject(identifier); err != nil {
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
					resp, err := newClient(c).ListUsers(
						client.ListOpts{Offset: c.Int("offset"), Limit: c.Int("limit")},
					)
					if err != nil {
						return err
					}
					output.Print(resp, c.String("output"))
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
					user, err := newClient(c).GetCurrentUser()
					if err != nil {
						return err
					}
					output.Print(user, c.String("output"))
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
					user, err := newClient(c).GetUser(id)
					if err != nil {
						return err
					}
					output.Print(user, c.String("output"))
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
					resp, err := newClient(c).ListTimeEntries(
						c.String("project"),
						c.Int("issue"),
						client.ListOpts{Offset: c.Int("offset"), Limit: c.Int("limit")},
					)
					if err != nil {
						return err
					}
					output.Print(resp, c.String("output"))
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

					entry, err := newClient(c).CreateTimeEntry(req)
					if err != nil {
						return err
					}
					output.Print(entry, c.String("output"))
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
					if err := newClient(c).DeleteTimeEntry(id); err != nil {
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
					resp, err := newClient(c).ListWikiPages(c.String("project"))
					if err != nil {
						return err
					}
					output.Print(resp, c.String("output"))
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
					page, err := newClient(c).GetWikiPage(c.String("project"), c.String("page"))
					if err != nil {
						return err
					}
					output.Print(page, c.String("output"))
					return nil
				},
			},
		},
	}
}

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
					resp, err := newClient(c).ListTrackers()
					if err != nil {
						return err
					}
					if c.String("output") == "json" {
						output.Print(resp, "json")
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

func statusCommand() *cli.Command {
	return &cli.Command{
		Name:  "status",
		Usage: "List issue statuses",
		Description: `Display all available issue statuses with their numeric IDs.

Use the ID with --status-id when listing or filtering issues.`,
		Commands: []*cli.Command{
			{
				Name:      "list",
				Usage:     "List issue statuses",
				ArgsUsage: " ",
				Description: `List all issue statuses configured in Redmine.

Output shows ID, name, and whether the status is a closed state.
Use --output json for machine-parseable output.`,
				Action: func(ctx context.Context, c *cli.Command) error {
					resp, err := newClient(c).ListIssueStatuses()
					if err != nil {
						return err
					}
					if c.String("output") == "json" {
						output.Print(resp, "json")
					} else {
						for _, s := range resp.IssueStatuses {
							closed := ""
							if s.IsClosed {
								closed = " (closed)"
							}
							fmt.Printf("% 4d  %s%s\n", s.ID, s.Name, closed)
						}
					}
					return nil
				},
			},
		},
	}
}

func priorityCommand() *cli.Command {
	return &cli.Command{
		Name:  "priority",
		Usage: "List issue priorities",
		Description: `Display all available issue priorities with their numeric IDs.

Use the ID with --priority-id when creating or filtering issues.`,
		Commands: []*cli.Command{
			{
				Name:      "list",
				Usage:     "List issue priorities",
				ArgsUsage: " ",
				Description: `List all issue priorities configured in Redmine.

Output shows ID and name. Default priority is marked with an asterisk.
Use --output json for machine-parseable output.`,
				Action: func(ctx context.Context, c *cli.Command) error {
					resp, err := newClient(c).ListIssuePriorities()
					if err != nil {
						return err
					}
					if c.String("output") == "json" {
						output.Print(resp, "json")
					} else {
						for _, p := range resp.IssuePriorities {
							def := ""
							if p.IsDefault {
								def = " *"
							}
							fmt.Printf("% 4d  %s%s\n", p.ID, p.Name, def)
						}
					}
					return nil
				},
			},
		},
	}
}

func categoryCommand() *cli.Command {
	return &cli.Command{
		Name:  "category",
		Usage: "List issue categories",
		Description: `Display issue categories for a project with their numeric IDs.

Use the ID with --category-id when listing or filtering issues.`,
		Commands: []*cli.Command{
			{
				Name:      "list",
				Usage:     "List categories for a project",
				ArgsUsage: " ",
				Description: `List all issue categories for a given project.

The --project flag is required. Output shows ID, name, and assignee.
Use --output json for machine-parseable output.`,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "project", Usage: "Project identifier", Required: true},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					resp, err := newClient(c).ListIssueCategories(c.String("project"))
					if err != nil {
						return err
					}
					if c.String("output") == "json" {
						output.Print(resp, "json")
					} else {
						for _, cat := range resp.IssueCategories {
							assigned := ""
							if cat.AssignedTo != nil {
								assigned = " → " + cat.AssignedTo.Name
							}
							fmt.Printf("% 4d  %s%s\n", cat.ID, cat.Name, assigned)
						}
					}
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
					resp, err := newClient(c).ListVersions(c.String("project"))
					if err != nil {
						return err
					}
					output.Print(resp, c.String("output"))
					return nil
				},
			},
		},
	}
}

func parseCustomFields(raw []string) ([]client.CustomFieldValue, error) {
	fields := make([]client.CustomFieldValue, 0, len(raw))
	for _, s := range raw {
		f, err := client.ParseCustomField(s)
		if err != nil {
			return nil, err
		}
		fields = append(fields, f)
	}
	return fields, nil
}

func parseInts(raw []string, flagName string) ([]int, error) {
	ids := make([]int, 0, len(raw))
	for _, s := range raw {
		id, err := strconv.Atoi(s)
		if err != nil {
			return nil, fmt.Errorf("invalid --%s value %q: %w", flagName, s, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}
