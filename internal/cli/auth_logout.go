package cli

import (
	"errors"

	"github.com/jc00ke/harvest/internal/config"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

func newAuthLogoutCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove stored Harvest credentials from the OS keyring",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			err := config.DeleteCredentialsFromKeyring()
			if errors.Is(err, keyring.ErrNotFound) {
				return renderMessage(out(cmd), "No credentials were stored in the keyring.")
			}
			if err != nil {
				return err
			}
			return renderMessage(out(cmd), "Removed credentials from keyring.")
		},
	}
}
