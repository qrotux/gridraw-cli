package cli

import "github.com/spf13/cobra"

// Execute runs the root command and returns the error for main to classify.
func Execute() error {
	root := &cobra.Command{
		Use:           "gridraw",
		Short:         "CLI access to gridraw table sources",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().String("config", "", "configuration profile to use")
	root.PersistentFlags().String("config-file", "", "read this configuration file only")
	return root.Execute()
}
