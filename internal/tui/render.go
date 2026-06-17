package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/jc00ke/harvest/internal/harvest"
)

// shellWidth returns the standard box width for the shell.
func (m Model) shellWidth() int {
	width := 65
	if m.width > 0 {
		width = min(m.width-2, 80)
	}
	return width
}

// renderTitleBar renders the title bar with date navigation.
func (m Model) renderTitleBar() string {
	width := m.shellWidth()

	dateStr := m.currentDate.Format("Mon, Jan 2, 2006")
	dateNav := ArrowNavStyle.Render("◀ ") + DateStyle.Render(dateStr) + ArrowNavStyle.Render(" ▶")

	titleText := "  " + TitleStyle.Render("🌾 Harvest Time Tracker")
	titleSuffix := dateNav + "  "
	spacerWidth := width - 2 - lipgloss.Width(titleText) - lipgloss.Width(titleSuffix)
	if spacerWidth < 1 {
		spacerWidth = 1
	}
	return titleText + strings.Repeat(" ", spacerWidth) + titleSuffix
}

// buildShellBox wraps content in a styled box border with parameterized footer keybindings.
func (m Model) buildShellBox(content string, width int, footerKeys []string) string {
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)

	// Top border
	top := "┌" + strings.Repeat("─", width-2) + "┐"

	// Content lines with side borders
	lines := strings.Split(content, "\n")
	var boxedLines []string
	boxedLines = append(boxedLines, borderStyle.Render(top))

	for _, line := range lines {
		padded := line
		if lipgloss.Width(padded) > width-2 {
			padded = truncateStyledLine(padded, width-2)
		}
		// Pad to the interior width so the right border stays aligned. Truncation
		// may land a column short (e.g. when an ellipsis can't fully fill the gap).
		if lineWidth := lipgloss.Width(padded); lineWidth < width-2 {
			padded += strings.Repeat(" ", width-2-lineWidth)
		}
		boxedLines = append(boxedLines, borderStyle.Render("│")+padded+borderStyle.Render("│"))
	}

	// Footer with keybindings
	footerSeparator := borderStyle.Render("├" + strings.Repeat("─", width-2) + "┤")

	footerText := " " + strings.Join(footerKeys, " ")
	footerWidth := lipgloss.Width(footerText)
	if footerWidth > width-2 {
		footerText = truncateStyledLine(footerText, width-2)
	}
	footerWidth = lipgloss.Width(footerText)
	if footerWidth < width-2 {
		footerText = footerText + strings.Repeat(" ", width-2-footerWidth)
	}

	// Bottom border
	bottom := borderStyle.Render("└" + strings.Repeat("─", width-2) + "┘")

	boxedLines = append(boxedLines, footerSeparator)
	boxedLines = append(boxedLines, borderStyle.Render("│")+footerText+borderStyle.Render("│"))
	boxedLines = append(boxedLines, bottom)

	return strings.Join(boxedLines, "\n")
}

// renderLoadingView renders the loading screen shown during startup.
func (m Model) renderLoadingView() string {
	width := m.shellWidth()

	titleBar := m.renderTitleBar()

	loadingMsg := "  " + m.spinner.View() + " " + AccentText.Render("Harvesting your data...")

	contentLines := []string{
		titleBar,
		"",
		loadingMsg,
		"",
	}

	content := strings.Join(contentLines, "\n")

	footerKeys := []string{
		RenderKeybinding("ctrl+c", "quit"),
	}

	return m.buildShellBox(content, width, footerKeys)
}

