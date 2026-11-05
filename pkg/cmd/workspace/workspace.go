// Package workspace implements bk workspace commands.
package workspace

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/ClaudioLazaro/bitbucket-cli/internal/cmdutil"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/tableprinter"
)

// NewCmdWorkspace returns the workspace command group.
func NewCmdWorkspace(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace",
		Short: "Work with Bitbucket workspaces",
		Long:  "List workspaces, view one workspace, and inspect its members.",
	}
	cmd.AddCommand(newCmdList(f), newCmdView(f), newCmdMembers(f))
	return cmd
}

func newCmdList(f *cmdutil.Factory) *cobra.Command {
	var limit int
	var out *cmdutil.OutputFlags
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List workspaces",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := f.App()
			if err != nil {
				return err
			}
			workspaces, err := app.Client.ListWorkspaces(cmd.Context(), limit)
			if err != nil {
				return err
			}
			if out.IsJSON() {
				return out.Print(app.IO.Out, workspaces)
			}
			t := tableprinter.New(app.IO).Header("SLUG", "NAME", "PRIVATE", "PERSONAL")
			for _, workspace := range workspaces {
				t.AddRow(workspace.Slug, workspace.Name, strconv.FormatBool(workspace.IsPrivate), strconv.FormatBool(workspace.IsPersonal))
			}
			return t.Render()
		},
	}
	out = cmdutil.AddOutputFlags(cmd)
	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of workspaces (0 for all)")
	return cmd
}

func newCmdView(f *cmdutil.Factory) *cobra.Command {
	var web bool
	var out *cmdutil.OutputFlags
	cmd := &cobra.Command{
		Use:   "view [SLUG]",
		Short: "View a workspace",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := f.App()
			if err != nil {
				return err
			}
			slug := ""
			if len(args) > 0 {
				slug = args[0]
			}
			if slug == "" {
				slug = app.Settings.Workspace
			}
			if slug == "" {
				return fmt.Errorf("workspace is required: pass SLUG or use --workspace")
			}
			workspace, err := app.Client.GetWorkspace(cmd.Context(), slug)
			if err != nil {
				return err
			}
			if web {
				return app.OpenBrowser(fmt.Sprintf("https://bitbucket.org/%s/", workspace.Slug))
			}
			if out.IsJSON() {
				return out.Print(app.IO.Out, workspace)
			}
			cs := app.IO.ColorScheme()
			fmt.Fprintln(app.IO.Out, cs.Bold(workspace.Name))
			fmt.Fprintf(app.IO.Out, "Slug:    %s\n", workspace.Slug)
			fmt.Fprintf(app.IO.Out, "UUID:    %s\n", workspace.UUID)
			fmt.Fprintf(app.IO.Out, "Private: %t\n", workspace.IsPrivate)
			fmt.Fprintf(app.IO.Out, "URL:     https://bitbucket.org/%s/\n", workspace.Slug)
			return nil
		},
	}
	out = cmdutil.AddOutputFlags(cmd)
	cmd.Flags().BoolVarP(&web, "web", "w", false, "Open the workspace in a browser")
	return cmd
}

func newCmdMembers(f *cmdutil.Factory) *cobra.Command {
	var limit int
	var out *cmdutil.OutputFlags
	cmd := &cobra.Command{
		Use:   "members [SLUG]",
		Short: "List workspace members",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := f.App()
			if err != nil {
				return err
			}
			slug := ""
			if len(args) > 0 {
				slug = args[0]
			}
			if slug == "" {
				slug = app.Settings.Workspace
			}
			if slug == "" {
				return fmt.Errorf("workspace is required: pass SLUG or use --workspace")
			}
			members, err := app.Client.ListWorkspaceMembers(cmd.Context(), slug, limit)
			if err != nil {
				return err
			}
			if out.IsJSON() {
				return out.Print(app.IO.Out, members)
			}
			t := tableprinter.New(app.IO).Header("NAME", "UUID")
			for _, member := range members {
				t.AddRow(member.User.Label(), member.User.UUID)
			}
			return t.Render()
		},
	}
	out = cmdutil.AddOutputFlags(cmd)
	cmd.Flags().IntVarP(&limit, "limit", "L", 100, "Maximum number of members (0 for all)")
	return cmd
}
