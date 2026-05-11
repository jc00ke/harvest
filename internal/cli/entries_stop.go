package cli

import "github.com/spf13/cobra"

func newEntriesStopCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stop ENTRY_ID",
		Short: "Stop the timer on a running time entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseEntryID(args[0])
			if err != nil {
				return err
			}
			client, _, err := authedClient()
			if err != nil {
				return err
			}
			entry, err := client.StopTimeEntry(id)
			if err != nil {
				return err
			}
			return renderEntry(out(cmd), entry)
		},
	}
}
