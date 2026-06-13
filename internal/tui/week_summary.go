package tui

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jc00ke/harvest/internal/harvest"
)

// weekEntriesFetchedMsg carries the result of fetching a week of time entries.
type weekEntriesFetchedMsg struct {
	entries []harvest.TimeEntry
	err     error
}

// fetchWeekEntriesCmd fetches all time entries for the week starting at weekStart.
func fetchWeekEntriesCmd(client *harvest.Client, weekStart time.Time) tea.Cmd {
	return func() tea.Msg {
		from := weekStart.Format("2006-01-02")
		to := weekStart.AddDate(0, 0, 6).Format("2006-01-02")
		entries, err := client.FetchTimeEntries(context.Background(), from, to)
		return weekEntriesFetchedMsg{entries: entries, err: err}
	}
}

// mondayOf returns the Monday of the week containing t, truncated to midnight.
func mondayOf(t time.Time) time.Time {
	offset := (int(t.Weekday()) + 6) % 7
	monday := t.AddDate(0, 0, -offset)
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, monday.Location())
}

// projectHours aggregates hours for one client → project pair.
type projectHours struct {
	name  string
	hours float64
}

// weekSummary holds per-day and per-project hour totals for one week.
type weekSummary struct {
	dayHours [7]float64
	projects []projectHours
	total    float64
}

// summarizeWeek buckets entries by weekday (Monday first) and groups them by
// client → project, sorted by hours descending then name. Entries dated
// outside the week starting at weekStart are ignored.
func summarizeWeek(entries []harvest.TimeEntry, weekStart time.Time) weekSummary {
	var dayDates [7]string
	for i := range dayDates {
		dayDates[i] = weekStart.AddDate(0, 0, i).Format("2006-01-02")
	}

	var s weekSummary
	byProject := make(map[string]float64)
	for _, e := range entries {
		dayIndex := -1
		for i, d := range dayDates {
			if e.SpentDate == d {
				dayIndex = i
				break
			}
		}
		if dayIndex < 0 {
			continue
		}
		s.dayHours[dayIndex] += e.Hours
		s.total += e.Hours
		byProject[e.Client.Name+" → "+e.Project.Name] += e.Hours
	}

	for name, hours := range byProject {
		s.projects = append(s.projects, projectHours{name: name, hours: hours})
	}
	sort.Slice(s.projects, func(i, j int) bool {
		if s.projects[i].hours != s.projects[j].hours {
			return s.projects[i].hours > s.projects[j].hours
		}
		return s.projects[i].name < s.projects[j].name
	})
	return s
}

// handleWeekSummaryKeys handles key presses in the weekly summary view.
func (m Model) handleWeekSummaryKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	keys := DefaultKeyMap()

	switch msg.String() {
	case "esc", "q", "w":
		m.currentView = ViewList
		return m, nil
	}

	switch {
	case key.Matches(msg, keys.PrevDay):
		m.weekStart = m.weekStart.AddDate(0, 0, -7)
		m.weekLoading = true
		return m, fetchWeekEntriesCmd(m.harvestClient, m.weekStart)

	case key.Matches(msg, keys.NextDay):
		m.weekStart = m.weekStart.AddDate(0, 0, 7)
		m.weekLoading = true
		return m, fetchWeekEntriesCmd(m.harvestClient, m.weekStart)

	case key.Matches(msg, keys.Today):
		m.weekStart = mondayOf(time.Now())
		m.weekLoading = true
		return m, fetchWeekEntriesCmd(m.harvestClient, m.weekStart)
	}

	return m, nil
}

// renderWeekTitleBar renders the title bar with week range navigation.
func (m Model) renderWeekTitleBar() string {
	width := m.shellWidth()

	weekEnd := m.weekStart.AddDate(0, 0, 6)
	rangeStr := m.weekStart.Format("Jan 2") + " – " + weekEnd.Format("Jan 2, 2006")
	weekNav := ArrowNavStyle.Render("◀ ") + DateStyle.Render(rangeStr) + ArrowNavStyle.Render(" ▶")

	titleText := "  " + TitleStyle.Render("🌾 Harvest Time Tracker")
	titleSuffix := weekNav + "  "
	spacerWidth := width - 2 - lipgloss.Width(titleText) - lipgloss.Width(titleSuffix)
	if spacerWidth < 1 {
		spacerWidth = 1
	}
	return titleText + strings.Repeat(" ", spacerWidth) + titleSuffix
}

