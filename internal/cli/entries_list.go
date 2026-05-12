package cli

import (
	"time"

	"github.com/spf13/cobra"
)

func newEntriesListCommand() *cobra.Command {
	var date string
	var week bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List time entries for a date (defaults to today)",
		Long: "List time entries for a date (defaults to today).\n\n" +
			"Use --week to list entries for the 7-day window starting at --date.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			from, err := parseDate(date)
			if err != nil {
				return err
			}
			if from == "" {
				from = todayDate()
			}
			to := from
			if week {
				t, _ := time.Parse(dateFormat, from)
				to = t.AddDate(0, 0, 6).Format(dateFormat)
			}
			client, _, err := authedClient()
			if err != nil {
				return err
			}
			entries, err := client.FetchTimeEntries(from, to)
			if err != nil {
				return err
			}
			return renderEntries(out(cmd), entries)
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "Date in YYYY-MM-DD format (default: today)")
	cmd.Flags().BoolVar(&week, "week", false, "Treat --date as the start of a 7-day window")
	return cmd
}
