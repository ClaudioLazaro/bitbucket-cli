// Command gen-docs writes docs/commands.md from the Cobra command tree.
package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/ClaudioLazaro/bitbucket-cli/internal/cmdutil"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/iostreams"
	"github.com/ClaudioLazaro/bitbucket-cli/pkg/cmd/root"
)

func main() {
	streams, _, _, _ := iostreams.Test()
	rootCmd := root.NewCmdRoot(cmdutil.NewFactory(streams))
	rootCmd.InitDefaultHelpFlag()
	rootCmd.InitDefaultVersionFlag()

	var buf bytes.Buffer
	buf.WriteString("# Referência de comandos\n\n")
	buf.WriteString("Gerado a partir da árvore Cobra por `make docs`. Não edite à mão.\n")
	buf.WriteString("A saída de `bk <comando> --help` é a fonte para a versão instalada.\n\n")

	buf.WriteString("## Flags globais\n\n")
	writeFlag(&buf, rootCmd.PersistentFlags().Lookup("hostname"))
	writeFlag(&buf, rootCmd.Flags().Lookup("help"))
	writeFlag(&buf, rootCmd.Flags().Lookup("version"))
	buf.WriteString("\n")

	for _, child := range rootCmd.Commands() {
		walk(&buf, child)
	}

	if err := os.MkdirAll("docs", 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	path := filepath.Join("docs", "commands.md")
	content := bytes.TrimRight(buf.Bytes(), "\n")
	content = append(content, '\n')
	if err := os.WriteFile(path, content, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s\n", path)
}

func walk(buf *bytes.Buffer, cmd *cobra.Command) {
	if cmd.Hidden || cmd.Deprecated != "" || cmd.Name() == "help" {
		return
	}
	fmt.Fprintf(buf, "## %s\n\n", cmd.CommandPath())
	if cmd.Short != "" {
		fmt.Fprintf(buf, "%s\n\n", cmd.Short)
	}
	long := strings.TrimSpace(cmd.Long)
	if long != "" && long != strings.TrimSpace(cmd.Short) {
		fmt.Fprintf(buf, "%s\n\n", long)
	}
	if cmd.Example != "" {
		fmt.Fprintf(buf, "```sh\n%s\n```\n\n", strings.TrimSpace(cmd.Example))
	}

	if cmd.HasAvailableSubCommands() {
		fmt.Fprintf(buf, "**Uso:** `%s <subcomando> [flags]`\n\n", cmd.CommandPath())
	} else {
		use := strings.TrimSuffix(cmd.CommandPath(), cmd.Name()) + cmd.Use
		fmt.Fprintf(buf, "**Uso:** `%s [flags]`\n\n", use)
	}

	var flags []*pflag.Flag
	cmd.NonInheritedFlags().VisitAll(func(flag *pflag.Flag) {
		if flag.Hidden || flag.Name == "help" {
			return
		}
		flags = append(flags, flag)
	})
	if len(flags) > 0 {
		buf.WriteString("**Flags:**\n\n")
		for _, flag := range flags {
			writeFlag(buf, flag)
		}
		buf.WriteString("\n")
	}

	for _, child := range cmd.Commands() {
		walk(buf, child)
	}
}

func writeFlag(buf *bytes.Buffer, flag *pflag.Flag) {
	if flag == nil || flag.Hidden {
		return
	}
	name := "--" + flag.Name
	typ := flag.Value.Type()
	if flag.NoOptDefVal != "" && typ != "bool" {
		name += "[=" + strings.ToUpper(typ) + "]"
	} else if typ != "bool" {
		name += " " + typ
	}
	if flag.Shorthand != "" {
		name = "-" + flag.Shorthand + ", " + name
	}
	if flag.DefValue != "" && flag.DefValue != "false" && flag.DefValue != "0" && flag.DefValue != "[]" && flag.NoOptDefVal == "" {
		name += " (default " + flag.DefValue + ")"
	}
	usage := strings.Join(strings.Fields(flag.Usage), " ")
	fmt.Fprintf(buf, "- `%s` - %s\n", name, usage)
}
