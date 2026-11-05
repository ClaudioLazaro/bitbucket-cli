package bitbucket

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// FlexString accepts either a JSON string or a JSON number. Bitbucket uses
// both shapes for snippet identifiers across API versions.
type FlexString string

// UnmarshalJSON implements json.Unmarshaler.
func (f *FlexString) UnmarshalJSON(b []byte) error {
	raw := strings.TrimSpace(string(b))
	if raw == "null" {
		*f = ""
		return nil
	}
	if strings.HasPrefix(raw, `"`) {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*f = FlexString(s)
		return nil
	}
	*f = FlexString(strings.Trim(raw, `"`))
	return nil
}

// String returns the raw value.
func (f FlexString) String() string { return string(f) }

// Link is one hypermedia reference.
type Link struct {
	Href string `json:"href"`
	Name string `json:"name"`
}

// Links groups the hypermedia references Bitbucket returns.
type Links struct {
	Self      Link   `json:"self"`
	HTML      Link   `json:"html"`
	Avatar    Link   `json:"avatar"`
	Clone     []Link `json:"clone"`
	Commits   Link   `json:"commits"`
	Diff      Link   `json:"diff"`
	Patch     Link   `json:"patch"`
	Approve   Link   `json:"approve"`
	Decline   Link   `json:"decline"`
	Merge     Link   `json:"merge"`
	Comments  Link   `json:"comments"`
	Statuses  Link   `json:"statuses"`
	Downloads Link   `json:"downloads"`
	Source    Link   `json:"source"`
	Hooks     Link   `json:"hooks"`
}

// Account is a Bitbucket user or access token identity.
type Account struct {
	Type        string `json:"type"`
	UUID        string `json:"uuid"`
	Username    string `json:"username"`
	Nickname    string `json:"nickname"`
	DisplayName string `json:"display_name"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Links       Links  `json:"links"`
}

// Label returns the best available human name for the account.
func (a *Account) Label() string {
	if a == nil {
		return ""
	}
	switch {
	case a.DisplayName != "":
		return a.DisplayName
	case a.Nickname != "":
		return a.Nickname
	case a.Username != "":
		return a.Username
	case a.Name != "":
		return a.Name
	case a.Slug != "":
		return a.Slug
	}
	return a.UUID
}

// Commit is a git commit as returned by Bitbucket.
type Commit struct {
	Type    string    `json:"type"`
	Hash    string    `json:"hash"`
	Message string    `json:"message"`
	Date    time.Time `json:"date"`
	Author  *Account  `json:"author"`
	Links   Links     `json:"links"`
}

// Branch is a git branch reference.
type Branch struct {
	Name                 string   `json:"name"`
	DisplayID            string   `json:"displayId"`
	ID                   string   `json:"id"`
	IsDefault            bool     `json:"isDefault"`
	DefaultMergeStrategy string   `json:"default_merge_strategy"`
	MergeStrategies      []string `json:"merge_strategies"`
	Target               *Commit  `json:"target"`
	LatestCommit         string   `json:"latestCommit"`
}

// Label returns the short branch name for Cloud and Data Center shapes.
func (b *Branch) Label() string {
	switch {
	case b.Name != "":
		return b.Name
	case b.DisplayID != "":
		return b.DisplayID
	case strings.HasPrefix(b.ID, "refs/heads/"):
		return strings.TrimPrefix(b.ID, "refs/heads/")
	}
	return b.ID
}

// Project is a Bitbucket Cloud project.
type Project struct {
	Type        string `json:"type"`
	UUID        string `json:"uuid"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsPrivate   bool   `json:"is_private"`
	Links       Links  `json:"links"`
}

// Repository is a Bitbucket repository.
type Repository struct {
	Type        string      `json:"type"`
	UUID        string      `json:"uuid"`
	FullName    string      `json:"full_name"`
	Name        string      `json:"name"`
	Slug        string      `json:"slug"`
	IsPrivate   bool        `json:"is_private"`
	SCM         string      `json:"scm"`
	Description string      `json:"description"`
	Language    string      `json:"language"`
	ForkPolicy  string      `json:"fork_policy"`
	Owner       *Account    `json:"owner"`
	Project     *Project    `json:"project"`
	MainBranch  *Branch     `json:"mainbranch"`
	Parent      *Repository `json:"parent"`
	Links       Links       `json:"links"`
	CreatedOn   time.Time   `json:"created_on"`
	UpdatedOn   time.Time   `json:"updated_on"`
}

