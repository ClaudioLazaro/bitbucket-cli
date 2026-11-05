// Package pr implements bk pr commands.
package pr

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ClaudioLazaro/bitbucket-cli/internal/bitbucket"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/cmdutil"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/git"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/tableprinter"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/text"
)

// NewCmdPR returns the pr command group.
func NewCmdPR(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pr",
		Short: "Work with pull requests",
		Long:  "List, view, create, review, merge, decline, and inspect pull requests.",
	}
	cmd.AddCommand(
		newCmdList(f),
		newCmdView(f),
		newCmdCreate(f),
		newCmdCheckout(f),
		newCmdMerge(f),
		newCmdDecline(f),
		newCmdReopen(f),
		newCmdApprove(f, true),
		newCmdApprove(f, false),
		newCmdRequestChanges(f),
		newCmdComment(f),
		newCmdComments(f),
		newCmdDiff(f),
		newCmdChecks(f),
		newCmdCommits(f),
	)
	return cmd
}

func newCmdList(f *cmdutil.Factory) *cobra.Command {
	var repoFlag, state, author, reviewer, base, head string
	var limit int
	var web bool
	var out *cmdutil.OutputFlags
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List pull requests",
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
			if web {
				return app.OpenBrowser(app.RepoHTMLURL(ref) + "/pull-requests/")
			}
			q := url.Values{}
			switch strings.ToLower(state) {
			case "", "open":
				q.Set("state", "OPEN")
			case "merged":
				q.Set("state", "MERGED")
			case "declined":
				q.Set("state", "DECLINED")
			case "all":
			default:
				return fmt.Errorf("invalid --state %q: use open, merged, declined, or all", state)
			}
			var clauses []string
			if author != "" {
				uuid, err := app.Client.ResolveUserUUID(ctx, author)
				if err != nil {
					return err
				}
				clauses = append(clauses, fmt.Sprintf(`author.uuid="%s"`, uuid))
			}
			if reviewer != "" {
				uuid, err := app.Client.ResolveUserUUID(ctx, reviewer)
				if err != nil {
					return err
				}
				clauses = append(clauses, fmt.Sprintf(`reviewers.uuid="%s"`, uuid))
			}
			if base != "" {
				clauses = append(clauses, fmt.Sprintf(`destination.branch.name="%s"`, escapeQueryValue(base)))
			}
			if head != "" {
				clauses = append(clauses, fmt.Sprintf(`source.branch.name="%s"`, escapeQueryValue(head)))
			}
			if len(clauses) > 0 {
				q.Set("q", strings.Join(clauses, " AND "))
			}
			prs, err := app.Client.ListPullRequests(ctx, ref.Owner, ref.Slug, q, limit)
			if err != nil {
				return err
			}
			if out.IsJSON() {
				return out.Print(app.IO.Out, prs)
			}
			t := tableprinter.New(app.IO).Header("ID", "TITLE", "BRANCH", "STATE", "AUTHOR", "UPDATED")
			for _, pr := range prs {
				t.AddRow(
					strconv.Itoa(pr.ID),
					text.Truncate(pr.Title, 60),
					text.Truncate(pr.SourceBranch(), 30),
					strings.ToLower(pr.State),
					pr.Author.Label(),
					text.RelativeTime(pr.UpdatedOn),
				)
			}
			return t.Render()
		},
	}
	out = cmdutil.AddOutputFlags(cmd)
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	cmd.Flags().StringVar(&state, "state", "open", "Filter by state: open, merged, declined, all")
	cmd.Flags().StringVarP(&author, "author", "a", "", "Filter by author (@me or username)")
	cmd.Flags().StringVar(&reviewer, "reviewer", "", "Filter by reviewer (@me or username)")
	cmd.Flags().StringVarP(&base, "base", "B", "", "Filter by destination branch")
	cmd.Flags().StringVarP(&head, "head", "H", "", "Filter by source branch")
	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of pull requests (0 for all)")
	cmd.Flags().BoolVarP(&web, "web", "w", false, "Open the pull request list in a browser")
	return cmd
}

