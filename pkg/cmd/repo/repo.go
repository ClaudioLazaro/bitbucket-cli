// Package repo implements bk repo commands.
package repo

import (
	"context"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/ClaudioLazaro/bitbucket-cli/internal/bitbucket"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/cmdutil"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/git"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/tableprinter"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/text"
)

// NewCmdRepo returns the repo command group.
func NewCmdRepo(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo",
		Short: "Work with Bitbucket repositories",
		Long:  "List, view, create, clone, fork, edit, and delete repositories.",
	}
	cmd.AddCommand(
		newCmdList(f),
		newCmdView(f),
		newCmdCreate(f),
		newCmdClone(f),
		newCmdFork(f),
		newCmdDelete(f),
		newCmdEdit(f),
	)
	return cmd
}

func newCmdList(f *cmdutil.Factory) *cobra.Command {
	var workspace string
	var limit int
	var role, sortBy string
	var out *cmdutil.OutputFlags
	cmd := &cobra.Command{
		Use:   "list [WORKSPACE]",
		Short: "List repositories in a workspace or project",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			app, err := f.App()
			if err != nil {
				return err
			}
			ws := workspace
			if len(args) == 1 {
				ws = args[0]
			}
			if ws == "" {
				ws = app.Settings.Workspace
			}
			if ws == "" {
				ws = app.Settings.Project
			}
			if ws == "" && app.IsCloud() {
				return fmt.Errorf("workspace is required: pass WORKSPACE or use --workspace")
			}
			q := url.Values{}
			if app.IsCloud() {
				if role != "" {
					q.Set("role", role)
				}
				if sortBy != "" {
					q.Set("sort", sortBy)
				}
			}
			repos, err := app.Client.ListRepositories(ctx, ws, q, limit)
			if err != nil {
				return err
			}
			if out.IsJSON() {
				return out.Print(app.IO.Out, repos)
			}
			t := tableprinter.New(app.IO).Header("NAME", "VISIBILITY", "UPDATED", "DESCRIPTION")
			for _, repo := range repos {
				t.AddRow(
					repo.SlugOrName(),
					visibility(repo.IsPrivate),
					text.RelativeTime(repo.UpdatedOn),
					text.Truncate(repo.Description, 60),
				)
			}
			return t.Render()
		},
	}
	out = cmdutil.AddOutputFlags(cmd)
	cmd.Flags().StringVarP(&workspace, "workspace", "w", "", "Workspace or project key")
	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of repositories (0 for all)")
	cmd.Flags().StringVar(&role, "role", "", "Filter by role: admin, contributor, member, owner")
	cmd.Flags().StringVar(&sortBy, "sort", "-updated_on", "Sort field")
	return cmd
}

func newCmdView(f *cmdutil.Factory) *cobra.Command {
	var repoFlag string
	var web bool
	var out *cmdutil.OutputFlags
	cmd := &cobra.Command{
		Use:   "view [OWNER/REPO]",
		Short: "View a repository",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			app, err := f.App()
			if err != nil {
				return err
			}
			ref, err := resolveArg(ctx, app, repoFlag, args)
			if err != nil {
				return err
			}
			repository, err := app.Client.GetRepository(ctx, ref.Owner, ref.Slug)
			if err != nil {
				return err
			}
			htmlURL := repository.HTMLURL()
			if htmlURL == "" {
				htmlURL = app.RepoHTMLURL(ref)
			}
			if web {
				return app.OpenBrowser(htmlURL)
			}
			if out.IsJSON() {
				return out.Print(app.IO.Out, repository)
			}

			cs := app.IO.ColorScheme()
			fmt.Fprintln(app.IO.Out, cs.Bold(repository.NameWithOwner()))
			if repository.Description != "" {
				fmt.Fprintf(app.IO.Out, "%s\n", repository.Description)
			}
			fmt.Fprintln(app.IO.Out)
			fmt.Fprintf(app.IO.Out, "Private:        %t\n", repository.IsPrivate)
			fmt.Fprintf(app.IO.Out, "SCM:            %s\n", repository.SCM)
			if branch := defaultBranch(repository); branch != "" {
				fmt.Fprintf(app.IO.Out, "Default branch: %s\n", branch)
			}
			if repository.Language != "" {
				fmt.Fprintf(app.IO.Out, "Language:       %s\n", repository.Language)
			}
			if !repository.UpdatedOn.IsZero() {
				fmt.Fprintf(app.IO.Out, "Updated:        %s\n", text.Date(repository.UpdatedOn))
			}
			fmt.Fprintf(app.IO.Out, "URL:            %s\n", htmlURL)
			if clone := repository.CloneURL("https"); clone != "" {
				fmt.Fprintf(app.IO.Out, "Clone (HTTPS):  %s\n", clone)
			}
			if clone := repository.CloneURL("ssh"); clone != "" {
				fmt.Fprintf(app.IO.Out, "Clone (SSH):    %s\n", clone)
			}
			return nil
		},
	}
	out = cmdutil.AddOutputFlags(cmd)
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	cmd.Flags().BoolVarP(&web, "web", "w", false, "Open the repository in a browser")
	return cmd
}

