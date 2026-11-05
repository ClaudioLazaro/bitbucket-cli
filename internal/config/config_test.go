package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ClaudioLazaro/bitbucket-cli/internal/config"
)

func TestNormalizeHost(t *testing.T) {
	tests := []struct {
		input string
		host  string
		kind  string
		api   string
	}{
		{"bitbucket.org", "bitbucket.org", config.KindCloud, config.CloudAPIURL},
		{"https://bitbucket.org/", "bitbucket.org", config.KindCloud, config.CloudAPIURL},
		{"api.bitbucket.org", "bitbucket.org", config.KindCloud, config.CloudAPIURL},
		{"https://bitbucket.example.com", "bitbucket.example.com", config.KindDataCenter, "https://bitbucket.example.com/rest/api/1.0"},
		{"bitbucket.example.com/bitbucket", "bitbucket.example.com", config.KindDataCenter, "https://bitbucket.example.com/bitbucket/rest/api/1.0"},
		{"https://bitbucket.example.com/rest/api/1.0", "bitbucket.example.com", config.KindDataCenter, "https://bitbucket.example.com/rest/api/1.0"},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			host, kind, api, err := config.NormalizeHost(test.input)
			if err != nil {
				t.Fatalf("NormalizeHost: %v", err)
			}
			if host != test.host || kind != test.kind || api != test.api {
				t.Fatalf("got (%q, %q, %q), want (%q, %q, %q)", host, kind, api, test.host, test.kind, test.api)
			}
		})
	}
}

func TestNormalizeHostRejectsEmpty(t *testing.T) {
	if _, _, _, err := config.NormalizeHost("   "); err == nil {
		t.Fatal("expected an error for an empty host")
	}
}

func TestResolveEnvironmentOverrides(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BK_HOST", "https://bitbucket.example.com/bitbucket")
	t.Setenv("BK_TOKEN", "secret")
	t.Setenv("BK_USERNAME", "me@example.com")
	t.Setenv("BK_AUTH_METHOD", "basic")
	t.Setenv("BK_WORKSPACE", "acme")
	t.Setenv("BK_PROJECT", "PROJ")
	t.Setenv("BK_REPO", "acme/widgets")

	cfg := &config.Config{Version: 1}
	settings, err := config.Resolve(cfg, dir)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if settings.Hostname != "bitbucket.example.com" {
		t.Fatalf("hostname = %q", settings.Hostname)
	}
	if settings.Kind != config.KindDataCenter {
		t.Fatalf("kind = %q", settings.Kind)
	}
	if settings.APIURL != "https://bitbucket.example.com/bitbucket/rest/api/1.0" {
		t.Fatalf("api URL = %q", settings.APIURL)
	}
	if settings.Token != "secret" || settings.TokenSource != "env" {
		t.Fatalf("token = %q source = %q", settings.Token, settings.TokenSource)
	}
	if settings.AuthMethod != config.AuthBasic {
		t.Fatalf("method = %q", settings.AuthMethod)
	}
	if settings.Workspace != "acme" || settings.Project != "PROJ" || settings.Repo != "acme/widgets" {
		t.Fatalf("defaults not applied: %+v", settings)
	}
}

func TestResolveCloudDefaultsToBasicWhenUsernameIsSet(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BK_HOST", "bitbucket.org")
	t.Setenv("BK_USERNAME", "me@example.com")
	t.Setenv("BK_TOKEN", "secret")
	t.Setenv("BK_AUTH_METHOD", "")

	cfg := &config.Config{Version: 1}
	settings, err := config.Resolve(cfg, dir)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if settings.AuthMethod != config.AuthBasic {
		t.Fatalf("method = %q, want basic", settings.AuthMethod)
	}
	if settings.APIURL != config.CloudAPIURL {
		t.Fatalf("api URL = %q", settings.APIURL)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BK_CONFIG_DIR", dir)

	cfg := &config.Config{Version: 1, ActiveHost: "bitbucket.org"}
	host := cfg.HostFor("bitbucket.org")
	host.User = "me@example.com"
	host.AuthMethod = config.AuthBasic
	host.Workspace = "acme"
	cfg.SetAlias("pl", "pr list")
	if err := cfg.Save(dir); err != nil {
		t.Fatalf("Save: %v", err)
	}

	path := filepath.Join(dir, "config.yml")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("permissions = %v, want 0600", info.Mode().Perm())
	}

	loaded, loadedDir, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loadedDir != dir {
		t.Fatalf("dir = %q, want %q", loadedDir, dir)
	}
	if loaded.ActiveHost != "bitbucket.org" {
		t.Fatalf("active host = %q", loaded.ActiveHost)
	}
	if loaded.Hosts["bitbucket.org"].Workspace != "acme" {
		t.Fatalf("workspace = %q", loaded.Hosts["bitbucket.org"].Workspace)
	}
	if loaded.Aliases["pl"] != "pr list" {
		t.Fatalf("alias = %q", loaded.Aliases["pl"])
	}
}