// SlugOrName returns the repository slug, deriving one when the API omits it.
func (r *Repository) SlugOrName() string {
	if r.Slug != "" {
		return r.Slug
	}
	if i := strings.LastIndex(r.FullName, "/"); i >= 0 {
		return r.FullName[i+1:]
	}
	return r.Name
}

// OwnerName returns the workspace slug or Data Center project key.
func (r *Repository) OwnerName() string {
	if r.Project != nil && r.Project.Key != "" {
		return r.Project.Key
	}
	if r.Owner != nil {
		if r.Owner.Nickname != "" {
			return r.Owner.Nickname
		}
		if r.Owner.Username != "" {
			return r.Owner.Username
		}
	}
	if i := strings.LastIndex(r.FullName, "/"); i > 0 {
		return r.FullName[:i]
	}
	return ""
}

// NameWithOwner returns owner/slug.
func (r *Repository) NameWithOwner() string {
	if r.FullName != "" {
		return r.FullName
	}
	if owner := r.OwnerName(); owner != "" {
		return owner + "/" + r.SlugOrName()
	}
	return r.SlugOrName()
}

// CloneURL returns the clone URL for the requested protocol ("https" or "ssh").
func (r *Repository) CloneURL(protocol string) string {
	for _, link := range r.Links.Clone {
		if strings.EqualFold(link.Name, protocol) && link.Href != "" {
			return link.Href
		}
	}
	if r.Links.HTML.Href != "" && protocol == "https" {
		return strings.TrimSuffix(r.Links.HTML.Href, "/") + ".git"
	}
	return ""
}

// HTMLURL returns the browser URL for the repository.
func (r *Repository) HTMLURL() string { return r.Links.HTML.Href }

// BranchRef points to a branch and commit in a pull request.
type BranchRef struct {
	Branch     *Branch     `json:"branch"`
	Commit     *Commit     `json:"commit"`
	Repository *Repository `json:"repository"`
}

// Participant records a reviewer's state on a pull request.
type Participant struct {
	Type     string   `json:"type"`
	User     *Account `json:"user"`
	Role     string   `json:"role"`
	Approved bool     `json:"approved"`
	State    string   `json:"state"`
}

// PullRequest is a Bitbucket Cloud pull request.
type PullRequest struct {
	Type              string         `json:"type"`
	ID                int            `json:"id"`
	Title             string         `json:"title"`
	Description       string         `json:"description"`
	Summary           string         `json:"summary"`
	State             string         `json:"state"`
	Author            *Account       `json:"author"`
	Source            *BranchRef     `json:"source"`
	Destination       *BranchRef     `json:"destination"`
	MergeCommit       *Commit        `json:"merge_commit"`
	CommentCount      int            `json:"comment_count"`
	TaskCount         int            `json:"task_count"`
	CloseSourceBranch bool           `json:"close_source_branch"`
	ClosedBy          *Account       `json:"closed_by"`
	Reason            string         `json:"reason"`
	Draft             bool           `json:"draft"`
	Queued            bool           `json:"queued"`
	Reviewers         []*Account     `json:"reviewers"`
	Participants      []*Participant `json:"participants"`
	CreatedOn         time.Time      `json:"created_on"`
	UpdatedOn         time.Time      `json:"updated_on"`
	Links             Links          `json:"links"`
}

// Body returns the pull request description from either API field.
func (p *PullRequest) Body() string {
	if p.Description != "" {
		return p.Description
	}
	return p.Summary
}

// SourceBranch returns the source branch name, if present.
func (p *PullRequest) SourceBranch() string {
	if p.Source != nil && p.Source.Branch != nil {
		return p.Source.Branch.Name
	}
	return ""
}

// DestinationBranch returns the destination branch name, if present.
func (p *PullRequest) DestinationBranch() string {
	if p.Destination != nil && p.Destination.Branch != nil {
		return p.Destination.Branch.Name
	}
	return ""
}

// CommentContent is the body of a pull request comment.
type CommentContent struct {
	Raw    string `json:"raw"`
	Markup string `json:"markup"`
	HTML   string `json:"html"`
}

