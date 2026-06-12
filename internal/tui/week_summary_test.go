package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jc00ke/harvest/internal/harvest"
)

var errTest = errors.New("test error")

func TestMondayOf(t *testing.T) {
	tests := []struct {
		name string
		date time.Time
		want string
	}{
		{
			name: "given a Wednesday then returns the preceding Monday",
			date: time.Date(2026, 6, 10, 15, 30, 0, 0, time.UTC),
			want: "2026-06-08",
		},
		{
			name: "given a Monday then returns the same day",
			date: time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC),
			want: "2026-06-08",
		},
		{
			name: "given a Sunday then returns the Monday six days earlier",
			date: time.Date(2026, 6, 14, 23, 59, 0, 0, time.UTC),
			want: "2026-06-08",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, want := mondayOf(tt.date).Format("2006-01-02"), tt.want; got != want {
				t.Errorf("mondayOf=%s, want=%s", got, want)
			}
		})
	}
}

func TestSummarizeWeek(t *testing.T) {
	weekStart := time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC) // Monday

	entries := []harvest.TimeEntry{
		{
			ID:        1,
			SpentDate: "2026-06-08",
			Hours:     2.5,
			Client:    harvest.TimeEntryClient{ID: 1, Name: "Acme"},
			Project:   harvest.TimeEntryProject{ID: 1, Name: "Website"},
		},
		{
			ID:        2,
			SpentDate: "2026-06-08",
			Hours:     1.5,
			Client:    harvest.TimeEntryClient{ID: 2, Name: "BigCorp"},
			Project:   harvest.TimeEntryProject{ID: 2, Name: "API"},
		},
		{
			ID:        3,
			SpentDate: "2026-06-10",
			Hours:     3.0,
			Client:    harvest.TimeEntryClient{ID: 1, Name: "Acme"},
			Project:   harvest.TimeEntryProject{ID: 1, Name: "Website"},
		},
		{
			ID:        4,
			SpentDate: "2026-06-14",
			Hours:     1.0,
			Client:    harvest.TimeEntryClient{ID: 1, Name: "Acme"},
			Project:   harvest.TimeEntryProject{ID: 3, Name: "Mobile"},
		},
	}

	summary := summarizeWeek(entries, weekStart)

	t.Run("buckets hours by weekday", func(t *testing.T) {
		if got, want := summary.dayHours[0], 4.0; got != want {
			t.Errorf("Monday hours=%f, want=%f", got, want)
		}
		if got, want := summary.dayHours[2], 3.0; got != want {
			t.Errorf("Wednesday hours=%f, want=%f", got, want)
		}
		if got, want := summary.dayHours[6], 1.0; got != want {
			t.Errorf("Sunday hours=%f, want=%f", got, want)
		}
		if got, want := summary.dayHours[1], 0.0; got != want {
			t.Errorf("Tuesday hours=%f, want=%f", got, want)
		}
	})

	t.Run("computes weekly total", func(t *testing.T) {
		if got, want := summary.total, 8.0; got != want {
			t.Errorf("total=%f, want=%f", got, want)
		}
	})

	t.Run("groups by project sorted by hours descending", func(t *testing.T) {
		if got, want := len(summary.projects), 3; got != want {
			t.Fatalf("project count=%d, want=%d", got, want)
		}
		if got, want := summary.projects[0].name, "Acme → Website"; got != want {
			t.Errorf("first project=%s, want=%s", got, want)
		}
		if got, want := summary.projects[0].hours, 5.5; got != want {
			t.Errorf("first project hours=%f, want=%f", got, want)
		}
		if got, want := summary.projects[1].name, "BigCorp → API"; got != want {
			t.Errorf("second project=%s, want=%s", got, want)
		}
	})

	t.Run("ignores entries outside the week", func(t *testing.T) {
		outside := append(entries, harvest.TimeEntry{
			ID:        5,
			SpentDate: "2026-06-15",
			Hours:     9.0,
			Client:    harvest.TimeEntryClient{ID: 1, Name: "Acme"},
			Project:   harvest.TimeEntryProject{ID: 1, Name: "Website"},
		})
		s := summarizeWeek(outside, weekStart)
		if got, want := s.total, 8.0; got != want {
			t.Errorf("total=%f, want=%f", got, want)
		}
	})
}