// renderStyledListView renders the main list view with proper styling.
func (m Model) renderStyledListView() string {
	// Calculate dimensions
	width := m.shellWidth()

	titleBar := m.renderTitleBar()

	// Calculate daily total with accent color (add elapsed time for running entry)
	totalHours := 0.0
	for _, entry := range m.timeEntries {
		if entry.IsRunning && !m.lastFetchTime.IsZero() {
			totalHours += entry.Hours + time.Since(m.lastFetchTime).Hours()
		} else {
			totalHours += entry.Hours
		}
	}
	totalStr := formatHoursSimple(totalHours)

	// Section header with Tokyo Night styling
	// Check if currentDate is today
	now := time.Now()
	isToday := m.currentDate.Year() == now.Year() &&
		m.currentDate.Month() == now.Month() &&
		m.currentDate.Day() == now.Day()

	headerText := "Today's Entries"
	if !isToday {
		headerText = m.currentDate.Format("Monday's Entries")
	}
	entriesText := SectionHeaderStyle.Render(headerText)
	totalLabelText := TotalLabel.Render("Total: ")
	totalValue := TotalValue.Render(totalStr)
	paddingWidth := width - lipgloss.Width(entriesText) - lipgloss.Width(totalLabelText) - lipgloss.Width(totalValue) - 4
	if paddingWidth < 1 {
		paddingWidth = 1
	}
	sectionHeader := "  " + entriesText + strings.Repeat(" ", paddingWidth) + totalLabelText + totalValue + "  "

	// Divider with Tokyo Night styling
	divider := "  " + RenderDividerWidth(width-4)

	// Handle loading state
	if m.loading {
		content := []string{
			titleBar,
			sectionHeader,
			divider,
			"    " + MutedText.Render("Loading..."),
			"",
		}
		return m.wrapInStyledBox(strings.Join(content, "\n"), width)
	}

	// Handle error state
	if m.errorMessage != "" {
		content := []string{
			titleBar,
			sectionHeader,
			divider,
			"    " + ErrorText.Render("Error: "+m.errorMessage),
			"",
		}
		return m.wrapInStyledBox(strings.Join(content, "\n"), width)
	}

	// Handle empty state
	if len(m.timeEntries) == 0 {
		content := []string{
			titleBar,
			sectionHeader,
			divider,
			"",
			"    " + RenderEmptyState(),
			"",
		}
		return m.wrapInStyledBox(strings.Join(content, "\n"), width)
	}

	// Render time entries with styles
	var entryLines []string
	for i, entry := range m.timeEntries {
		isSelected := i == m.selectedEntryIndex
		entryLines = append(entryLines, m.renderStyledTimeEntry(entry, isSelected, width-4))
	}

	// Build content
	contentLines := []string{titleBar, sectionHeader, divider}
	contentLines = append(contentLines, entryLines...)

	// Add status message with appropriate styling
	if statusLine := m.renderStatusLine(); statusLine != "" {
		contentLines = append(contentLines, "", statusLine)
	}

	return m.wrapInStyledBox(strings.Join(contentLines, "\n"), width)
}

// listViewFooterKeys returns the standard footer keybindings for the list view.
func listViewFooterKeys() []string {
	return []string{
		RenderKeybinding("n", "new"),
		RenderKeybinding("e", "edit"),
		RenderKeybinding("s", "timer"),
		RenderKeybinding("d", "del"),
		RenderKeybinding("w", "week"),
		RenderKeybinding("?", "help"),
		RenderKeybinding("q", "quit"),
	}
}

// wrapInStyledBox wraps content in a styled box border with list view footer keys.
func (m Model) wrapInStyledBox(content string, width int) string {
	return m.buildShellBox(content, width, listViewFooterKeys())
}

