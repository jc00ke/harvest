package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newEntriesDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete ENTRY_ID",
		Short: "Delete a time entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseEntryID(args[0])
			if err != nil {
				return err
			}
			client, _, err := authedClient(cmd.Context())
			if err != nil {
				return err
			}
			if err := client.DeleteTimeEntry(cmd.Context(), id); err != nil {
				return err
			}
			return renderMessage(out(cmd), fmt.Sprintf("Deleted time entry %d", id))
		},
	}
}
