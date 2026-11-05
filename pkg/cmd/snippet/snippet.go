// Package snippet implements bk snippet commands.
package snippet

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ClaudioLazaro/bitbucket-cli/internal/bitbucket"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/cmdutil"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/git"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/tableprinter"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/text"
)

// NewCmdSnippet returns the snippet command group.
func NewCmdSnippet(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snippet",
		Short: "Work with Bitbucket snippets",
		Long:  "List, view, create, clone, and delete snippets in a workspace.",
	}
	cmd.AddCommand(newCmdList(f), newCmdView(f), newCmdCreate(f), newCmdClone(f), newCmdDelete(f))
	return cmd
}

func newCmdList(f *cmdutil.Factory) *cobra.Command {
	var workspace, role string
	var limit int
	var out *cmdutil.OutputFlags
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List snippets",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := f.App()
			if err != nil {
				return err
			}
			ws, err := resolveWorkspace(app, workspace)
			if err != nil {
				return err
			}
			snippets, err := app.Client.ListSnippets(cmd.Context(), ws, role, limit)
			if err != nil {
				return err
			}
			if out.IsJSON() {
				return out.Print(app.IO.Out, snippets)
			}
			t := tableprinter.New(app.IO).Header("ID", "TITLE", "PRIVATE", "UPDATED")
			for _, snippet := range snippets {
				t.AddRow(snippet.ID.String(), text.Truncate(snippet.Title, 60), strconv.FormatBool(snippet.IsPrivate), text.RelativeTime(snippet.UpdatedOn))
			}
			return t.Render()
		},
	}
	out = cmdutil.AddOutputFlags(cmd)
	cmd.Flags().StringVarP(&workspace, "workspace", "w", "", "Workspace slug")
	cmd.Flags().StringVar(&role, "role", "", "Filter by role: owner, contributor, member")
	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of snippets (0 for all)")
	return cmd
}

func newCmdView(f *cmdutil.Factory) *cobra.Command {
	var workspace string
	var out *cmdutil.OutputFlags
	cmd := &cobra.Command{
		Use:   "view ID",
		Short: "View a snippet",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			app, err := f.App()
			if err != nil {
				return err
			}
			ws, err := resolveWorkspace(app, workspace)
			if err != nil {
				return err
			}
			snippet, err := app.Client.GetSnippet(ctx, ws, args[0])
			if err != nil {
				return err
			}
			if out.IsJSON() {
				return out.Print(app.IO.Out, snippet)
			}
			cs := app.IO.ColorScheme()
			fmt.Fprintln(app.IO.Out, cs.Bold(snippet.Title))
			fmt.Fprintf(app.IO.Out, "ID: %s | %s | private: %t\n", snippet.ID.String(), snippet.SCM, snippet.IsPrivate)
			if snippet.Links.HTML.Href != "" {
				fmt.Fprintln(app.IO.Out, snippet.Links.HTML.Href)
			}
			names := make([]string, 0, len(snippet.Files))
			for name := range snippet.Files {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				content := snippet.Files[name].Content
				if content == "" {
					data, err := app.Client.GetSnippetFile(ctx, ws, snippet.ID.String(), name)
					if err != nil {
						return err
					}
					content = string(data)
				}
				fmt.Fprintf(app.IO.Out, "\n--- %s ---\n%s\n", name, content)
			}
			return nil
		},
	}
	out = cmdutil.AddOutputFlags(cmd)
	cmd.Flags().StringVarP(&workspace, "workspace", "w", "", "Workspace slug")
	return cmd
}

