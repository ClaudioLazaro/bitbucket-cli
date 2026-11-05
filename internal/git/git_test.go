package git

import (
	"context"
	"os"
	"os/exec"
	"testing"
)

func TestParseRemoteURL(t *testing.T) {
	tests := []struct {
		raw   string
		host  string
		owner string
		slug  string
	}{
		{"https://bitbucket.org/acme/widgets.git", "bitbucket.org", "acme", "widgets"},
		{"https://bitbucket.org/acme/widgets", "bitbucket.org", "acme", "widgets"},
		{"git@bitbucket.org:acme/widgets.git", "bitbucket.org", "acme", "widgets"},
		{"ssh://git@bitbucket.org/acme/widgets.git", "bitbucket.org", "acme", "widgets"},
		{"https://dc.example.com/scm/PROJ/widgets.git", "dc.example.com", "PROJ", "widgets"},
		{"ssh://git@dc.example.com:7999/PROJ/widgets.git", "dc.example.com:7999", "PROJ", "widgets"},
		{"https://dc.example.com/projects/PROJ/repos/widgets", "dc.example.com", "PROJ", "widgets"},
	}
	for _, test := range tests {
		t.Run(test.raw, func(t *testing.T) {
			host, owner, slug, ok := ParseRemoteURL(test.raw)
			if !ok {
				t.Fatalf("not parsed")
			}
			if host != test.host || owner != test.owner || slug != test.slug {
				t.Fatalf("got (%q, %q, %q), want (%q, %q, %q)", host, owner, slug, test.host, test.owner, test.slug)
			}
		})
	}
}

func TestRepoFromRemotes(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "remote", "add", "origin", "git@bitbucket.org:acme/widgets.git")

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldDir)

	ref, err := RepoFromRemotes(context.Background(), "bitbucket.org")
	if err != nil {
		t.Fatalf("RepoFromRemotes: %v", err)
	}
	if ref.Owner != "acme" || ref.Slug != "widgets" {
		t.Fatalf("ref = %+v", ref)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
