// Package root assembles the bk command tree.
package root

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/ClaudioLazaro/bitbucket-cli/internal/build"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/cmdutil"
	"github.com/ClaudioLazaro/bitbucket-cli/pkg/cmd/alias"
	"github.com/ClaudioLazaro/bitbucket-cli/pkg/cmd/api"
	"github.com/ClaudioLazaro/bitbucket-cli/pkg/cmd/auth"
	"github.com/ClaudioLazaro/bitbucket-cli/pkg/cmd/branch"
	"github.com/ClaudioLazaro/bitbucket-cli/pkg/cmd/browse"
	"github.com/ClaudioLazaro/bitbucket-cli/pkg/cmd/completion"
	configcmd "github.com/ClaudioLazaro/bitbucket-cli/pkg/cmd/config"
	"github.com/ClaudioLazaro/bitbucket-cli/pkg/cmd/download"
	"github.com/ClaudioLazaro/bitbucket-cli/pkg/cmd/gpgkey"
	"github.com/ClaudioLazaro/bitbucket-cli/pkg/cmd/pipeline"
	"github.com/ClaudioLazaro/bitbucket-cli/pkg/cmd/pr"
	"github.com/ClaudioLazaro/bitbucket-cli/pkg/cmd/repo"
	"github.com/ClaudioLazaro/bitbucket-cli/pkg/cmd/search"
	"github.com/ClaudioLazaro/bitbucket-cli/pkg/cmd/snippet"
	"github.com/ClaudioLazaro/bitbucket-cli/pkg/cmd/sshkey"
	"github.com/ClaudioLazaro/bitbucket-cli/pkg/cmd/version"
	"github.com/ClaudioLazaro/bitbucket-cli/pkg/cmd/webhook"
	"github.com/ClaudioLazaro/bitbucket-cli/pkg/cmd/workspace"
)

// NewCmdRoot builds the root command with every command group attached.
func NewCmdRoot(f *cmdutil.Factory) *cobra.Command {
	var hostname string
	root := &cobra.Command{
		Use:   "bk <command> <subcommand> [flags]",
		Short: "Work with Bitbucket from the command line",
		Long: `bk is a command line interface for Bitbucket Cloud with a pluggable
layer for Bitbucket Data Center.

Inside a git clone, bk infers the repository from the git remote. Elsewhere,
pass -R OWNER/REPO or set BK_REPO.

Authentication uses Bitbucket API tokens (Basic auth with your Atlassian email)
or repository, project, workspace, and Data Center access tokens (Bearer auth).
Bitbucket Cloud app passwords were removed in July 2026.`,
		Version:       build.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if hostname != "" {
				return os.Setenv("BK_HOST", hostname)
			}
			return nil
		},
	}
	root.SetOut(f.IO.Out)
	root.SetErr(f.IO.ErrOut)
	root.SetIn(f.IO.In)
	root.SetVersionTemplate("bk version {{.Version}}\n")
	root.CompletionOptions.DisableDefaultCmd = true
	root.PersistentFlags().StringVar(&hostname, "hostname", "", "Bitbucket host to use for this command")

	root.AddCommand(
		auth.NewCmdAuth(f),
		repo.NewCmdRepo(f),
		pr.NewCmdPR(f),
		pipeline.NewCmdPipeline(f),
		workspace.NewCmdWorkspace(f),
		snippet.NewCmdSnippet(f),
		download.NewCmdDownload(f),
		webhook.NewCmdWebhook(f),
		branch.NewCmdBranch(f),
		sshkey.NewCmdSSHKey(f),
		gpgkey.NewCmdGPGKey(f),
		search.NewCmdSearch(f),
		api.NewCmdAPI(f),
		browse.NewCmdBrowse(f),
		configcmd.NewCmdConfig(f),
		alias.NewCmdAlias(f),
		completion.NewCmdCompletion(f),
		version.NewCmdVersion(f),
	)
	return root
}
