// Package sshkey implements bk ssh-key commands.
package sshkey

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ClaudioLazaro/bitbucket-cli/internal/cmdutil"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/tableprinter"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/text"
)

// NewCmdSSHKey returns the ssh-key command group.
func NewCmdSSHKey(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ssh-key",
		Short: "Manage SSH keys on your account",
	}
	cmd.AddCommand(newCmdList(f), newCmdAdd(f), newCmdDelete(f))
	return cmd
}

func newCmdList(f *cmdutil.Factory) *cobra.Command {
	var out *cmdutil.OutputFlags
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List SSH keys",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			app, err := f.App()
			if err != nil {
				return err
			}
			user, err := currentUserUUID(ctx, app)
			if err != nil {
				return err
			}
			keys, err := app.Client.ListSSHKeys(ctx, user)
			if err != nil {
				return err
			}
			if out.IsJSON() {
				return out.Print(app.IO.Out, keys)
			}
			t := tableprinter.New(app.IO).Header("ID", "LABEL", "FINGERPRINT", "CREATED")
			for _, key := range keys {
				t.AddRow(strconv.Itoa(key.ID), key.Label, key.Fingerprint, text.RelativeTime(key.CreatedOn))
			}
			return t.Render()
		},
	}
	out = cmdutil.AddOutputFlags(cmd)
	return cmd
}

func newCmdAdd(f *cmdutil.Factory) *cobra.Command {
	var title, keyFile string
	var out *cmdutil.OutputFlags
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add an SSH key",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			app, err := f.App()
			if err != nil {
				return err
			}
			user, err := currentUserUUID(ctx, app)
			if err != nil {
				return err
			}
			data, err := os.ReadFile(keyFile)
			if err != nil {
				return err
			}
			key, err := app.Client.AddSSHKey(ctx, user, title, strings.TrimSpace(string(data)))
			if err != nil {
				return err
			}
			if out.IsJSON() {
				return out.Print(app.IO.Out, key)
			}
			fmt.Fprintf(app.IO.Out, "Added SSH key %d (%s)\n", key.ID, key.Fingerprint)
			return nil
		},
	}
	out = cmdutil.AddOutputFlags(cmd)
	cmd.Flags().StringVar(&title, "title", "bk", "Key label")
	cmd.Flags().StringVar(&keyFile, "key-file", "", "Public key file")
	_ = cmd.MarkFlagRequired("key-file")
	return cmd
}

func newCmdDelete(f *cmdutil.Factory) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete ID",
		Short: "Delete an SSH key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			app, err := f.App()
			if err != nil {
				return err
			}
			user, err := currentUserUUID(ctx, app)
			if err != nil {
				return err
			}
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid SSH key ID %q", args[0])
			}
			if err := app.Confirm(fmt.Sprintf("Delete SSH key %d?", id), yes); err != nil {
				return err
			}
			if err := app.Client.DeleteSSHKey(ctx, user, id); err != nil {
				return err
			}
			fmt.Fprintf(app.IO.Out, "Deleted SSH key %d\n", id)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Confirm deletion")
	return cmd
}

func currentUserUUID(ctx context.Context, app *cmdutil.App) (string, error) {
	user, err := app.Client.GetCurrentUser(ctx)
	if err != nil {
		return "", err
	}
	if user.UUID == "" {
		return "", fmt.Errorf("could not determine your account UUID")
	}
	return user.UUID, nil
}
