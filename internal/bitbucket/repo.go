package bitbucket

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// RepoRef identifies a repository on a Bitbucket host.
type RepoRef struct {
	Host  string
	Owner string
	Slug  string
}

// String returns owner/slug.
func (r RepoRef) String() string { return r.Owner + "/" + r.Slug }

// IsZero reports whether the reference is unset.
func (r RepoRef) IsZero() bool { return r.Owner == "" || r.Slug == "" }

// CreateRepositoryInput is the editable set of repository fields.
type CreateRepositoryInput struct {
	SCM         string
	IsPrivate   bool
	Description string
	ProjectKey  string
}

// GetRepository fetches one repository.
func (c *Client) GetRepository(ctx context.Context, owner, slug string) (*Repository, error) {
	var repo Repository
	if err := c.Get(ctx, c.RepoPath(owner, slug), nil, &repo); err != nil {
		return nil, err
	}
	return &repo, nil
}

// ListRepositories lists repositories in a workspace or Data Center project.
// An empty workspace lists everything the credential can see on Data Center.
func (c *Client) ListRepositories(ctx context.Context, workspace string, q url.Values, limit int) ([]Repository, error) {
	var path string
	if c.IsCloud() {
		if workspace == "" {
			return nil, fmt.Errorf("workspace is required")
		}
		path = "/repositories/" + url.PathEscape(workspace)
	} else if workspace != "" {
		path = "/projects/" + url.PathEscape(workspace) + "/repos"
	} else {
		path = "/repos"
	}
	return GetAll[Repository](ctx, c, path, q, limit)
}

// CreateRepository creates a repository. On Cloud, workspace and slug are part
// of the URL. On Data Center, slug becomes the repository name in the project.
func (c *Client) CreateRepository(ctx context.Context, workspace, slug string, in CreateRepositoryInput) (*Repository, error) {
	scm := in.SCM
	if scm == "" {
		scm = "git"
	}
	if c.IsCloud() {
		body := map[string]any{
			"scm":        scm,
			"is_private": in.IsPrivate,
		}
		if in.Description != "" {
			body["description"] = in.Description
		}
		if in.ProjectKey != "" {
			body["project"] = map[string]string{"key": in.ProjectKey}
		}
		var repo Repository
		path := fmt.Sprintf("/repositories/%s/%s", url.PathEscape(workspace), url.PathEscape(slug))
		if err := c.Send(ctx, "POST", path, body, &repo); err != nil {
			return nil, err
		}
		return &repo, nil
	}

	body := map[string]any{
		"name":     slug,
		"scmId":    scm,
		"forkable": true,
	}
	if in.Description != "" {
		body["description"] = in.Description
	}
	var repo Repository
	path := fmt.Sprintf("/projects/%s/repos", url.PathEscape(workspace))
	if err := c.Send(ctx, "POST", path, body, &repo); err != nil {
		return nil, err
	}
	return &repo, nil
}

// UpdateRepository updates repository fields.
func (c *Client) UpdateRepository(ctx context.Context, owner, slug string, body map[string]any) (*Repository, error) {
	var repo Repository
	if err := c.Send(ctx, "PUT", c.RepoPath(owner, slug), body, &repo); err != nil {
		return nil, err
	}
	return &repo, nil
}

// DeleteRepository deletes a repository.
func (c *Client) DeleteRepository(ctx context.Context, owner, slug string) error {
	return c.Send(ctx, "DELETE", c.RepoPath(owner, slug), nil, nil)
}

// ForkRepository forks a Cloud repository.
func (c *Client) ForkRepository(ctx context.Context, owner, slug, workspace, name string) (*Repository, error) {
	if !c.IsCloud() {
		return nil, unsupported("repository fork")
	}
	body := map[string]any{}
	if name != "" {
		body["name"] = name
	}
	if workspace != "" {
		body["workspace"] = map[string]string{"slug": workspace}
	}
	var repo Repository
	if err := c.Send(ctx, "POST", c.RepoPath(owner, slug)+"/forks", body, &repo); err != nil {
		return nil, err
	}
	return &repo, nil
}

// ListBranches lists branches in a repository.
func (c *Client) ListBranches(ctx context.Context, owner, slug string, limit int) ([]Branch, error) {
	path := c.RepoPath(owner, slug) + "/refs/branches"
	if !c.IsCloud() {
		path = c.RepoPath(owner, slug) + "/branches"
	}
	return GetAll[Branch](ctx, c, path, nil, limit)
}

// ListProjects lists projects in a workspace (Cloud) or accessible projects
// (Data Center).
func (c *Client) ListProjects(ctx context.Context, workspace string, limit int) ([]Project, error) {
	var path string
	if c.IsCloud() {
		if workspace == "" {
			return nil, fmt.Errorf("workspace is required")
		}
		path = "/workspaces/" + url.PathEscape(workspace) + "/projects"
	} else {
		path = "/projects"
	}
	return GetAll[Project](ctx, c, path, nil, limit)
}

// GetCurrentUser returns the authenticated identity.
func (c *Client) GetCurrentUser(ctx context.Context) (*Account, error) {
	if !c.IsCloud() {
		return nil, unsupported("current user lookup")
	}
	var user Account
	if err := c.Get(ctx, "/user", nil, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// RepositoryFromInput parses "owner/slug" or "host/owner/slug".
func RepositoryFromInput(input, defaultHost string) (owner, slug, host string, err error) {
	input = strings.Trim(strings.TrimSpace(input), "/")
	if input == "" {
		return "", "", "", fmt.Errorf("empty repository")
	}
	parts := strings.Split(input, "/")
	switch len(parts) {
	case 2:
		return parts[0], parts[1], defaultHost, nil
	case 3:
		return parts[1], parts[2], strings.ToLower(parts[0]), nil
	default:
		return "", "", "", fmt.Errorf("invalid repository %q: expected OWNER/REPO", input)
	}
}
