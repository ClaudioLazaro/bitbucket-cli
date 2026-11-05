// Package config loads and persists bk configuration and resolves the
// effective host, credentials, and repository defaults for a command.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// KindCloud identifies Bitbucket Cloud hosts.
	KindCloud = "cloud"
	// KindDataCenter identifies Bitbucket Data Center or Server hosts.
	KindDataCenter = "dc"
	// CloudAPIURL is the Bitbucket Cloud REST 2.0 base URL.
	CloudAPIURL = "https://api.bitbucket.org/2.0"
	// DefaultHost is the hostname used when none is configured.
	DefaultHost = "bitbucket.org"
	// AuthBasic uses HTTP Basic with an Atlassian email and API token.
	AuthBasic = "basic"
	// AuthBearer uses a Bearer access token.
	AuthBearer = "bearer"
)

// ErrNoHost is returned when no host has been configured or selected.
var ErrNoHost = errors.New("no Bitbucket host configured: run `bk auth login`")

// Config is the on-disk configuration file.
type Config struct {
	Version    int               `yaml:"version"`
	ActiveHost string            `yaml:"active_host,omitempty"`
	Hosts      map[string]*Host  `yaml:"hosts,omitempty"`
	Aliases    map[string]string `yaml:"aliases,omitempty"`
}

// Host holds non-secret settings for one Bitbucket host.
type Host struct {
	Kind       string `yaml:"kind"`
	User       string `yaml:"user,omitempty"`
	AuthMethod string `yaml:"auth_method,omitempty"`
	Workspace  string `yaml:"workspace,omitempty"`
	Project    string `yaml:"project,omitempty"`
	APIURL     string `yaml:"api_url,omitempty"`
}

// Settings is the effective configuration for one command invocation, after
// merging the config file with environment variables.
type Settings struct {
	ConfigDir   string
	Hostname    string
	Kind        string
	APIURL      string
	Username    string
	Token       string
	AuthMethod  string
	TokenSource string
	Workspace   string
	Project     string
	Repo        string
	ActiveHost  string
}

// IsCloud reports whether the effective host is Bitbucket Cloud.
func (s Settings) IsCloud() bool { return s.Kind == KindCloud }

// Dir returns the configuration directory, honoring BK_CONFIG_DIR.
func Dir() (string, error) {
	if d := os.Getenv("BK_CONFIG_DIR"); d != "" {
		return d, nil
	}
	d, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "bk"), nil
}

// Load reads the configuration file. A missing file yields an empty config.
func Load() (*Config, string, error) {
	dir, err := Dir()
	if err != nil {
		return nil, "", err
	}
	cfg := &Config{Version: 1}
	path := filepath.Join(dir, "config.yml")
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, dir, nil
		}
		return nil, dir, err
	}
	if err := yaml.Unmarshal(b, cfg); err != nil {
		return nil, dir, fmt.Errorf("parse %s: %w", path, err)
	}
	if cfg.Hosts == nil {
		cfg.Hosts = map[string]*Host{}
	}
	if cfg.Aliases == nil {
		cfg.Aliases = map[string]string{}
	}
	return cfg, dir, nil
}

// Save writes the configuration atomically with owner-only permissions.
func (c *Config) Save(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	c.Version = 1
	b, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".config-*.yml")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, filepath.Join(dir, "config.yml"))
}

// HostFor returns the settings for hostname, creating an entry when absent.
func (c *Config) HostFor(hostname string) *Host {
	if c.Hosts == nil {
		c.Hosts = map[string]*Host{}
	}
	if h, ok := c.Hosts[hostname]; ok && h != nil {
		return h
	}
	h := &Host{}
	if _, kind, api, err := NormalizeHost(hostname); err == nil {
		h.Kind = kind
		h.APIURL = api
	}
	c.Hosts[hostname] = h
	return h
}

// SetAlias stores an alias expansion.
func (c *Config) SetAlias(name, expansion string) {
	if c.Aliases == nil {
		c.Aliases = map[string]string{}
	}
	c.Aliases[name] = expansion
}

