// Package completion implements shell completion generation.
package completion

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ClaudioLazaro/bitbucket-cli/internal/cmdutil"
)

// NewCmdCompletion returns the completion command.
func NewCmdCompletion(f *cmdutil.Factory) *cobra.Command {
	var noDescriptions bool
	cmd := &cobra.Command{
		Use:       "completion [bash|zsh|fish|powershell]",
		Short:     "Generate shell completion scripts",
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		RunE: func(cmd *cobra.Command, args []string) error {
			root := cmd.Root()
			switch args[0] {
			case "bash":
				return root.GenBashCompletionV2(f.IO.Out, !noDescriptions)
			case "zsh":
				if noDescriptions {
					return root.GenZshCompletionNoDesc(f.IO.Out)
				}
				return root.GenZshCompletion(f.IO.Out)
			case "fish":
				return root.GenFishCompletion(f.IO.Out, !noDescriptions)
			case "powershell":
				if noDescriptions {
					return root.GenPowerShellCompletion(f.IO.Out)
				}
				return root.GenPowerShellCompletionWithDesc(f.IO.Out)
			default:
				return fmt.Errorf("unsupported shell %q", args[0])
			}
		},
	}
	cmd.Flags().BoolVar(&noDescriptions, "no-descriptions", false, "Disable completion descriptions")
	return cmd
}
