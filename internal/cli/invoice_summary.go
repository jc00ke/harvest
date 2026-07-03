package cli

import (
	"fmt"
	"sort"
	"time"

	"github.com/jc00ke/harvest/internal/harvest"
)

// invoiceTaskSummary aggregates one person's hours on one task. Amount is the
// billable dollar total (hours × billable rate over the billable entries).
type invoiceTaskSummary struct {
	Task   string  `json:"task"`
	Hours  float64 `json:"hours"`
	Amount float64 `json:"amount"`
}

// invoicePersonSummary aggregates one person's hours and billable amount
// across tasks.
type invoicePersonSummary struct {
	Person string               `json:"person"`
	Hours  float64              `json:"hours"`
	Amount float64              `json:"amount"`
	Tasks  []invoiceTaskSummary `json:"tasks"`
}

// summarizeInvoice groups entries by person and task, summing hours and
// billable amounts. People are keyed by user ID and sorted by name; tasks
// likewise within each person.
func summarizeInvoice(entries []harvest.TimeEntry) []invoicePersonSummary {
	type taskKey struct{ userID, taskID int }
	people := make(map[int]*invoicePersonSummary)
	tasks := make(map[taskKey]*invoiceTaskSummary)
	for _, e := range entries {
		p, ok := people[e.User.ID]
		if !ok {
			p = &invoicePersonSummary{Person: e.User.Name}
			people[e.User.ID] = p
		}
		p.Hours += e.Hours

		k := taskKey{e.User.ID, e.Task.ID}
		ts, ok := tasks[k]
		if !ok {
			ts = &invoiceTaskSummary{Task: e.Task.Name}
			tasks[k] = ts
		}
		ts.Hours += e.Hours

		if e.IsBillable {
			amount := e.Hours * e.BillableRate
			p.Amount += amount
			ts.Amount += amount
		}
	}

	summaries := make([]invoicePersonSummary, 0, len(people))
	for userID, p := range people {
		for k, ts := range tasks {
			if k.userID == userID {
				p.Tasks = append(p.Tasks, *ts)
			}
		}
		sort.Slice(p.Tasks, func(i, j int) bool { return p.Tasks[i].Task < p.Tasks[j].Task })
		summaries = append(summaries, *p)
	}
	sort.Slice(summaries, func(i, j int) bool { return summaries[i].Person < summaries[j].Person })
	return summaries
}

// parseMonth converts a YYYY-MM string into the month's inclusive first and
// last day in YYYY-MM-DD format.
func parseMonth(s string) (from, to string, err error) {
	t, err := time.Parse("2006-01", s)
	if err != nil {
		return "", "", fmt.Errorf("invalid month %q: expected YYYY-MM", s)
	}
	return t.Format(dateFormat), t.AddDate(0, 1, -1).Format(dateFormat), nil
}
