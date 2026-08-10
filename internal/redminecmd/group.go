package redminecmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/tom-redmine/go-redmine-cli/internal/client"
	"github.com/tom-redmine/go-redmine-cli/internal/output"
	"github.com/urfave/cli/v3"
)

func groupCommand() *cli.Command {
	return &cli.Command{
		Name:  "group",
		Usage: "Manage groups",
		Description: `List, create, and manage Redmine user groups.

Groups can be assigned to projects as members and granted roles.`,
		Commands: []*cli.Command{
			{
				Name:      "list",
				Usage:     "List groups",
				ArgsUsage: " ",
				Description: `List all groups.

By default built-in groups are hidden; pass --builtin to include them
(admins only). Supports pagination with --limit and --offset flags.`,
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "builtin", Usage: "Include built-in groups (admins only)"},
					&cli.IntFlag{Name: "limit", Usage: "Max results", Value: 25},
					&cli.IntFlag{Name: "offset", Usage: "Result offset", Value: 0},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					cl, err := newClient(c)
					if err != nil {
						return err
					}
					opts := client.ListOpts{Offset: c.Int("offset"), Limit: c.Int("limit")}
					var resp *client.GroupListResponse
					if c.Bool("builtin") {
						resp, err = cl.ListGroupsBuiltin(ctx, opts)
					} else {
						resp, err = cl.ListGroups(ctx, opts)
					}
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
				Usage:     "Get a group",
				ArgsUsage: "<id>",
				Description: `Show a single group by its numeric ID.

Use --include to expand associated data (users, memberships).`,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "include", Usage: "Associated data to include (users, memberships)"},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					id, err := strconv.Atoi(c.Args().First())
					if err != nil {
						return fmt.Errorf("invalid group ID: %w", err)
					}
					cl, err := newClient(c)
					if err != nil {
						return err
					}
					group, err := cl.GetGroup(ctx, id, c.String("include"))
					if err != nil {
						return err
					}
					if err := output.Print(group, c.String("output")); err != nil {
						return err
					}
					return nil
				},
			},
			{
				Name:      "create",
				Usage:     "Create a group",
				ArgsUsage: " ",
				Description: `Create a new group with the given name.

Optionally seed it with users via --users (comma-separated user IDs).`,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "name", Usage: "Group name", Required: true},
					&cli.StringFlag{Name: "users", Usage: "Comma-separated user IDs"},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					var req client.GroupCreateRequest
					req.Group.Name = c.String("name")
					if c.IsSet("users") {
						userIDs, err := parseInts(splitCSV(c.String("users")), "users")
						if err != nil {
							return err
						}
						req.Group.UserIds = userIDs
					}
					cl, err := newClient(c)
					if err != nil {
						return err
					}
					group, err := cl.CreateGroup(ctx, req)
					if err != nil {
						return err
					}
					if err := output.Print(group, c.String("output")); err != nil {
						return err
					}
					return nil
				},
			},
			{
				Name:      "update",
				Usage:     "Update a group",
				ArgsUsage: "<id>",
				Description: `Rename a group or replace its members.

Pass --name to rename. Use --users (comma-separated IDs) to set the
full member list; --add-users and --remove-users adjust it incrementally
(membership changes require the admin role).`,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "name", Usage: "New group name"},
					&cli.StringFlag{Name: "add-users", Usage: "Comma-separated user IDs to add"},
					&cli.StringFlag{Name: "remove-users", Usage: "Comma-separated user IDs to remove"},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					id, err := strconv.Atoi(c.Args().First())
					if err != nil {
						return fmt.Errorf("invalid group ID: %w", err)
					}
					cl, err := newClient(c)
					if err != nil {
						return err
					}
					req := client.GroupUpdateRequest{}
					if c.IsSet("name") {
						name := c.String("name")
						req.Group.Name = &name
					}
					if c.IsSet("add-users") || c.IsSet("remove-users") {
						group, err := cl.GetGroup(ctx, id, "users")
						if err != nil {
							return err
						}
						addIDs, err := parseInts(splitCSV(c.String("add-users")), "add-users")
						if err != nil {
							return err
						}
						removeIDs, err := parseInts(splitCSV(c.String("remove-users")), "remove-users")
						if err != nil {
							return err
						}
						req.Group.UserIds = diffUserIDs(group.Users, addIDs, removeIDs)
					}
					group, err := cl.UpdateGroup(ctx, id, req)
					if err != nil {
						return err
					}
					if err := output.Print(group, c.String("output")); err != nil {
						return err
					}
					return nil
				},
			},
			{
				Name:      "delete",
				Usage:     "Delete a group",
				ArgsUsage: "<id>",
				Description: `Permanently delete a group by its numeric ID.
This action cannot be undone.`,
				Action: func(ctx context.Context, c *cli.Command) error {
					id, err := strconv.Atoi(c.Args().First())
					if err != nil {
						return fmt.Errorf("invalid group ID: %w", err)
					}
					cl, err := newClient(c)
					if err != nil {
						return err
					}
					if err := cl.DeleteGroup(ctx, id); err != nil {
						return err
					}
					fmt.Fprintf(os.Stdout, "Group %d deleted\n", id)
					return nil
				},
			},
		},
	}
}

func splitCSV(s string) []string {
	var parts []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

func diffUserIDs(current []client.IDName, addIDs, removeIDs []int) []int {
	remove := map[int]bool{}
	for _, id := range removeIDs {
		remove[id] = true
	}
	keep := map[int]bool{}
	for _, u := range current {
		keep[u.ID] = true
	}
	for _, id := range addIDs {
		keep[id] = true
	}
	var result []int
	for _, u := range current {
		if !remove[u.ID] && keep[u.ID] {
			result = append(result, u.ID)
		}
	}
	for _, id := range addIDs {
		if !remove[id] && !containsInt(result, id) {
			result = append(result, id)
		}
	}
	return result
}

func containsInt(list []int, v int) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