func newCmdCreate(f *cmdutil.Factory) *cobra.Command {
	var description, project, scm string
	var public, clone bool
	var out *cmdutil.OutputFlags
	cmd := &cobra.Command{
		Use:   "create OWNER/REPO",
		Short: "Create a repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			app, err := f.App()
			if err != nil {
				return err
			}
			owner, slug, _, err := bitbucket.RepositoryFromInput(args[0], app.Settings.Hostname)
			if err != nil {
				return err
			}
			projectKey := project
			if projectKey == "" {
				projectKey = app.Settings.Project
			}
			repository, err := app.Client.CreateRepository(ctx, owner, slug, bitbucket.CreateRepositoryInput{
				SCM:         scm,
				IsPrivate:   !public,
				Description: description,
				ProjectKey:  projectKey,
			})
			if err != nil {
				return err
			}
			if out.IsJSON() {
				return out.Print(app.IO.Out, repository)
			}
			htmlURL := repository.HTMLURL()
			if htmlURL == "" {
				htmlURL = app.RepoHTMLURL(bitbucket.RepoRef{Owner: owner, Slug: slug})
			}
			fmt.Fprintf(app.IO.Out, "Created repository %s\n%s\n", repository.NameWithOwner(), htmlURL)
			if clone {
				cloneURL := repository.CloneURL("https")
				if cloneURL == "" {
					return fmt.Errorf("repository created but no HTTPS clone URL was returned")
				}
				if err := git.Clone(ctx, cloneURL, slug, app.IO.Out, app.IO.ErrOut); err != nil {
					return err
				}
				fmt.Fprintf(app.IO.Out, "Cloned to %s\n", slug)
			}
			return nil
		},
	}
	out = cmdutil.AddOutputFlags(cmd)
	cmd.Flags().StringVarP(&description, "description", "d", "", "Repository description")
	cmd.Flags().StringVar(&project, "project", "", "Cloud project key")
	cmd.Flags().StringVar(&scm, "scm", "git", "Repository SCM")
	cmd.Flags().BoolVar(&public, "public", false, "Create a public repository")
	cmd.Flags().BoolVar(&clone, "clone", false, "Clone the repository after creating it")
	return cmd
}

func newCmdClone(f *cmdutil.Factory) *cobra.Command {
	var repoFlag string
	var ssh bool
	cmd := &cobra.Command{
		Use:   "clone [OWNER/REPO] [DIRECTORY]",
		Short: "Clone a repository",
		Args:  cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			app, err := f.App()
			if err != nil {
				return err
			}
			ref, err := resolveArg(ctx, app, repoFlag, args)
			if err != nil {
				return err
			}
			repository, err := app.Client.GetRepository(ctx, ref.Owner, ref.Slug)
			if err != nil {
				return err
			}
			protocol := "https"
			if ssh {
				protocol = "ssh"
			}
			cloneURL := repository.CloneURL(protocol)
			if cloneURL == "" {
				return fmt.Errorf("repository has no %s clone URL", protocol)
			}
			dir := ref.Slug
			if len(args) == 2 {
				dir = args[1]
			}
			if err := git.Clone(ctx, cloneURL, dir, app.IO.Out, app.IO.ErrOut); err != nil {
				return err
			}
			fmt.Fprintf(app.IO.Out, "Cloned to %s\n", dir)
			return nil
		},
	}
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	cmd.Flags().BoolVar(&ssh, "ssh", false, "Clone using SSH")
	return cmd
}

