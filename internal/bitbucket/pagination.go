package bitbucket

import (
	"context"
	"net/url"
	"strconv"
)

// Page is a paginated Bitbucket collection. Cloud uses next cursors; Data
// Center uses start/nextPageStart offsets.
type Page[T any] struct {
	Values        []T    `json:"values"`
	Next          string `json:"next"`
	Page          int    `json:"page"`
	Pagelen       int    `json:"pagelen"`
	Size          int    `json:"size"`
	IsLastPage    bool   `json:"isLastPage"`
	NextPageStart *int   `json:"nextPageStart"`
}

// GetAll follows pagination until limit items are collected (0 means all) or
// the collection ends.
func GetAll[T any](ctx context.Context, c *Client, path string, q url.Values, limit int) ([]T, error) {
	var out []T
	nextURL := ""
	start := -1

	for pageNo := 0; pageNo < 1000; pageNo++ {
		query := url.Values{}
		for key, values := range q {
			query[key] = append([]string(nil), values...)
		}
		requestPath := path
		if nextURL != "" {
			requestPath = nextURL
			query = nil
		} else if start >= 0 {
			query.Set("start", strconv.Itoa(start))
		}

		if limit > 0 {
			remaining := limit - len(out)
			if remaining <= 0 {
				break
			}
			if remaining > 100 {
				remaining = 100
			}
			if c.IsCloud() {
				query.Set("pagelen", strconv.Itoa(remaining))
			} else {
				query.Set("limit", strconv.Itoa(remaining))
			}
		} else if pageNo == 0 {
			if c.IsCloud() {
				query.Set("pagelen", "100")
			} else {
				query.Set("limit", "100")
			}
		}

		var page Page[T]
		if err := c.Get(ctx, requestPath, query, &page); err != nil {
			return nil, err
		}
		out = append(out, page.Values...)
		if limit > 0 && len(out) >= limit {
			return out[:limit], nil
		}
		if page.Next != "" {
			nextURL = page.Next
			start = -1
			continue
		}
		if !page.IsLastPage && page.NextPageStart != nil {
			nextURL = ""
			start = *page.NextPageStart
			continue
		}
		break
	}
	return out, nil
}

// ListWorkspaces lists workspaces the authenticated user belongs to.
func (c *Client) ListWorkspaces(ctx context.Context, limit int) ([]Workspace, error) {
	if !c.IsCloud() {
		return nil, unsupported("workspace list")
	}
	return GetAll[Workspace](ctx, c, "/workspaces", nil, limit)
}
