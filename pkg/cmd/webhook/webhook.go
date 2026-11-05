// Package webhook implements bk webhook commands.
package webhook

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ClaudioLazaro/bitbucket-cli/internal/cmdutil"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/tableprinter"
)

// NewCmdWebhook returns the webhook command group.
func NewCmdWebhook(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "webhook",
		Short: "Manage repository webhooks",
	}
	cmd.AddCommand(newCmdList(f), newCmdView(f), newCmdCreate(f), newCmdDelete(f))
	return cmd
}

func newCmdList(f *cmdutil.Factory) *cobra.Command {
	var repoFlag string
	var limit int
	var out *cmdutil.OutputFlags
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List webhooks",
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
			hooks, err := app.Client.ListWebhooks(ctx, ref.Owner, ref.Slug, limit)
			if err != nil {
				return err
			}
			if out.IsJSON() {
				return out.Print(app.IO.Out, hooks)
			}
			t := tableprinter.New(app.IO).Header("UUID", "ACTIVE", "URL", "EVENTS")
			for _, hook := range hooks {
				t.AddRow(hook.UUID, strconv.FormatBool(hook.Active), hook.URL, strings.Join(hook.Events, ","))
			}
			return t.Render()
		},
	}
	out = cmdutil.AddOutputFlags(cmd)
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of webhooks (0 for all)")
	return cmd
}

func newCmdView(f *cmdutil.Factory) *cobra.Command {
	var repoFlag string
	var out *cmdutil.OutputFlags
	cmd := &cobra.Command{
		Use:   "view UUID",
		Short: "View a webhook",
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
			hook, err := app.Client.GetWebhook(ctx, ref.Owner, ref.Slug, args[0])
			if err != nil {
				return err
			}
			if out.IsJSON() {
				return out.Print(app.IO.Out, hook)
			}
			fmt.Fprintf(app.IO.Out, "UUID:        %s\n", hook.UUID)
			fmt.Fprintf(app.IO.Out, "Description: %s\n", hook.Description)
			fmt.Fprintf(app.IO.Out, "URL:         %s\n", hook.URL)
			fmt.Fprintf(app.IO.Out, "Active:      %t\n", hook.Active)
			fmt.Fprintf(app.IO.Out, "Events:      %s\n", strings.Join(hook.Events, ", "))
			return nil
		},
	}
	out = cmdutil.AddOutputFlags(cmd)
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	return cmd
}

func newCmdCreate(f *cmdutil.Factory) *cobra.Command {
	var repoFlag, targetURL, description, secret string
	var events []string
	var out *cmdutil.OutputFlags
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a webhook",
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
			hook, err := app.Client.CreateWebhook(ctx, ref.Owner, ref.Slug, description, targetURL, events, secret)
			if err != nil {
				return err
			}
			if out.IsJSON() {
				return out.Print(app.IO.Out, hook)
			}
			fmt.Fprintf(app.IO.Out, "Created webhook %s\n", hook.UUID)
			return nil
		},
	}
	out = cmdutil.AddOutputFlags(cmd)
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	cmd.Flags().StringVar(&targetURL, "url", "", "Webhook target URL")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Webhook description")
	cmd.Flags().StringArrayVar(&events, "event", []string{"repo:push"}, "Event to subscribe to (repeatable)")
	cmd.Flags().StringVar(&secret, "secret", "", "Webhook secret")
	_ = cmd.MarkFlagRequired("url")
	return cmd
}

func newCmdDelete(f *cmdutil.Factory) *cobra.Command {
	var repoFlag string
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete UUID",
		Short: "Delete a webhook",
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
			if err := app.Confirm(fmt.Sprintf("Delete webhook %s?", args[0]), yes); err != nil {
				return err
			}
			if err := app.Client.DeleteWebhook(ctx, ref.Owner, ref.Slug, args[0]); err != nil {
				return err
			}
			fmt.Fprintf(app.IO.Out, "Deleted webhook %s\n", args[0])
			return nil
		},
	}
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Confirm deletion")
	return cmd
}
