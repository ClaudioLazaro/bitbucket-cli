// Command bk is a Bitbucket command line interface.
package main

import (
	"errors"
	"os"

	"github.com/ClaudioLazaro/bitbucket-cli/internal/cmdutil"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/iostreams"
	"github.com/ClaudioLazaro/bitbucket-cli/pkg/cmd/root"
)

func main() {
	ios := iostreams.System()
	f := cmdutil.NewFactory(ios)
	rootCmd := root.NewCmdRoot(f)

	args, err := cmdutil.ExpandAlias(os.Args[1:], f)
	if err != nil {
		cmdutil.PrintError(ios, err)
		os.Exit(1)
	}
	rootCmd.SetArgs(args)
	if err := rootCmd.Execute(); err != nil {
		cmdutil.PrintError(ios, err)
		var exitErr *cmdutil.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}
		os.Exit(1)
	}
}