func TestWeekSummaryKeyOpensView(t *testing.T) {
	t.Run("given list view when 'w' pressed then opens week summary and fetches entries", func(t *testing.T) {
		model := newTestModel()
		model.currentView = ViewList
		model.currentDate = time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC) // Wednesday

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}}
		newModel, cmd := model.handleListViewKeys(msg)
		m := newModel.(Model)

		if got, want := m.currentView, ViewWeekSummary; got != want {
			t.Errorf("currentView=%v, want=%v", got, want)
		}
		if got, want := m.weekStart.Format("2006-01-02"), "2026-06-08"; got != want {
			t.Errorf("weekStart=%s, want=%s", got, want)
		}
		if !m.weekLoading {
			t.Error("weekLoading=false, want=true")
		}
		if cmd == nil {
			t.Error("cmd=nil, want fetch command")
		}
	})
}

func TestWeekSummaryNavigation(t *testing.T) {
	setup := func() Model {
		model := newTestModel()
		model.currentView = ViewWeekSummary
		model.weekStart = time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC)
		return model
	}

	t.Run("given week view when left pressed then navigates to previous week", func(t *testing.T) {
		model := setup()
		msg := tea.KeyMsg{Type: tea.KeyLeft}
		newModel, cmd := model.handleWeekSummaryKeys(msg)
		m := newModel.(Model)

		if got, want := m.weekStart.Format("2006-01-02"), "2026-06-01"; got != want {
			t.Errorf("weekStart=%s, want=%s", got, want)
		}
		if cmd == nil {
			t.Error("cmd=nil, want fetch command")
		}
	})

	t.Run("given week view when right pressed then navigates to next week", func(t *testing.T) {
		model := setup()
		msg := tea.KeyMsg{Type: tea.KeyRight}
		newModel, cmd := model.handleWeekSummaryKeys(msg)
		m := newModel.(Model)

		if got, want := m.weekStart.Format("2006-01-02"), "2026-06-15"; got != want {
			t.Errorf("weekStart=%s, want=%s", got, want)
		}
		if cmd == nil {
			t.Error("cmd=nil, want fetch command")
		}
	})

	t.Run("given week view when 't' pressed then navigates to current week", func(t *testing.T) {
		model := setup()
		model.weekStart = time.Date(2020, 1, 6, 0, 0, 0, 0, time.UTC)
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}
		newModel, cmd := model.handleWeekSummaryKeys(msg)
		m := newModel.(Model)

		if got, want := m.weekStart.Format("2006-01-02"), mondayOf(time.Now()).Format("2006-01-02"); got != want {
			t.Errorf("weekStart=%s, want=%s", got, want)
		}
		if cmd == nil {
			t.Error("cmd=nil, want fetch command")
		}
	})

	t.Run("given week view when esc pressed then returns to list view", func(t *testing.T) {
		model := setup()
		msg := tea.KeyMsg{Type: tea.KeyEscape}
		newModel, _ := model.handleWeekSummaryKeys(msg)
		m := newModel.(Model)

		if got, want := m.currentView, ViewList; got != want {
			t.Errorf("currentView=%v, want=%v", got, want)
		}
	})

	t.Run("given week view when 'w' pressed then returns to list view", func(t *testing.T) {
		model := setup()
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}}
		newModel, _ := model.handleWeekSummaryKeys(msg)
		m := newModel.(Model)

		if got, want := m.currentView, ViewList; got != want {
			t.Errorf("currentView=%v, want=%v", got, want)
		}
	})
}

