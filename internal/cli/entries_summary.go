package cli

import (
	"sort"
	"time"

	"github.com/planetargon/harvest-tui/internal/harvest"
)

// entrySummary aggregates time entries for one client on one date.
type entrySummary struct {
	Client string  `json:"client"`
	Date   string  `json:"date"`
	Notes  string  `json:"notes"`
	Hours  float64 `json:"hours"`
}

// summarizeEntries groups entries by (client, date), summing hours and joining
// non-empty notes with ", ". Results are sorted by client name, then date.
func summarizeEntries(entries []harvest.TimeEntry) []entrySummary {
	type key struct{ client, date string }
	groups := make(map[key]*entrySummary)
	for _, e := range entries {
		k := key{e.Client.Name, e.SpentDate}
		s, ok := groups[k]
		if !ok {
			s = &entrySummary{Client: e.Client.Name, Date: e.SpentDate}
			groups[k] = s
		}
		if e.Notes != "" {
			if s.Notes != "" {
				s.Notes += ", "
			}
			s.Notes += e.Notes
		}
		s.Hours += e.Hours
	}
	summaries := make([]entrySummary, 0, len(groups))
	for _, s := range groups {
		summaries = append(summaries, *s)
	}
	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].Client != summaries[j].Client {
			return summaries[i].Client < summaries[j].Client
		}
		return summaries[i].Date < summaries[j].Date
	})
	return summaries
}

// mondayOfWeek returns the Monday of the week containing t in YYYY-MM-DD format.
func mondayOfWeek(t time.Time) string {
	offset := (int(t.Weekday()) + 6) % 7
	return t.AddDate(0, 0, -offset).Format(dateFormat)
}