// InlineComment locates a comment in a file.
type InlineComment struct {
	Path string `json:"path"`
	From *int   `json:"from"`
	To   *int   `json:"to"`
}

// Comment is a pull request comment.
type Comment struct {
	Type      string         `json:"type"`
	ID        int            `json:"id"`
	Content   CommentContent `json:"content"`
	User      *Account       `json:"user"`
	CreatedOn time.Time      `json:"created_on"`
	UpdatedOn time.Time      `json:"updated_on"`
	Deleted   bool           `json:"deleted"`
	Parent    *Comment       `json:"parent"`
	Inline    *InlineComment `json:"inline"`
	Links     Links          `json:"links"`
}

// Status is a commit build status.
type Status struct {
	Type        string    `json:"type"`
	Key         string    `json:"key"`
	Refname     string    `json:"refname"`
	URL         string    `json:"url"`
	State       string    `json:"state"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedOn   time.Time `json:"created_on"`
	UpdatedOn   time.Time `json:"updated_on"`
	Links       Links     `json:"links"`
}

// PipelineSelector identifies which pipeline definition runs.
type PipelineSelector struct {
	Type    string `json:"type"`
	Pattern string `json:"pattern"`
}

// PipelineTarget describes what a pipeline ran against.
type PipelineTarget struct {
	Type     string            `json:"type"`
	RefType  string            `json:"ref_type"`
	RefName  string            `json:"ref_name"`
	Commit   *Commit           `json:"commit"`
	Selector *PipelineSelector `json:"selector"`
}

// PipelineStateResult is the outcome of a pipeline or step.
type PipelineStateResult struct {
	Name string `json:"name"`
}

// PipelineState is the current state of a pipeline or step.
type PipelineState struct {
	Type   string               `json:"type"`
	Name   string               `json:"name"`
	Result *PipelineStateResult `json:"result"`
}

// PipelineTrigger describes how a pipeline started.
type PipelineTrigger struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

// Pipeline is one Bitbucket Pipelines run.
type Pipeline struct {
	Type             string           `json:"type"`
	UUID             string           `json:"uuid"`
	BuildNumber      int              `json:"build_number"`
	Creator          *Account         `json:"creator"`
	Repository       *Repository      `json:"repository"`
	Target           *PipelineTarget  `json:"target"`
	Trigger          *PipelineTrigger `json:"trigger"`
	State            *PipelineState   `json:"state"`
	Variables        []Variable       `json:"variables"`
	CreatedOn        time.Time        `json:"created_on"`
	CompletedOn      *time.Time       `json:"completed_on"`
	BuildSecondsUsed int              `json:"build_seconds_used"`
	Links            Links            `json:"links"`
}

// Status returns a compact pipeline status: PENDING, IN_PROGRESS, SUCCESSFUL,
// FAILED, STOPPED, EXPIRED, or ERROR.
func (p *Pipeline) Status() string {
	if p.State == nil {
		return ""
	}
	if p.State.Result != nil && p.State.Result.Name != "" {
		return p.State.Result.Name
	}
	return p.State.Name
}

// Step is one step of a pipeline.
type Step struct {
	Type              string         `json:"type"`
	UUID              string         `json:"uuid"`
	Name              string         `json:"name"`
	State             *PipelineState `json:"state"`
	StartedOn         *time.Time     `json:"started_on"`
	CompletedOn       *time.Time     `json:"completed_on"`
	DurationInSeconds int            `json:"duration_in_seconds"`
	Links             Links          `json:"links"`
}

// Status returns a compact step status.
func (s *Step) Status() string {
	if s.State == nil {
		return ""
	}
	if s.State.Result != nil && s.State.Result.Name != "" {
		return s.State.Result.Name
	}
	return s.State.Name
}

// Variable is a pipeline configuration variable.
type Variable struct {
	Type    string `json:"type"`
	UUID    string `json:"uuid"`
	Key     string `json:"key"`
	Value   string `json:"value"`
	Secured bool   `json:"secured"`
}

// Workspace is a Bitbucket Cloud workspace.
type Workspace struct {
	Type       string `json:"type"`
	UUID       string `json:"uuid"`
	Slug       string `json:"slug"`
	Name       string `json:"name"`
	IsPrivate  bool   `json:"is_private"`
	IsPersonal bool   `json:"is_personal"`
	Links      Links  `json:"links"`
}

// WorkspaceMember is a membership entry returned by the workspace members API.
type WorkspaceMember struct {
	Type      string     `json:"type"`
	User      *Account   `json:"user"`
	Workspace *Workspace `json:"workspace"`
}

// SnippetFile is one file inside a snippet.
type SnippetFile struct {
	Content string `json:"content"`
	Size    int    `json:"size"`
	Links   Links  `json:"links"`
}

// Snippet is a Bitbucket Cloud snippet.
type Snippet struct {
	Type      string                 `json:"type"`
	ID        FlexString             `json:"id"`
	Title     string                 `json:"title"`
	SCM       string                 `json:"scm"`
	IsPrivate bool                   `json:"is_private"`
	Owner     *Account               `json:"owner"`
	Creator   *Account               `json:"creator"`
	CreatedOn time.Time              `json:"created_on"`
	UpdatedOn time.Time              `json:"updated_on"`
	Files     map[string]SnippetFile `json:"files"`
	Links     Links                  `json:"links"`
}

// Webhook is a repository webhook subscription.
type Webhook struct {
	Type        string    `json:"type"`
	UUID        string    `json:"uuid"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	Active      bool      `json:"active"`
	Events      []string  `json:"events"`
	Secret      string    `json:"secret"`
	CreatedAt   time.Time `json:"created_at"`
	Links       Links     `json:"links"`
}

