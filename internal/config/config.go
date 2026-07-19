package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds the CLI configuration including authentication.
type Config struct {
	URL  string     `yaml:"url"`
	Auth AuthConfig `yaml:"auth"`
}

// AuthConfig specifies how to authenticate with Redmine.
type AuthConfig struct {
	Type     string `yaml:"type"`     // "api_key", "basic", "oauth2"
	APIKey   string `yaml:"key"`      // for api_key
	Username string `yaml:"username"` // for basic
	Password string `yaml:"password"` // for basic
	Token    string `yaml:"token"`    // for oauth2
}

// Load reads the config from the default location.
// It checks XDG_CONFIG_HOME/redmine-cli/config.yaml first,
// then falls back to ~/.redmine-cli.yaml.
func Load() (*Config, error) {
	paths := configPaths()

	var data []byte
	var err error
	for _, p := range paths {
		data, err = os.ReadFile(p)
		if err == nil {
			break
		}
	}
	if data == nil {
		return nil, fmt.Errorf("no config file found (tried %v)", paths)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if cfg.URL == "" {
		return nil, fmt.Errorf("config: url is required")
	}

	if err := cfg.Auth.validate(); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	return &cfg, nil
}

func (a *AuthConfig) validate() error {
	switch a.Type {
	case "api_key":
		if a.APIKey == "" {
			return fmt.Errorf("auth.key is required for api_key auth")
		}
	case "basic":
		if a.Username == "" || a.Password == "" {
			return fmt.Errorf("auth.username and auth.password are required for basic auth")
		}
	case "oauth2":
		if a.Token == "" {
			return fmt.Errorf("auth.token is required for oauth2 auth")
		}
	case "":
		return fmt.Errorf("auth.type is required (api_key, basic, or oauth2)")
	default:
		return fmt.Errorf("unknown auth.type %q (must be api_key, basic, or oauth2)", a.Type)
	}
	return nil
}

func configPaths() []string {
	var paths []string

	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		paths = append(paths, filepath.Join(xdg, "redmine-cli", "config.yaml"))
	} else if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".config", "redmine-cli", "config.yaml"))
	}

	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".redmine-cli.yaml"))
	}

	return paths
}
