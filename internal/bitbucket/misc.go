package bitbucket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
)

// CreateSnippetInput describes a snippet to create.
type CreateSnippetInput struct {
	Title     string
	IsPrivate bool
	Files     map[string]string
}

// ListSnippets lists snippets in a workspace.
func (c *Client) ListSnippets(ctx context.Context, workspace, role string, limit int) ([]Snippet, error) {
	if !c.IsCloud() {
		return nil, unsupported("snippet list")
	}
	q := url.Values{}
	if role != "" {
		q.Set("role", role)
	}
	return GetAll[Snippet](ctx, c, "/snippets/"+url.PathEscape(workspace), q, limit)
}

// GetSnippet fetches one snippet.
func (c *Client) GetSnippet(ctx context.Context, workspace, id string) (*Snippet, error) {
	if !c.IsCloud() {
		return nil, unsupported("snippet view")
	}
	var snippet Snippet
	if err := c.Get(ctx, "/snippets/"+url.PathEscape(workspace)+"/"+url.PathEscape(id), nil, &snippet); err != nil {
		return nil, err
	}
	return &snippet, nil
}

// CreateSnippet creates a snippet. Snippets are uploaded as a multipart form
// with one "file" part per file, as required by the Bitbucket API.
func (c *Client) CreateSnippet(ctx context.Context, workspace string, in CreateSnippetInput) (*Snippet, error) {
	if !c.IsCloud() {
		return nil, unsupported("snippet create")
	}
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	if err := writer.WriteField("title", in.Title); err != nil {
		return nil, err
	}
	if err := writer.WriteField("is_private", strconv.FormatBool(in.IsPrivate)); err != nil {
		return nil, err
	}
	if err := writer.WriteField("scm", "git"); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(in.Files))
	for name := range in.Files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		part, err := writer.CreateFormFile("file", name)
		if err != nil {
			return nil, err
		}
		if _, err := io.WriteString(part, in.Files[name]); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	var snippet Snippet
	path := "/snippets/" + url.PathEscape(workspace)
	if err := c.SendRaw(ctx, "POST", path, buf.Bytes(), writer.FormDataContentType(), &snippet); err != nil {
		return nil, err
	}
	return &snippet, nil
}

// DeleteSnippet deletes a snippet.
func (c *Client) DeleteSnippet(ctx context.Context, workspace, id string) error {
	if !c.IsCloud() {
		return unsupported("snippet delete")
	}
	return c.Send(ctx, "DELETE", "/snippets/"+url.PathEscape(workspace)+"/"+url.PathEscape(id), nil, nil)
}

// GetSnippetFile returns the raw content of one file in a snippet.
func (c *Client) GetSnippetFile(ctx context.Context, workspace, id, path string) ([]byte, error) {
	if !c.IsCloud() {
		return nil, unsupported("snippet view")
	}
	escaped := strings.Split(strings.TrimPrefix(path, "/"), "/")
	for i := range escaped {
		escaped[i] = url.PathEscape(escaped[i])
	}
	endpoint := "/snippets/" + url.PathEscape(workspace) + "/" + url.PathEscape(id) + "/files/" + strings.Join(escaped, "/")
	return c.GetRaw(ctx, endpoint, nil, "text/plain")
}

// ListDownloads lists repository download files.
func (c *Client) ListDownloads(ctx context.Context, owner, slug string, limit int) ([]Download, error) {
	if !c.IsCloud() {
		return nil, unsupported("downloads")
	}
	return GetAll[Download](ctx, c, c.RepoPath(owner, slug)+"/downloads", nil, limit)
}

// UploadDownload uploads one file to the repository downloads directory.
func (c *Client) UploadDownload(ctx context.Context, owner, slug, filename string, content io.Reader) (*Download, error) {
	if !c.IsCloud() {
		return nil, unsupported("downloads")
	}
	tmp, err := os.CreateTemp("", "bk-upload-*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	mw := multipart.NewWriter(tmp)
	part, err := mw.CreateFormFile("files", filename)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, content); err != nil {
		return nil, err
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}
	total, err := tmp.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	rawURL, err := c.resolveURL(c.RepoPath(owner, slug) + "/downloads")
	if err != nil {
		return nil, err
	}
	req, err := c.newRequest(ctx, "POST", rawURL, nil, mw.FormDataContentType(), nil)
	if err != nil {
		return nil, err
	}
	req.Body = tmp
	req.ContentLength = total
	if c.Debug && c.DebugOut != nil {
		fmt.Fprintf(c.DebugOut, "> POST %s (upload %s, %d bytes)\n", req.URL.Redacted(), filename, total)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, NewHTTPError(resp, "POST", req.URL.String(), data)
	}
	var download Download
	if err := decodeJSON(data, &download); err != nil {
		return nil, err
	}
	return &download, nil
}

