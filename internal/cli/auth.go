package cli

import "github.com/spf13/cobra"

func newAuthCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage Harvest credentials stored in the OS keyring",
	}
	cmd.AddCommand(
		newAuthLoginCommand(),
		newAuthLogoutCommand(),
		newAuthStatusCommand(),
	)
	return cmd
}
