package cli

import "github.com/spf13/cobra"

func newMeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "me",
		Short: "Show the authenticated Harvest user",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, user, err := authedClient(cmd.Context())
			if err != nil {
				return err
			}
			return renderUser(out(cmd), user)
		},
	}
}
