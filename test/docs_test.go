package test

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/ClaudioLazaro/bitbucket-cli/internal/cmdutil"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/iostreams"
	"github.com/ClaudioLazaro/bitbucket-cli/pkg/cmd/root"
)

// TestCommandReferenceCoversEveryCommand fails when a command is added or
// renamed without regenerating docs/commands.md with `make docs`.
func TestCommandReferenceCoversEveryCommand(t *testing.T) {
	data, err := os.ReadFile("../docs/commands.md")
	if err != nil {
		t.Fatalf("read docs/commands.md: %v (run `make docs`)", err)
	}
	docs := string(data)

	streams, _, _, _ := iostreams.Test()
	rootCmd := root.NewCmdRoot(cmdutil.NewFactory(streams))

	var missing []string
	var walk func(*cobra.Command)
	walk = func(cmd *cobra.Command) {
		for _, child := range cmd.Commands() {
			if child.Hidden || child.Deprecated != "" || child.Name() == "help" {
				continue
			}
			heading := "## " + child.CommandPath() + "\n"
			if !strings.Contains(docs, heading) {
				missing = append(missing, child.CommandPath())
			}
			walk(child)
		}
	}
	walk(rootCmd)

	if len(missing) > 0 {
		t.Fatalf("docs/commands.md is stale; run `make docs`. Missing: %s", strings.Join(missing, ", "))
	}
}
