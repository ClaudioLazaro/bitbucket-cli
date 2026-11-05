// Package git wraps the git commands bk needs for repository discovery,
// cloning, and pull request checkouts.
package git

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os/exec"
	"strings"

	"github.com/ClaudioLazaro/bitbucket-cli/internal/bitbucket"
)

// Remote is one configured git remote.
type Remote struct {
	Name string
	URL  string
}

// Run executes git with args and returns its trimmed standard output.
func Run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return strings.TrimSpace(string(out)), nil
}

// IsRepository reports whether the current directory is inside a git work tree.
func IsRepository(ctx context.Context) bool {
	_, err := Run(ctx, "rev-parse", "--git-dir")
	return err == nil
}

// Remotes lists configured remotes, de-duplicated by name and URL.
func Remotes(ctx context.Context) ([]Remote, error) {
	out, err := Run(ctx, "remote", "-v")
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var remotes []Remote
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		key := fields[0] + "\x00" + fields[1]
		if seen[key] {
			continue
		}
		seen[key] = true
		remotes = append(remotes, Remote{Name: fields[0], URL: fields[1]})
	}
	return remotes, nil
}

// CurrentBranch returns the checked out branch, failing on a detached HEAD.
func CurrentBranch(ctx context.Context) (string, error) {
	out, err := Run(ctx, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	if out == "HEAD" {
		return "", fmt.Errorf("HEAD is detached; pass --head or check out a branch")
	}
	return out, nil
}

// LastCommitSubject returns the subject of HEAD.
func LastCommitSubject(ctx context.Context) (string, error) {
	return Run(ctx, "log", "-1", "--pretty=%s")
}

// LastCommitBody returns the body of HEAD.
func LastCommitBody(ctx context.Context) (string, error) {
	return Run(ctx, "log", "-1", "--pretty=%b")
}

// BranchExists reports whether a local branch exists.
func BranchExists(ctx context.Context, name string) bool {
	_, err := Run(ctx, "show-ref", "--verify", "--quiet", "refs/heads/"+name)
	return err == nil
}

// Fetch fetches refspec from remote.
func Fetch(ctx context.Context, remote, refspec string) error {
	_, err := Run(ctx, "fetch", remote, refspec)
	return err
}

// Checkout switches to an existing branch.
func Checkout(ctx context.Context, branch string) error {
	_, err := Run(ctx, "checkout", branch)
	return err
}

// CloneClone runs git clone, streaming progress to stdout and stderr.
func Clone(ctx context.Context, cloneURL, dir string, stdout, stderr io.Writer) error {
	args := []string{"clone", cloneURL}
	if dir != "" {
		args = append(args, dir)
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

// ParseRemoteURL extracts a Bitbucket host, owner, and repository slug from a
// git remote URL. Both Cloud and Data Center URL shapes are accepted.
func ParseRemoteURL(raw string) (host, owner, slug string, ok bool) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimSuffix(raw, "/")
	if raw == "" {
		return "", "", "", false
	}

	var path string
	if strings.HasPrefix(raw, "git@") || (!strings.Contains(raw, "://") && strings.Contains(raw, ":")) {
		parts := strings.SplitN(raw, ":", 2)
		if len(parts) != 2 {
			return "", "", "", false
		}
		host = strings.TrimPrefix(parts[0], "git@")
		path = parts[1]
	} else {
		u, err := url.Parse(raw)
		if err != nil || u.Host == "" {
			return "", "", "", false
		}
		host = u.Host
		path = strings.TrimPrefix(u.Path, "/")
	}
	path = strings.TrimSuffix(path, ".git")
	parts := strings.Split(path, "/")
	switch {
	case len(parts) >= 4 && parts[0] == "projects" && parts[2] == "repos":
		owner, slug = parts[1], parts[3]
	case len(parts) >= 3 && parts[0] == "scm":
		owner, slug = parts[1], parts[2]
	case len(parts) >= 2:
		owner, slug = parts[len(parts)-2], parts[len(parts)-1]
	default:
		return "", "", "", false
	}
	return host, owner, slug, true
}

// RepoFromRemotes discovers the repository from git remotes that point at the
// configured host, preferring origin.
func RepoFromRemotes(ctx context.Context, hostname string) (bitbucket.RepoRef, error) {
	remotes, err := Remotes(ctx)
	if err != nil {
		return bitbucket.RepoRef{}, fmt.Errorf("not a git repository and no repository specified; use -R OWNER/REPO")
	}
	ordered := make([]Remote, 0, len(remotes))
	for _, remote := range remotes {
		if remote.Name == "origin" {
			ordered = append(ordered, remote)
		}
	}
	for _, remote := range remotes {
		if remote.Name != "origin" {
			ordered = append(ordered, remote)
		}
	}
	var candidates []string
	for _, remote := range ordered {
		host, owner, slug, ok := ParseRemoteURL(remote.URL)
		if !ok {
			continue
		}
		candidates = append(candidates, remote.URL)
		if hostMatches(host, hostname) {
			return bitbucket.RepoRef{Host: hostname, Owner: owner, Slug: slug}, nil
		}
	}
	if len(candidates) > 0 {
		return bitbucket.RepoRef{}, fmt.Errorf("no git remote points at %s; use -R OWNER/REPO", hostname)
	}
	return bitbucket.RepoRef{}, fmt.Errorf("no Bitbucket git remote found; use -R OWNER/REPO")
}

func hostMatches(remoteHost, configured string) bool {
	remoteHost = strings.ToLower(remoteHost)
	configured = strings.ToLower(configured)
	if remoteHost == configured {
		return true
	}
	// Ignore default ports when comparing.
	remoteHost = strings.TrimSuffix(remoteHost, ":443")
	remoteHost = strings.TrimSuffix(remoteHost, ":22")
	configured = strings.TrimSuffix(configured, ":443")
	configured = strings.TrimSuffix(configured, ":22")
	return remoteHost == configured
}
