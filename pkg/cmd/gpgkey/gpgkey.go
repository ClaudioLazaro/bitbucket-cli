// Package gpgkey implements bk gpg-key commands.
package gpgkey

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ClaudioLazaro/bitbucket-cli/internal/cmdutil"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/tableprinter"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/text"
)

// NewCmdGPGKey returns the gpg-key command group.
func NewCmdGPGKey(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gpg-key",
		Short: "Manage GPG keys on your account",
	}
	cmd.AddCommand(newCmdList(f), newCmdAdd(f), newCmdDelete(f))
	return cmd
}

func newCmdList(f *cmdutil.Factory) *cobra.Command {
	var out *cmdutil.OutputFlags
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List GPG keys",
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
			keys, err := app.Client.ListGPGKeys(ctx, user)
			if err != nil {
				return err
			}
			if out.IsJSON() {
				return out.Print(app.IO.Out, keys)
			}
			t := tableprinter.New(app.IO).Header("FINGERPRINT", "KEY ID", "NAME", "CREATED")
			for _, key := range keys {
				t.AddRow(key.Fingerprint, key.KeyID, key.Name, text.RelativeTime(key.CreatedOn))
			}
			return t.Render()
		},
	}
	out = cmdutil.AddOutputFlags(cmd)
	return cmd
}

func newCmdAdd(f *cmdutil.Factory) *cobra.Command {
	var keyFile string
	var out *cmdutil.OutputFlags
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a GPG key",
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
			key, err := app.Client.AddGPGKey(ctx, user, strings.TrimSpace(string(data)))
			if err != nil {
				return err
			}
			if out.IsJSON() {
				return out.Print(app.IO.Out, key)
			}
			fmt.Fprintf(app.IO.Out, "Added GPG key %s\n", key.Fingerprint)
			return nil
		},
	}
	out = cmdutil.AddOutputFlags(cmd)
	cmd.Flags().StringVar(&keyFile, "key-file", "", "ASCII-armored public key file")
	_ = cmd.MarkFlagRequired("key-file")
	return cmd
}

func newCmdDelete(f *cmdutil.Factory) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete FINGERPRINT",
		Short: "Delete a GPG key",
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
			if err := app.Confirm(fmt.Sprintf("Delete GPG key %s?", args[0]), yes); err != nil {
				return err
			}
			if err := app.Client.DeleteGPGKey(ctx, user, args[0]); err != nil {
				return err
			}
			fmt.Fprintf(app.IO.Out, "Deleted GPG key %s\n", args[0])
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
