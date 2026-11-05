// Package bitbucket implements a small, typed client for the Bitbucket Cloud
// REST 2.0 API with a pluggable path layer for Data Center.
package bitbucket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ClaudioLazaro/bitbucket-cli/internal/config"
)

// Client talks to one Bitbucket host.
type Client struct {
	BaseURL    *url.URL
	HTTP       *http.Client
	Token      string
	Username   string
	AuthMethod string
	Kind       string
	UserAgent  string

	// Debug enables request logging to DebugOut.
	Debug    bool
	DebugOut io.Writer
}

// NewClient builds a client from resolved settings.
func NewClient(settings config.Settings) (*Client, error) {
	base, err := url.Parse(settings.APIURL)
	if err != nil {
		return nil, fmt.Errorf("invalid API URL %q: %w", settings.APIURL, err)
	}
	if base.Scheme != "https" && base.Scheme != "http" {
		return nil, fmt.Errorf("invalid API URL %q: scheme must be http or https", settings.APIURL)
	}
	c := &Client{
		BaseURL:    base,
		HTTP:       &http.Client{Timeout: 60 * time.Second},
		Token:      settings.Token,
		Username:   settings.Username,
		AuthMethod: settings.AuthMethod,
		Kind:       settings.Kind,
		UserAgent:  "bk",
	}
	c.Debug = os.Getenv("BK_HTTP_DEBUG") != ""
	c.DebugOut = os.Stderr
	return c, nil
}

// IsCloud reports whether the client targets Bitbucket Cloud.
func (c *Client) IsCloud() bool { return c.Kind == config.KindCloud }

// RepoPath returns the REST path for a repository.
func (c *Client) RepoPath(owner, slug string) string {
	if c.IsCloud() {
		return fmt.Sprintf("/repositories/%s/%s", url.PathEscape(owner), url.PathEscape(slug))
	}
	return fmt.Sprintf("/projects/%s/repos/%s", url.PathEscape(owner), url.PathEscape(slug))
}

func (c *Client) resolveURL(path string) (string, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		u, err := url.Parse(path)
		if err != nil {
			return "", err
		}
		if !strings.EqualFold(u.Host, c.BaseURL.Host) {
			return "", fmt.Errorf("refusing to call a different host: %s", u.Host)
		}
		return u.String(), nil
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return strings.TrimRight(c.BaseURL.String(), "/") + path, nil
}

func mergeQuery(rawURL string, q url.Values) (string, error) {
	if len(q) == 0 {
		return rawURL, nil
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	values := u.Query()
	for key, list := range q {
		values.Del(key)
		for _, value := range list {
			values.Add(key, value)
		}
	}
	u.RawQuery = values.Encode()
	return u.String(), nil
}

func (c *Client) newRequest(ctx context.Context, method, rawURL string, body []byte, contentType string, headers http.Header) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, rawURL, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for key, values := range headers {
		req.Header.Del(key)
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}
	switch strings.ToLower(c.AuthMethod) {
	case config.AuthBasic:
		if c.Username == "" || c.Token == "" {
			return nil, fmt.Errorf("basic auth requires a username and token")
		}
		req.SetBasicAuth(c.Username, c.Token)
	default:
		if c.Token != "" {
			req.Header.Set("Authorization", "Bearer "+c.Token)
		}
	}
	return req, nil
}

// Do sends a request with retries for throttling and transient server errors.
func (c *Client) Do(ctx context.Context, method, path string, body []byte, contentType string, headers http.Header) (*http.Response, []byte, error) {
	rawURL, err := c.resolveURL(path)
	if err != nil {
		return nil, nil, err
	}
	return c.doURL(ctx, method, rawURL, body, contentType, headers)
}

func (c *Client) doURL(ctx context.Context, method, rawURL string, body []byte, contentType string, headers http.Header) (*http.Response, []byte, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		req, err := c.newRequest(ctx, method, rawURL, body, contentType, headers)
		if err != nil {
			return nil, nil, err
		}
		if c.Debug && c.DebugOut != nil {
			fmt.Fprintf(c.DebugOut, "> %s %s\n", method, req.URL.Redacted())
		}
		resp, err := c.HTTP.Do(req)
		if err != nil {
			lastErr = err
			if attempt < 2 && ctx.Err() == nil {
				time.Sleep(backoff(attempt))
				continue
			}
			return nil, nil, err
		}
		data, readErr := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
		resp.Body.Close()
		if c.Debug && c.DebugOut != nil {
			fmt.Fprintf(c.DebugOut, "< %d %s\n", resp.StatusCode, req.URL.Redacted())
		}
		if readErr != nil {
			return nil, nil, readErr
		}
		if isRetryableStatus(resp.StatusCode) && attempt < 2 {
			time.Sleep(retryDelay(resp, attempt))
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return resp, data, NewHTTPError(resp, method, req.URL.String(), data)
		}
		return resp, data, nil
	}
	return nil, nil, lastErr
}

