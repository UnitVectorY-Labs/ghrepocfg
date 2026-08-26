// Package github is the small GitHub.com REST client used by ghrepocfg.
package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const apiBase = "https://api.github.com"

type Client struct {
	http  *http.Client
	token string
	base  string
}

func NewClient(token string) *Client {
	return &Client{http: &http.Client{Timeout: 30 * time.Second}, token: token, base: apiBase}
}

type APIError struct {
	Method, Path, Message, DocumentationURL string
	Status                                  int
}

func (e *APIError) Error() string {
	msg := fmt.Sprintf("GitHub API %s %s failed (%d)", e.Method, e.Path, e.Status)
	if e.Message != "" {
		msg += ": " + e.Message
	}
	return msg
}

func (c *Client) request(ctx context.Context, method, path string, body any, out any) (http.Header, error) {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, r)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "ghrepocfg")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GitHub API request failed: %w", err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var detail struct {
			Message, DocumentationURL string
			Errors                    json.RawMessage
		}
		_ = json.Unmarshal(b, &detail)
		return resp.Header, &APIError{Method: method, Path: path, Status: resp.StatusCode, Message: detail.Message, DocumentationURL: detail.DocumentationURL}
	}
	if out != nil && len(b) > 0 {
		if err := json.Unmarshal(b, out); err != nil {
			return nil, fmt.Errorf("decode GitHub response for %s: %w", path, err)
		}
	}
	return resp.Header, nil
}

func (c *Client) paged(ctx context.Context, path string, out any) error {
	// All collection readers use per_page=100 and follow GitHub's Link header.
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	next := path + sep + "per_page=100"
	var aggregate []json.RawMessage
	for next != "" {
		var page []json.RawMessage
		headers, err := c.request(ctx, http.MethodGet, next, nil, &page)
		if err != nil {
			return err
		}
		aggregate = append(aggregate, page...)
		next = nextLink(headers.Get("Link"))
	}
	b, err := json.Marshal(aggregate)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

func nextLink(link string) string {
	for _, part := range strings.Split(link, ",") {
		pieces := strings.Split(strings.TrimSpace(part), ";")
		if len(pieces) < 2 || !strings.Contains(pieces[1], `rel="next"`) {
			continue
		}
		u, err := url.Parse(strings.Trim(pieces[0], "<>"))
		if err == nil {
			return u.RequestURI()
		}
	}
	return ""
}

func repoPath(owner, repo, suffix string) string {
	return "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(repo) + suffix
}

func IsStatus(err error, status int) bool {
	var e *APIError
	return errors.As(err, &e) && e.Status == status
}

func intPath(id int64) string { return strconv.FormatInt(id, 10) }
