package client

import (
	"bytes"
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
// Retry attempts are logged to stderr.
func (c *Client) doWithRetry(ctx context.Context, createReq func() (*http.Request, error)) (*http.Response, error) {
	maxAttempts := c.Retries + 1
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			if lastErr != nil {
				fmt.Fprintf(logWriter, "retry %d/%d: %v\n", attempt, maxAttempts-1, lastErr)
			}
			time.Sleep(backoff(attempt))
		}

		req, err := createReq()
		if err != nil {
			return nil, err
		}

		ctx, cancel := context.WithTimeout(ctx, c.MaxTime)
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
				Retries: c.Retries,
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
				Retries:    c.Retries,
			}
			continue
		}

		return resp, nil
	}

	return nil, lastErr
}

var logWriter io.Writer = io.Discard

// SetLogWriter redirects retry log output to the given writer.
// By default retry logs are discarded. Pass os.Stderr to enable.
func SetLogWriter(w io.Writer) { logWriter = w }

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
func (c *Client) get(ctx context.Context, path string, params url.Values, target any) error {
	u := c.BaseURL + path
	if params != nil && len(params) > 0 {
		u += "?" + params.Encode()
	}

	resp, err := c.doWithRetry(ctx, func() (*http.Request, error) {
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
		raw, err := io.ReadAll(resp.Body)
		if err != nil {
			return &DecodeError{
				Method:      "GET",
				URL:         u,
				StatusCode:  resp.StatusCode,
				BodySnippet: "",
				Err:         fmt.Errorf("reading response body: %w", err),
			}
		}
		if err := json.Unmarshal(raw, target); err != nil {
			return &DecodeError{
				Method:      "GET",
				URL:         u,
				StatusCode:  resp.StatusCode,
				BodySnippet: string(raw),
				Err:         err,
			}
		}
	}
	return nil
}

// post performs a POST request with a JSON body and decodes the response.
func (c *Client) post(ctx context.Context, path string, body any, target any) error {
	return c.doWithBody(ctx, "POST", path, body, target)
}

// put performs a PUT request with a JSON body.
func (c *Client) put(ctx context.Context, path string, body any, target any) error {
	return c.doWithBody(ctx, "PUT", path, body, target)
}

// delete performs a DELETE request.
func (c *Client) delete(ctx context.Context, path string) error {
	u := c.BaseURL + path
	resp, err := c.doWithRetry(ctx, func() (*http.Request, error) {
		return http.NewRequest("DELETE", u, nil)
	})
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *Client) doWithBody(ctx context.Context, method, path string, body any, target any) error {
	u := c.BaseURL + path

	resp, err := c.doWithRetry(ctx, func() (*http.Request, error) {
		var bodyReader io.Reader
		if body != nil {
			jsonBytes, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("marshaling request body: %w", err)
			}
			bodyReader = bytes.NewReader(jsonBytes)
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
		raw, err := io.ReadAll(resp.Body)
		if err != nil {
			return &DecodeError{
				Method:      method,
				URL:         u,
				StatusCode:  resp.StatusCode,
				BodySnippet: "",
				Err:         fmt.Errorf("reading response body: %w", err),
			}
		}
		if err := json.Unmarshal(raw, target); err != nil {
			return &DecodeError{
				Method:      method,
				URL:         u,
				StatusCode:  resp.StatusCode,
				BodySnippet: string(raw),
				Err:         err,
			}
		}
	}
	return nil
}

// RequestError represents a network-level failure including timeouts.
type RequestError struct {
	Err     error
	Method  string
	URL     string
	Timeout time.Duration
	Attempt int
	Retries int
}

func (e *RequestError) Error() string {
	msg := fmt.Sprintf("%s %s", e.Method, e.URL)
	reason := e.Err.Error()
	if contextMsg := contextCanceledReason(reason); contextMsg != "" {
		reason = contextMsg
	}
	if e.Retries > 0 {
		return fmt.Sprintf("%s: %s (after %d retries, %s timeout)", msg, reason, e.Retries, e.Timeout)
	}
	return fmt.Sprintf("%s: %s (timeout %s)", msg, reason, e.Timeout)
}

func contextCanceledReason(s string) string {
	if strings.Contains(s, "context deadline exceeded") {
		return "request timed out"
	}
	if strings.Contains(s, "connection refused") {
		return "connection refused"
	}
	if strings.Contains(s, "no such host") || strings.Contains(s, "DNS") || strings.Contains(s, "lookup") {
		return "DNS resolution failed — check your Redmine URL"
	}
	if strings.Contains(s, "EOF") {
		return "unexpected EOF — server closed connection"
	}
	return ""
}

func (e *RequestError) Unwrap() error { return e.Err }

// APIError represents a non-2xx response from the Redmine API.
type APIError struct {
	Body       string
	Method     string
	URL        string
	StatusCode int
	Attempt    int
	Retries    int
}

func (e *APIError) Error() string {
	msg := fmt.Sprintf("%s %s: HTTP %d", e.Method, e.URL, e.StatusCode)
	if e.Retries > 0 && e.Attempt > 1 {
		msg += fmt.Sprintf(" (after %d retries)", e.Retries)
	}
	if e.Body != "" {
		msg += ": " + truncate(e.Body, 200)
	}
	return msg
}

// DecodeError represents a failure to decode a successful HTTP response.
type DecodeError struct {
	Err         error
	Method      string
	URL         string
	BodySnippet string
	StatusCode  int
}

func (e *DecodeError) Error() string {
	msg := fmt.Sprintf("%s %s: HTTP %d: decoding response: %v", e.Method, e.URL, e.StatusCode, e.Err)
	if e.BodySnippet != "" {
		msg += " | body: " + truncate(e.BodySnippet, 100)
	}
	return msg
}

func (e *DecodeError) Unwrap() error { return e.Err }

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