func newCmdFork(f *cmdutil.Factory) *cobra.Command {
	var repoFlag, name, workspace string
	var clone bool
	var out *cmdutil.OutputFlags
	cmd := &cobra.Command{
		Use:   "fork [OWNER/REPO]",
		Short: "Fork a repository",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			app, err := f.App()
			if err != nil {
				return err
			}
			ref, err := resolveArg(ctx, app, repoFlag, args)
			if err != nil {
				return err
			}
			repository, err := app.Client.ForkRepository(ctx, ref.Owner, ref.Slug, workspace, name)
			if err != nil {
				return err
			}
			if out.IsJSON() {
				return out.Print(app.IO.Out, repository)
			}
			htmlURL := repository.HTMLURL()
			if htmlURL == "" {
				htmlURL = app.RepoHTMLURL(bitbucket.RepoRef{Owner: repository.OwnerName(), Slug: repository.SlugOrName()})
			}
			fmt.Fprintf(app.IO.Out, "Forked to %s\n%s\n", repository.NameWithOwner(), htmlURL)
			if clone {
				cloneURL := repository.CloneURL("https")
				if cloneURL == "" {
					return fmt.Errorf("fork created but no HTTPS clone URL was returned")
				}
				dir := repository.SlugOrName()
				if err := git.Clone(ctx, cloneURL, dir, app.IO.Out, app.IO.ErrOut); err != nil {
					return err
				}
				fmt.Fprintf(app.IO.Out, "Cloned to %s\n", dir)
			}
			return nil
		},
	}
	out = cmdutil.AddOutputFlags(cmd)
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	cmd.Flags().StringVar(&name, "name", "", "Name for the fork")
	cmd.Flags().StringVarP(&workspace, "workspace", "w", "", "Destination workspace")
	cmd.Flags().BoolVar(&clone, "clone", false, "Clone the fork after creating it")
	return cmd
}

func newCmdDelete(f *cmdutil.Factory) *cobra.Command {
	var repoFlag string
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete [OWNER/REPO]",
		Short: "Delete a repository",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			app, err := f.App()
			if err != nil {
				return err
			}
			ref, err := resolveArg(ctx, app, repoFlag, args)
			if err != nil {
				return err
			}
			if err := app.Confirm(fmt.Sprintf("Delete repository %s?", ref), yes); err != nil {
				return err
			}
			if err := app.Client.DeleteRepository(ctx, ref.Owner, ref.Slug); err != nil {
				return err
			}
			fmt.Fprintf(app.IO.Out, "Deleted repository %s\n", ref)
			return nil
		},
	}
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Confirm deletion")
	return cmd
}

func newCmdEdit(f *cmdutil.Factory) *cobra.Command {
	var repoFlag, description, defaultBranch string
	var public, private bool
	var out *cmdutil.OutputFlags
	cmd := &cobra.Command{
		Use:   "edit [OWNER/REPO]",
		Short: "Edit repository settings",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			app, err := f.App()
			if err != nil {
				return err
			}
			ref, err := resolveArg(ctx, app, repoFlag, args)
			if err != nil {
				return err
			}
			body := map[string]any{}
			if cmd.Flags().Changed("description") {
				body["description"] = description
			}
			if public && private {
				return fmt.Errorf("--public and --private are mutually exclusive")
			}
			if public {
				body["is_private"] = false
			}
			if private {
				body["is_private"] = true
			}
			if defaultBranch != "" {
				if err := app.RequireCloud("changing the default branch"); err != nil {
					return err
				}
				body["mainbranch"] = map[string]string{"name": defaultBranch}
			}
			if len(body) == 0 {
				return fmt.Errorf("nothing to update: pass --description, --public, --private, or --default-branch")
			}
			repository, err := app.Client.UpdateRepository(ctx, ref.Owner, ref.Slug, body)
			if err != nil {
				return err
			}
			if out.IsJSON() {
				return out.Print(app.IO.Out, repository)
			}
			htmlURL := repository.HTMLURL()
			if htmlURL == "" {
				htmlURL = app.RepoHTMLURL(ref)
			}
			fmt.Fprintf(app.IO.Out, "Updated repository %s\n%s\n", ref, htmlURL)
			return nil
		},
	}
	out = cmdutil.AddOutputFlags(cmd)
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Repository description")
	cmd.Flags().BoolVar(&public, "public", false, "Make the repository public")
	cmd.Flags().BoolVar(&private, "private", false, "Make the repository private")
	cmd.Flags().StringVar(&defaultBranch, "default-branch", "", "Default branch name")
	return cmd
}

func resolveArg(ctx context.Context, app *cmdutil.App, repoFlag string, args []string) (bitbucket.RepoRef, error) {
	override := repoFlag
	if len(args) > 0 {
		override = args[0]
	}
	return app.ResolveRepo(ctx, override)
}

func defaultBranch(repository *bitbucket.Repository) string {
	if repository.MainBranch != nil {
		return repository.MainBranch.Label()
	}
	return ""
}

func visibility(private bool) string {
	if private {
		return "private"
	}
	return "public"
}
