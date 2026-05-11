package cli

import "github.com/spf13/cobra"

func newEntriesListCommand() *cobra.Command {
	var date string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List time entries for a date (defaults to today)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			d, err := parseDate(date)
			if err != nil {
				return err
			}
			if d == "" {
				d = todayDate()
			}
			client, _, err := authedClient()
			if err != nil {
				return err
			}
			entries, err := client.FetchTimeEntries(d)
			if err != nil {
				return err
			}
			return renderEntries(out(cmd), entries)
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "Date in YYYY-MM-DD format (default: today)")
	return cmd
}