// Group is a Bitbucket user group reference.
type Group struct {
	Type  string `json:"type"`
	Slug  string `json:"slug"`
	Name  string `json:"name"`
	Links Links  `json:"links"`
}

// BranchRestriction is a branch permission rule.
type BranchRestriction struct {
	Type            string     `json:"type"`
	ID              int        `json:"id"`
	Kind            string     `json:"kind"`
	Pattern         string     `json:"pattern"`
	BranchMatchKind string     `json:"branch_match_kind"`
	Value           *int       `json:"value"`
	Users           []*Account `json:"users"`
	Groups          []*Group   `json:"groups"`
	Links           Links      `json:"links"`
}

// SSHKey is an account SSH key.
type SSHKey struct {
	Type        string    `json:"type"`
	ID          int       `json:"id"`
	Label       string    `json:"label"`
	Key         string    `json:"key"`
	Fingerprint string    `json:"fingerprint"`
	CreatedOn   time.Time `json:"created_on"`
	LastUsed    time.Time `json:"last_used"`
	Links       Links     `json:"links"`
}

// GPGKey is an account GPG key.
type GPGKey struct {
	Type        string    `json:"type"`
	Fingerprint string    `json:"fingerprint"`
	KeyID       string    `json:"key_id"`
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Emails      []string  `json:"emails"`
	CreatedOn   time.Time `json:"created_on"`
	ExpiresOn   time.Time `json:"expires_on"`
	Links       Links     `json:"links"`
}

// Download is one file in the repository downloads directory.
type Download struct {
	Name  string `json:"name"`
	Size  int64  `json:"size"`
	Links Links  `json:"links"`
}

// SearchResult is one code search hit.
type SearchResult struct {
	Type string `json:"type"`
	File struct {
		Path  string `json:"path"`
		Type  string `json:"type"`
		Links Links  `json:"links"`
	} `json:"file"`
	ContentMatches []struct {
		Lines []struct {
			Line       string `json:"line"`
			LineNumber int    `json:"line_number"`
			Segments   []struct {
				Text string `json:"text"`
			} `json:"segments"`
		} `json:"lines"`
	} `json:"content_matches"`
}

// Line returns the first matching line, trimmed.
func (r SearchResult) Line() string {
	if len(r.ContentMatches) == 0 || len(r.ContentMatches[0].Lines) == 0 {
		return ""
	}
	return strings.TrimSpace(r.ContentMatches[0].Lines[0].Line)
}

// LineNumber returns the first matching line number.
func (r SearchResult) LineNumber() int {
	if len(r.ContentMatches) == 0 || len(r.ContentMatches[0].Lines) == 0 {
		return 0
	}
	return r.ContentMatches[0].Lines[0].LineNumber
}

// Ref builds a repository reference string.
func Ref(owner, slug string) string {
	return fmt.Sprintf("%s/%s", owner, slug)
}