// Get fetches path with query q and decodes the JSON response into out.
func (c *Client) Get(ctx context.Context, path string, q url.Values, out any) error {
	rawURL, err := c.resolveURL(path)
	if err != nil {
		return err
	}
	rawURL, err = mergeQuery(rawURL, q)
	if err != nil {
		return err
	}
	_, data, err := c.doURL(ctx, http.MethodGet, rawURL, nil, "", nil)
	if err != nil {
		return err
	}
	return decodeJSON(data, out)
}

// Send issues a JSON request and decodes the response into out when non-nil.
func (c *Client) Send(ctx context.Context, method, path string, body any, out any) error {
	var data []byte
	if body != nil {
		var err error
		data, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	_, respData, err := c.Do(ctx, method, path, data, "application/json", nil)
	if err != nil {
		return err
	}
	return decodeJSON(respData, out)
}

// GetRaw fetches path and returns the response body untouched. It follows
// redirects manually so a signed storage URL is fetched without Bitbucket
// credentials, and retries with a wildcard Accept when the host rejects the
// requested format with 406.
func (c *Client) GetRaw(ctx context.Context, path string, q url.Values, accept string) ([]byte, error) {
	rawURL, err := c.resolveURL(path)
	if err != nil {
		return nil, err
	}
	rawURL, err = mergeQuery(rawURL, q)
	if err != nil {
		return nil, err
	}
	headers := http.Header{}
	if accept != "" {
		headers.Set("Accept", accept)
	}

	resp, data, err := c.doRaw(ctx, rawURL, headers)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotAcceptable && accept != "*/*" {
		headers.Set("Accept", "*/*")
		resp, data, err = c.doRaw(ctx, rawURL, headers)
		if err != nil {
			return nil, err
		}
	}
	if isRedirectStatus(resp.StatusCode) {
		location := resp.Header.Get("Location")
		if location == "" {
			return nil, fmt.Errorf("HTTP %d without Location from %s", resp.StatusCode, rawURL)
		}
		base := resp.Request.URL
		if base == nil {
			base, _ = url.Parse(rawURL)
		}
		target, err := base.Parse(location)
		if err != nil {
			return nil, fmt.Errorf("invalid redirect location %q: %w", location, err)
		}
		if target.Scheme != "http" && target.Scheme != "https" {
			return nil, fmt.Errorf("unsupported redirect scheme %q", target.Scheme)
		}
		return c.downloadRaw(ctx, target.String())
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, NewHTTPError(resp, http.MethodGet, rawURL, data)
	}
	return data, nil
}

func (c *Client) doRaw(ctx context.Context, rawURL string, headers http.Header) (*http.Response, []byte, error) {
	client := c.client(false)
	req, err := c.newRequest(ctx, http.MethodGet, rawURL, nil, "", headers)
	if err != nil {
		return nil, nil, err
	}
	if c.Debug && c.DebugOut != nil {
		fmt.Fprintf(c.DebugOut, "> GET %s\n", req.URL.Redacted())
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	data, readErr := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	resp.Body.Close()
	if c.Debug && c.DebugOut != nil {
		fmt.Fprintf(c.DebugOut, "< %d %s\n", resp.StatusCode, req.URL.Redacted())
	}
	if readErr != nil {
		return nil, nil, readErr
	}
	return resp, data, nil
}

func (c *Client) downloadRaw(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", c.UserAgent)
	if c.Debug && c.DebugOut != nil {
		fmt.Fprintf(c.DebugOut, "> GET %s (redirect, no auth)\n", req.URL.Redacted())
	}
	resp, err := c.client(true).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return nil, err
	}
	if c.Debug && c.DebugOut != nil {
		fmt.Fprintf(c.DebugOut, "< %d %s\n", resp.StatusCode, req.URL.Redacted())
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, NewHTTPError(resp, http.MethodGet, rawURL, data)
	}
	return data, nil
}

func (c *Client) client(followRedirects bool) *http.Client {
	base := c.HTTP
	if base == nil {
		base = http.DefaultClient
	}
	if followRedirects {
		return base
	}
	client := *base
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &client
}

func isRedirectStatus(status int) bool {
	switch status {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther,
		http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		return true
	}
	return false
}

// SendRaw issues a request with a caller-provided body and content type.
func (c *Client) SendRaw(ctx context.Context, method, path string, body []byte, contentType string, out any) error {
	_, data, err := c.Do(ctx, method, path, body, contentType, nil)
	if err != nil {
		return err
	}
	return decodeJSON(data, out)
}

func decodeJSON(data []byte, out any) error {
	if out == nil || len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func isRetryableStatus(status int) bool {
	switch status {
	case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	}
	return false
}

func retryDelay(resp *http.Response, attempt int) time.Duration {
	if v := resp.Header.Get("Retry-After"); v != "" {
		if seconds, err := strconv.Atoi(v); err == nil && seconds >= 0 {
			return time.Duration(seconds) * time.Second
		}
		if when, err := http.ParseTime(v); err == nil {
			if d := time.Until(when); d > 0 {
				return d
			}
		}
	}
	return backoff(attempt)
}

func backoff(attempt int) time.Duration {
	return time.Duration(1<<uint(attempt)) * 500 * time.Millisecond
}
