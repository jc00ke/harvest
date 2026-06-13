package cli

import (
	"github.com/jc00ke/harvest/internal/harvest"
	"github.com/spf13/cobra"
)

func newEntriesCreateCommand() *cobra.Command {
	var (
		projectID int
		taskID    int
		hours     float64
		notes     string
		date      string
		billable  bool
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new time entry",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			d, err := parseDate(date)
			if err != nil {
				return err
			}
			if d == "" {
				d = todayDate()
			}
			req := harvest.CreateTimeEntryRequest{
				ProjectID: projectID,
				TaskID:    taskID,
				SpentDate: d,
				Hours:     hours,
				Notes:     notes,
			}
			if cmd.Flags().Changed("billable") {
				b := billable
				req.IsBillable = &b
			}

			client, _, err := authedClient(cmd.Context())
			if err != nil {
				return err
			}
			entry, err := client.CreateTimeEntry(cmd.Context(), req)
			if err != nil {
				return err
			}
			return renderEntry(out(cmd), entry)
		},
	}
	cmd.Flags().IntVar(&projectID, "project", 0, "Project ID (required)")
	cmd.Flags().IntVar(&taskID, "task", 0, "Task ID (required)")
	cmd.Flags().Float64Var(&hours, "hours", 0, "Hours worked (e.g. 1.5)")
	cmd.Flags().StringVar(&notes, "notes", "", "Notes for the time entry")
	cmd.Flags().StringVar(&date, "date", "", "Date in YYYY-MM-DD format (default: today)")
	cmd.Flags().BoolVar(&billable, "billable", false, "Mark entry as billable; omit to use the project default")
	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("task")
	return cmd
}
