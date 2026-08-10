package redminecmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tom-redmine/go-redmine-cli/internal/client"
	"github.com/tom-redmine/go-redmine-cli/internal/output"
	"github.com/urfave/cli/v3"
)

func attachmentCommand() *cli.Command {
	return &cli.Command{
		Name:  "attachment",
		Usage: "Manage attachments",
		Description: `View, update, and delete file attachments.

Attachments are files uploaded to issues, wiki pages, and other objects.
The attachment's download URL is exposed as content_url on the get command.`,
		Commands: []*cli.Command{
			{
				Name:      "get",
				Usage:     "Get an attachment by ID",
				ArgsUsage: "<id>",
				Description: `Retrieve attachment metadata.

With --output FILE, the file is downloaded to FILE instead and the metadata
is not printed.`,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "output-file", Aliases: []string{"o"}, Usage: "Download the file to this path"},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					id, err := parseArgID(c, "attachment id")
					if err != nil {
						return err
					}
					cl, err := newClient(c)
					if err != nil {
						return err
					}
					att, err := cl.GetAttachment(ctx, id)
					if err != nil {
						return err
					}

					outFile := c.String("output-file")
					if outFile == "" {
						return output.Print(att, c.String("output"))
					}

					contentURL := att.ContentURL
					if contentURL == nil || *contentURL == "" {
						return fmt.Errorf("attachment %d has no content_url", id)
					}
					f, err := os.Create(outFile)
					if err != nil {
						return fmt.Errorf("creating %s: %w", outFile, err)
					}
					defer f.Close()
					if err := cl.Download(ctx, *contentURL, f); err != nil {
						return err
					}
					fmt.Fprintf(os.Stdout, "Downloaded %s to %s\n", displayName(att.Filename, *contentURL), outFile)
					return nil
				},
			},
			{
				Name:      "update",
				Usage:     "Update an attachment",
				ArgsUsage: "<id>",
				Description: `Update attachment metadata, e.g. its description.`,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "description", Usage: "New description"},
					&cli.StringFlag{Name: "filename", Usage: "New filename"},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					id, err := parseArgID(c, "attachment id")
					if err != nil {
						return err
					}
					var req client.AttachmentUpdateRequest
					if c.IsSet("description") {
						d := c.String("description")
						req.Attachment.Description = &d
					}
					if c.IsSet("filename") {
						f := c.String("filename")
						req.Attachment.Filename = &f
					}
					if req.Attachment.Description == nil && req.Attachment.Filename == nil {
						return fmt.Errorf("at least one of --description or --filename is required")
					}

					cl, err := newClient(c)
					if err != nil {
						return err
					}
					if err := cl.UpdateAttachment(ctx, id, req); err != nil {
						return err
					}
					fmt.Fprintf(os.Stdout, "Attachment %d updated\n", id)
					return nil
				},
			},
			{
				Name:      "delete",
				Usage:     "Delete an attachment",
				ArgsUsage: "<id>",
				Description: `Permanently delete an attachment.

This removes the stored file and cannot be undone.`,
				Action: func(ctx context.Context, c *cli.Command) error {
					id, err := parseArgID(c, "attachment id")
					if err != nil {
						return err
					}
					cl, err := newClient(c)
					if err != nil {
						return err
					}
					if err := cl.DeleteAttachment(ctx, id); err != nil {
						return err
					}
					fmt.Fprintf(os.Stdout, "Attachment %d deleted\n", id)
					return nil
				},
			},
		},
	}
}

func uploadCommand() *cli.Command {
	return &cli.Command{
		Name:      "upload",
		Usage:     "Upload a file and print its token",
		ArgsUsage: " ",
		Description: `Upload a file to Redmine's /uploads.json endpoint and print the
resulting upload token.

The token can be used as the 'uploads' field when creating an issue via the
Redmine API (e.g. with curl or a scripting layer on top of this CLI).`,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "file", Usage: "Path to the file to upload", Required: true},
			&cli.StringFlag{Name: "filename", Usage: "Filename to report to Redmine (default: the local filename)"},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			file := c.String("file")
			filename := c.String("filename")
			if filename == "" {
				filename = filepath.Base(file)
			}

			cl, err := newClient(c)
			if err != nil {
				return err
			}
			token, err := cl.UploadAttachment(ctx, file, filename)
			if err != nil {
				return err
			}
			if err := output.Print(token, c.String("output")); err != nil {
				return err
			}
			return nil
		},
	}
}

func parseArgID(c *cli.Command, what string) (int, error) {
	raw := strings.TrimSpace(c.Args().First())
	if raw == "" {
		return 0, fmt.Errorf("%s is required", what)
	}
	id, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: must be a number", what, raw)
	}
	return id, nil
}

func displayName(filename *string, fallback string) string {
	if filename != nil && *filename != "" {
		return *filename
	}
	parts := strings.Split(fallback, "/")
	return parts[len(parts)-1]
}
