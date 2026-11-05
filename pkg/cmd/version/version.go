// Package version implements bk version.
package version

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/ClaudioLazaro/bitbucket-cli/internal/build"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/cmdutil"
)

// NewCmdVersion returns the version command.
func NewCmdVersion(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version of bk",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(f.IO.Out, "bk version %s\n", build.Version)
			fmt.Fprintf(f.IO.Out, "commit: %s\n", build.Commit)
			fmt.Fprintf(f.IO.Out, "built:  %s\n", build.Date)
			fmt.Fprintf(f.IO.Out, "go:     %s\n", runtime.Version())
		},
	}
}
