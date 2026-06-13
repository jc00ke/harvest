package cli

import "github.com/spf13/cobra"

func newEntriesStartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "start ENTRY_ID",
		Short: "Start the timer on an existing time entry",
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
			entry, err := client.RestartTimeEntry(cmd.Context(), id)
			if err != nil {
				return err
			}
			return renderEntry(out(cmd), entry)
		},
	}
}
