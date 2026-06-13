package demo

import (
	"testing"
	"time"

	"github.com/jc00ke/harvest/internal/harvest"
)

// newTestClient starts a demo server and returns a harvest client pointed at
// it. The server is closed automatically when the test finishes.
func newTestClient(t *testing.T) *harvest.Client {
	t.Helper()
	server := NewServer(time.Date(2026, 6, 12, 9, 0, 0, 0, time.UTC))
	t.Cleanup(server.Close)
	client := harvest.NewClient("demo", "demo")
	client.SetBaseURL(server.URL)
	return client
}

func TestValidateAuthReturnsDemoUser(t *testing.T) {
	client := newTestClient(t)

	user, err := client.ValidateAuth(t.Context())
	if err != nil {
		t.Fatalf("ValidateAuth() error: %v", err)
	}
	if got, want := user.FirstName+" "+user.LastName, "Demo User"; got != want {
		t.Errorf("user name=%s, want=%s", got, want)
	}
}

func TestFetchProjects(t *testing.T) {
	client := newTestClient(t)

	projects, err := client.FetchProjects(t.Context())
	if err != nil {
		t.Fatalf("FetchProjects() error: %v", err)
	}
	if got, want := len(projects), 3; got != want {
		t.Errorf("len(projects)=%d, want=%d", got, want)
	}
}

func TestFetchTaskAssignmentsCoverEveryProject(t *testing.T) {
	client := newTestClient(t)

	projects, err := client.FetchProjects(t.Context())
	if err != nil {
		t.Fatalf("FetchProjects() error: %v", err)
	}
	assignments, err := client.FetchTaskAssignments(t.Context())
	if err != nil {
		t.Fatalf("FetchTaskAssignments() error: %v", err)
	}

	combined := harvest.AggregateProjectsWithTasks(projects, assignments)
	if got, want := len(combined), len(projects); got != want {
		t.Errorf("projects with tasks=%d, want=%d", got, want)
	}
}

func TestFetchTimeEntriesForSeedDate(t *testing.T) {
	client := newTestClient(t)
	if _, err := client.ValidateAuth(t.Context()); err != nil {
		t.Fatalf("ValidateAuth() error: %v", err)
	}

	entries, err := client.FetchTimeEntries(t.Context(), "2026-06-12", "2026-06-12")
	if err != nil {
		t.Fatalf("FetchTimeEntries() error: %v", err)
	}
	if got, want := len(entries), 3; got != want {
		t.Fatalf("len(entries)=%d, want=%d", got, want)
	}

	running := 0
	for _, e := range entries {
		if e.IsRunning {
			running++
		}
	}
	if got, want := running, 1; got != want {
		t.Errorf("running entries=%d, want=%d", got, want)
	}
}

func TestFetchTimeEntriesFiltersByDateRange(t *testing.T) {
	client := newTestClient(t)
	if _, err := client.ValidateAuth(t.Context()); err != nil {
		t.Fatalf("ValidateAuth() error: %v", err)
	}

	entries, err := client.FetchTimeEntries(t.Context(), "2020-01-01", "2020-01-01")
	if err != nil {
		t.Fatalf("FetchTimeEntries() error: %v", err)
	}
	if got, want := len(entries), 0; got != want {
		t.Errorf("len(entries)=%d, want=%d", got, want)
	}
}

func TestCreateTimeEntryResolvesNamesAndPersists(t *testing.T) {
	client := newTestClient(t)
	if _, err := client.ValidateAuth(t.Context()); err != nil {
		t.Fatalf("ValidateAuth() error: %v", err)
	}

	projects, err := client.FetchProjects(t.Context())
	if err != nil {
		t.Fatalf("FetchProjects() error: %v", err)
	}
	assignments, err := client.FetchTaskAssignments(t.Context())
	if err != nil {
		t.Fatalf("FetchTaskAssignments() error: %v", err)
	}
	combined := harvest.AggregateProjectsWithTasks(projects, assignments)
	project := combined[0].Project
	task := combined[0].Tasks[0]

	created, err := client.CreateTimeEntry(t.Context(), harvest.CreateTimeEntryRequest{
		ProjectID: project.ID,
		TaskID:    task.ID,
		SpentDate: "2026-06-12",
		Hours:     1.5,
		Notes:     "Created in demo mode",
	})
	if err != nil {
		t.Fatalf("CreateTimeEntry() error: %v", err)
	}
	if got, want := created.Project.Name, project.Name; got != want {
		t.Errorf("created.Project.Name=%s, want=%s", got, want)
	}
	if got, want := created.Task.Name, task.Name; got != want {
		t.Errorf("created.Task.Name=%s, want=%s", got, want)
	}
	if got, want := created.Client.Name, project.Client.Name; got != want {
		t.Errorf("created.Client.Name=%s, want=%s", got, want)
	}

	entries, err := client.FetchTimeEntries(t.Context(), "2026-06-12", "2026-06-12")
	if err != nil {
		t.Fatalf("FetchTimeEntries() error: %v", err)
	}
	if got, want := len(entries), 4; got != want {
		t.Errorf("len(entries) after create=%d, want=%d", got, want)
	}
}

