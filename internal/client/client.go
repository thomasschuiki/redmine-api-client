package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/tom-redmine/go-redmine-cli/internal/config"
)

// Client is an HTTP client for the Redmine API.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	Auth       config.AuthConfig
}

// New creates a new Redmine API client.
func New(cfg *config.Config) *Client {
	return &Client{
		BaseURL:    strings.TrimRight(cfg.URL, "/"),
		HTTPClient: http.DefaultClient,
		Auth:       cfg.Auth,
	}
}

// do executes an HTTP request and returns the response.
// It injects authentication and returns structured errors.
func (c *Client) do(req *http.Request) (*http.Response, error) {
	c.authenticate(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(body),
		}
	}

	return resp, nil
}

func (c *Client) authenticate(req *http.Request) {
	switch c.Auth.Type {
	case "api_key":
		req.Header.Set("X-Redmine-API-Key", c.Auth.APIKey)
	case "basic":
		req.SetBasicAuth(c.Auth.Username, c.Auth.Password)
	case "oauth2":
		req.Header.Set("Authorization", "Bearer "+c.Auth.Token)
	}
}

// get performs a GET request and decodes the JSON response into target.
func (c *Client) get(path string, params url.Values, target any) error {
	u := c.BaseURL + path
	if params != nil && len(params) > 0 {
		u += "?" + params.Encode()
	}

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if target != nil {
		if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}
	return nil
}

// post performs a POST request with a JSON body and decodes the response.
func (c *Client) post(path string, body any, target any) error {
	return c.doWithBody("POST", path, body, target)
}

// put performs a PUT request with a JSON body.
func (c *Client) put(path string, body any, target any) error {
	return c.doWithBody("PUT", path, body, target)
}

// delete performs a DELETE request.
func (c *Client) delete(path string) error {
	u := c.BaseURL + path
	req, err := http.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *Client) doWithBody(method, path string, body any, target any) error {
	u := c.BaseURL + path

	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = strings.NewReader(string(jsonBytes))
	}

	req, err := http.NewRequest(method, u, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if target != nil {
		if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}
	return nil
}

// APIError represents a non-2xx response from the Redmine API.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	if e.Body != "" {
		return fmt.Sprintf("API error %d: %s", e.StatusCode, truncate(e.Body, 200))
	}
	return fmt.Sprintf("API error %d", e.StatusCode)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// ListOpts holds common pagination options.
type ListOpts struct {
	Offset int
	Limit  int
}

// Params converts ListOpts to url.Values.
func (o ListOpts) Params() url.Values {
	v := url.Values{}
	if o.Offset > 0 {
		v.Set("offset", fmt.Sprintf("%d", o.Offset))
	}
	if o.Limit > 0 {
		v.Set("limit", fmt.Sprintf("%d", o.Limit))
	}
	return v
}
