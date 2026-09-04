package cli

import (
	"context"

	"github.com/qrotux/gridraw-cli/internal/client"
	"github.com/qrotux/gridraw-cli/internal/config"
	"github.com/spf13/cobra"
)

// Execute runs the root command and returns the error for main to classify.
func Execute() error {
	return newRootCmd().ExecuteContext(context.Background())
}

// newRootCmd assembles the command tree.
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "gridraw",
		Short:         "CLI access to gridraw table sources",
		Version:       versionString(),
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().String("config", "", "configuration profile to use")
	root.PersistentFlags().String("config-file", "", "read this configuration file only")
	root.AddCommand(newConfigCmd(), newListCmd(), newGridCmd(), newFromCmd(), newWhereTopic())
	return root
}

// loadConfig honours --config-file, falling back to discovery.
func loadConfig(cmd *cobra.Command) (*config.Config, error) {
	path, _ := cmd.Flags().GetString("config-file")
	if path != "" {
		return config.LoadFile(path)
	}
	return config.Load()
}

// profileFor returns the profile selected by --config, or the current one.
func profileFor(cmd *cobra.Command) (config.Profile, error) {
	cfg, err := loadConfig(cmd)
	if err != nil {
		return config.Profile{}, err
	}
	name, _ := cmd.Flags().GetString("config")
	p, _, err := cfg.Profile(name)
	return p, err
}

// apiFor builds the client and the descriptor cache for the selected profile.
func apiFor(cmd *cobra.Command) (*client.Client, config.Profile, *client.Cache, error) {
	cfg, err := loadConfig(cmd)
	if err != nil {
		return nil, config.Profile{}, nil, err
	}
	selected, _ := cmd.Flags().GetString("config")
	profile, name, err := cfg.Profile(selected)
	if err != nil {
		return nil, config.Profile{}, nil, err
	}
	// A machine with no resolvable cache root still runs: the empty Dir turns
	// every cache read and write into a no-op.
	dir, _ := client.DefaultDir()
	cache := &client.Cache{Dir: dir, Profile: name, TTL: client.DefaultTTL}
	return client.New(profile.Host, profile.Header, nil), profile, cache, nil
}
