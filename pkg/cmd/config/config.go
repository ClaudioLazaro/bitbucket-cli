// Package configcmd implements bk config commands.
package configcmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/ClaudioLazaro/bitbucket-cli/internal/cmdutil"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/config"
)

// NewCmdConfig returns the config command group.
func NewCmdConfig(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage bk configuration",
		Long:  "Read and write the configuration file. Credentials are never stored here.",
	}
	cmd.AddCommand(newCmdList(f), newCmdGet(f), newCmdSet(f))
	return cmd
}

func newCmdList(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Print the configuration file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := config.Load()
			if err != nil {
				return err
			}
			data, err := yaml.Marshal(cfg)
			if err != nil {
				return err
			}
			_, err = f.IO.Out.Write(data)
			return err
		},
	}
	return cmd
}

func newCmdGet(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get KEY",
		Short: "Print a configuration value",
		Long:  "Valid keys: active_host, host, workspace, project, user, kind, api_url.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := config.Load()
			if err != nil {
				return err
			}
			value, err := getValue(cfg, args[0])
			if err != nil {
				return err
			}
			fmt.Fprintln(f.IO.Out, value)
			return nil
		},
	}
	return cmd
}

func newCmdSet(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set KEY VALUE",
		Short: "Set a configuration value",
		Long:  "Valid keys: active_host, workspace, project, user, api_url. workspace and project apply to the active host.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, dir, err := config.Load()
			if err != nil {
				return err
			}
			key, value := args[0], args[1]
			if key == "active_host" {
				hostname, _, _, err := config.NormalizeHost(value)
				if err != nil {
					return err
				}
				cfg.HostFor(hostname)
				cfg.ActiveHost = hostname
				if err := cfg.Save(dir); err != nil {
					return err
				}
				fmt.Fprintf(f.IO.Out, "active_host=%s\n", hostname)
				return nil
			}

			hostname := activeHostname(cfg)
			if hostname == "" {
				return fmt.Errorf("no active host: run `bk auth login` or `bk config set active_host HOST`")
			}
			host := cfg.HostFor(hostname)
			switch key {
			case "workspace":
				host.Workspace = value
			case "project":
				host.Project = value
			case "user":
				host.User = value
			case "api_url":
				host.APIURL = value
			default:
				return fmt.Errorf("invalid key %q: use active_host, workspace, project, user, or api_url", key)
			}
			if err := cfg.Save(dir); err != nil {
				return err
			}
			fmt.Fprintf(f.IO.Out, "%s=%s (%s)\n", key, value, hostname)
			return nil
		},
	}
	return cmd
}

func getValue(cfg *config.Config, key string) (string, error) {
	if key == "active_host" {
		return cfg.ActiveHost, nil
	}
	hostname := activeHostname(cfg)
	if hostname == "" {
		return "", fmt.Errorf("no active host")
	}
	host := cfg.Hosts[hostname]
	if host == nil {
		return "", fmt.Errorf("no configuration for host %s", hostname)
	}
	switch key {
	case "host":
		return hostname, nil
	case "workspace":
		return host.Workspace, nil
	case "project":
		return host.Project, nil
	case "user":
		return host.User, nil
	case "api_url":
		return host.APIURL, nil
	case "kind":
		return host.Kind, nil
	default:
		return "", fmt.Errorf("invalid key %q: use active_host, host, workspace, project, user, kind, or api_url", key)
	}
}

func activeHostname(cfg *config.Config) string {
	if cfg.ActiveHost != "" {
		return cfg.ActiveHost
	}
	if len(cfg.Hosts) == 1 {
		for hostname := range cfg.Hosts {
			return hostname
		}
	}
	return ""
}