func newCmdView(f *cmdutil.Factory) *cobra.Command {
	var repoFlag string
	var web, withComments bool
	var limit int
	var out *cmdutil.OutputFlags
	cmd := &cobra.Command{
		Use:   "view [ID]",
		Short: "View a pull request",
		Args:  cobra.MaximumNArgs(1),
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
			id, pr, err := resolvePullRequest(ctx, app, ref, firstArg(args))
			if err != nil {
				return err
			}
			if web {
				return app.OpenBrowser(pullRequestURL(app, ref, pr))
			}
			var comments []bitbucket.Comment
			if withComments {
				comments, err = app.Client.ListComments(ctx, ref.Owner, ref.Slug, id, limit)
				if err != nil {
					return err
				}
			}
			if out.IsJSON() {
				return out.Print(app.IO.Out, struct {
					*bitbucket.PullRequest
					Comments []bitbucket.Comment `json:"comments,omitempty"`
				}{PullRequest: pr, Comments: comments})
			}

			cs := app.IO.ColorScheme()
			fmt.Fprintf(app.IO.Out, "%s #%d %s\n", cs.Magenta("pull request"), pr.ID, cs.Bold(pr.Title))
			meta := []string{
				strings.ToLower(pr.State),
				pr.Author.Label(),
				pr.SourceBranch() + " -> " + pr.DestinationBranch(),
				text.RelativeTime(pr.UpdatedOn),
			}
			fmt.Fprintln(app.IO.Out, strings.Join(nonEmpty(meta), " | "))
			fmt.Fprintln(app.IO.Out, pullRequestURL(app, ref, pr))
			if body := strings.TrimSpace(pr.Body()); body != "" {
				fmt.Fprintln(app.IO.Out)
				fmt.Fprintln(app.IO.Out, body)
			}
			if withComments {
				fmt.Fprintf(app.IO.Out, "\nComments (%d)\n", len(comments))
				for _, comment := range comments {
					if comment.Deleted {
						continue
					}
					fmt.Fprintf(app.IO.Out, "\n%s commented %s\n", comment.User.Label(), text.RelativeTime(comment.CreatedOn))
					fmt.Fprintln(app.IO.Out, text.Indent(strings.TrimSpace(comment.Content.Raw), "  "))
				}
			}
			return nil
		},
	}
	out = cmdutil.AddOutputFlags(cmd)
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	cmd.Flags().BoolVarP(&web, "web", "w", false, "Open the pull request in a browser")
	cmd.Flags().BoolVarP(&withComments, "comments", "c", false, "Include comments")
	cmd.Flags().IntVarP(&limit, "limit", "L", 100, "Maximum number of comments")
	return cmd
}

