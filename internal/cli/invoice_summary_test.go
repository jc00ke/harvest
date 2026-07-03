package cli

import (
	"strings"
	"testing"

	"github.com/jc00ke/harvest/internal/harvest"
)

func TestSummarizeInvoice(t *testing.T) {
	entry := func(userID int, user, task string, taskID int, hours, rate float64) harvest.TimeEntry {
		return harvest.TimeEntry{
			User:         harvest.TimeEntryUser{ID: userID, Name: user},
			Task:         harvest.TimeEntryTask{ID: taskID, Name: task},
			Hours:        hours,
			IsBillable:   rate != 0,
			BillableRate: rate,
		}
	}

	t.Run("given entries from several people when summarized then groups by person and task sorted by name", func(t *testing.T) {
		entries := []harvest.TimeEntry{
			entry(2, "Sam Chen", "Design", 201, 2.0, 150),
			entry(1, "Alex Rivera", "Development", 202, 1.5, 160),
			entry(1, "Alex Rivera", "Design", 201, 0.5, 150),
			entry(1, "Alex Rivera", "Development", 202, 2.0, 160),
		}

		summaries := summarizeInvoice(entries)

		if got, want := len(summaries), 2; got != want {
			t.Fatalf("len(summaries)=%d, want=%d", got, want)
		}
		if got, want := summaries[0].Person, "Alex Rivera"; got != want {
			t.Errorf("first person=%s, want=%s", got, want)
		}
		if got, want := summaries[0].Hours, 4.0; got != want {
			t.Errorf("person hours=%f, want=%f", got, want)
		}
		if got, want := len(summaries[0].Tasks), 2; got != want {
			t.Fatalf("len(tasks)=%d, want=%d", got, want)
		}
		if got, want := summaries[0].Tasks[0].Task, "Design"; got != want {
			t.Errorf("first task=%s, want=%s", got, want)
		}
		if got, want := summaries[0].Tasks[1].Hours, 3.5; got != want {
			t.Errorf("development hours=%f, want=%f", got, want)
		}
		if got, want := summaries[0].Amount, 635.0; got != want {
			t.Errorf("person amount=%f, want=%f", got, want)
		}
		if got, want := summaries[0].Tasks[1].Amount, 560.0; got != want {
			t.Errorf("development amount=%f, want=%f", got, want)
		}
		if got, want := summaries[1].Person, "Sam Chen"; got != want {
			t.Errorf("second person=%s, want=%s", got, want)
		}
	})

	t.Run("given a non-billable entry when summarized then its hours count but its amount does not", func(t *testing.T) {
		entries := []harvest.TimeEntry{
			entry(1, "Alex Rivera", "Development", 202, 2.0, 160),
			{
				User:  harvest.TimeEntryUser{ID: 1, Name: "Alex Rivera"},
				Task:  harvest.TimeEntryTask{ID: 203, Name: "Meetings"},
				Hours: 1.0,
			},
		}

		summaries := summarizeInvoice(entries)

		if got, want := summaries[0].Hours, 3.0; got != want {
			t.Errorf("person hours=%f, want=%f", got, want)
		}
		if got, want := summaries[0].Amount, 320.0; got != want {
			t.Errorf("person amount=%f, want=%f", got, want)
		}
	})

	t.Run("given no entries when summarized then returns an empty slice", func(t *testing.T) {
		if got, want := len(summarizeInvoice(nil)), 0; got != want {
			t.Errorf("len(summaries)=%d, want=%d", got, want)
		}
	})
}

func TestParseMonth(t *testing.T) {
	t.Run("given a valid month when parsed then returns first and last day", func(t *testing.T) {
		from, to, err := parseMonth("2026-06")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got, want := from, "2026-06-01"; got != want {
			t.Errorf("from=%s, want=%s", got, want)
		}
		if got, want := to, "2026-06-30"; got != want {
			t.Errorf("to=%s, want=%s", got, want)
		}
	})

	t.Run("given February of a leap year when parsed then ends on the 29th", func(t *testing.T) {
		_, to, err := parseMonth("2028-02")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got, want := to, "2028-02-29"; got != want {
			t.Errorf("to=%s, want=%s", got, want)
		}
	})

	t.Run("given December when parsed then ends on the 31st of the same year", func(t *testing.T) {
		_, to, err := parseMonth("2026-12")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got, want := to, "2026-12-31"; got != want {
			t.Errorf("to=%s, want=%s", got, want)
		}
	})

	t.Run("given a malformed month when parsed then returns a format error", func(t *testing.T) {
		_, _, err := parseMonth("June 2026")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if got, want := err.Error(), "expected YYYY-MM"; !strings.Contains(got, want) {
			t.Errorf("error=%q, want substring %q", got, want)
		}
	})
}
