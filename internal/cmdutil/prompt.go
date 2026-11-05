package cmdutil

import (
	"fmt"
	"strings"

	"github.com/ClaudioLazaro/bitbucket-cli/internal/config"
)

// Confirm asks for confirmation unless assumeYes is set. Non-interactive runs
// must pass --yes explicitly.
func (a *App) Confirm(message string, assumeYes bool) error {
	if assumeYes {
		return nil
	}
	if !a.IO.IsStdinTTY() {
		return fmt.Errorf("%s: pass --yes to confirm in non-interactive mode", message)
	}
	answer, err := a.IO.ReadLine(message + " [y/N] ")
	if err != nil {
		return err
	}
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes":
		return nil
	default:
		return fmt.Errorf("aborted")
	}
}

// PromptText asks for a non-secret value when standard input is a terminal.
func (a *App) PromptText(label string) (string, error) {
	if !a.IO.IsStdinTTY() {
		return "", fmt.Errorf("%s: pass the value as a flag in non-interactive mode", label)
	}
	return a.IO.ReadLine(label + ": ")
}

var builtinCommands = map[string]bool{
	"auth": true, "repo": true, "pr": true, "pipeline": true, "workspace": true,
	"snippet": true, "download": true, "webhook": true, "branch": true,
	"ssh-key": true, "gpg-key": true, "search": true, "api": true, "browse": true,
	"config": true, "alias": true, "completion": true, "version": true, "help": true,
}

// IsBuiltinCommand reports whether name is a top-level bk command.
func IsBuiltinCommand(name string) bool { return builtinCommands[name] }

// ExpandAlias replaces a leading alias with its configured expansion.
func ExpandAlias(args []string, f *Factory) ([]string, error) {
	if len(args) == 0 || builtinCommands[args[0]] {
		return args, nil
	}
	cfg, _, err := config.Load()
	if err != nil {
		return nil, err
	}
	expansion, ok := cfg.Aliases[args[0]]
	if !ok {
		return args, nil
	}
	fields := strings.Fields(expansion)
	if len(fields) == 0 {
		return args, nil
	}
	return append(fields, args[1:]...), nil
}
