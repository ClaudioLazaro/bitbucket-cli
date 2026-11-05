// Package pipeline implements bk pipeline commands.
package pipeline

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ClaudioLazaro/bitbucket-cli/internal/bitbucket"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/cmdutil"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/git"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/tableprinter"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/text"
)

// NewCmdPipeline returns the pipeline command group.
func NewCmdPipeline(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pipeline",
		Short: "Work with Bitbucket Pipelines",
		Long:  "List, inspect, run, stop, and read logs from Bitbucket Pipelines, and manage pipeline variables.",
	}
	cmd.AddCommand(
		newCmdList(f),
		newCmdView(f),
		newCmdRun(f),
		newCmdStop(f),
		newCmdLogs(f),
		newCmdVariable(f),
	)
	return cmd
}

func newCmdList(f *cmdutil.Factory) *cobra.Command {
	var repoFlag, branch, status string
	var limit int
	var out *cmdutil.OutputFlags
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List pipeline runs",
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
			q := url.Values{"sort": []string{"-created_on"}}
			if branch != "" {
				q.Set("target.ref_name", branch)
			}
			if status != "" {
				q.Set("status", strings.ToUpper(status))
			}
			pipelines, err := app.Client.ListPipelines(ctx, ref.Owner, ref.Slug, q, limit)
			if err != nil {
				return err
			}
			if out.IsJSON() {
				return out.Print(app.IO.Out, pipelines)
			}
			t := tableprinter.New(app.IO).Header("BUILD", "STATUS", "BRANCH", "TRIGGER", "CREATED", "DURATION")
			for _, pipeline := range pipelines {
				t.AddRow(
					strconv.Itoa(pipeline.BuildNumber),
					pipeline.Status(),
					branchOf(&pipeline),
					triggerOf(&pipeline),
					text.RelativeTime(pipeline.CreatedOn),
					formatDuration(pipeline.BuildSecondsUsed),
				)
			}
			return t.Render()
		},
	}
	out = cmdutil.AddOutputFlags(cmd)
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	cmd.Flags().StringVarP(&branch, "branch", "b", "", "Filter by branch")
	cmd.Flags().StringVar(&status, "status", "", "Filter by status (SUCCESSFUL, FAILED, IN_PROGRESS, ...)")
	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of pipelines (0 for all)")
	return cmd
}

func newCmdView(f *cmdutil.Factory) *cobra.Command {
	var repoFlag string
	var web bool
	var out *cmdutil.OutputFlags
	cmd := &cobra.Command{
		Use:   "view UUID|BUILD",
		Short: "View a pipeline run",
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
			uuid, err := resolvePipeline(ctx, app, ref, args[0])
			if err != nil {
				return err
			}
			pipeline, err := app.Client.GetPipeline(ctx, ref.Owner, ref.Slug, uuid)
			if err != nil {
				return err
			}
			if web {
				return app.OpenBrowser(app.PipelineHTMLURL(ref, pipeline.BuildNumber))
			}
			steps, err := app.Client.ListSteps(ctx, ref.Owner, ref.Slug, uuid)
			if err != nil {
				return err
			}
			if out.IsJSON() {
				return out.Print(app.IO.Out, struct {
					*bitbucket.Pipeline
					Steps []bitbucket.Step `json:"steps"`
				}{Pipeline: pipeline, Steps: steps})
			}

			cs := app.IO.ColorScheme()
			fmt.Fprintf(app.IO.Out, "%s #%d %s\n", cs.Magenta("pipeline"), pipeline.BuildNumber, cs.Bold(pipeline.Status()))
			fmt.Fprintf(app.IO.Out, "%s | %s | %s | %s\n",
				branchOf(pipeline),
				triggerOf(pipeline),
				text.Date(pipeline.CreatedOn),
				formatDuration(pipeline.BuildSecondsUsed),
			)
			fmt.Fprintln(app.IO.Out, pipeline.UUID)
			if pipeline.Links.HTML.Href != "" {
				fmt.Fprintln(app.IO.Out, pipeline.Links.HTML.Href)
			}
			if len(steps) > 0 {
				fmt.Fprintln(app.IO.Out)
				t := tableprinter.New(app.IO).Header("#", "STATUS", "STEP", "DURATION")
				for i, step := range steps {
					t.AddRow(strconv.Itoa(i+1), step.Status(), step.Name, formatDuration(step.DurationInSeconds))
				}
				if err := t.Render(); err != nil {
					return err
				}
			}
			return nil
		},
	}
	out = cmdutil.AddOutputFlags(cmd)
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	cmd.Flags().BoolVarP(&web, "web", "w", false, "Open the pipeline in a browser")
	return cmd
}