// renderStyledTimeEntry renders a single time entry with Tokyo Night styling.
func (m Model) renderStyledTimeEntry(entry harvest.TimeEntry, isSelected bool, maxWidth int) string {
	var lines []string

	// Build the entry path
	clientName := truncateString(entry.Client.Name, 20)
	projectName := truncateString(entry.Project.Name, 25)
	taskName := truncateString(entry.Task.Name, 20)

	// For running entries, add elapsed time since last fetch for a live display
	displayHours := entry.Hours
	if entry.IsRunning && !m.lastFetchTime.IsZero() {
		elapsed := time.Since(m.lastFetchTime).Hours()
		displayHours += elapsed
	}

	// formatHoursSimple renders HH:MM (5 columns) for any duration under 100
	// hours, which covers every per-entry value.
	const durationCellWidth = 5

	// Build styled components with optional selected background
	var entryPath, styledDuration, indicator string
	if isSelected {
		bg := selectedBg
		entryPath = ClientStyle.Background(bg).Render(clientName) +
			ArrowStyle.Background(bg).Render(" → ") +
			ProjectStyle.Background(bg).Render(projectName) +
			ArrowStyle.Background(bg).Render(" → ") +
			TaskStyle.Background(bg).Render(taskName)

		// Durations are a constant HH:MM width, so the cell is sized to fit
		// exactly (no internal padding between the indicator and the digits).
		if entry.IsRunning {
			styledDuration = RunningDurationStyle.Background(bg).Width(durationCellWidth).Render(formatHoursSimple(displayHours))
		} else {
			styledDuration = DurationStyle.Background(bg).Width(durationCellWidth).Render(formatHoursSimple(displayHours))
		}

		// Indicator sits to the left of the duration. Its separating space is
		// styled with the row background so it doesn't render as an unstyled gap
		// on the highlighted row.
		sep := lipgloss.NewStyle().Background(bg).Render(" ")
		if entry.IsRunning {
			indicator = RunningDot.Background(bg).Render("●") + sep
		} else if entry.IsLocked {
			indicator = LockedIcon.Background(bg).Render("🔒") + sep
		}
	} else {
		entryPath = RenderEntryPath(clientName, projectName, taskName)

		if entry.IsRunning {
			styledDuration = RunningDurationStyle.Width(durationCellWidth).Render(formatHoursSimple(displayHours))
		} else if entry.IsLocked {
			styledDuration = DurationStyle.Copy().Foreground(mutedText).Width(durationCellWidth).Render(formatHoursSimple(displayHours))
		} else {
			styledDuration = DurationStyle.Width(durationCellWidth).Render(formatHoursSimple(displayHours))
		}

		// Indicator sits to the left of the duration.
		if entry.IsRunning {
			indicator = RunningDot.Render("●") + " "
		} else if entry.IsLocked {
			indicator = LockedIcon.Render("🔒") + " "
		}
	}

	// Both row styles add 3 columns of left overhead (selected: 1 border + 2
	// padding, unselected: 3 padding) and render to a total width of maxWidth,
	// so the content area each holds is maxWidth-3. Right-align the
	// indicator+duration within that area, reserving a trailing column so the
	// duration never abuts the box border. Sizing padding identically for both
	// states keeps the duration column aligned regardless of selection.
	const leftOverhead = 3
	const trailingGap = 1
	contentWidth := maxWidth - leftOverhead
	pathWidth := lipgloss.Width(entryPath)
	durationWidth := lipgloss.Width(styledDuration)
	indicatorWidth := lipgloss.Width(indicator)
	padding := contentWidth - pathWidth - durationWidth - indicatorWidth - trailingGap
	if padding < 1 {
		padding = 1
	}

	// Build the entry line
	var entryLine string
	if isSelected {
		// Selected entry with accent bar and full-width background. Width is
		// maxWidth-1 because the left border adds the remaining column.
		bgSpacer := lipgloss.NewStyle().Background(selectedBg).Render(strings.Repeat(" ", padding))
		entryContent := entryPath + bgSpacer + indicator + styledDuration
		entryLine = SelectedEntry.Width(maxWidth - 1).Render(entryContent)
	} else {
		// Unselected entry with left padding.
		entryContent := entryPath + strings.Repeat(" ", padding) + indicator + styledDuration
		if entry.IsLocked {
			entryContent = LockedEntryStyle.Render(entryContent)
		}
		entryLine = UnselectedEntry.Width(maxWidth).Render(entryContent)
	}

	lines = append(lines, entryLine)

	// Notes line with Tokyo Night styling
	if entry.Notes != "" {
		notesText := RenderNotes(truncateString(entry.Notes, maxWidth-8))
		// Indent notes to align with entry content (3 chars from entry style + 2 indent)
		lines = append(lines, "     "+notesText)
	}

	return strings.Join(lines, "\n")
}

// truncateStyledLine truncates a styled line to fit within maxWidth, preserving
// ANSI styling and appending an ellipsis when the line is clipped.
func truncateStyledLine(line string, maxWidth int) string {
	if lipgloss.Width(line) <= maxWidth {
		return line
	}
	if maxWidth <= 1 {
		return ansi.Truncate(line, maxWidth, "")
	}
	return ansi.Truncate(line, maxWidth, "…")
}

// max returns the maximum of two integers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