// DeleteDownload removes a file from the downloads directory.
func (c *Client) DeleteDownload(ctx context.Context, owner, slug, filename string) error {
	if !c.IsCloud() {
		return unsupported("downloads")
	}
	return c.Send(ctx, "DELETE", c.RepoPath(owner, slug)+"/downloads/"+url.PathEscape(filename), nil, nil)
}

// ListWebhooks lists repository webhooks.
func (c *Client) ListWebhooks(ctx context.Context, owner, slug string, limit int) ([]Webhook, error) {
	if !c.IsCloud() {
		return nil, unsupported("webhooks")
	}
	return GetAll[Webhook](ctx, c, c.RepoPath(owner, slug)+"/hooks", nil, limit)
}

// GetWebhook fetches one webhook.
func (c *Client) GetWebhook(ctx context.Context, owner, slug, uuid string) (*Webhook, error) {
	if !c.IsCloud() {
		return nil, unsupported("webhooks")
	}
	var hook Webhook
	if err := c.Get(ctx, c.RepoPath(owner, slug)+"/hooks/"+url.PathEscape(uuid), nil, &hook); err != nil {
		return nil, err
	}
	return &hook, nil
}

// CreateWebhook registers a repository webhook.
func (c *Client) CreateWebhook(ctx context.Context, owner, slug, description, targetURL string, events []string, secret string) (*Webhook, error) {
	if !c.IsCloud() {
		return nil, unsupported("webhooks")
	}
	if targetURL == "" {
		return nil, fmt.Errorf("webhook URL is required")
	}
	if len(events) == 0 {
		events = []string{"repo:push"}
	}
	body := map[string]any{
		"description": description,
		"url":         targetURL,
		"active":      true,
		"events":      events,
	}
	if secret != "" {
		body["secret"] = secret
	}
	var hook Webhook
	if err := c.Send(ctx, "POST", c.RepoPath(owner, slug)+"/hooks", body, &hook); err != nil {
		return nil, err
	}
	return &hook, nil
}

// DeleteWebhook removes a repository webhook.
func (c *Client) DeleteWebhook(ctx context.Context, owner, slug, uuid string) error {
	if !c.IsCloud() {
		return unsupported("webhooks")
	}
	return c.Send(ctx, "DELETE", c.RepoPath(owner, slug)+"/hooks/"+url.PathEscape(uuid), nil, nil)
}

// CreateBranchRestrictionInput describes a branch restriction rule.
type CreateBranchRestrictionInput struct {
	Kind            string
	Pattern         string
	BranchMatchKind string
	Value           *int
	Users           []string
	Groups          []string
}

// ListBranchRestrictions lists branch restriction rules.
func (c *Client) ListBranchRestrictions(ctx context.Context, owner, slug string, limit int) ([]BranchRestriction, error) {
	if !c.IsCloud() {
		return nil, unsupported("branch restrictions")
	}
	return GetAll[BranchRestriction](ctx, c, c.RepoPath(owner, slug)+"/branch-restrictions", nil, limit)
}

// CreateBranchRestriction creates a branch restriction rule.
func (c *Client) CreateBranchRestriction(ctx context.Context, owner, slug string, in CreateBranchRestrictionInput) (*BranchRestriction, error) {
	if !c.IsCloud() {
		return nil, unsupported("branch restrictions")
	}
	matchKind := in.BranchMatchKind
	if matchKind == "" {
		matchKind = "glob"
	}
	body := map[string]any{
		"kind":              in.Kind,
		"pattern":           in.Pattern,
		"branch_match_kind": matchKind,
	}
	if in.Value != nil {
		body["value"] = *in.Value
	}
	if len(in.Users) > 0 {
		users := make([]map[string]string, 0, len(in.Users))
		for _, uuid := range in.Users {
			users = append(users, map[string]string{"uuid": uuid})
		}
		body["users"] = users
	}
	if len(in.Groups) > 0 {
		groups := make([]map[string]string, 0, len(in.Groups))
		for _, group := range in.Groups {
			groups = append(groups, map[string]string{"slug": group})
		}
		body["groups"] = groups
	}
	var restriction BranchRestriction
	if err := c.Send(ctx, "POST", c.RepoPath(owner, slug)+"/branch-restrictions", body, &restriction); err != nil {
		return nil, err
	}
	return &restriction, nil
}

// DeleteBranchRestriction deletes a branch restriction rule.
func (c *Client) DeleteBranchRestriction(ctx context.Context, owner, slug string, id int) error {
	if !c.IsCloud() {
		return unsupported("branch restrictions")
	}
	return c.Send(ctx, "DELETE", fmt.Sprintf("%s/branch-restrictions/%d", c.RepoPath(owner, slug), id), nil, nil)
}

