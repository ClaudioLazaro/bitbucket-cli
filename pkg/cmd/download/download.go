// Package download implements bk download commands.
package download

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/ClaudioLazaro/bitbucket-cli/internal/cmdutil"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/tableprinter"
)

// NewCmdDownload returns the download command group.
func NewCmdDownload(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download",
		Short: "Manage repository download files",
		Long:  "List, upload, and delete files in the repository downloads directory.",
	}
	cmd.AddCommand(newCmdList(f), newCmdUpload(f), newCmdDelete(f))
	return cmd
}

func newCmdList(f *cmdutil.Factory) *cobra.Command {
	var repoFlag string
	var limit int
	var out *cmdutil.OutputFlags
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List download files",
		Args:  cobra.NoArgs,
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
			files, err := app.Client.ListDownloads(ctx, ref.Owner, ref.Slug, limit)
			if err != nil {
				return err
			}
			if out.IsJSON() {
				return out.Print(app.IO.Out, files)
			}
			t := tableprinter.New(app.IO).Header("NAME", "SIZE", "URL")
			for _, file := range files {
				t.AddRow(file.Name, humanSize(file.Size), file.Links.Self.Href)
			}
			return t.Render()
		},
	}
	out = cmdutil.AddOutputFlags(cmd)
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of files (0 for all)")
	return cmd
}

func newCmdUpload(f *cmdutil.Factory) *cobra.Command {
	var repoFlag, name string
	var out *cmdutil.OutputFlags
	cmd := &cobra.Command{
		Use:   "upload FILE",
		Short: "Upload a file to the downloads directory",
		Args:  cobra.ExactArgs(1),
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
			file, err := os.Open(args[0])
			if err != nil {
				return err
			}
			defer file.Close()
			filename := name
			if filename == "" {
				filename = filepath.Base(args[0])
			}
			download, err := app.Client.UploadDownload(ctx, ref.Owner, ref.Slug, filename, file)
			if err != nil {
				return err
			}
			if out.IsJSON() {
				return out.Print(app.IO.Out, download)
			}
			fmt.Fprintf(app.IO.Out, "Uploaded %s\n", filename)
			return nil
		},
	}
	out = cmdutil.AddOutputFlags(cmd)
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	cmd.Flags().StringVar(&name, "name", "", "Remote file name (default: base name of FILE)")
	return cmd
}

func newCmdDelete(f *cmdutil.Factory) *cobra.Command {
	var repoFlag string
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete NAME",
		Short: "Delete a download file",
		Args:  cobra.ExactArgs(1),
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
			if err := app.Confirm(fmt.Sprintf("Delete download %s?", args[0]), yes); err != nil {
				return err
			}
			if err := app.Client.DeleteDownload(ctx, ref.Owner, ref.Slug, args[0]); err != nil {
				return err
			}
			fmt.Fprintf(app.IO.Out, "Deleted download %s\n", args[0])
			return nil
		},
	}
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Confirm deletion")
	return cmd
}

func humanSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit && exp < 3; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}
