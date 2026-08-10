package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/tom-redmine/go-redmine-cli/internal/models"
)

// Attachment represents a Redmine file attachment.
type Attachment = models.Attachment

// AttachmentUpdateRequest is the request body for updating an attachment.
type AttachmentUpdateRequest = models.AttachmentUpdateRequest

// GetAttachment returns a single attachment by ID.
func (c *Client) GetAttachment(ctx context.Context, id int) (*Attachment, error) {
	var result struct {
		Attachment Attachment `json:"attachment"`
	}
	path := fmt.Sprintf("/attachments/%d.json", id)
	if err := c.get(ctx, path, nil, &result); err != nil {
		return nil, err
	}
	return &result.Attachment, nil
}

// UpdateAttachment updates an existing attachment (e.g. its description).
func (c *Client) UpdateAttachment(ctx context.Context, id int, req AttachmentUpdateRequest) error {
	path := fmt.Sprintf("/attachments/%d.json", id)
	return c.put(ctx, path, req, nil)
}

// DeleteAttachment deletes an attachment.
func (c *Client) DeleteAttachment(ctx context.Context, id int) error {
	path := fmt.Sprintf("/attachments/%d.json", id)
	return c.delete(ctx, path)
}

// Download streams an authenticated content URL (e.g. an attachment's
// content_url) into w. The URL may be absolute or relative to BaseURL.
func (c *Client) Download(ctx context.Context, contentPath string, w io.Writer) error {
	u := contentPath
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		u = c.BaseURL + u
	}
	resp, err := c.doWithRetry(ctx, func() (*http.Request, error) {
		req, err := http.NewRequest("GET", u, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/octet-stream")
		return req, nil
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		return &APIError{Method: "GET", URL: u, StatusCode: resp.StatusCode, Body: string(body)}
	}
	_, err = io.Copy(w, resp.Body)
	return err
}

// UploadToken is the response from POSTing a file to /uploads.json.
type UploadToken struct {
	Upload struct {
		ID    int    `json:"id"`
		Token string `json:"token"`
	} `json:"upload"`
}

// UploadAttachment posts a file's raw bytes to /uploads.json and returns the
// upload token for use in issue creation. The body is application/octet-stream
// (Redmine uploads are not JSON), so this bypasses the JSON helpers.
func (c *Client) UploadAttachment(ctx context.Context, path, filename string) (*UploadToken, error) {
	u := c.BaseURL + "/uploads.json"
	if filename != "" {
		u += "?filename=" + url.QueryEscape(filename)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	resp, err := c.doWithRetry(ctx, func() (*http.Request, error) {
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("rewinding %s for request: %w", path, err)
		}
		req, err := http.NewRequest("POST", u, f)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/octet-stream")
		req.Header.Set("Accept", "application/json")
		return req, nil
	})
	if err != nil {
		return nil, err
	}

	var token UploadToken
	if err := c.decodeResponse("POST", u, resp, &token); err != nil {
		return nil, err
	}
	return &token, nil
}