func TestWeekEntriesFetchedMsg(t *testing.T) {
	t.Run("given successful fetch then stores entries and clears loading", func(t *testing.T) {
		model := newTestModel()
		model.currentView = ViewWeekSummary
		model.weekLoading = true

		entries := []harvest.TimeEntry{{ID: 1, SpentDate: "2026-06-08", Hours: 2.0}}
		newModel, _ := model.Update(weekEntriesFetchedMsg{entries: entries})
		m := newModel.(Model)

		if got, want := len(m.weekEntries), 1; got != want {
			t.Errorf("weekEntries count=%d, want=%d", got, want)
		}
		if m.weekLoading {
			t.Error("weekLoading=true, want=false")
		}
	})

	t.Run("given failed fetch then sets error message", func(t *testing.T) {
		model := newTestModel()
		model.currentView = ViewWeekSummary
		model.weekLoading = true

		newModel, _ := model.Update(weekEntriesFetchedMsg{err: errTest})
		m := newModel.(Model)

		if m.weekLoading {
			t.Error("weekLoading=true, want=false")
		}
		if got := m.errorMessage; !strings.Contains(got, "test error") {
			t.Errorf("errorMessage=%q, want it to contain %q", got, "test error")
		}
	})
}

func TestRenderWeekSummaryView(t *testing.T) {
	model := newTestModel()
	model.currentView = ViewWeekSummary
	model.weekStart = time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC)
	model.weekEntries = []harvest.TimeEntry{
		{
			ID:        1,
			SpentDate: "2026-06-08",
			Hours:     2.5,
			Client:    harvest.TimeEntryClient{ID: 1, Name: "Acme"},
			Project:   harvest.TimeEntryProject{ID: 1, Name: "Website"},
		},
		{
			ID:        2,
			SpentDate: "2026-06-10",
			Hours:     3.0,
			Client:    harvest.TimeEntryClient{ID: 1, Name: "Acme"},
			Project:   harvest.TimeEntryProject{ID: 1, Name: "Website"},
		},
	}

	output := model.View()

	t.Run("shows the week range in the header", func(t *testing.T) {
		if got, want := output, "Jun 8 – Jun 14, 2026"; !strings.Contains(got, want) {
			t.Errorf("output missing week range %q", want)
		}
	})

	t.Run("shows day rows with totals", func(t *testing.T) {
		for _, want := range []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"} {
			if !strings.Contains(output, want) {
				t.Errorf("output missing day label %q", want)
			}
		}
		if got, want := output, "2:30"; !strings.Contains(got, want) {
			t.Errorf("output missing Monday total %q", want)
		}
		if got, want := output, "3:00"; !strings.Contains(got, want) {
			t.Errorf("output missing Wednesday total %q", want)
		}
	})

	t.Run("shows weekly total", func(t *testing.T) {
		if got, want := output, "5:30"; !strings.Contains(got, want) {
			t.Errorf("output missing weekly total %q", want)
		}
	})

	t.Run("shows project breakdown", func(t *testing.T) {
		if got, want := output, "Acme"; !strings.Contains(got, want) {
			t.Errorf("output missing client name %q", want)
		}
		if got, want := output, "Website"; !strings.Contains(got, want) {
			t.Errorf("output missing project name %q", want)
		}
	})

	t.Run("given loading state then shows loading message", func(t *testing.T) {
		loadingModel := model
		loadingModel.weekLoading = true
		if got, want := loadingModel.View(), "Loading"; !strings.Contains(got, want) {
			t.Errorf("output missing %q while loading", want)
		}
	})

	t.Run("given no entries then shows empty message", func(t *testing.T) {
		emptyModel := model
		emptyModel.weekEntries = nil
		if got, want := emptyModel.View(), "No time tracked this week"; !strings.Contains(got, want) {
			t.Errorf("output missing empty state %q", want)
		}
	})
}