func TestCreateTimeEntryRejectsUnknownProject(t *testing.T) {
	client := newTestClient(t)

	_, err := client.CreateTimeEntry(t.Context(), harvest.CreateTimeEntryRequest{
		ProjectID: 999999,
		TaskID:    1,
		SpentDate: "2026-06-12",
		Hours:     1,
	})
	if err == nil {
		t.Error("CreateTimeEntry() with unknown project returned nil error, want error")
	}
}

func TestUpdateTimeEntry(t *testing.T) {
	client := newTestClient(t)
	if _, err := client.ValidateAuth(t.Context()); err != nil {
		t.Fatalf("ValidateAuth() error: %v", err)
	}

	entries, err := client.FetchTimeEntries(t.Context(), "2026-06-12", "2026-06-12")
	if err != nil {
		t.Fatalf("FetchTimeEntries() error: %v", err)
	}

	notes := "Updated notes"
	hours := 2.25
	updated, err := client.UpdateTimeEntry(t.Context(), entries[0].ID, harvest.UpdateTimeEntryRequest{
		Notes: &notes,
		Hours: &hours,
	})
	if err != nil {
		t.Fatalf("UpdateTimeEntry() error: %v", err)
	}
	if got, want := updated.Notes, notes; got != want {
		t.Errorf("updated.Notes=%s, want=%s", got, want)
	}
	if got, want := updated.Hours, hours; got != want {
		t.Errorf("updated.Hours=%f, want=%f", got, want)
	}
}

func TestDeleteTimeEntry(t *testing.T) {
	client := newTestClient(t)
	if _, err := client.ValidateAuth(t.Context()); err != nil {
		t.Fatalf("ValidateAuth() error: %v", err)
	}

	entries, err := client.FetchTimeEntries(t.Context(), "2026-06-12", "2026-06-12")
	if err != nil {
		t.Fatalf("FetchTimeEntries() error: %v", err)
	}
	before := len(entries)

	if err := client.DeleteTimeEntry(t.Context(), entries[0].ID); err != nil {
		t.Fatalf("DeleteTimeEntry() error: %v", err)
	}

	entries, err = client.FetchTimeEntries(t.Context(), "2026-06-12", "2026-06-12")
	if err != nil {
		t.Fatalf("FetchTimeEntries() error: %v", err)
	}
	if got, want := len(entries), before-1; got != want {
		t.Errorf("len(entries) after delete=%d, want=%d", got, want)
	}
}

func TestRestartAndStopTimeEntry(t *testing.T) {
	client := newTestClient(t)
	if _, err := client.ValidateAuth(t.Context()); err != nil {
		t.Fatalf("ValidateAuth() error: %v", err)
	}

	entries, err := client.FetchTimeEntries(t.Context(), "2026-06-12", "2026-06-12")
	if err != nil {
		t.Fatalf("FetchTimeEntries() error: %v", err)
	}
	var stopped harvest.TimeEntry
	for _, e := range entries {
		if !e.IsRunning {
			stopped = e
			break
		}
	}

	restarted, err := client.RestartTimeEntry(t.Context(), stopped.ID)
	if err != nil {
		t.Fatalf("RestartTimeEntry() error: %v", err)
	}
	if got, want := restarted.IsRunning, true; got != want {
		t.Errorf("restarted.IsRunning=%t, want=%t", got, want)
	}

	halted, err := client.StopTimeEntry(t.Context(), stopped.ID)
	if err != nil {
		t.Fatalf("StopTimeEntry() error: %v", err)
	}
	if got, want := halted.IsRunning, false; got != want {
		t.Errorf("halted.IsRunning=%t, want=%t", got, want)
	}
}
