package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tom-redmine/go-redmine-cli/internal/config"
)

// Client is an HTTP client for the Redmine API.
type Client struct {
	BaseURL        string
	HTTPClient     *http.Client
	Auth           config.AuthConfig
	ConnectTimeout time.Duration
	MaxTime        time.Duration
	Retries        int
}

// New creates a new Redmine API client with timeout and retry settings.
func New(cfg *config.Config, connectTimeout, maxTime time.Duration, retries int) *Client {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   connectTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   connectTimeout,
		ResponseHeaderTimeout: maxTime,
	}
	return &Client{
		BaseURL:        strings.TrimRight(cfg.URL, "/"),
		HTTPClient:     &http.Client{Transport: transport},
		Auth:           cfg.Auth,
		ConnectTimeout: connectTimeout,
		MaxTime:        maxTime,
		Retries:        retries,
	}
}

// doWithRetry executes a request factory with retries and per-request timeout.
// Retries on network errors and 5xx responses with exponential backoff.
func (c *Client) doWithRetry(createReq func() (*http.Request, error)) (*http.Response, error) {
	maxAttempts := c.Retries + 1
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(backoff(attempt))
		}

		req, err := createReq()
		if err != nil {
			return nil, err
		}

		ctx, cancel := context.WithTimeout(context.Background(), c.MaxTime)
		req = req.WithContext(ctx)
		c.authenticate(req)

		resp, err := c.HTTPClient.Do(req)
		cancel()

		if err != nil {
			lastErr = &RequestError{
				Method:  req.Method,
				URL:     req.URL.String(),
				Err:     err,
				Timeout: c.MaxTime,
				Attempt: attempt + 1,
			}
			continue
		}

		if resp.StatusCode >= 500 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastErr = &APIError{
				StatusCode: resp.StatusCode,
				Body:       string(body),
				Method:     req.Method,
				URL:        req.URL.String(),
				Attempt:    attempt + 1,
			}
			continue
		}

		return resp, nil
	}

	return nil, lastErr
}

func backoff(attempt int) time.Duration {
	return time.Duration(1<<(attempt-1)) * time.Second
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

	resp, err := c.doWithRetry(func() (*http.Request, error) {
		req, err := http.NewRequest("GET", u, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		return req, nil
	})
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
	resp, err := c.doWithRetry(func() (*http.Request, error) {
		return http.NewRequest("DELETE", u, nil)
	})
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *Client) doWithBody(method, path string, body any, target any) error {
	u := c.BaseURL + path

	resp, err := c.doWithRetry(func() (*http.Request, error) {
		var bodyReader io.Reader
		if body != nil {
			jsonBytes, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("marshaling request body: %w", err)
			}
			bodyReader = strings.NewReader(string(jsonBytes))
		}
		req, err := http.NewRequest(method, u, bodyReader)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		return req, nil
	})
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

// RequestError represents a network-level failure including timeouts.
type RequestError struct {
	Method  string
	URL     string
	Err     error
	Timeout time.Duration
	Attempt int
}

func (e *RequestError) Error() string {
	return fmt.Sprintf("%s %s failed (attempt %d/%d, timeout %s): %v",
		e.Method, e.URL, e.Attempt, e.Attempt, e.Timeout, e.Err)
}

func (e *RequestError) Unwrap() error { return e.Err }

// APIError represents a non-2xx response from the Redmine API.
type APIError struct {
	StatusCode int
	Body       string
	Method     string
	URL        string
	Attempt    int
}

func (e *APIError) Error() string {
	msg := fmt.Sprintf("%s %s: HTTP %d", e.Method, e.URL, e.StatusCode)
	if e.Attempt > 1 {
		msg += fmt.Sprintf(" (attempt %d)", e.Attempt)
	}
	if e.Body != "" {
		msg += ": " + truncate(e.Body, 200)
	}
	return msg
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
