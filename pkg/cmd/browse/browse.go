// Package browse opens Bitbucket resources in a browser.
package browse

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ClaudioLazaro/bitbucket-cli/internal/cmdutil"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/git"
)

// NewCmdBrowse returns the browse command.
func NewCmdBrowse(f *cmdutil.Factory) *cobra.Command {
	var repoFlag, branch, commit string
	var pr, pipeline int
	var noOpen bool
	cmd := &cobra.Command{
		Use:   "browse [PATH]",
		Short: "Open a repository, file, pull request, or pipeline in a browser",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			app, err := f.App()
			if err != nil {
				return err
			}
			ref, err := app.ResolveRepo(ctx, repoFlag)
			if err != nil {
				return err
			}
			selected := 0
			if pr > 0 {
				selected++
			}
			if commit != "" {
				selected++
			}
			if pipeline > 0 {
				selected++
			}
			if selected > 1 {
				return fmt.Errorf("use only one of --pr, --commit, and --pipeline")
			}

			var target string
			switch {
			case pr > 0:
				target = app.PullRequestHTMLURL(ref, pr)
			case commit != "":
				target = app.RepoHTMLURL(ref) + "/commits/" + commit
			case pipeline > 0:
				target = app.PipelineHTMLURL(ref, pipeline)
			case branch != "" || len(args) > 0:
				name := branch
				if name == "" {
					name, err = git.CurrentBranch(ctx)
					if err != nil {
						return fmt.Errorf("could not determine the current branch: pass --branch")
					}
				}
				path := ""
				if len(args) > 0 {
					path = args[0]
				}
				target = app.SourceHTMLURL(ref, name, path)
			default:
				target = app.RepoHTMLURL(ref)
			}

			if noOpen || !app.IO.IsStdoutTTY() {
				fmt.Fprintln(app.IO.Out, target)
				return nil
			}
			return app.OpenBrowser(target)
		},
	}
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	cmd.Flags().StringVarP(&branch, "branch", "b", "", "Branch to browse (default current branch)")
	cmd.Flags().StringVar(&commit, "commit", "", "Commit hash to browse")
	cmd.Flags().IntVar(&pr, "pr", 0, "Pull request ID to browse")
	cmd.Flags().IntVar(&pipeline, "pipeline", 0, "Pipeline build number to browse")
	cmd.Flags().BoolVar(&noOpen, "no-open", false, "Print the URL instead of opening a browser")
	return cmd
}