func newCmdCreate(f *cmdutil.Factory) *cobra.Command {
	var repoFlag, title, body, bodyFile, base, head string
	var closeSource, draft, fill, web bool
	var reviewers []string
	var out *cmdutil.OutputFlags
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a pull request",
		Long:  "Open a pull request from the current branch (or --head) into the repository default branch (or --base).",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			app, err := f.App()
			if err != nil {
				return err
			}
			if err := app.RequireCloud("pull requests"); err != nil {
				return err
			}
			ref, err := app.ResolveRepo(ctx, repoFlag)
			if err != nil {
				return err
			}
			repository, err := app.Client.GetRepository(ctx, ref.Owner, ref.Slug)
			if err != nil {
				return err
			}

			source := head
			if source == "" {
				source, err = git.CurrentBranch(ctx)
				if err != nil {
					return fmt.Errorf("could not determine the current branch: %w", err)
				}
			}
			destination := base
			if destination == "" {
				destination = defaultBranch(repository)
			}
			if destination == "" {
				return fmt.Errorf("could not determine the default branch: pass --base")
			}
			if source == destination {
				return fmt.Errorf("source and destination branches are both %q", source)
			}

			if fill && title == "" {
				title, err = git.LastCommitSubject(ctx)
				if err != nil {
					return err
				}
			}
			if title == "" {
				title, err = app.PromptText("Title")
				if err != nil || strings.TrimSpace(title) == "" {
					return fmt.Errorf("title is required: pass --title")
				}
				title = strings.TrimSpace(title)
			}

			if bodyFile != "" {
				body, err = readBody(bodyFile, app.IO.In)
				if err != nil {
					return err
				}
			} else if fill && body == "" {
				body, _ = git.LastCommitBody(ctx)
			}

			uuids := make([]string, 0, len(reviewers))
			for _, reviewer := range reviewers {
				uuid, err := app.Client.ResolveUserUUID(ctx, reviewer)
				if err != nil {
					return fmt.Errorf("resolve reviewer %q: %w", reviewer, err)
				}
				uuids = append(uuids, uuid)
			}

			pr, err := app.Client.CreatePullRequest(ctx, ref.Owner, ref.Slug, bitbucket.CreatePullRequestInput{
				Title:             title,
				Description:       body,
				SourceBranch:      source,
				DestinationBranch: destination,
				CloseSourceBranch: closeSource,
				Draft:             draft,
				Reviewers:         uuids,
			})
			if err != nil {
				return err
			}
			if out.IsJSON() {
				return out.Print(app.IO.Out, pr)
			}
			htmlURL := pullRequestURL(app, ref, pr)
			fmt.Fprintf(app.IO.Out, "Created pull request #%d\n%s\n", pr.ID, htmlURL)
			if web {
				return app.OpenBrowser(htmlURL)
			}
			return nil
		},
	}
	out = cmdutil.AddOutputFlags(cmd)
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	cmd.Flags().StringVarP(&title, "title", "t", "", "Pull request title")
	cmd.Flags().StringVarP(&body, "body", "b", "", "Pull request description")
	cmd.Flags().StringVarP(&bodyFile, "body-file", "F", "", "Read the description from a file (- for stdin)")
	cmd.Flags().StringVarP(&base, "base", "B", "", "Destination branch")
	cmd.Flags().StringVarP(&head, "head", "H", "", "Source branch")
	cmd.Flags().StringArrayVarP(&reviewers, "reviewer", "r", nil, "Request a review from a username (repeatable)")
	cmd.Flags().BoolVar(&closeSource, "close-source-branch", false, "Close the source branch after merging")
	cmd.Flags().BoolVar(&draft, "draft", false, "Create as a draft pull request")
	cmd.Flags().BoolVarP(&fill, "fill", "f", false, "Use the last commit subject and body")
	cmd.Flags().BoolVarP(&web, "web", "w", false, "Open the pull request in a browser")
	return cmd
}

func newCmdCheckout(f *cmdutil.Factory) *cobra.Command {
	var repoFlag string
	cmd := &cobra.Command{
		Use:   "checkout [ID]",
		Short: "Check out a pull request branch",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			app, err := f.App()
			if err != nil {
				return err
			}
			if err := app.RequireCloud("pull request checkout"); err != nil {
				return err
			}
			ref, err := app.ResolveRepo(ctx, repoFlag)
			if err != nil {
				return err
			}
			id, pr, err := resolvePullRequest(ctx, app, ref, firstArg(args))
			if err != nil {
				return err
			}
			branch := pr.SourceBranch()
			if branch == "" {
				return fmt.Errorf("pull request #%d has no source branch", id)
			}
			if git.BranchExists(ctx, branch) {
				if err := git.Checkout(ctx, branch); err != nil {
					return err
				}
				fmt.Fprintf(app.IO.Out, "Switched to branch %s (pull request #%d)\n", branch, id)
				return nil
			}

			remote := "origin"
			if pr.Source.Repository != nil && pr.Destination != nil && pr.Destination.Repository != nil {
				if !strings.EqualFold(pr.Source.Repository.UUID, pr.Destination.Repository.UUID) {
					if cloneURL := pr.Source.Repository.CloneURL("https"); cloneURL != "" {
						remote = cloneURL
					}
				}
			}
			if err := git.Fetch(ctx, remote, branch+":"+branch); err != nil {
				return err
			}
			if err := git.Checkout(ctx, branch); err != nil {
				return err
			}
			fmt.Fprintf(app.IO.Out, "Switched to branch %s (pull request #%d)\n", branch, id)
			return nil
		},
	}
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	return cmd
}

