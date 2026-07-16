package cli

import (
	"github.com/spf13/cobra"
)

func NewRootCmd(version string) *cobra.Command {
	v := newConfig()
	var configFile string

	cmd := &cobra.Command{
		Use:           "skillctl",
		Short:         "Manage AI agent skills as OCI images",
		Long:          "skillctl builds, pushes, pulls, and manages the lifecycle of AI agent skills stored as OCI images.",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			return loadConfig(v, configFile)
		},
	}
	cmd.PersistentFlags().StringVar(&configFile, "config", "", "configuration file (default: ./config.yaml or user config directory)")

	cmd.AddCommand(newValidateCmd())
	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newBuildCmd())
	cmd.AddCommand(newPushCmd())
	cmd.AddCommand(newPullCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newInspectCmd())
	cmd.AddCommand(newPromoteCmd())
	cmd.AddCommand(newDemoteCmd())
	cmd.AddCommand(newInstallCmd())
	cmd.AddCommand(newTagCmd())
	cmd.AddCommand(newPruneCmd())
	cmd.AddCommand(newRmCmd())
	cmd.AddCommand(newUpgradeCmd())
	cmd.AddCommand(newServeCmd(v))
	cmd.AddCommand(newCollectionCmd())

	return cmd
}
