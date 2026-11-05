// Package auth implements bk auth commands.
package auth

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	authstore "github.com/ClaudioLazaro/bitbucket-cli/internal/auth"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/bitbucket"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/cmdutil"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/config"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/tableprinter"
)

// NewCmdAuth returns the auth command group.
func NewCmdAuth(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate with a Bitbucket host",
		Long:  "Log in, log out, and inspect the credentials bk uses for each Bitbucket host.",
	}
	cmd.AddCommand(newCmdLogin(f), newCmdLogout(f), newCmdStatus(f), newCmdToken(f))
	return cmd
}

type loginOptions struct {
	Hostname    string
	Username    string
	WithToken   bool
	AccessToken bool
	AuthMethod  string
	NoVerify    bool
	Workspace   string
	Project     string
}

func newCmdLogin(f *cmdutil.Factory) *cobra.Command {
	opts := &loginOptions{}
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in to Bitbucket",
		Long: `Store a credential for a Bitbucket host.

Bitbucket Cloud removed app passwords in July 2026. Use an Atlassian API token
created with Bitbucket scopes: the username is your Atlassian account email and
the password is the token. Repository, project, and workspace access tokens use
Bearer authentication and need no username.

Data Center personal access tokens use Bearer authentication by default.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogin(cmd.Context(), f, opts)
		},
	}
	cmd.Flags().StringVar(&opts.Hostname, "hostname", "", "Bitbucket host (default bitbucket.org)")
	cmd.Flags().StringVarP(&opts.Username, "username", "u", "", "Atlassian account email for API tokens")
	cmd.Flags().BoolVar(&opts.WithToken, "with-token", false, "Read the token from standard input")
	cmd.Flags().BoolVar(&opts.AccessToken, "access-token", false, "Treat the token as a Bearer access token")
	cmd.Flags().StringVar(&opts.AuthMethod, "auth-method", "", "Authentication method: basic or bearer")
	cmd.Flags().BoolVar(&opts.NoVerify, "no-verify", false, "Do not validate the credential against the host")
	cmd.Flags().StringVarP(&opts.Workspace, "workspace", "w", "", "Default Cloud workspace")
	cmd.Flags().StringVarP(&opts.Project, "project", "p", "", "Default Data Center project key")
	return cmd
}

func runLogin(ctx context.Context, f *cmdutil.Factory, opts *loginOptions) error {
	hostInput := opts.Hostname
	if hostInput == "" {
		hostInput = os.Getenv("BK_HOST")
	}
	if hostInput == "" {
		hostInput = config.DefaultHost
	}
	hostname, kind, apiURL, err := config.NormalizeHost(hostInput)
	if err != nil {
		return err
	}

	cfg, dir, err := config.Load()
	if err != nil {
		return err
	}
	hostCfg := cfg.HostFor(hostname)
	if hostCfg.Kind == "" {
		hostCfg.Kind = kind
	}
	if hostCfg.APIURL != "" {
		apiURL = hostCfg.APIURL
	} else {
		hostCfg.APIURL = apiURL
	}
	if override := os.Getenv("BK_API_URL"); override != "" {
		apiURL = strings.TrimRight(override, "/")
	}
	kind = hostCfg.Kind

	method := strings.ToLower(strings.TrimSpace(opts.AuthMethod))
	if method == "" {
		switch {
		case opts.AccessToken:
			method = config.AuthBearer
		case kind == config.KindDataCenter && opts.Username != "":
			method = config.AuthBasic
		case kind == config.KindDataCenter:
			method = config.AuthBearer
		default:
			method = config.AuthBasic
		}
	}
	if method != config.AuthBasic && method != config.AuthBearer {
		return fmt.Errorf("invalid --auth-method %q: use basic or bearer", method)
	}

	username := opts.Username
	if username == "" {
		username = os.Getenv("BK_USERNAME")
	}
	if username == "" {
		username = hostCfg.User
	}
	if method == config.AuthBasic && username == "" {
		if !f.IO.IsStdinTTY() {
			return fmt.Errorf("basic authentication requires --username (your Atlassian account email)")
		}
		username, err = f.IO.ReadLine("Atlassian account email: ")
		if err != nil {
			return err
		}
		username = strings.TrimSpace(username)
		if username == "" {
			return fmt.Errorf("username is required for basic authentication")
		}
	}
	if method == config.AuthBearer {
		username = ""
	}

	var token string
	if opts.WithToken {
		data, err := io.ReadAll(io.LimitReader(f.IO.In, 1<<20))
		if err != nil {
			return err
		}
		token = strings.TrimSpace(string(data))
	} else {
		if !f.IO.IsStdinTTY() {
			return fmt.Errorf("no terminal available: pass --with-token and pipe the token on standard input")
		}
		token, err = f.IO.ReadPassword("Paste your token: ")
		if err != nil {
			return err
		}
		token = strings.TrimSpace(token)
	}
	if token == "" {
		return fmt.Errorf("empty token")
	}

	settings := config.Settings{
		Hostname:   hostname,
		Kind:       kind,
		APIURL:     apiURL,
		Username:   username,
		Token:      token,
		AuthMethod: method,
	}
	client, err := bitbucket.NewClient(settings)
	if err != nil {
		return err
	}
	if opts.NoVerify {
		fmt.Fprintln(f.IO.ErrOut, "Warning: credential was not verified (--no-verify).")
	} else {
		result, err := verifyCredential(ctx, client)
		if err != nil {
			return fmt.Errorf("credential verification failed: %w", err)
		}
		if result != "" {
			fmt.Fprintf(f.IO.ErrOut, "Authenticated: %s\n", result)
		}
	}

	store := authstore.NewStore(dir)
	backend, err := store.Set(hostname, authstore.Credential{Token: token, Username: username, Method: method})
	if err != nil {
		return err
	}

	hostCfg.User = username
	hostCfg.AuthMethod = method
	hostCfg.Kind = kind
	if opts.Workspace != "" {
		hostCfg.Workspace = opts.Workspace
	}
	if opts.Project != "" {
		hostCfg.Project = opts.Project
	}
	cfg.ActiveHost = hostname
	if err := cfg.Save(dir); err != nil {
		return err
	}

	identity := username
	if identity == "" {
		identity = "access token"
	}
	fmt.Fprintf(f.IO.Out, "Logged in to %s as %s (credential stored in %s)\n", hostname, identity, backend)
	if backend == "file" {
		fmt.Fprintf(f.IO.ErrOut, "Warning: no OS keyring available; token stored in %s with mode 0600.\n", store.FilePath())
	}
	return nil
}

func verifyCredential(ctx context.Context, client *bitbucket.Client) (string, error) {
	if client.IsCloud() {
		user, err := client.GetCurrentUser(ctx)
		if err == nil {
			return "ok (" + user.Label() + ")", nil
		}
		if bitbucket.IsForbidden(err) && client.AuthMethod == config.AuthBearer {
			return "ok (identity not exposed by this token)", nil
		}
		return "", err
	}
	if _, err := client.ListRepositories(ctx, "", nil, 1); err != nil {
		return "", err
	}
	return "ok", nil
}

func newCmdLogout(f *cmdutil.Factory) *cobra.Command {
	var hostname string
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Remove the stored credential for a host",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, dir, err := config.Load()
			if err != nil {
				return err
			}
			host := hostname
			if host == "" {
				host = activeHost(cfg)
			}
			if host == "" {
				return fmt.Errorf("no host selected: pass --hostname")
			}
			normalized, _, _, err := config.NormalizeHost(host)
			if err != nil {
				return err
			}
			store := authstore.NewStore(dir)
			if err := store.Delete(normalized); err != nil {
				return err
			}
			if hc := cfg.Hosts[normalized]; hc != nil {
				hc.User = ""
				hc.AuthMethod = ""
			}
			if err := cfg.Save(dir); err != nil {
				return err
			}
			fmt.Fprintf(f.IO.Out, "Logged out of %s\n", normalized)
			if os.Getenv("BK_TOKEN") != "" {
				fmt.Fprintln(f.IO.ErrOut, "Warning: BK_TOKEN is set and will keep authenticating requests.")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&hostname, "hostname", "", "Bitbucket host")
	return cmd
}

type statusEntry struct {
	Host        string `json:"host"`
	Kind        string `json:"kind"`
	User        string `json:"user,omitempty"`
	Auth        string `json:"auth,omitempty"`
	TokenSource string `json:"token_source,omitempty"`
	Active      bool   `json:"active"`
	Status      string `json:"status"`
}

func newCmdStatus(f *cmdutil.Factory) *cobra.Command {
	var hostnameFlag string
	var out *cmdutil.OutputFlags
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show authentication status for configured hosts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, dir, err := config.Load()
			if err != nil {
				return err
			}
			hosts, err := hostsToCheck(cfg, hostnameFlag)
			if err != nil {
				return err
			}
			if len(hosts) == 0 {
				fmt.Fprintln(f.IO.Out, "No Bitbucket hosts configured. Run `bk auth login`.")
				return nil
			}
			store := authstore.NewStore(dir)
			active := activeHost(cfg)
			entries := make([]statusEntry, 0, len(hosts))
			for _, host := range hosts {
				entries = append(entries, checkHost(ctx, store, cfg, host, active))
			}
			if out.IsJSON() {
				return out.Print(f.IO.Out, entries)
			}
			t := tableprinter.New(f.IO).Header("HOST", "KIND", "USER", "AUTH", "TOKEN", "ACTIVE", "STATUS")
			for _, entry := range entries {
				mark := ""
				if entry.Active {
					mark = "*"
				}
				t.AddRow(entry.Host, entry.Kind, entry.User, entry.Auth, entry.TokenSource, mark, entry.Status)
			}
			return t.Render()
		},
	}
	out = cmdutil.AddOutputFlags(cmd)
	cmd.Flags().StringVar(&hostnameFlag, "hostname", "", "Check only this host")
	return cmd
}

func checkHost(ctx context.Context, store *authstore.Store, cfg *config.Config, hostname, active string) statusEntry {
	entry := statusEntry{Host: hostname, Active: hostname == active}
	hostCfg := cfg.Hosts[hostname]
	_, kind, apiURL, err := config.NormalizeHost(hostname)
	if err != nil {
		entry.Status = err.Error()
		return entry
	}
	settings := config.Settings{Hostname: hostname, Kind: kind, APIURL: apiURL}
	if hostCfg != nil {
		if hostCfg.Kind != "" {
			settings.Kind = hostCfg.Kind
		}
		if hostCfg.APIURL != "" {
			settings.APIURL = hostCfg.APIURL
		}
		settings.Username = hostCfg.User
		settings.AuthMethod = hostCfg.AuthMethod
	}
	if hostname == active {
		if v := os.Getenv("BK_USERNAME"); v != "" {
			settings.Username = v
		}
		if v := os.Getenv("BK_AUTH_METHOD"); v != "" {
			settings.AuthMethod = v
		}
		if v := os.Getenv("BK_API_URL"); v != "" {
			settings.APIURL = v
		}
		if v := os.Getenv("BK_TOKEN"); v != "" {
			settings.Token = v
			settings.TokenSource = "env"
		}
	}
	entry.Kind = settings.Kind
	entry.Auth = settings.AuthMethod

	if settings.Token == "" {
		cred, source, err := store.Get(hostname)
		if err != nil {
			entry.Status = err.Error()
			return entry
		}
		if cred.Token != "" {
			settings.Token = cred.Token
			settings.TokenSource = source
			settings.Username = cred.Username
			settings.AuthMethod = cred.Method
		}
	}
	entry.User = settings.Username
	entry.Auth = settings.AuthMethod
	entry.TokenSource = settings.TokenSource
	if settings.Token == "" {
		entry.Status = "not logged in"
		return entry
	}
	client, err := bitbucket.NewClient(settings)
	if err != nil {
		entry.Status = err.Error()
		return entry
	}
	result, err := verifyCredential(ctx, client)
	if err != nil {
		entry.Status = err.Error()
		return entry
	}
	entry.Status = result
	return entry
}

func newCmdToken(f *cmdutil.Factory) *cobra.Command {
	var hostnameFlag string
	cmd := &cobra.Command{
		Use:   "token",
		Short: "Print the stored token for a host",
		Long:  "Print the token exactly as stored so scripts can reuse it. Treat the output as a secret.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, dir, err := config.Load()
			if err != nil {
				return err
			}
			host := hostnameFlag
			if host == "" {
				host = activeHost(cfg)
			}
			if host == "" {
				return fmt.Errorf("no host selected: pass --hostname")
			}
			normalized, _, _, err := config.NormalizeHost(host)
			if err != nil {
				return err
			}
			if normalized == activeHost(cfg) {
				if token := os.Getenv("BK_TOKEN"); token != "" {
					fmt.Fprintln(f.IO.Out, token)
					return nil
				}
			}
			cred, _, err := authstore.NewStore(dir).Get(normalized)
			if err != nil {
				return err
			}
			if cred.Token == "" {
				return fmt.Errorf("no stored token for %s", normalized)
			}
			fmt.Fprintln(f.IO.Out, cred.Token)
			return nil
		},
	}
	cmd.Flags().StringVar(&hostnameFlag, "hostname", "", "Bitbucket host")
	return cmd
}

func hostsToCheck(cfg *config.Config, override string) ([]string, error) {
	if override != "" {
		host, _, _, err := config.NormalizeHost(override)
		if err != nil {
			return nil, err
		}
		return []string{host}, nil
	}
	if raw := os.Getenv("BK_HOST"); raw != "" {
		host, _, _, err := config.NormalizeHost(raw)
		if err != nil {
			return nil, err
		}
		return []string{host}, nil
	}
	if cfg.ActiveHost != "" {
		return []string{cfg.ActiveHost}, nil
	}
	hosts := make([]string, 0, len(cfg.Hosts))
	for host := range cfg.Hosts {
		hosts = append(hosts, host)
	}
	sort.Strings(hosts)
	return hosts, nil
}

func activeHost(cfg *config.Config) string {
	if raw := os.Getenv("BK_HOST"); raw != "" {
		if host, _, _, err := config.NormalizeHost(raw); err == nil {
			return host
		}
	}
	if cfg.ActiveHost != "" {
		return cfg.ActiveHost
	}
	if len(cfg.Hosts) == 1 {
		for host := range cfg.Hosts {
			return host
		}
	}
	return ""
}