func newCmdMerge(f *cmdutil.Factory) *cobra.Command {
	var repoFlag, strategy, message string
	var closeSource bool
	var out *cmdutil.OutputFlags
	cmd := &cobra.Command{
		Use:   "merge [ID]",
		Short: "Merge a pull request",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			app, err := f.App()
			if err != nil {
				return err
			}
			if err := app.RequireCloud("pull request merge"); err != nil {
				return err
			}
			ref, err := app.ResolveRepo(ctx, repoFlag)
			if err != nil {
				return err
			}
			switch strategy {
			case "merge_commit", "squash", "fast_forward", "squash_fast_forward", "rebase_fast_forward", "rebase_merge":
			default:
				return fmt.Errorf("invalid --strategy %q", strategy)
			}
			id, _, err := resolvePullRequest(ctx, app, ref, firstArg(args))
			if err != nil {
				return err
			}
			pr, err := app.Client.MergePullRequest(ctx, ref.Owner, ref.Slug, id, strategy, message, closeSource)
			if err != nil {
				return err
			}
			if out.IsJSON() {
				return out.Print(app.IO.Out, pr)
			}
			htmlURL := app.PullRequestHTMLURL(ref, id)
			if pr != nil && pr.Links.HTML.Href != "" {
				htmlURL = pr.Links.HTML.Href
			}
			fmt.Fprintf(app.IO.Out, "Merged pull request #%d\n%s\n", id, htmlURL)
			return nil
		},
	}
	out = cmdutil.AddOutputFlags(cmd)
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	cmd.Flags().StringVarP(&strategy, "strategy", "s", "merge_commit", "Merge strategy: merge_commit, squash, fast_forward")
	cmd.Flags().StringVarP(&message, "message", "m", "", "Merge commit message")
	cmd.Flags().BoolVar(&closeSource, "close-source-branch", false, "Close the source branch after merging")
	return cmd
}

