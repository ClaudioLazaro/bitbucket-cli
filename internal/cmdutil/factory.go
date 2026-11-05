// Package cmdutil wires configuration, credentials, and the API client for
// command implementations.
package cmdutil

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ClaudioLazaro/bitbucket-cli/internal/auth"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/bitbucket"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/browser"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/config"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/git"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/iostreams"
)

// Factory builds the runtime application for a command.
type Factory struct {
	IO  *iostreams.IOStreams
	Now func() time.Time
}

// NewFactory returns a factory bound to the given streams.
func NewFactory(io *iostreams.IOStreams) *Factory {
	return &Factory{IO: io, Now: time.Now}
}

// App is the resolved runtime context for one command invocation.
type App struct {
	IO               *iostreams.IOStreams
	Config           *config.Config
	ConfigDir        string
	Settings         config.Settings
	Client           *bitbucket.Client
	Credentials      auth.Credential
	CredentialSource string
	Store            *auth.Store
	Now              func() time.Time
}

// App loads configuration and credentials and builds the API client.
func (f *Factory) App() (*App, error) {
	cfg, dir, err := config.Load()
	if err != nil {
		return nil, err
	}
	settings, err := config.Resolve(cfg, dir)
	if err != nil {
		return nil, err
	}

	store := auth.NewStore(dir)
	cred := auth.Credential{Token: settings.Token, Username: settings.Username, Method: settings.AuthMethod}
	source := settings.TokenSource
	if cred.Token == "" {
		stored, storedSource, err := store.Get(settings.Hostname)
		if err != nil {
			return nil, err
		}
		if stored.Token != "" {
			cred = stored
			source = storedSource
			if cred.Username != "" {
				settings.Username = cred.Username
			}
			if cred.Method != "" {
				settings.AuthMethod = cred.Method
			}
		}
	}
	if cred.Token == "" {
		return nil, fmt.Errorf("not logged in to %s: run `bk auth login --hostname %s`", settings.Hostname, settings.Hostname)
	}
	if cred.Method == "" {
		cred.Method = settings.AuthMethod
	}
	if cred.Method == config.AuthBasic && settings.Username == "" {
		return nil, fmt.Errorf("basic authentication requires a username: run `bk auth login` again or set BK_USERNAME")
	}

	settings.Token = cred.Token
	settings.TokenSource = source
	client, err := bitbucket.NewClient(settings)
	if err != nil {
		return nil, err
	}
	now := f.Now
	if now == nil {
		now = time.Now
	}
	return &App{
		IO:               f.IO,
		Config:           cfg,
		ConfigDir:        dir,
		Settings:         settings,
		Client:           client,
		Credentials:      cred,
		CredentialSource: source,
		Store:            store,
		Now:              now,
	}, nil
}

// IsCloud reports whether the active host is Bitbucket Cloud.
func (a *App) IsCloud() bool { return a.Settings.IsCloud() }

// RequireCloud returns a descriptive error for Cloud-only operations on Data
// Center hosts.
func (a *App) RequireCloud(operation string) error {
	if a.IsCloud() {
		return nil
	}
	return &bitbucket.UnsupportedError{Operation: operation, Kind: "Data Center"}
}

// ResolveRepo resolves the target repository from an explicit override, the
// BK_REPO environment variable, or the git remotes.
func (a *App) ResolveRepo(ctx context.Context, override string) (bitbucket.RepoRef, error) {
	override = strings.TrimSpace(override)
	if override == "" {
		override = a.Settings.Repo
	}
	if override != "" {
		owner, slug, host, err := bitbucket.RepositoryFromInput(override, a.Settings.Hostname)
		if err != nil {
			return bitbucket.RepoRef{}, err
		}
		return bitbucket.RepoRef{Host: host, Owner: owner, Slug: slug}, nil
	}
	return git.RepoFromRemotes(ctx, a.Settings.Hostname)
}

// RepoHTMLURL returns the browser URL for a repository.
func (a *App) RepoHTMLURL(ref bitbucket.RepoRef) string {
	if a.IsCloud() {
		return fmt.Sprintf("https://bitbucket.org/%s/%s", ref.Owner, ref.Slug)
	}
	base := strings.TrimSuffix(a.Settings.APIURL, "/rest/api/1.0")
	return fmt.Sprintf("%s/projects/%s/repos/%s", base, ref.Owner, ref.Slug)
}

// PullRequestHTMLURL returns the browser URL for a pull request.
func (a *App) PullRequestHTMLURL(ref bitbucket.RepoRef, id int) string {
	return fmt.Sprintf("%s/pull-requests/%d", a.RepoHTMLURL(ref), id)
}

// SourceHTMLURL returns the browser URL for a file or directory on a branch.
func (a *App) SourceHTMLURL(ref bitbucket.RepoRef, branch, path string) string {
	base := fmt.Sprintf("%s/src/%s", a.RepoHTMLURL(ref), branch)
	if path != "" {
		base += "/" + strings.TrimPrefix(path, "/")
	}
	return base
}

// PipelineHTMLURL returns the browser URL for a pipeline build number.
func (a *App) PipelineHTMLURL(ref bitbucket.RepoRef, buildNumber int) string {
	if a.IsCloud() {
		return fmt.Sprintf("%s/addon/pipelines/home#!/results/%d", a.RepoHTMLURL(ref), buildNumber)
	}
	return a.RepoHTMLURL(ref)
}

// OpenBrowser opens rawURL, printing it when no browser is available.
func (a *App) OpenBrowser(rawURL string) error {
	if !a.IO.IsStdoutTTY() || !browser.Available() {
		fmt.Fprintln(a.IO.Out, rawURL)
		return nil
	}
	if err := browser.Open(rawURL); err != nil {
		fmt.Fprintln(a.IO.Out, rawURL)
		return fmt.Errorf("could not open browser: %w", err)
	}
	return nil
}