func newCmdRun(f *cmdutil.Factory) *cobra.Command {
	var repoFlag, branch, custom string
	var variables []string
	var web bool
	var out *cmdutil.OutputFlags
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run a pipeline",
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
			if branch == "" {
				branch, err = git.CurrentBranch(ctx)
				if err != nil {
					return fmt.Errorf("could not determine the current branch: pass --branch")
				}
			}
			vars := map[string]string{}
			for _, pair := range variables {
				key, value, ok := strings.Cut(pair, "=")
				if !ok || key == "" {
					return fmt.Errorf("invalid --var %q: expected KEY=VALUE", pair)
				}
				vars[key] = value
			}
			pipeline, err := app.Client.RunPipeline(ctx, ref.Owner, ref.Slug, bitbucket.RunPipelineInput{
				Branch:        branch,
				CustomPattern: custom,
				Variables:     vars,
			})
			if err != nil {
				return err
			}
			if out.IsJSON() {
				return out.Print(app.IO.Out, pipeline)
			}
			htmlURL := pipeline.Links.HTML.Href
			if htmlURL == "" {
				htmlURL = app.PipelineHTMLURL(ref, pipeline.BuildNumber)
			}
			fmt.Fprintf(app.IO.Out, "Started pipeline #%d\n%s\n", pipeline.BuildNumber, htmlURL)
			if web {
				return app.OpenBrowser(htmlURL)
			}
			return nil
		},
	}
	out = cmdutil.AddOutputFlags(cmd)
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	cmd.Flags().StringVarP(&branch, "branch", "b", "", "Branch to run (default current branch)")
	cmd.Flags().StringVar(&custom, "custom", "", "Custom pipeline pattern")
	cmd.Flags().StringArrayVar(&variables, "var", nil, "Pipeline variable KEY=VALUE (repeatable)")
	cmd.Flags().BoolVarP(&web, "web", "w", false, "Open the pipeline in a browser")
	return cmd
}

func newCmdStop(f *cmdutil.Factory) *cobra.Command {
	var repoFlag string
	cmd := &cobra.Command{
		Use:   "stop UUID|BUILD",
		Short: "Stop a running pipeline",
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
			uuid, err := resolvePipeline(ctx, app, ref, args[0])
			if err != nil {
				return err
			}
			if err := app.Client.StopPipeline(ctx, ref.Owner, ref.Slug, uuid); err != nil {
				return err
			}
			fmt.Fprintf(app.IO.Out, "Stopped pipeline %s\n", uuid)
			return nil
		},
	}
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	return cmd
}

func newCmdLogs(f *cmdutil.Factory) *cobra.Command {
	var repoFlag string
	var stepSelector string
	cmd := &cobra.Command{
		Use:   "logs UUID|BUILD",
		Short: "Print pipeline step logs",
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
			uuid, err := resolvePipeline(ctx, app, ref, args[0])
			if err != nil {
				return err
			}
			steps, err := app.Client.ListSteps(ctx, ref.Owner, ref.Slug, uuid)
			if err != nil {
				return err
			}
			if len(steps) == 0 {
				return fmt.Errorf("pipeline %s has no steps yet", uuid)
			}
			selected, err := selectSteps(steps, stepSelector)
			if err != nil {
				return err
			}
			for _, step := range selected {
				if len(selected) > 1 {
					fmt.Fprintf(app.IO.Out, "=== step %d: %s (%s) ===\n", stepPosition(steps, step.UUID)+1, step.Name, step.Status())
				}
				if err := app.Client.StreamStepLog(ctx, app.IO.Out, ref.Owner, ref.Slug, uuid, step.UUID); err != nil {
					return err
				}
				if len(selected) > 1 {
					fmt.Fprintln(app.IO.Out)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	cmd.Flags().StringVar(&stepSelector, "step", "", "Step to print: 1-based index, name, or UUID")
	return cmd
}

func selectSteps(steps []bitbucket.Step, selector string) ([]bitbucket.Step, error) {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return steps, nil
	}
	if index, err := strconv.Atoi(selector); err == nil {
		if index < 1 || index > len(steps) {
			return nil, fmt.Errorf("step %d does not exist (pipeline has %d steps: %s)", index, len(steps), stepList(steps))
		}
		return steps[index-1 : index], nil
	}

	normalized := strings.Trim(selector, "{}")
	var exact []bitbucket.Step
	for _, step := range steps {
		if strings.EqualFold(strings.Trim(step.UUID, "{}"), normalized) || strings.EqualFold(step.Name, selector) {
			exact = append(exact, step)
		}
	}
	if len(exact) == 1 {
		return exact, nil
	}
	if len(exact) > 1 {
		return nil, fmt.Errorf("%q matches %d steps; use the 1-based index (%s)", selector, len(exact), stepList(exact))
	}

	var partial []bitbucket.Step
	for _, step := range steps {
		if strings.Contains(strings.ToLower(step.Name), strings.ToLower(selector)) {
			partial = append(partial, step)
		}
	}
	if len(partial) == 1 {
		return partial, nil
	}
	if len(partial) > 1 {
		return nil, fmt.Errorf("%q matches %d steps; use the 1-based index (%s)", selector, len(partial), stepList(partial))
	}
	return nil, fmt.Errorf("no step matches %q (available: %s)", selector, stepList(steps))
}

func stepList(steps []bitbucket.Step) string {
	items := make([]string, 0, len(steps))
	for i, step := range steps {
		name := step.Name
		if name == "" {
			name = "(unnamed)"
		}
		items = append(items, fmt.Sprintf("%d=%s", i+1, name))
	}
	return strings.Join(items, ", ")
}

func stepPosition(steps []bitbucket.Step, uuid string) int {
	for i, step := range steps {
		if step.UUID == uuid {
			return i
		}
	}
	return -1
}

func newCmdVariable(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "variable",
		Short: "Manage pipeline variables",
	}
	cmd.AddCommand(newCmdVariableList(f), newCmdVariableSet(f), newCmdVariableDelete(f))
	return cmd
}

func newCmdVariableList(f *cmdutil.Factory) *cobra.Command {
	var repoFlag string
	var out *cmdutil.OutputFlags
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List pipeline variables",
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
			variables, err := app.Client.ListPipelineVariables(ctx, ref.Owner, ref.Slug)
			if err != nil {
				return err
			}
			if out.IsJSON() {
				return out.Print(app.IO.Out, variables)
			}
			t := tableprinter.New(app.IO).Header("KEY", "VALUE", "SECURED", "UUID")
			for _, variable := range variables {
				value := variable.Value
				if variable.Secured {
					value = "****"
				}
				t.AddRow(variable.Key, text.Truncate(value, 50), strconv.FormatBool(variable.Secured), variable.UUID)
			}
			return t.Render()
		},
	}
	out = cmdutil.AddOutputFlags(cmd)
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	return cmd
}

