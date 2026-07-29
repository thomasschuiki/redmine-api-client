package redminecmd

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/tom-redmine/go-redmine-cli/internal/client"
	"github.com/tom-redmine/go-redmine-cli/internal/config"
	"github.com/urfave/cli/v3"
)

var redmineClient *client.Client

func newClient(c *cli.Command) (*client.Client, error) {
	if redmineClient == nil {
		cfg, err := config.Load()
		if err != nil {
			return nil, err
		}
		connectTimeout, err := time.ParseDuration(c.String("connect-timeout"))
		if err != nil {
			return nil, fmt.Errorf("invalid --connect-timeout: %w", err)
		}
		maxTime, err := time.ParseDuration(c.String("max-time"))
		if err != nil {
			return nil, fmt.Errorf("invalid --max-time: %w", err)
		}
		redmineClient = client.New(cfg, connectTimeout, maxTime, c.Int("retries"))
		client.SetLogWriter(os.Stderr)
	}
	return redmineClient, nil
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
			&cli.StringFlag{
				Name:  "output",
				Usage: "Output format (yaml, json)",
				Value: "yaml",
				Validator: func(v string) error {
					if v != "yaml" && v != "json" {
						return fmt.Errorf("invalid output format %q — must be yaml or json", v)
					}
					return nil
				},
			},
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
