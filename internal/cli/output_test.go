package cli

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/jc00ke/harvest/internal/harvest"
)

// update regenerates the golden files: go test ./internal/cli -update
var update = flag.Bool("update", false, "update golden files")

// checkGolden compares got against testdata/<name>.golden, rewriting the
// file first when -update is set.
func checkGolden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	if *update {
		if err := os.MkdirAll("testdata", 0755); err != nil {
			t.Fatalf("mkdir testdata: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden (run `go test ./internal/cli -update` to regenerate): %v", err)
	}
	if got != string(want) {
		t.Errorf("output does not match %s (run with -update to regenerate):\ngot:\n%s\nwant:\n%s", path, got, want)
	}
}

func goldenEntries() []harvest.TimeEntry {
	return []harvest.TimeEntry{
		{
			ID: 1001, SpentDate: "2026-06-10", Hours: 0.5, Notes: "Sprint planning",
			Client:  harvest.TimeEntryClient{ID: 11, Name: "Acme Corp"},
			Project: harvest.TimeEntryProject{ID: 101, Name: "Website Redesign"},
			Task:    harvest.TimeEntryTask{ID: 203, Name: "Meetings"},
		},
		{
			ID: 1002, SpentDate: "2026-06-10", Hours: 2.25, Notes: "Homepage hero and nav implementation", IsRunning: true, IsBillable: true,
			Client:  harvest.TimeEntryClient{ID: 12, Name: "Globex"},
			Project: harvest.TimeEntryProject{ID: 102, Name: "Mobile App"},
			Task:    harvest.TimeEntryTask{ID: 202, Name: "Development"},
		},
		{
			ID: 1003, SpentDate: "2026-06-09", Hours: 1.75, Notes: "A very long note that should be truncated in the table output because it exceeds the column limit", IsLocked: true,
			Client:  harvest.TimeEntryClient{ID: 13, Name: "Initech"},
			Project: harvest.TimeEntryProject{ID: 103, Name: "Maintenance Retainer"},
			Task:    harvest.TimeEntryTask{ID: 204, Name: "QA"},
		},
	}
}

func TestRenderGolden(t *testing.T) {
	t.Run("given time entries when rendered as a table then output matches the golden file", func(t *testing.T) {
		var buf bytes.Buffer
		if err := renderEntries(&buf, goldenEntries()); err != nil {
			t.Fatalf("render: %v", err)
		}
		checkGolden(t, "entries_table", buf.String())
	})

	t.Run("given a single entry when rendered as a detail view then output matches the golden file", func(t *testing.T) {
		entries := goldenEntries()
		var buf bytes.Buffer
		if err := renderEntry(&buf, &entries[1]); err != nil {
			t.Fatalf("render: %v", err)
		}
		checkGolden(t, "entry_detail", buf.String())
	})

	t.Run("given projects with tasks when rendered as a table then output matches the golden file", func(t *testing.T) {
		projects := []harvest.ProjectWithTasks{
			{
				Project: harvest.Project{ID: 101, Name: "Website Redesign", Client: harvest.ProjectClient{ID: 11, Name: "Acme Corp"}},
				Tasks:   []harvest.Task{{ID: 201, Name: "Design"}, {ID: 202, Name: "Development"}},
			},
			{
				Project: harvest.Project{ID: 102, Name: "Mobile App", Client: harvest.ProjectClient{ID: 12, Name: "Globex"}},
				Tasks:   []harvest.Task{{ID: 204, Name: "QA"}},
			},
		}
		var buf bytes.Buffer
		if err := renderProjects(&buf, projects); err != nil {
			t.Fatalf("render: %v", err)
		}
		checkGolden(t, "projects_table", buf.String())
	})

	t.Run("given invoice summaries when rendered as a table then output matches the golden file", func(t *testing.T) {
		summaries := []invoicePersonSummary{
			{
				Person: "Alex Rivera",
				Hours:  6.5,
				Tasks: []invoiceTaskSummary{
					{Task: "Design", Hours: 2.5},
					{Task: "Development", Hours: 3},
					{Task: "Meetings", Hours: 1},
				},
			},
			{
				Person: "Sam Chen",
				Hours:  2,
				Tasks:  []invoiceTaskSummary{{Task: "Design", Hours: 2}},
			},
		}
		var buf bytes.Buffer
		if err := renderInvoice(&buf, summaries); err != nil {
			t.Fatalf("render: %v", err)
		}
		checkGolden(t, "invoice_table", buf.String())
	})

	t.Run("given a user when rendered as a table then output matches the golden file", func(t *testing.T) {
		user := harvest.User{ID: 1, FirstName: "Demo", LastName: "User", Email: "demo@example.com"}
		var buf bytes.Buffer
		if err := renderUser(&buf, &user); err != nil {
			t.Fatalf("render: %v", err)
		}
		checkGolden(t, "user_table", buf.String())
	})
}