// renderWeekSummaryView renders the weekly summary view.
func (m Model) renderWeekSummaryView() string {
	width := m.shellWidth()

	titleBar := m.renderWeekTitleBar()

	summary := summarizeWeek(m.weekEntries, m.weekStart)

	// Section header with right-aligned weekly total
	headerText := SectionHeaderStyle.Render("Weekly Summary")
	totalLabelText := TotalLabel.Render("Total: ")
	totalValue := TotalValue.Render(formatHoursSimple(summary.total))
	paddingWidth := width - lipgloss.Width(headerText) - lipgloss.Width(totalLabelText) - lipgloss.Width(totalValue) - 4
	if paddingWidth < 1 {
		paddingWidth = 1
	}
	sectionHeader := "  " + headerText + strings.Repeat(" ", paddingWidth) + totalLabelText + totalValue + "  "

	divider := "  " + RenderDividerWidth(width-4)

	footerKeys := []string{
		RenderKeybinding("←/→", "prev/next week"),
		RenderKeybinding("t", "this week"),
		RenderKeybinding("esc", "back"),
	}

	if m.weekLoading {
		content := strings.Join([]string{
			titleBar,
			sectionHeader,
			divider,
			"    " + MutedText.Render("Loading..."),
			"",
		}, "\n")
		return m.buildShellBox(content, width, footerKeys)
	}

	if m.errorMessage != "" {
		content := strings.Join([]string{
			titleBar,
			sectionHeader,
			divider,
			"    " + ErrorText.Render("Error: "+m.errorMessage),
			"",
		}, "\n")
		return m.buildShellBox(content, width, footerKeys)
	}

	contentLines := []string{titleBar, sectionHeader, divider, ""}

	if summary.total == 0 {
		contentLines = append(contentLines,
			"    "+EmptyState.Render("No time tracked this week."),
			"",
		)
		content := strings.Join(contentLines, "\n")
		return m.buildShellBox(content, width, footerKeys)
	}

	contentLines = append(contentLines, m.renderWeekDayRows(summary, width)...)
	contentLines = append(contentLines, "", divider, "")
	contentLines = append(contentLines, m.renderWeekProjectRows(summary, width)...)

	content := strings.Join(contentLines, "\n")
	return m.buildShellBox(content, width, footerKeys)
}

// renderWeekDayRows renders one row per weekday with hours and a scaled bar.
func (m Model) renderWeekDayRows(summary weekSummary, width int) []string {
	maxHours := 0.0
	for _, h := range summary.dayHours {
		if h > maxHours {
			maxHours = h
		}
	}

	maxBarWidth := width - 24
	if maxBarWidth > 32 {
		maxBarWidth = 32
	}
	if maxBarWidth < 4 {
		maxBarWidth = 4
	}

	today := time.Now().Format("2006-01-02")

	var lines []string
	for i, hours := range summary.dayHours {
		day := m.weekStart.AddDate(0, 0, i)
		label := day.Format("Mon Jan 2")
		isToday := day.Format("2006-01-02") == today

		labelStyle := MutedText
		if isToday {
			labelStyle = AccentText
		}

		durationStr := formatHoursSimple(hours)
		var duration string
		if hours > 0 {
			duration = DurationStyle.Render(durationStr)
		} else {
			duration = DurationStyle.Foreground(dimText).Render(durationStr)
		}

		bar := ""
		if hours > 0 && maxHours > 0 {
			barWidth := int(hours / maxHours * float64(maxBarWidth))
			if barWidth < 1 {
				barWidth = 1
			}
			bar = "  " + SuccessText.Render(strings.Repeat("█", barWidth))
		}

		lines = append(lines, "    "+labelStyle.Render(padRight(label, 11))+duration+bar)
	}
	return lines
}

// renderWeekProjectRows renders the per-project breakdown sorted by hours.
func (m Model) renderWeekProjectRows(summary weekSummary, width int) []string {
	lines := []string{"  " + SectionHeaderStyle.Render("By Project")}
	for _, p := range summary.projects {
		name := truncateString(p.name, width-18)
		duration := DurationStyle.Render(formatHoursSimple(p.hours))
		padding := width - lipgloss.Width(name) - lipgloss.Width(duration) - 8
		if padding < 1 {
			padding = 1
		}
		lines = append(lines, "    "+TaskStyle.Render(name)+strings.Repeat(" ", padding)+duration)
	}
	return lines
}

// padRight pads s with spaces to the given width.
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
