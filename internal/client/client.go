// Package client speaks the gridraw HTTP protocol.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	// maxErrorBody caps how much of a non-JSON error body reaches the user.
	maxErrorBody = 2048
	// maxErrorRead caps how much of it is read at all.
	maxErrorRead = 64 << 10
)

// HTTPError is a non-2xx response.
type HTTPError struct {
	Status int
	// Msg is the server's own message, or the response body when it carried no
	// {"error": …} envelope.
	Msg string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("server returned %d: %s", e.Status, e.Msg)
}

// HTTPStatus lets internal/cli classify the error without importing this package.
func (e *HTTPError) HTTPStatus() int { return e.Status }

// Client talks to one grids base URL.
type Client struct {
	base   string
	header string
	http   *http.Client
}

// New returns a client for the base grids URL; a trailing slash is trimmed.
// Pass nil for hc to get a client with a 60s timeout.
func New(base, header string, hc *http.Client) *Client {
	if hc == nil {
		hc = &http.Client{Timeout: 60 * time.Second}
	}
	return &Client{base: strings.TrimRight(base, "/"), header: header, http: hc}
}

func (c *Client) do(ctx context.Context, method, path string, body any) ([]byte, error) {
	var rdr io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("cannot encode the request: %w", err)
		}
		rdr = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, rdr)
	if err != nil {
		return nil, err
	}
	if c.header != "" {
		req.Header.Set("Authorization", c.header)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		// A failing proxy can answer with megabytes of HTML, so the body is
		// bounded; the window is well above maxErrorBody because a long but
		// legitimate {"error": …} envelope still has to parse.
		body, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorRead))
		if err != nil {
			return nil, err
		}
		return nil, &HTTPError{Status: resp.StatusCode, Msg: errorMessage(body)}
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

// errorMessage prefers the server's {"error": …} field and falls back to the
// raw body, which may be a proxy's HTML.
func errorMessage(raw []byte) string {
	var envelope struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(raw, &envelope) == nil && envelope.Error != "" {
		return envelope.Error
	}
	msg := strings.TrimSpace(string(raw))
	if len(msg) > maxErrorBody {
		// Cutting by bytes can split a rune; dropping the remnant keeps the
		// message valid UTF-8.
		msg = strings.ToValidUTF8(msg[:maxErrorBody], "") + "…"
	}
	if msg == "" {
		msg = "empty response body"
	}
	return msg
}
