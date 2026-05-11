package cli

import (
	"github.com/planetargon/harvest-tui/internal/harvest"
	"github.com/spf13/cobra"
)

func newEntriesEditCommand() *cobra.Command {
	var (
		projectID int
		taskID    int
		hours     float64
		notes     string
		date      string
		billable  bool
	)
	cmd := &cobra.Command{
		Use:   "edit ENTRY_ID",
		Short: "Edit an existing time entry; only specified flags are updated",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseEntryID(args[0])
			if err != nil {
				return err
			}

			req := harvest.UpdateTimeEntryRequest{}
			flags := cmd.Flags()
			if flags.Changed("project") {
				p := projectID
				req.ProjectID = &p
			}
			if flags.Changed("task") {
				t := taskID
				req.TaskID = &t
			}
			if flags.Changed("hours") {
				h := hours
				req.Hours = &h
			}
			if flags.Changed("notes") {
				n := notes
				req.Notes = &n
			}
			if flags.Changed("date") {
				d, err := parseDate(date)
				if err != nil {
					return err
				}
				req.SpentDate = &d
			}
			if flags.Changed("billable") {
				b := billable
				req.IsBillable = &b
			}

			client, _, err := authedClient()
			if err != nil {
				return err
			}
			entry, err := client.UpdateTimeEntry(id, req)
			if err != nil {
				return err
			}
			return renderEntry(out(cmd), entry)
		},
	}
	cmd.Flags().IntVar(&projectID, "project", 0, "New project ID")
	cmd.Flags().IntVar(&taskID, "task", 0, "New task ID")
	cmd.Flags().Float64Var(&hours, "hours", 0, "New hours value")
	cmd.Flags().StringVar(&notes, "notes", "", "New notes")
	cmd.Flags().StringVar(&date, "date", "", "New date in YYYY-MM-DD format")
	cmd.Flags().BoolVar(&billable, "billable", false, "Mark entry as billable (true) or non-billable (false)")
	return cmd
}