func newCmdDecline(f *cmdutil.Factory) *cobra.Command {
	var repoFlag, reason string
	cmd := &cobra.Command{
		Use:   "decline [ID]",
		Short: "Decline a pull request",
		Args:  cobra.MaximumNArgs(1),
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
			id, _, err := resolvePullRequest(ctx, app, ref, firstArg(args))
			if err != nil {
				return err
			}
			if _, err := app.Client.DeclinePullRequest(ctx, ref.Owner, ref.Slug, id); err != nil {
				return err
			}
			fmt.Fprintf(app.IO.Out, "Declined pull request #%d\n", id)
			if reason != "" {
				if _, err := app.Client.AddComment(ctx, ref.Owner, ref.Slug, id, "Declined: "+reason); err != nil {
					fmt.Fprintf(app.IO.ErrOut, "Warning: declined, but could not add the reason as a comment: %v\n", err)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	cmd.Flags().StringVar(&reason, "reason", "", "Reason for declining (posted as a comment)")
	return cmd
}

func newCmdReopen(f *cmdutil.Factory) *cobra.Command {
	var repoFlag string
	cmd := &cobra.Command{
		Use:   "reopen [ID]",
		Short: "Reopen a declined pull request",
		Args:  cobra.MaximumNArgs(1),
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
			id, _, err := resolvePullRequest(ctx, app, ref, firstArg(args))
			if err != nil {
				return err
			}
			if _, err := app.Client.SetPullRequestState(ctx, ref.Owner, ref.Slug, id, "OPEN"); err != nil {
				return err
			}
			fmt.Fprintf(app.IO.Out, "Reopened pull request #%d\n", id)
			return nil
		},
	}
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	return cmd
}

func newCmdApprove(f *cmdutil.Factory, approve bool) *cobra.Command {
	var repoFlag string
	use, short := "approve [ID]", "Approve a pull request"
	if !approve {
		use, short = "unapprove [ID]", "Remove your approval from a pull request"
	}
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.MaximumNArgs(1),
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
			id, _, err := resolvePullRequest(ctx, app, ref, firstArg(args))
			if err != nil {
				return err
			}
			if err := app.Client.SetReviewAction(ctx, ref.Owner, ref.Slug, id, "approve", approve); err != nil {
				return err
			}
			if approve {
				fmt.Fprintf(app.IO.Out, "Approved pull request #%d\n", id)
			} else {
				fmt.Fprintf(app.IO.Out, "Removed approval from pull request #%d\n", id)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	return cmd
}

func newCmdRequestChanges(f *cmdutil.Factory) *cobra.Command {
	var repoFlag string
	var clear bool
	cmd := &cobra.Command{
		Use:   "request-changes [ID]",
		Short: "Request changes on a pull request",
		Args:  cobra.MaximumNArgs(1),
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
			id, _, err := resolvePullRequest(ctx, app, ref, firstArg(args))
			if err != nil {
				return err
			}
			if err := app.Client.SetReviewAction(ctx, ref.Owner, ref.Slug, id, "request-changes", !clear); err != nil {
				return err
			}
			if clear {
				fmt.Fprintf(app.IO.Out, "Cleared change request on pull request #%d\n", id)
			} else {
				fmt.Fprintf(app.IO.Out, "Requested changes on pull request #%d\n", id)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	cmd.Flags().BoolVar(&clear, "clear", false, "Clear your change request")
	return cmd
}

func newCmdComment(f *cmdutil.Factory) *cobra.Command {
	var repoFlag, body, bodyFile string
	cmd := &cobra.Command{
		Use:   "comment [ID]",
		Short: "Comment on a pull request",
		Args:  cobra.MaximumNArgs(1),
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
			id, _, err := resolvePullRequest(ctx, app, ref, firstArg(args))
			if err != nil {
				return err
			}
			if bodyFile != "" {
				body, err = readBody(bodyFile, app.IO.In)
				if err != nil {
					return err
				}
			}
			if strings.TrimSpace(body) == "" {
				body, err = app.PromptText("Comment")
				if err != nil {
					return fmt.Errorf("comment body is required: pass --body or --body-file")
				}
			}
			if strings.TrimSpace(body) == "" {
				return fmt.Errorf("comment body is empty")
			}
			comment, err := app.Client.AddComment(ctx, ref.Owner, ref.Slug, id, body)
			if err != nil {
				return err
			}
			fmt.Fprintf(app.IO.Out, "Commented on pull request #%d\n", id)
			if comment.Links.HTML.Href != "" {
				fmt.Fprintln(app.IO.Out, comment.Links.HTML.Href)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	cmd.Flags().StringVarP(&body, "body", "b", "", "Comment body")
	cmd.Flags().StringVarP(&bodyFile, "body-file", "F", "", "Read the comment from a file (- for stdin)")
	return cmd
}

func newCmdComments(f *cmdutil.Factory) *cobra.Command {
	var repoFlag string
	var limit int
	var out *cmdutil.OutputFlags
	cmd := &cobra.Command{
		Use:   "comments [ID]",
		Short: "List pull request comments",
		Args:  cobra.MaximumNArgs(1),
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
			id, _, err := resolvePullRequest(ctx, app, ref, firstArg(args))
			if err != nil {
				return err
			}
			comments, err := app.Client.ListComments(ctx, ref.Owner, ref.Slug, id, limit)
			if err != nil {
				return err
			}
			if out.IsJSON() {
				return out.Print(app.IO.Out, comments)
			}
			for _, comment := range comments {
				if comment.Deleted {
					continue
				}
				fmt.Fprintf(app.IO.Out, "%s  %s\n", comment.User.Label(), text.RelativeTime(comment.CreatedOn))
				fmt.Fprintln(app.IO.Out, text.Indent(strings.TrimSpace(comment.Content.Raw), "  "))
				fmt.Fprintln(app.IO.Out)
			}
			return nil
		},
	}
	out = cmdutil.AddOutputFlags(cmd)
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	cmd.Flags().IntVarP(&limit, "limit", "L", 100, "Maximum number of comments")
	return cmd
}

func newCmdDiff(f *cmdutil.Factory) *cobra.Command {
	var repoFlag string
	cmd := &cobra.Command{
		Use:   "diff [ID]",
		Short: "Show the diff of a pull request",
		Args:  cobra.MaximumNArgs(1),
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
			id, _, err := resolvePullRequest(ctx, app, ref, firstArg(args))
			if err != nil {
				return err
			}
			diff, err := app.Client.GetPullRequestDiff(ctx, ref.Owner, ref.Slug, id)
			if err != nil {
				return err
			}
			_, err = app.IO.Out.Write(diff)
			return err
		},
	}
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	return cmd
}

func newCmdChecks(f *cmdutil.Factory) *cobra.Command {
	var repoFlag string
	var limit int
	var out *cmdutil.OutputFlags
	cmd := &cobra.Command{
		Use:   "checks [ID]",
		Short: "Show build statuses for a pull request",
		Long:  "List the commit statuses reported for the pull request head and exit 1 when any status failed.",
		Args:  cobra.MaximumNArgs(1),
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
			id, _, err := resolvePullRequest(ctx, app, ref, firstArg(args))
			if err != nil {
				return err
			}
			statuses, err := app.Client.ListPullRequestStatuses(ctx, ref.Owner, ref.Slug, id, limit)
			if err != nil {
				return err
			}
			if out.IsJSON() {
				if err := out.Print(app.IO.Out, statuses); err != nil {
					return err
				}
			} else if len(statuses) == 0 {
				fmt.Fprintln(app.IO.Out, "No checks reported.")
			} else {
				t := tableprinter.New(app.IO).Header("NAME", "STATE", "DESCRIPTION", "URL")
				for _, status := range statuses {
					t.AddRow(status.Name, status.State, text.Truncate(status.Description, 40), status.URL)
				}
				if err := t.Render(); err != nil {
					return err
				}
			}
			failed := 0
			for _, status := range statuses {
				if strings.EqualFold(status.State, "FAILED") {
					failed++
				}
			}
			if failed > 0 {
				return &cmdutil.ExitError{Code: 1, Err: fmt.Errorf("%d check(s) failed", failed)}
			}
			return nil
		},
	}
	out = cmdutil.AddOutputFlags(cmd)
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	cmd.Flags().IntVarP(&limit, "limit", "L", 100, "Maximum number of statuses")
	return cmd
}

func newCmdCommits(f *cmdutil.Factory) *cobra.Command {
	var repoFlag string
	var limit int
	var out *cmdutil.OutputFlags
	cmd := &cobra.Command{
		Use:   "commits [ID]",
		Short: "List commits in a pull request",
		Args:  cobra.MaximumNArgs(1),
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
			id, _, err := resolvePullRequest(ctx, app, ref, firstArg(args))
			if err != nil {
				return err
			}
			commits, err := app.Client.ListPullRequestCommits(ctx, ref.Owner, ref.Slug, id, limit)
			if err != nil {
				return err
			}
			if out.IsJSON() {
				return out.Print(app.IO.Out, commits)
			}
			t := tableprinter.New(app.IO).Header("HASH", "MESSAGE", "AUTHOR", "DATE")
			for _, commit := range commits {
				hash := commit.Hash
				if len(hash) > 8 {
					hash = hash[:8]
				}
				t.AddRow(hash, text.Truncate(strings.SplitN(commit.Message, "\n", 2)[0], 60), commit.Author.Label(), text.Date(commit.Date))
			}
			return t.Render()
		},
	}
	out = cmdutil.AddOutputFlags(cmd)
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	cmd.Flags().IntVarP(&limit, "limit", "L", 100, "Maximum number of commits")
	return cmd
}

func resolvePullRequest(ctx context.Context, app *cmdutil.App, ref bitbucket.RepoRef, arg string) (int, *bitbucket.PullRequest, error) {
	if arg != "" {
		id, err := strconv.Atoi(arg)
		if err != nil || id <= 0 {
			return 0, nil, fmt.Errorf("invalid pull request ID %q", arg)
		}
		pr, err := app.Client.GetPullRequest(ctx, ref.Owner, ref.Slug, id)
		if err != nil {
			return 0, nil, err
		}
		return id, pr, nil
	}
	branch, err := git.CurrentBranch(ctx)
	if err != nil {
		return 0, nil, fmt.Errorf("no pull request ID given: %w", err)
	}
	pr, err := app.Client.FindOpenPullRequestForBranch(ctx, ref.Owner, ref.Slug, branch)
	if err != nil {
		return 0, nil, err
	}
	return pr.ID, pr, nil
}

func pullRequestURL(app *cmdutil.App, ref bitbucket.RepoRef, pr *bitbucket.PullRequest) string {
	if pr != nil && pr.Links.HTML.Href != "" {
		return pr.Links.HTML.Href
	}
	id := 0
	if pr != nil {
		id = pr.ID
	}
	return app.PullRequestHTMLURL(ref, id)
}

func defaultBranch(repository *bitbucket.Repository) string {
	if repository != nil && repository.MainBranch != nil {
		return repository.MainBranch.Label()
	}
	return ""
}

func firstArg(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return ""
}

func escapeQueryValue(value string) string {
	return strings.ReplaceAll(value, `"`, `\"`)
}

func nonEmpty(values []string) []string {
	out := values[:0]
	for _, value := range values {
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func readBody(path string, stdin io.Reader) (string, error) {
	if path == "-" {
		data, err := io.ReadAll(io.LimitReader(stdin, 4<<20))
		return string(data), err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