// NormalizeHost validates a hostname or URL and returns its canonical
// hostname, kind, and REST API base URL.
func NormalizeHost(raw string) (hostname, kind, apiURL string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", "", errors.New("empty hostname")
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "", "", "", fmt.Errorf("invalid host %q", raw)
	}
	host := strings.ToLower(u.Host)
	switch host {
	case "bitbucket.org", "www.bitbucket.org", "api.bitbucket.org":
		return DefaultHost, KindCloud, CloudAPIURL, nil
	}

	path := strings.TrimSuffix(u.Path, "/")
	path = strings.TrimSuffix(path, "/rest/api/1.0")
	base := u.Scheme + "://" + host + path
	return host, KindDataCenter, base + "/rest/api/1.0", nil
}

// Resolve merges the config file with BK_* environment variables.
func Resolve(cfg *Config, dir string) (Settings, error) {
	s := Settings{ConfigDir: dir, ActiveHost: cfg.ActiveHost}

	hostname := ""
	if raw := os.Getenv("BK_HOST"); raw != "" {
		h, kind, api, err := NormalizeHost(raw)
		if err != nil {
			return s, err
		}
		hostname, s.Kind, s.APIURL = h, kind, api
	} else {
		hostname = cfg.ActiveHost
		if hostname == "" && len(cfg.Hosts) == 1 {
			for name := range cfg.Hosts {
				hostname = name
			}
		}
		if hostname == "" {
			return s, ErrNoHost
		}
		_, kind, api, err := NormalizeHost(hostname)
		if err != nil {
			return s, err
		}
		s.Kind, s.APIURL = kind, api
	}
	s.Hostname = hostname

	if hc := cfg.Hosts[hostname]; hc != nil {
		if hc.Kind != "" {
			if hc.Kind != KindCloud && hc.Kind != KindDataCenter {
				return s, fmt.Errorf("host %s has invalid kind %q", hostname, hc.Kind)
			}
			s.Kind = hc.Kind
		}
		if hc.APIURL != "" {
			s.APIURL = strings.TrimRight(hc.APIURL, "/")
		}
		s.Username = hc.User
		s.AuthMethod = hc.AuthMethod
		s.Workspace = hc.Workspace
		s.Project = hc.Project
	}

	if v := os.Getenv("BK_API_URL"); v != "" {
		s.APIURL = strings.TrimRight(v, "/")
	}
	if v := os.Getenv("BK_USERNAME"); v != "" {
		s.Username = v
	}
	if v := os.Getenv("BK_AUTH_METHOD"); v != "" {
		m, err := normalizeMethod(v)
		if err != nil {
			return s, err
		}
		s.AuthMethod = m
	}
	if v := os.Getenv("BK_WORKSPACE"); v != "" {
		s.Workspace = v
	}
	if v := os.Getenv("BK_PROJECT"); v != "" {
		s.Project = v
	}
	if v := os.Getenv("BK_REPO"); v != "" {
		s.Repo = v
	}
	if v := os.Getenv("BK_TOKEN"); v != "" {
		s.Token = v
		s.TokenSource = "env"
	}

	if s.AuthMethod == "" {
		if s.Username != "" && s.Token != "" {
			s.AuthMethod = AuthBasic
		} else {
			s.AuthMethod = AuthBearer
		}
	}
	if s.APIURL == "" {
		if s.Kind == KindCloud {
			s.APIURL = CloudAPIURL
		} else {
			return s, fmt.Errorf("host %s has no REST API URL", hostname)
		}
	}
	if s.Kind == KindDataCenter && s.Username == "" && s.AuthMethod == AuthBasic {
		return s, fmt.Errorf("Data Center basic auth requires a username (BK_USERNAME)")
	}
	return s, nil
}

func normalizeMethod(m string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(m)) {
	case AuthBasic, "api-token", "password":
		return AuthBasic, nil
	case AuthBearer, "token", "access-token", "pat", "oauth":
		return AuthBearer, nil
	default:
		return "", fmt.Errorf("invalid auth method %q: use basic or bearer", m)
	}
}

// RepoFromSettings builds a repository reference from defaults and BK_REPO.
func RepoFromSettings(s Settings) (owner, slug string, ok bool) {
	if s.Repo == "" {
		return "", "", false
	}
	parts := strings.Split(strings.Trim(s.Repo, "/"), "/")
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}
