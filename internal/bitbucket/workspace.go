package bitbucket

import (
	"context"
	"net/url"
)

// GetWorkspace fetches one workspace.
func (c *Client) GetWorkspace(ctx context.Context, slug string) (*Workspace, error) {
	if !c.IsCloud() {
		return nil, unsupported("workspace view")
	}
	var workspace Workspace
	if err := c.Get(ctx, "/workspaces/"+url.PathEscape(slug), nil, &workspace); err != nil {
		return nil, err
	}
	return &workspace, nil
}

// ListWorkspaceMembers lists the members of a workspace.
func (c *Client) ListWorkspaceMembers(ctx context.Context, slug string, limit int) ([]WorkspaceMember, error) {
	if !c.IsCloud() {
		return nil, unsupported("workspace members")
	}
	return GetAll[WorkspaceMember](ctx, c, "/workspaces/"+url.PathEscape(slug)+"/members", nil, limit)
}
