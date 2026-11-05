// Package search implements bk search commands.
package search

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ClaudioLazaro/bitbucket-cli/internal/cmdutil"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/tableprinter"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/text"
)

// NewCmdSearch returns the search command group.
func NewCmdSearch(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search Bitbucket",
	}
	cmd.AddCommand(newCmdCode(f))
	return cmd
}

func newCmdCode(f *cmdutil.Factory) *cobra.Command {
	var workspace string
	var limit int
	var out *cmdutil.OutputFlags
	cmd := &cobra.Command{
		Use:   "code QUERY",
		Short: "Search code in a workspace",
		Long: `Search code in a Cloud workspace.

Atlassian announced the deprecation of the workspace code search endpoint for
November 2026. Data Center has no equivalent public API.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := f.App()
			if err != nil {
				return err
			}
			if err := app.RequireCloud("code search"); err != nil {
				return err
			}
			ws := workspace
			if ws == "" {
				ws = app.Settings.Workspace
			}
			if ws == "" {
				return fmt.Errorf("workspace is required: pass --workspace")
			}
			query := strings.Join(args, " ")
			fmt.Fprintln(app.IO.ErrOut, "Warning: Atlassian announced the deprecation of this code search endpoint for November 2026.")
			results, err := app.Client.SearchCode(cmd.Context(), ws, query, limit)
			if err != nil {
				return err
			}
			if out.IsJSON() {
				return out.Print(app.IO.Out, results)
			}
			t := tableprinter.New(app.IO).Header("PATH", "LINE", "TEXT")
			for _, result := range results {
				t.AddRow(text.Truncate(result.File.Path, 60), strconv.Itoa(result.LineNumber()), text.Truncate(result.Line(), 80))
			}
			return t.Render()
		},
	}
	out = cmdutil.AddOutputFlags(cmd)
	cmd.Flags().StringVarP(&workspace, "workspace", "w", "", "Workspace slug")
	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of results (0 for all)")
	return cmd
}
