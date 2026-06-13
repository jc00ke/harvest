package cli

import (
	"time"

	"github.com/spf13/cobra"
)

func newEntriesListCommand() *cobra.Command {
	var date string
	var week bool
	var summary bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List time entries for a date (defaults to today)",
		Long: "List time entries for a date (defaults to today).\n\n" +
			"Use --week to list entries for the 7-day window starting at --date.\n" +
			"Use --summary to aggregate a week's entries per client per day; without\n" +
			"--date the window starts on the Monday of the current week.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			from, err := parseDate(date)
			if err != nil {
				return err
			}
			if from == "" {
				if summary {
					from = mondayOfWeek(time.Now())
				} else {
					from = todayDate()
				}
			}
			to := from
			if week || summary {
				t, _ := time.Parse(dateFormat, from)
				to = t.AddDate(0, 0, 6).Format(dateFormat)
			}
			client, _, err := authedClient(cmd.Context())
			if err != nil {
				return err
			}
			entries, err := client.FetchTimeEntries(cmd.Context(), from, to)
			if err != nil {
				return err
			}
			if summary {
				return renderEntrySummaries(out(cmd), summarizeEntries(entries))
			}
			return renderEntries(out(cmd), entries)
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "Date in YYYY-MM-DD format (default: today)")
	cmd.Flags().BoolVar(&week, "week", false, "Treat --date as the start of a 7-day window")
	cmd.Flags().BoolVar(&summary, "summary", false, "Aggregate the week per client per day (implies --week; default start: Monday of current week)")
	return cmd
}
