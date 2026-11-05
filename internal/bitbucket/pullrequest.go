package bitbucket

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// CreatePullRequestInput holds the fields needed to open a pull request.
type CreatePullRequestInput struct {
	Title             string
	Description       string
	SourceBranch      string
	DestinationBranch string
	CloseSourceBranch bool
	Draft             bool
	// Reviewers are account UUIDs.
	Reviewers []string
}

func (c *Client) prPath(owner, slug string) string {
	return c.RepoPath(owner, slug) + "/pullrequests"
}

func (c *Client) prItemPath(owner, slug string, id int) string {
	return fmt.Sprintf("%s/%d", c.prPath(owner, slug), id)
}

// ListPullRequests lists pull requests. Callers pass state and q filters.
func (c *Client) ListPullRequests(ctx context.Context, owner, slug string, q url.Values, limit int) ([]PullRequest, error) {
	if !c.IsCloud() {
		return nil, unsupported("pull request list")
	}
	return GetAll[PullRequest](ctx, c, c.prPath(owner, slug), q, limit)
}

// GetPullRequest fetches one pull request.
func (c *Client) GetPullRequest(ctx context.Context, owner, slug string, id int) (*PullRequest, error) {
	if !c.IsCloud() {
		return nil, unsupported("pull request view")
	}
	var pr PullRequest
	if err := c.Get(ctx, c.prItemPath(owner, slug, id), nil, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

// FindOpenPullRequestForBranch returns the first open pull request whose
// source branch matches branch.
func (c *Client) FindOpenPullRequestForBranch(ctx context.Context, owner, slug, branch string) (*PullRequest, error) {
	q := url.Values{}
	q.Set("state", "OPEN")
	q.Set("q", fmt.Sprintf(`source.branch.name="%s"`, strings.ReplaceAll(branch, `"`, `\"`)))
	prs, err := c.ListPullRequests(ctx, owner, slug, q, 1)
	if err != nil {
		return nil, err
	}
	if len(prs) == 0 {
		return nil, fmt.Errorf("no open pull request found for branch %q", branch)
	}
	return &prs[0], nil
}

// FindOpenPullRequestBetween returns the open pull request from source into
// destination, or nil when there is none.
//
// Bitbucket Cloud does not reject a second POST for a source/destination pair
// that already has an open pull request: it silently updates the existing one
// (title, description, reviewers). Callers that intend to create must check
// first, or they overwrite someone's pull request.
func (c *Client) FindOpenPullRequestBetween(ctx context.Context, owner, slug, source, destination string) (*PullRequest, error) {
	q := url.Values{}
	q.Set("state", "OPEN")
	q.Set("q", fmt.Sprintf(`source.branch.name="%s" AND destination.branch.name="%s"`,
		strings.ReplaceAll(source, `"`, `\"`), strings.ReplaceAll(destination, `"`, `\"`)))
	prs, err := c.ListPullRequests(ctx, owner, slug, q, 1)
	if err != nil {
		return nil, err
	}
	if len(prs) == 0 {
		return nil, nil
	}
	return &prs[0], nil
}

// UpdatePullRequestInput holds the fields changed by UpdatePullRequest. Empty
// strings and a nil Reviewers slice are left untouched.
type UpdatePullRequestInput struct {
	Title             string
	Description       string
	CloseSourceBranch bool
	Reviewers         []string
}

// UpdatePullRequest edits an existing pull request.
func (c *Client) UpdatePullRequest(ctx context.Context, owner, slug string, id int, in UpdatePullRequestInput) (*PullRequest, error) {
	if !c.IsCloud() {
		return nil, unsupported("pull request update")
	}
	body := map[string]any{}
	if in.Title != "" {
		body["title"] = in.Title
	}
	if in.Description != "" {
		body["description"] = in.Description
	}
	if in.CloseSourceBranch {
		body["close_source_branch"] = true
	}
	if in.Reviewers != nil {
		reviewers := make([]map[string]string, 0, len(in.Reviewers))
		for _, uuid := range in.Reviewers {
			reviewers = append(reviewers, map[string]string{"uuid": uuid})
		}
		body["reviewers"] = reviewers
	}
	var pr PullRequest
	if err := c.Send(ctx, "PUT", c.prItemPath(owner, slug, id), body, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

// CreatePullRequest opens a pull request.
func (c *Client) CreatePullRequest(ctx context.Context, owner, slug string, in CreatePullRequestInput) (*PullRequest, error) {
	if !c.IsCloud() {
		return nil, unsupported("pull request create")
	}
	body := map[string]any{
		"title": in.Title,
		"source": map[string]any{
			"branch": map[string]string{"name": in.SourceBranch},
		},
		"destination": map[string]any{
			"branch": map[string]string{"name": in.DestinationBranch},
		},
	}
	if in.Description != "" {
		body["description"] = in.Description
	}
	if in.CloseSourceBranch {
		body["close_source_branch"] = true
	}
	if in.Draft {
		body["draft"] = true
	}
	if len(in.Reviewers) > 0 {
		reviewers := make([]map[string]string, 0, len(in.Reviewers))
		for _, uuid := range in.Reviewers {
			reviewers = append(reviewers, map[string]string{"uuid": uuid})
		}
		body["reviewers"] = reviewers
	}
	var pr PullRequest
	if err := c.Send(ctx, "POST", c.prPath(owner, slug), body, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

// MergePullRequest merges a pull request.
func (c *Client) MergePullRequest(ctx context.Context, owner, slug string, id int, strategy, message string, closeSourceBranch bool) (*PullRequest, error) {
	if !c.IsCloud() {
		return nil, unsupported("pull request merge")
	}
	body := map[string]any{}
	if strategy != "" {
		body["merge_strategy"] = strategy
	}
	if message != "" {
		body["message"] = message
	}
	if closeSourceBranch {
		body["close_source_branch"] = true
	}
	var pr PullRequest
	if err := c.Send(ctx, "POST", c.prItemPath(owner, slug, id)+"/merge", body, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

// DeclinePullRequest declines a pull request. The decline endpoint does not
// accept a reason, so callers that want one should post a comment.
func (c *Client) DeclinePullRequest(ctx context.Context, owner, slug string, id int) (*PullRequest, error) {
	if !c.IsCloud() {
		return nil, unsupported("pull request decline")
	}
	var pr PullRequest
	if err := c.Send(ctx, "POST", c.prItemPath(owner, slug, id)+"/decline", nil, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

// SetPullRequestState reopens a pull request.
func (c *Client) SetPullRequestState(ctx context.Context, owner, slug string, id int, state string) (*PullRequest, error) {
	if !c.IsCloud() {
		return nil, unsupported("pull request state change")
	}
	var pr PullRequest
	body := map[string]string{"state": state}
	if err := c.Send(ctx, "PUT", c.prItemPath(owner, slug, id), body, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

// SetReviewAction approves, unapproves, requests changes on, or clears a
// change request for a pull request.
func (c *Client) SetReviewAction(ctx context.Context, owner, slug string, id int, action string, enabled bool) error {
	if !c.IsCloud() {
		return unsupported("pull request review")
	}
	method := "POST"
	if !enabled {
		method = "DELETE"
	}
	return c.Send(ctx, method, c.prItemPath(owner, slug, id)+"/"+action, nil, nil)
}

// ListComments lists pull request comments.
func (c *Client) ListComments(ctx context.Context, owner, slug string, id int, limit int) ([]Comment, error) {
	if !c.IsCloud() {
		return nil, unsupported("pull request comments")
	}
	return GetAll[Comment](ctx, c, c.prItemPath(owner, slug, id)+"/comments", nil, limit)
}

// AddComment posts a raw comment on a pull request.
func (c *Client) AddComment(ctx context.Context, owner, slug string, id int, body string) (*Comment, error) {
	if !c.IsCloud() {
		return nil, unsupported("pull request comment")
	}
	payload := map[string]any{"content": map[string]string{"raw": body}}
	var comment Comment
	path := c.prItemPath(owner, slug, id) + "/comments"
	if err := c.Send(ctx, "POST", path, payload, &comment); err != nil {
		return nil, err
	}
	return &comment, nil
}

// GetPullRequestDiff returns the unified diff for a pull request.
func (c *Client) GetPullRequestDiff(ctx context.Context, owner, slug string, id int) ([]byte, error) {
	if !c.IsCloud() {
		return nil, unsupported("pull request diff")
	}
	return c.GetRaw(ctx, c.prItemPath(owner, slug, id)+"/diff", nil, "text/plain")
}

// ListPullRequestStatuses lists build statuses for the pull request head.
func (c *Client) ListPullRequestStatuses(ctx context.Context, owner, slug string, id int, limit int) ([]Status, error) {
	if !c.IsCloud() {
		return nil, unsupported("pull request checks")
	}
	return GetAll[Status](ctx, c, c.prItemPath(owner, slug, id)+"/statuses", nil, limit)
}

// ListPullRequestCommits lists commits in a pull request.
func (c *Client) ListPullRequestCommits(ctx context.Context, owner, slug string, id int, limit int) ([]Commit, error) {
	if !c.IsCloud() {
		return nil, unsupported("pull request commits")
	}
	return GetAll[Commit](ctx, c, c.prItemPath(owner, slug, id)+"/commits", nil, limit)
}

// ResolveUserUUID resolves a username, nickname, or UUID to an account UUID.
func (c *Client) ResolveUserUUID(ctx context.Context, user string) (string, error) {
	if strings.HasPrefix(user, "{") && strings.HasSuffix(user, "}") {
		return user, nil
	}
	if user == "@me" || user == "me" {
		me, err := c.GetCurrentUser(ctx)
		if err != nil {
			return "", err
		}
		return me.UUID, nil
	}
	if !c.IsCloud() {
		return "", unsupported("user lookup")
	}
	var account Account
	if err := c.Get(ctx, "/users/"+url.PathEscape(user), nil, &account); err != nil {
		return "", err
	}
	if account.UUID == "" {
		return "", fmt.Errorf("user %q has no UUID", user)
	}
	return account.UUID, nil
}
