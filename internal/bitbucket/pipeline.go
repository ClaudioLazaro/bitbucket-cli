package bitbucket

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

// RunPipelineInput selects the pipeline to run and its variables.
type RunPipelineInput struct {
	Branch        string
	CustomPattern string
	Variables     map[string]string
}

var pipelineUUIDPattern = regexp.MustCompile(`^\{?[0-9a-fA-F-]{8,}\}?$`)

func normalizePipelineUUID(uuid string) (string, error) {
	uuid = strings.TrimSpace(uuid)
	if uuid == "" {
		return "", fmt.Errorf("pipeline UUID is required")
	}
	if !pipelineUUIDPattern.MatchString(uuid) {
		return "", fmt.Errorf("invalid pipeline UUID %q", uuid)
	}
	if !strings.HasPrefix(uuid, "{") {
		uuid = "{" + uuid + "}"
	}
	return uuid, nil
}

func (c *Client) pipelinesPath(owner, slug string) string {
	return c.RepoPath(owner, slug) + "/pipelines"
}

// ListPipelines lists pipeline runs.
func (c *Client) ListPipelines(ctx context.Context, owner, slug string, q url.Values, limit int) ([]Pipeline, error) {
	if !c.IsCloud() {
		return nil, unsupported("pipeline list")
	}
	return GetAll[Pipeline](ctx, c, c.pipelinesPath(owner, slug), q, limit)
}

// GetPipeline fetches one pipeline run.
func (c *Client) GetPipeline(ctx context.Context, owner, slug, uuid string) (*Pipeline, error) {
	if !c.IsCloud() {
		return nil, unsupported("pipeline view")
	}
	uuid, err := normalizePipelineUUID(uuid)
	if err != nil {
		return nil, err
	}
	var pipeline Pipeline
	if err := c.Get(ctx, c.pipelinesPath(owner, slug)+"/"+url.PathEscape(uuid), nil, &pipeline); err != nil {
		return nil, err
	}
	return &pipeline, nil
}

// RunPipeline starts a pipeline for a branch or a custom pipeline pattern.
func (c *Client) RunPipeline(ctx context.Context, owner, slug string, in RunPipelineInput) (*Pipeline, error) {
	if !c.IsCloud() {
		return nil, unsupported("pipeline run")
	}
	if in.Branch == "" {
		return nil, fmt.Errorf("branch is required")
	}
	target := map[string]any{
		"type":     "pipeline_ref_target",
		"ref_type": "branch",
		"ref_name": in.Branch,
		"selector": map[string]string{
			"type":    "branches",
			"pattern": in.Branch,
		},
	}
	if in.CustomPattern != "" {
		target["selector"] = map[string]string{
			"type":    "custom",
			"pattern": in.CustomPattern,
		}
	}
	body := map[string]any{"target": target}
	if len(in.Variables) > 0 {
		keys := make([]string, 0, len(in.Variables))
		for key := range in.Variables {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		vars := make([]map[string]any, 0, len(keys))
		for _, key := range keys {
			vars = append(vars, map[string]any{
				"key":     key,
				"value":   in.Variables[key],
				"secured": false,
			})
		}
		body["variables"] = vars
	}
	var pipeline Pipeline
	if err := c.Send(ctx, "POST", c.pipelinesPath(owner, slug), body, &pipeline); err != nil {
		return nil, err
	}
	return &pipeline, nil
}

// StopPipeline cancels a running pipeline.
func (c *Client) StopPipeline(ctx context.Context, owner, slug, uuid string) error {
	if !c.IsCloud() {
		return unsupported("pipeline stop")
	}
	uuid, err := normalizePipelineUUID(uuid)
	if err != nil {
		return err
	}
	return c.Send(ctx, "POST", c.pipelinesPath(owner, slug)+"/"+url.PathEscape(uuid)+"/stopPipeline", nil, nil)
}

// ListSteps lists the steps of a pipeline.
func (c *Client) ListSteps(ctx context.Context, owner, slug, uuid string) ([]Step, error) {
	if !c.IsCloud() {
		return nil, unsupported("pipeline steps")
	}
	uuid, err := normalizePipelineUUID(uuid)
	if err != nil {
		return nil, err
	}
	return GetAll[Step](ctx, c, c.pipelinesPath(owner, slug)+"/"+url.PathEscape(uuid)+"/steps", nil, 100)
}

// GetStepLog returns the raw log of one pipeline step.
func (c *Client) GetStepLog(ctx context.Context, owner, slug, pipelineUUID, stepUUID string) ([]byte, error) {
	if !c.IsCloud() {
		return nil, unsupported("pipeline logs")
	}
	pipelineUUID, err := normalizePipelineUUID(pipelineUUID)
	if err != nil {
		return nil, err
	}
	path := c.pipelinesPath(owner, slug) + "/" + url.PathEscape(pipelineUUID) + "/steps/" + url.PathEscape(stepUUID) + "/log"
	return c.GetRaw(ctx, path, nil, "text/plain")
}

// StreamStepLog writes the raw log of one step to w.
func (c *Client) StreamStepLog(ctx context.Context, w io.Writer, owner, slug, pipelineUUID, stepUUID string) error {
	data, err := c.GetStepLog(ctx, owner, slug, pipelineUUID, stepUUID)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// ListPipelineVariables lists repository-level pipeline variables.
func (c *Client) ListPipelineVariables(ctx context.Context, owner, slug string) ([]Variable, error) {
	if !c.IsCloud() {
		return nil, unsupported("pipeline variables")
	}
	return GetAll[Variable](ctx, c, c.RepoPath(owner, slug)+"/pipelines_config/variables", nil, 0)
}

// SetPipelineVariable creates or updates a repository-level pipeline variable.
func (c *Client) SetPipelineVariable(ctx context.Context, owner, slug, key, value string, secured bool) (*Variable, error) {
	if !c.IsCloud() {
		return nil, unsupported("pipeline variables")
	}
	existing, err := c.ListPipelineVariables(ctx, owner, slug)
	if err != nil {
		return nil, err
	}
	body := map[string]any{"key": key, "value": value, "secured": secured}
	for _, variable := range existing {
		if variable.Key != key {
			continue
		}
		var updated Variable
		path := c.RepoPath(owner, slug) + "/pipelines_config/variables/" + url.PathEscape(variable.UUID)
		if err := c.Send(ctx, "PUT", path, body, &updated); err != nil {
			return nil, err
		}
		return &updated, nil
	}
	var created Variable
	if err := c.Send(ctx, "POST", c.RepoPath(owner, slug)+"/pipelines_config/variables", body, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

// DeletePipelineVariable deletes a repository-level pipeline variable by UUID.
func (c *Client) DeletePipelineVariable(ctx context.Context, owner, slug, uuid string) error {
	if !c.IsCloud() {
		return unsupported("pipeline variables")
	}
	if uuid == "" {
		return fmt.Errorf("variable UUID is required")
	}
	return c.Send(ctx, "DELETE", c.RepoPath(owner, slug)+"/pipelines_config/variables/"+url.PathEscape(uuid), nil, nil)
}
