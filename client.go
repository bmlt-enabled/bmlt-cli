package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	Root    string // ends with "/main_server/" (or equivalent)
	Format  string // json, jsonp, tsml, csv
	Timeout time.Duration
	HTTP    *http.Client
}

func NewClient(root string, timeout time.Duration) *Client {
	return &Client{
		Root:    normalizeRoot(root),
		Format:  "json",
		Timeout: timeout,
		HTTP:    &http.Client{Timeout: timeout},
	}
}

// normalizeRoot ensures the URL ends with a slash. We do not force /main_server/
// — some servers mount elsewhere — but we strip a trailing client_interface/...
// fragment if the user pasted one.
func normalizeRoot(root string) string {
	root = strings.TrimSpace(root)
	if i := strings.Index(root, "/client_interface"); i >= 0 {
		root = root[:i]
	}
	if !strings.HasSuffix(root, "/") {
		root += "/"
	}
	return root
}

// BuildURL composes a full Semantic API URL.
func (c *Client) BuildURL(switcher string, params url.Values) string {
	if params == nil {
		params = url.Values{}
	}
	params.Set("switcher", switcher)
	return fmt.Sprintf("%sclient_interface/%s/?%s", c.Root, c.Format, params.Encode())
}

// Fetch performs the GET and returns the raw body.
func (c *Client) Fetch(switcher string, params url.Values) ([]byte, string, error) {
	u := c.BuildURL(switcher, params)
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, u, err
	}
	req.Header.Set("User-Agent", "bmlt-cli/"+version)
	req.Header.Set("Accept", acceptFor(c.Format))

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, u, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, u, fmt.Errorf("read body: %w", err)
	}
	if resp.StatusCode >= 400 {
		return body, u, fmt.Errorf("HTTP %d from %s\n%s", resp.StatusCode, u, truncate(string(body), 500))
	}
	return body, u, nil
}

// FetchJSON fetches and decodes into v.
func (c *Client) FetchJSON(switcher string, params url.Values, v any) (string, error) {
	body, u, err := c.Fetch(switcher, params)
	if err != nil {
		return u, err
	}
	if err := json.Unmarshal(body, v); err != nil {
		return u, fmt.Errorf("decode JSON: %w\n%s", err, truncate(string(body), 300))
	}
	return u, nil
}

func acceptFor(format string) string {
	switch format {
	case "csv":
		return "text/csv"
	case "tsml":
		return "application/json"
	default:
		return "application/json"
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