func newCmdVariableSet(f *cmdutil.Factory) *cobra.Command {
	var repoFlag string
	var secured bool
	cmd := &cobra.Command{
		Use:   "set KEY VALUE",
		Short: "Create or update a pipeline variable",
		Args:  cobra.ExactArgs(2),
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
			variable, err := app.Client.SetPipelineVariable(ctx, ref.Owner, ref.Slug, args[0], args[1], secured)
			if err != nil {
				return err
			}
			fmt.Fprintf(app.IO.Out, "Set pipeline variable %s (%s)\n", variable.Key, variable.UUID)
			return nil
		},
	}
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	cmd.Flags().BoolVar(&secured, "secured", false, "Mark the variable as secured")
	return cmd
}

func newCmdVariableDelete(f *cmdutil.Factory) *cobra.Command {
	var repoFlag string
	cmd := &cobra.Command{
		Use:   "delete UUID",
		Short: "Delete a pipeline variable",
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
			if err := app.Client.DeletePipelineVariable(ctx, ref.Owner, ref.Slug, args[0]); err != nil {
				return err
			}
			fmt.Fprintf(app.IO.Out, "Deleted pipeline variable %s\n", args[0])
			return nil
		},
	}
	cmd.Flags().StringVarP(&repoFlag, "repo", "R", "", "Repository as OWNER/REPO")
	return cmd
}

func resolvePipeline(ctx context.Context, app *cmdutil.App, ref bitbucket.RepoRef, arg string) (string, error) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return "", fmt.Errorf("pipeline UUID or build number is required")
	}
	if _, err := strconv.Atoi(arg); err != nil {
		return arg, nil
	}
	build, _ := strconv.Atoi(arg)
	pipelines, err := app.Client.ListPipelines(ctx, ref.Owner, ref.Slug, url.Values{"sort": []string{"-created_on"}}, 100)
	if err != nil {
		return "", err
	}
	for _, pipeline := range pipelines {
		if pipeline.BuildNumber == build {
			return pipeline.UUID, nil
		}
	}
	return "", fmt.Errorf("pipeline build %d not found in the latest 100 runs", build)
}

func branchOf(pipeline *bitbucket.Pipeline) string {
	if pipeline == nil || pipeline.Target == nil {
		return ""
	}
	return pipeline.Target.RefName
}

func triggerOf(pipeline *bitbucket.Pipeline) string {
	if pipeline == nil || pipeline.Trigger == nil {
		return ""
	}
	if pipeline.Trigger.Name != "" {
		return pipeline.Trigger.Name
	}
	return pipeline.Trigger.Type
}

func formatDuration(seconds int) string {
	if seconds <= 0 {
		return ""
	}
	d := time.Duration(seconds) * time.Second
	if d < time.Minute {
		return fmt.Sprintf("%ds", seconds)
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
}
