// Package branch implements bk branch commands, including branch restrictions.
package branch

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/ClaudioLazaro/bitbucket-cli/internal/bitbucket"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/cmdutil"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/tableprinter"
)

// NewCmdBranch returns the branch command group.
func NewCmdBranch(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "branch",
		Short: "Work with branches and branch restrictions",
	}
	cmd.AddCommand(newCmdList(f), newCmdRestriction(f))
	return cmd
}

func newCmdList(f *cmdutil.Factory) *cobra.Command {
	var repoFlag string
	var limit int
	var out *cmdutil.OutputFlags
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List branches",
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
			branches, err := app.Client.ListBranches(ctx, ref.Owner, ref.Slug, limit)
			if err != nil {
				return err
			}
			if out.IsJSON() {
				return out.Print(app.IO.Out, branches)
			}
			t := tableprinter.New(app.IO).Header("NAME", "COMMIT", "DEFAULT")
			for _, item := range branches {
				hash := item.LatestCommit
				if item.Target != nil && item.Target.Hash != "" {
					hash = item.Target.Hash
				}
				if len(hash) > 8 {
					hash = hash[:8]
				}
				t.AddRow(item.Label(), hash, strconv.FormatBool(item.IsDefault))
			}
			return t.Render()
		},
	}
	out = cmdutil.AddOutputFlags(cmd)
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of branches (0 for all)")
	return cmd
}

func newCmdRestriction(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restriction",
		Short: "Manage branch restrictions",
	}
	cmd.AddCommand(newCmdRestrictionList(f), newCmdRestrictionCreate(f), newCmdRestrictionDelete(f))
	return cmd
}

func newCmdRestrictionList(f *cmdutil.Factory) *cobra.Command {
	var repoFlag string
	var limit int
	var out *cmdutil.OutputFlags
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List branch restrictions",
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
			rules, err := app.Client.ListBranchRestrictions(ctx, ref.Owner, ref.Slug, limit)
			if err != nil {
				return err
			}
			if out.IsJSON() {
				return out.Print(app.IO.Out, rules)
			}
			t := tableprinter.New(app.IO).Header("ID", "KIND", "PATTERN", "MATCH", "VALUE")
			for _, rule := range rules {
				value := ""
				if rule.Value != nil {
					value = strconv.Itoa(*rule.Value)
				}
				t.AddRow(strconv.Itoa(rule.ID), rule.Kind, rule.Pattern, rule.BranchMatchKind, value)
			}
			return t.Render()
		},
	}
	out = cmdutil.AddOutputFlags(cmd)
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	cmd.Flags().IntVarP(&limit, "limit", "L", 100, "Maximum number of rules (0 for all)")
	return cmd
}

func newCmdRestrictionCreate(f *cmdutil.Factory) *cobra.Command {
	var repoFlag, kind, pattern, matchKind string
	var value int
	var users, groups []string
	var out *cmdutil.OutputFlags
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a branch restriction",
		Long:  "Create a branch restriction. Common kinds: push, delete, force, restrict_merges, require_approvals_to_merge, require_passing_builds_to_merge.",
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
			uuids := make([]string, 0, len(users))
			for _, user := range users {
				uuid, err := app.Client.ResolveUserUUID(ctx, user)
				if err != nil {
					return fmt.Errorf("resolve user %q: %w", user, err)
				}
				uuids = append(uuids, uuid)
			}
			var valuePtr *int
			if cmd.Flags().Changed("value") {
				valuePtr = &value
			}
			rule, err := app.Client.CreateBranchRestriction(ctx, ref.Owner, ref.Slug, bitbucket.CreateBranchRestrictionInput{
				Kind:            kind,
				Pattern:         pattern,
				BranchMatchKind: matchKind,
				Value:           valuePtr,
				Users:           uuids,
				Groups:          groups,
			})
			if err != nil {
				return err
			}
			if out.IsJSON() {
				return out.Print(app.IO.Out, rule)
			}
			fmt.Fprintf(app.IO.Out, "Created branch restriction %d (%s on %s)\n", rule.ID, rule.Kind, rule.Pattern)
			return nil
		},
	}
	out = cmdutil.AddOutputFlags(cmd)
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	cmd.Flags().StringVar(&kind, "kind", "", "Restriction kind")
	cmd.Flags().StringVar(&pattern, "pattern", "", "Branch pattern")
	cmd.Flags().StringVar(&matchKind, "match", "glob", "Pattern match kind: glob, regex, branching_model")
	cmd.Flags().IntVar(&value, "value", 0, "Numeric value for the restriction (e.g. approvals required)")
	cmd.Flags().StringArrayVar(&users, "user", nil, "User allowed or required by the rule (repeatable)")
	cmd.Flags().StringArrayVar(&groups, "group", nil, "Group slug allowed or required by the rule (repeatable)")
	_ = cmd.MarkFlagRequired("kind")
	_ = cmd.MarkFlagRequired("pattern")
	return cmd
}

func newCmdRestrictionDelete(f *cmdutil.Factory) *cobra.Command {
	var repoFlag string
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete ID",
		Short: "Delete a branch restriction",
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
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid restriction ID %q", args[0])
			}
			if err := app.Confirm(fmt.Sprintf("Delete branch restriction %d?", id), yes); err != nil {
				return err
			}
			if err := app.Client.DeleteBranchRestriction(ctx, ref.Owner, ref.Slug, id); err != nil {
				return err
			}
			fmt.Fprintf(app.IO.Out, "Deleted branch restriction %d\n", id)
			return nil
		},
	}
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Confirm deletion")
	return cmd
}