func newCmdCreate(f *cmdutil.Factory) *cobra.Command {
	var workspace, title string
	var files []string
	var private bool
	var out *cmdutil.OutputFlags
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a snippet",
		Long:  "Create a snippet from one or more local files. Use --file NAME=PATH to control the snippet file name.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := f.App()
			if err != nil {
				return err
			}
			ws, err := resolveWorkspace(app, workspace)
			if err != nil {
				return err
			}
			if len(files) == 0 {
				return fmt.Errorf("at least one --file is required")
			}
			if title == "" {
				title, err = app.PromptText("Title")
				if err != nil || strings.TrimSpace(title) == "" {
					return fmt.Errorf("title is required: pass --title")
				}
			}
			contents := map[string]string{}
			for _, spec := range files {
				name, path := parseFileSpec(spec)
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				contents[name] = string(data)
			}
			snippet, err := app.Client.CreateSnippet(cmd.Context(), ws, bitbucket.CreateSnippetInput{
				Title:     title,
				IsPrivate: private,
				Files:     contents,
			})
			if err != nil {
				return err
			}
			if out.IsJSON() {
				return out.Print(app.IO.Out, snippet)
			}
			htmlURL := snippet.Links.HTML.Href
			if htmlURL == "" {
				htmlURL = fmt.Sprintf("https://bitbucket.org/snippets/%s/%s", ws, snippet.ID.String())
			}
			fmt.Fprintf(app.IO.Out, "Created snippet %s\n%s\n", snippet.ID.String(), htmlURL)
			return nil
		},
	}
	out = cmdutil.AddOutputFlags(cmd)
	cmd.Flags().StringVarP(&workspace, "workspace", "w", "", "Workspace slug")
	cmd.Flags().StringVarP(&title, "title", "t", "", "Snippet title")
	cmd.Flags().StringArrayVarP(&files, "file", "f", nil, "File to include (PATH or NAME=PATH, repeatable)")
	cmd.Flags().BoolVar(&private, "private", false, "Create a private snippet")
	return cmd
}

func newCmdClone(f *cmdutil.Factory) *cobra.Command {
	var workspace string
	cmd := &cobra.Command{
		Use:   "clone ID [DIRECTORY]",
		Short: "Clone a snippet as a git repository",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := f.App()
			if err != nil {
				return err
			}
			ws, err := resolveWorkspace(app, workspace)
			if err != nil {
				return err
			}
			cloneURL := fmt.Sprintf("https://bitbucket.org/snippets/%s/%s.git", ws, args[0])
			dir := args[0]
			if len(args) == 2 {
				dir = args[1]
			}
			if err := git.Clone(cmd.Context(), cloneURL, dir, app.IO.Out, app.IO.ErrOut); err != nil {
				return err
			}
			fmt.Fprintf(app.IO.Out, "Cloned to %s\n", dir)
			return nil
		},
	}
	cmd.Flags().StringVarP(&workspace, "workspace", "w", "", "Workspace slug")
	return cmd
}

func newCmdDelete(f *cmdutil.Factory) *cobra.Command {
	var workspace string
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete ID",
		Short: "Delete a snippet",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := f.App()
			if err != nil {
				return err
			}
			ws, err := resolveWorkspace(app, workspace)
			if err != nil {
				return err
			}
			if err := app.Confirm(fmt.Sprintf("Delete snippet %s?", args[0]), yes); err != nil {
				return err
			}
			if err := app.Client.DeleteSnippet(cmd.Context(), ws, args[0]); err != nil {
				return err
			}
			fmt.Fprintf(app.IO.Out, "Deleted snippet %s\n", args[0])
			return nil
		},
	}
	cmd.Flags().StringVarP(&workspace, "workspace", "w", "", "Workspace slug")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Confirm deletion")
	return cmd
}

func resolveWorkspace(app *cmdutil.App, override string) (string, error) {
	if override != "" {
		return override, nil
	}
	if app.Settings.Workspace != "" {
		return app.Settings.Workspace, nil
	}
	return "", fmt.Errorf("workspace is required: pass --workspace")
}

func parseFileSpec(spec string) (name, path string) {
	if index := strings.Index(spec, "="); index > 0 {
		return spec[:index], spec[index+1:]
	}
	return filepath.Base(spec), spec
}
