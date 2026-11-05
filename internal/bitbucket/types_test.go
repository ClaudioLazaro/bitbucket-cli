package bitbucket

import (
	"encoding/json"
	"testing"
)

func TestTextContentAcceptsObjectAndString(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"object", `{"raw":"hello","markup":"markdown","html":"<p>hello</p>"}`, "hello"},
		{"string", `"hello"`, "hello"},
		{"null", `null`, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var content TextContent
			if err := json.Unmarshal([]byte(test.raw), &content); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			if content.Raw != test.want {
				t.Fatalf("raw = %q, want %q", content.Raw, test.want)
			}
		})
	}
}

func TestPullRequestSummaryObject(t *testing.T) {
	payload := `{
		"id": 7,
		"title": "Add widgets",
		"state": "OPEN",
		"summary": {"raw": "summary body", "markup": "markdown", "html": "<p>summary body</p>"}
	}`
	var pr PullRequest
	if err := json.Unmarshal([]byte(payload), &pr); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if pr.Body() != "summary body" {
		t.Fatalf("body = %q", pr.Body())
	}
	if pr.Summary.Markup != "markdown" || pr.Summary.HTML == "" {
		t.Fatalf("summary = %+v", pr.Summary)
	}

	encoded, err := json.Marshal(pr)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var generic map[string]any
	if err := json.Unmarshal(encoded, &generic); err != nil {
		t.Fatalf("Unmarshal encoded: %v", err)
	}
	if generic["summary"] != "summary body" {
		t.Fatalf("summary JSON = %#v, want a string", generic["summary"])
	}
}

func TestPullRequestBodyPrefersDescription(t *testing.T) {
	pr := PullRequest{
		Description: TextContent{Raw: "description body"},
		Summary:     TextContent{Raw: "summary body"},
	}
	if pr.Body() != "description body" {
		t.Fatalf("body = %q", pr.Body())
	}
}
