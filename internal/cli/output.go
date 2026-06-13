package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/jc00ke/harvest/internal/harvest"
)

// renderJSON marshals v to JSON and writes it to w with a trailing newline.
func renderJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// newTabWriter returns a tabwriter configured for the CLI's table output.
func newTabWriter(w io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
}

// entryStatus returns a short status label for a time entry.
func entryStatus(e harvest.TimeEntry) string {
	switch {
	case e.IsRunning:
		return "running"
	case e.IsLocked:
		return "locked"
	default:
		return ""
	}
}

// formatHours renders hours as H:MM (e.g. 1.5 -> 1:30).
func formatHours(hours float64) string {
	totalMinutes := int(hours*60 + 0.5)
	h := totalMinutes / 60
	m := totalMinutes % 60
	return fmt.Sprintf("%d:%02d", h, m)
}

// truncate clips s to max runes and appends "…" if truncated.
func truncate(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 1 {
		return string(r[:max])
	}
	return string(r[:max-1]) + "…"
}

// renderEntries writes a list of time entries as JSON or a table.
func renderEntries(w io.Writer, entries []harvest.TimeEntry) error {
	if outputJSON {
		return renderJSON(w, entries)
	}
	if len(entries) == 0 {
		_, err := fmt.Fprintln(w, "(no time entries)")
		return err
	}
	tw := newTabWriter(w)
	fmt.Fprintln(tw, "ID\tDATE\tHOURS\tSTATUS\tCLIENT\tPROJECT\tTASK\tNOTES")
	for _, e := range entries {
		fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			e.ID,
			e.SpentDate,
			formatHours(e.Hours),
			entryStatus(e),
			e.Client.Name,
			e.Project.Name,
			e.Task.Name,
			truncate(e.Notes, 60),
		)
	}
	return tw.Flush()
}

// renderEntrySummaries writes per-client per-day summaries as JSON or a table.
func renderEntrySummaries(w io.Writer, summaries []entrySummary) error {
	if outputJSON {
		return renderJSON(w, summaries)
	}
	if len(summaries) == 0 {
		_, err := fmt.Fprintln(w, "(no time entries)")
		return err
	}
	tw := newTabWriter(w)
	fmt.Fprintln(tw, "CLIENT\tDATE\tHOURS\tNOTES")
	for _, s := range summaries {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
			s.Client,
			s.Date,
			formatHours(s.Hours),
			truncate(s.Notes, 60),
		)
	}
	return tw.Flush()
}

// renderEntry writes a single time entry as JSON or a key/value table.
func renderEntry(w io.Writer, e *harvest.TimeEntry) error {
	if outputJSON {
		return renderJSON(w, e)
	}
	tw := newTabWriter(w)
	fmt.Fprintf(tw, "ID\t%d\n", e.ID)
	fmt.Fprintf(tw, "DATE\t%s\n", e.SpentDate)
	fmt.Fprintf(tw, "HOURS\t%s\n", formatHours(e.Hours))
	fmt.Fprintf(tw, "STATUS\t%s\n", entryStatus(*e))
	fmt.Fprintf(tw, "BILLABLE\t%t\n", e.IsBillable)
	fmt.Fprintf(tw, "CLIENT\t%s\n", e.Client.Name)
	fmt.Fprintf(tw, "PROJECT\t%s (%d)\n", e.Project.Name, e.Project.ID)
	fmt.Fprintf(tw, "TASK\t%s (%d)\n", e.Task.Name, e.Task.ID)
	fmt.Fprintf(tw, "NOTES\t%s\n", e.Notes)
	return tw.Flush()
}

// renderProjects writes project+task assignments as JSON or a table with one
// row per (project, task) pair so users can copy IDs into `entries create`.
func renderProjects(w io.Writer, projects []harvest.ProjectWithTasks) error {
	if outputJSON {
		return renderJSON(w, projects)
	}
	if len(projects) == 0 {
		_, err := fmt.Fprintln(w, "(no projects)")
		return err
	}
	tw := newTabWriter(w)
	fmt.Fprintln(tw, "PROJECT_ID\tCLIENT\tPROJECT\tTASK_ID\tTASK")
	for _, p := range projects {
		for _, task := range p.Tasks {
			fmt.Fprintf(tw, "%d\t%s\t%s\t%d\t%s\n",
				p.Project.ID,
				p.Project.Client.Name,
				p.Project.Name,
				task.ID,
				task.Name,
			)
		}
	}
	return tw.Flush()
}

// renderUser writes the authenticated user as JSON or a key/value table.
func renderUser(w io.Writer, u *harvest.User) error {
	if outputJSON {
		return renderJSON(w, u)
	}
	tw := newTabWriter(w)
	fmt.Fprintf(tw, "ID\t%d\n", u.ID)
	fmt.Fprintf(tw, "NAME\t%s %s\n", u.FirstName, u.LastName)
	fmt.Fprintf(tw, "EMAIL\t%s\n", u.Email)
	return tw.Flush()
}

// renderMessage writes a confirmation message as JSON ({"message": ...}) or plain text.
func renderMessage(w io.Writer, msg string) error {
	if outputJSON {
		return renderJSON(w, map[string]string{"message": msg})
	}
	_, err := fmt.Fprintln(w, msg)
	return err
}