// ListSSHKeys lists SSH keys for a user.
func (c *Client) ListSSHKeys(ctx context.Context, user string) ([]SSHKey, error) {
	if !c.IsCloud() {
		return nil, unsupported("ssh keys")
	}
	return GetAll[SSHKey](ctx, c, "/users/"+url.PathEscape(user)+"/ssh-keys", nil, 0)
}

// AddSSHKey adds an SSH key to a user.
func (c *Client) AddSSHKey(ctx context.Context, user, label, key string) (*SSHKey, error) {
	if !c.IsCloud() {
		return nil, unsupported("ssh keys")
	}
	body := map[string]string{"key": key, "label": label}
	var sshKey SSHKey
	if err := c.Send(ctx, "POST", "/users/"+url.PathEscape(user)+"/ssh-keys", body, &sshKey); err != nil {
		return nil, err
	}
	return &sshKey, nil
}

// DeleteSSHKey deletes an SSH key by numeric ID.
func (c *Client) DeleteSSHKey(ctx context.Context, user string, id int) error {
	if !c.IsCloud() {
		return unsupported("ssh keys")
	}
	return c.Send(ctx, "DELETE", fmt.Sprintf("/users/%s/ssh-keys/%d", url.PathEscape(user), id), nil, nil)
}

// ListGPGKeys lists GPG keys for a user.
func (c *Client) ListGPGKeys(ctx context.Context, user string) ([]GPGKey, error) {
	if !c.IsCloud() {
		return nil, unsupported("gpg keys")
	}
	return GetAll[GPGKey](ctx, c, "/users/"+url.PathEscape(user)+"/gpg-keys", nil, 0)
}

// AddGPGKey adds a GPG key to a user.
func (c *Client) AddGPGKey(ctx context.Context, user, key string) (*GPGKey, error) {
	if !c.IsCloud() {
		return nil, unsupported("gpg keys")
	}
	body := map[string]string{"key": key}
	var gpgKey GPGKey
	if err := c.Send(ctx, "POST", "/users/"+url.PathEscape(user)+"/gpg-keys", body, &gpgKey); err != nil {
		return nil, err
	}
	return &gpgKey, nil
}

// DeleteGPGKey deletes a GPG key by fingerprint.
func (c *Client) DeleteGPGKey(ctx context.Context, user, fingerprint string) error {
	if !c.IsCloud() {
		return unsupported("gpg keys")
	}
	return c.Send(ctx, "DELETE", "/users/"+url.PathEscape(user)+"/gpg-keys/"+url.PathEscape(fingerprint), nil, nil)
}

// ListCommitStatuses lists statuses for a commit.
func (c *Client) ListCommitStatuses(ctx context.Context, owner, slug, commit string, limit int) ([]Status, error) {
	if !c.IsCloud() {
		return nil, unsupported("commit statuses")
	}
	return GetAll[Status](ctx, c, fmt.Sprintf("%s/commit/%s/statuses", c.RepoPath(owner, slug), url.PathEscape(commit)), nil, limit)
}

// CreateCommitStatusInput is the payload for a build status.
type CreateCommitStatusInput struct {
	Key         string
	State       string
	URL         string
	Name        string
	Description string
}

// CreateCommitStatus posts a build status for a commit.
func (c *Client) CreateCommitStatus(ctx context.Context, owner, slug, commit string, in CreateCommitStatusInput) (*Status, error) {
	if !c.IsCloud() {
		return nil, unsupported("commit statuses")
	}
	body := map[string]any{
		"key":   in.Key,
		"state": strings.ToUpper(in.State),
	}
	if in.URL != "" {
		body["url"] = in.URL
	}
	if in.Name != "" {
		body["name"] = in.Name
	}
	if in.Description != "" {
		body["description"] = in.Description
	}
	var status Status
	path := fmt.Sprintf("%s/commit/%s/statuses/build", c.RepoPath(owner, slug), url.PathEscape(commit))
	if err := c.Send(ctx, "POST", path, body, &status); err != nil {
		return nil, err
	}
	return &status, nil
}

// SearchCode searches code in a workspace. Atlassian has announced the
// deprecation of this endpoint for November 2026.
func (c *Client) SearchCode(ctx context.Context, workspace, query string, limit int) ([]SearchResult, error) {
	if !c.IsCloud() {
		return nil, unsupported("code search")
	}
	if workspace == "" {
		return nil, fmt.Errorf("workspace is required")
	}
	q := url.Values{"search_query": []string{query}}
	return GetAll[SearchResult](ctx, c, "/workspaces/"+url.PathEscape(workspace)+"/search/code", q, limit)
}

// RawJSON marshals v without HTML escaping.
func RawJSON(v any) ([]byte, error) {
	var buf strings.Builder
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}
