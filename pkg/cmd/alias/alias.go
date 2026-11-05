// Package alias implements bk alias commands.
package alias

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ClaudioLazaro/bitbucket-cli/internal/cmdutil"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/config"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/tableprinter"
)

// NewCmdAlias returns the alias command group.
func NewCmdAlias(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alias",
		Short: "Create and manage command aliases",
	}
	cmd.AddCommand(newCmdList(f), newCmdSet(f), newCmdDelete(f))
	return cmd
}

func newCmdList(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List aliases",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := config.Load()
			if err != nil {
				return err
			}
			names := make([]string, 0, len(cfg.Aliases))
			for name := range cfg.Aliases {
				names = append(names, name)
			}
			sort.Strings(names)
			t := tableprinter.New(f.IO).Header("ALIAS", "EXPANSION")
			for _, name := range names {
				t.AddRow(name, cfg.Aliases[name])
			}
			return t.Render()
		},
	}
	return cmd
}

func newCmdSet(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set NAME EXPANSION...",
		Short: "Create an alias",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if cmdutil.IsBuiltinCommand(name) {
				return fmt.Errorf("%q is a built-in command and cannot be aliased", name)
			}
			if strings.ContainsAny(name, " \t") {
				return fmt.Errorf("alias names cannot contain spaces")
			}
			cfg, dir, err := config.Load()
			if err != nil {
				return err
			}
			expansion := strings.Join(args[1:], " ")
			cfg.SetAlias(name, expansion)
			if err := cfg.Save(dir); err != nil {
				return err
			}
			fmt.Fprintf(f.IO.Out, "Added alias %s -> %s\n", name, expansion)
			return nil
		},
	}
	return cmd
}

func newCmdDelete(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete NAME",
		Short: "Delete an alias",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, dir, err := config.Load()
			if err != nil {
				return err
			}
			name := args[0]
			if _, ok := cfg.Aliases[name]; !ok {
				return fmt.Errorf("alias %q does not exist", name)
			}
			delete(cfg.Aliases, name)
			if err := cfg.Save(dir); err != nil {
				return err
			}
			fmt.Fprintf(f.IO.Out, "Deleted alias %s\n", name)
			return nil
		},
	}
	return cmd
}
