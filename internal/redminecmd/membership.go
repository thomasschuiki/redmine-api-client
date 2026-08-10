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

func membershipCommand() *cli.Command {
	return &cli.Command{
		Name:  "membership",
		Usage: "Manage project memberships",
		Description: `Add users or groups to projects and manage their roles.

Memberships link a principal (user or group) to a project with one or
more roles. Use redmine role list to find role IDs.`,
		Commands: []*cli.Command{
			{
				Name:      "list",
				Usage:     "List memberships of a project",
				ArgsUsage: " ",
				Description: `List all members of a project and their roles.

--project is required and accepts an identifier or numeric ID.
Supports pagination with --limit and --offset flags.`,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "project", Usage: "Project identifier", Required: true},
					&cli.IntFlag{Name: "limit", Usage: "Max results", Value: 25},
					&cli.IntFlag{Name: "offset", Usage: "Result offset", Value: 0},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					cl, err := newClient(c)
					if err != nil {
						return err
					}
					project, err := cl.ResolveProject(ctx, c.String("project"))
					if err != nil {
						return err
					}
					resp, err := cl.ListMemberships(ctx,
						strconv.Itoa(project.ID),
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
				Name:      "add",
				Usage:     "Add a user or group to a project",
				ArgsUsage: " ",
				Description: `Add a member to a project with the given roles.

--project is required. Specify a principal with --user or --group and
assign roles with --roles (comma-separated role IDs).
Use redmine role list to find role IDs.`,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "project", Usage: "Project identifier", Required: true},
					&cli.IntFlag{Name: "user", Usage: "User ID to add"},
					&cli.IntFlag{Name: "group", Usage: "Group ID to add"},
					&cli.StringFlag{Name: "roles", Usage: "Comma-separated role IDs", Required: true},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if !c.IsSet("user") && !c.IsSet("group") {
						return fmt.Errorf("specify a principal with --user or --group")
					}
					var req client.MembershipCreateRequest
					if c.IsSet("user") {
						v := c.Int("user")
						req.Membership.UserID = &v
					}
					if c.IsSet("group") {
						v := c.Int("group")
						req.Membership.GroupID = &v
					}
					roleIDs, err := parseRoleIDs(c.String("roles"))
					if err != nil {
						return err
					}
					req.Membership.RoleIds = roleIDs

					cl, err := newClient(c)
					if err != nil {
						return err
					}
					project, err := cl.ResolveProject(ctx, c.String("project"))
					if err != nil {
						return err
					}
					membership, err := cl.CreateMembership(ctx, strconv.Itoa(project.ID), req)
					if err != nil {
						return err
					}
					if err := output.Print(membership, c.String("output")); err != nil {
						return err
					}
					return nil
				},
			},
			{
				Name:      "remove",
				Usage:     "Remove a membership",
				ArgsUsage: "<id>",
				Description: `Permanently remove a member from a project by membership ID.
This action cannot be undone.`,
				Action: func(ctx context.Context, c *cli.Command) error {
					id, err := strconv.Atoi(c.Args().First())
					if err != nil {
						return fmt.Errorf("invalid membership ID: %w", err)
					}
					cl, err := newClient(c)
					if err != nil {
						return err
					}
					if err := cl.DeleteMembership(ctx, id); err != nil {
						return err
					}
					fmt.Fprintf(os.Stdout, "Membership %d removed\n", id)
					return nil
				},
			},
		},
	}
}

func parseRoleIDs(s string) ([]int, error) {
	var ids []int
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid role ID %q: %w", part, err)
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("--roles requires at least one role ID")
	}
	return ids, nil
}
