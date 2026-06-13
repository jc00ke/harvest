package cli

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jc00ke/harvest/internal/harvest"
)

func entry(client, date, notes string, hours float64) harvest.TimeEntry {
	return harvest.TimeEntry{
		SpentDate: date,
		Hours:     hours,
		Notes:     notes,
		Client:    harvest.TimeEntryClient{Name: client},
	}
}

func TestSummarizeEntries(t *testing.T) {
	t.Run("given several entries for the same client and date when summarized then hours are summed and notes joined", func(t *testing.T) {
		got := summarizeEntries([]harvest.TimeEntry{
			entry("Acme", "2026-06-08", "standup", 0.25),
			entry("Acme", "2026-06-08", "code review", 1.25),
			entry("Acme", "2026-06-08", "pairing", 2.5),
			entry("Acme", "2026-06-08", "deploy", 0.75),
		})
		want := []entrySummary{
			{Client: "Acme", Date: "2026-06-08", Notes: "standup, code review, pairing, deploy", Hours: 4.75},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("given entries across clients and dates when summarized then each group sums its own hours and groups are sorted by client then date", func(t *testing.T) {
		got := summarizeEntries([]harvest.TimeEntry{
			entry("Zenith", "2026-06-08", "deploy", 2),
			entry("Acme", "2026-06-09", "design", 1),
			entry("Acme", "2026-06-08", "standup", 0.5),
			entry("Zenith", "2026-06-08", "incident response", 1.75),
			entry("Acme", "2026-06-09", "design review", 0.25),
			entry("Acme", "2026-06-08", "retro", 1.5),
			entry("Zenith", "2026-06-08", "postmortem", 0.5),
		})
		want := []entrySummary{
			{Client: "Acme", Date: "2026-06-08", Notes: "standup, retro", Hours: 2},
			{Client: "Acme", Date: "2026-06-09", Notes: "design, design review", Hours: 1.25},
			{Client: "Zenith", Date: "2026-06-08", Notes: "deploy, incident response, postmortem", Hours: 4.25},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("given an entry with empty notes when summarized then it is skipped in the joined notes", func(t *testing.T) {
		got := summarizeEntries([]harvest.TimeEntry{
			entry("Acme", "2026-06-08", "", 1),
			entry("Acme", "2026-06-08", "standup", 0.5),
		})
		if got, want := got[0].Notes, "standup"; got != want {
			t.Errorf("notes=%q, want=%q", got, want)
		}
	})

	t.Run("given no entries when summarized then the result is empty", func(t *testing.T) {
		if got := summarizeEntries(nil); len(got) != 0 {
			t.Errorf("got %+v, want empty", got)
		}
	})
}

func TestMondayOfWeek(t *testing.T) {
	cases := []struct {
		name string
		day  string
		want string
	}{
		{"given a Monday when computed then it returns the same day", "2026-06-08", "2026-06-08"},
		{"given a Thursday when computed then it returns the preceding Monday", "2026-06-11", "2026-06-08"},
		{"given a Sunday when computed then it returns the Monday six days earlier", "2026-06-14", "2026-06-08"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			day, err := time.Parse(dateFormat, tc.day)
			if err != nil {
				t.Fatalf("parse %q: %v", tc.day, err)
			}
			if got := mondayOfWeek(day); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRenderEntrySummaries(t *testing.T) {
	summaries := []entrySummary{
		{Client: "Acme", Date: "2026-06-08", Notes: "standup, code review", Hours: 1.75},
	}

	t.Run("given summaries when rendered as a table then it has a header and one row per group", func(t *testing.T) {
		var buf bytes.Buffer
		if err := renderEntrySummaries(&buf, summaries); err != nil {
			t.Fatalf("render: %v", err)
		}
		out := buf.String()
		for _, want := range []string{"CLIENT", "DATE", "HOURS", "NOTES", "Acme", "2026-06-08", "1:45", "standup, code review"} {
			if !strings.Contains(out, want) {
				t.Errorf("output missing %q:\n%s", want, out)
			}
		}
	})

	t.Run("given no summaries when rendered as a table then it prints a placeholder", func(t *testing.T) {
		var buf bytes.Buffer
		if err := renderEntrySummaries(&buf, nil); err != nil {
			t.Fatalf("render: %v", err)
		}
		if got := buf.String(); got != "(no time entries)\n" {
			t.Errorf("got %q, want %q", got, "(no time entries)\n")
		}
	})

	t.Run("given the json flag when rendered then it emits the aggregated objects", func(t *testing.T) {
		outputJSON = true
		defer func() { outputJSON = false }()
		var buf bytes.Buffer
		if err := renderEntrySummaries(&buf, summaries); err != nil {
			t.Fatalf("render: %v", err)
		}
		var got []entrySummary
		if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal: %v\noutput: %s", err, buf.String())
		}
		if !reflect.DeepEqual(got, summaries) {
			t.Errorf("got %+v, want %+v", got, summaries)
		}
	})
}
