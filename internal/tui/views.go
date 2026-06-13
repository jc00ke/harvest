// Per-view render functions and status-line helpers.
package tui

import (
	"fmt"
	"strings"
	"time"
)

func (m Model) renderProjectSelectView() string {
	width := m.shellWidth()

	titleBar := m.renderTitleBar()

	// Breadcrumb header
	breadcrumb := "  " + AccentText.Render("New Time Entry") + ArrowStyle.Render(" → ") + MutedText.Render("Step 1: Choose Project")

	divider := "  " + RenderDividerWidth(width-4)

	// Render the list
	listView := m.projectList.View()

	content := strings.Join([]string{titleBar, breadcrumb, divider, "", listView}, "\n")

	footerKeys := []string{
		RenderKeybinding("↑↓", "navigate"),
		RenderKeybinding("/", "filter"),
		RenderKeybinding("enter", "select"),
		RenderKeybinding("esc", "back"),
	}

	return m.buildShellBox(content, width, footerKeys)
}

func (m Model) renderTaskSelectView() string {
	width := m.shellWidth()

	titleBar := m.renderTitleBar()

	// Breadcrumb header
	var breadcrumb string
	if m.editingEntry != nil {
		breadcrumb = "  " + AccentText.Render("Edit Time Entry") + ArrowStyle.Render(" → ") + MutedText.Render("Change Task")
	} else {
		breadcrumb = "  " + AccentText.Render("New Time Entry") + ArrowStyle.Render(" → ") + MutedText.Render("Step 2: Choose Task")
	}

	// Show selected project
	projectInfo := ""
	if m.selectedProject != nil {
		projectInfo = "  " + MutedText.Render(fmt.Sprintf("Project: %s → %s",
			m.selectedProject.Client.Name,
			m.selectedProject.Name))
	}

	divider := "  " + RenderDividerWidth(width-4)

	// Render the list
	listView := m.taskList.View()

	content := strings.Join([]string{titleBar, breadcrumb, projectInfo, divider, "", listView}, "\n")

	footerKeys := []string{
		RenderKeybinding("↑↓", "navigate"),
		RenderKeybinding("/", "filter"),
		RenderKeybinding("enter", "select"),
		RenderKeybinding("esc", "back"),
	}

	return m.buildShellBox(content, width, footerKeys)
}

func (m Model) renderEditView() string {
	width := m.shellWidth()

	titleBar := m.renderTitleBar()

	// Breadcrumb
	breadcrumb := "  " + AccentText.Render("Edit Time Entry")

	// Entry info (client → project breadcrumb, task is now an editable field)
	info := ""
	if m.editingEntry != nil {
		info = "  " + MutedText.Render(fmt.Sprintf("%s → %s",
			m.editingEntry.Client.Name,
			m.editingEntry.Project.Name))
	}

	divider := "  " + RenderDividerWidth(width-4)

	// Build field views
	taskLabel := fieldLabel("Task:", m.editCurrentField == 0)
	taskName := ""
	if m.editTask != nil {
		taskName = m.editTask.Name
	}
	taskView := taskName
	if m.editCurrentField == 0 {
		taskView = taskName + MutedText.Render("  (press enter to change)")
	}

	notesLabel := fieldLabel("Notes:", m.editCurrentField == 1)
	var notesView string
	if m.editNotesInput != nil {
		notesView = m.editNotesInput.View()
	} else {
		notesView = m.editNotes
	}

	durationLabel := fieldLabel("Duration:", m.editCurrentField == 2)
	var durationView string
	if m.editDurationInput != nil {
		durationView = m.editDurationInput.View()
	} else {
		durationView = m.editHours
	}

	// Status message if any
	statusLine := m.renderStatusLine()

	contentLines := []string{
		titleBar,
		breadcrumb,
		info,
		divider,
		"",
		"  " + taskLabel + " " + taskView,
		"",
		"  " + notesLabel + " " + notesView,
		"",
		"  " + durationLabel + " " + durationView,
	}
	if statusLine != "" {
		contentLines = append(contentLines, "", statusLine)
	}

	content := strings.Join(contentLines, "\n")

	footerKeys := []string{
		RenderKeybinding("tab", "next field"),
	}
	if m.editCurrentField == 0 {
		footerKeys = append(footerKeys, RenderKeybinding("enter", "select"))
	}
	footerKeys = append(footerKeys,
		RenderKeybinding("ctrl+s", "save"),
		RenderKeybinding("esc", "cancel"),
	)

	return m.buildShellBox(content, width, footerKeys)
}

// fieldLabel renders a field label with ▶ indicator if active.
func fieldLabel(label string, active bool) string {
	if active {
		return AccentText.Render("▶ " + label)
	}
	return MutedText.Render("  " + label)
}

func (m Model) renderConfirmDeleteView() string {
	width := m.shellWidth()

	titleBar := m.renderTitleBar()

	breadcrumb := "  " + ErrorText.Render("Confirm Delete")

	divider := "  " + RenderDividerWidth(width-4)

	// Entry details
	var detailLines []string
	detailLines = append(detailLines, "  "+AccentText.Render("Are you sure you want to delete this entry?"))
	if m.editingEntry != nil {
		detailLines = append(detailLines, "")
		if m.editingEntry.Notes != "" {
			detailLines = append(detailLines, "  "+MutedText.Render("Notes: "+m.editingEntry.Notes))
		}
		detailLines = append(detailLines, "  "+MutedText.Render("Duration: "+formatHoursSimple(m.editingEntry.Hours)))
	}

	contentLines := []string{titleBar, breadcrumb, divider, ""}
	contentLines = append(contentLines, detailLines...)

	content := strings.Join(contentLines, "\n")

	footerKeys := []string{
		RenderKeybinding("y", "confirm"),
		RenderKeybinding("n", "cancel"),
		RenderKeybinding("esc", "cancel"),
	}

	return m.buildShellBox(content, width, footerKeys)
}

func (m Model) renderHelpView() string {
	width := m.shellWidth()

	titleBar := m.renderTitleBar()

	breadcrumb := "  " + AccentText.Render("Help")

	divider := "  " + RenderDividerWidth(width-4)

	// Keybindings organized by category
	contentLines := []string{
		titleBar,
		breadcrumb,
		divider,
		"",
		"  " + AccentText.Render("Navigation"),
		"    ↑/k       Move up",
		"    ↓/j       Move down",
		"    ←/h       Previous day",
		"    →/l       Next day",
		"    t         Jump to today",
		"    w         Weekly summary",
		"",
		"  " + AccentText.Render("Time Entry Actions"),
		"    n         New entry",
		"    e         Edit entry",
		"    d         Delete entry",
		"    s         Start/stop timer",
		"",
		"  " + AccentText.Render("General"),
		"    ?         Toggle this help",
		"    q/Esc     Quit/Go back",
		"    Ctrl+C    Force quit",
	}

	content := strings.Join(contentLines, "\n")

	footerKeys := []string{
		RenderKeybinding("?", "close"),
		RenderKeybinding("esc", "back"),
	}

	return m.buildShellBox(content, width, footerKeys)
}

func (m Model) renderNotesInputView() string {
	width := m.shellWidth()

	titleBar := m.renderTitleBar()

	breadcrumb := "  " + AccentText.Render("New Time Entry") + ArrowStyle.Render(" → ") + MutedText.Render("Step 3: Enter Notes")

	// Selected project and task info
	info := ""
	if m.selectedProject != nil && m.selectedTask != nil {
		info = "  " + MutedText.Render(fmt.Sprintf("%s → %s → %s",
			m.selectedProject.Client.Name,
			m.selectedProject.Name,
			m.selectedTask.Name))
	}

	divider := "  " + RenderDividerWidth(width-4)

	// Input field
	inputView := ""
	if m.notesInput != nil {
		inputView = "  " + m.notesInput.View()
	}

	content := strings.Join([]string{titleBar, breadcrumb, info, divider, "", inputView}, "\n")

	footerKeys := []string{
		RenderKeybinding("enter", "continue"),
		RenderKeybinding("esc", "cancel"),
	}

	return m.buildShellBox(content, width, footerKeys)
}

func (m Model) renderDurationInputView() string {
	width := m.shellWidth()

	titleBar := m.renderTitleBar()

	breadcrumb := "  " + AccentText.Render("New Time Entry") + ArrowStyle.Render(" → ") + MutedText.Render("Step 4: Enter Duration")

	// Selected info
	info := ""
	if m.selectedProject != nil && m.selectedTask != nil {
		info = "  " + MutedText.Render(fmt.Sprintf("%s → %s → %s",
			m.selectedProject.Client.Name,
			m.selectedProject.Name,
			m.selectedTask.Name))
	}

	// Notes info
	notesInfo := ""
	if m.newEntryNotes != "" {
		notesInfo = "  " + MutedText.Render("Notes: "+m.newEntryNotes)
	}

	divider := "  " + RenderDividerWidth(width-4)

	// Input field
	inputView := ""
	if m.durationInput != nil {
		inputView = "  " + m.durationInput.View()
	}

	// Status message
	statusLine := m.renderStatusLine()

	contentLines := []string{titleBar, breadcrumb, info, notesInfo, divider, "", inputView}
	if statusLine != "" {
		contentLines = append(contentLines, "", statusLine)
	}

	content := strings.Join(contentLines, "\n")

	footerKeys := []string{
		RenderKeybinding("enter", "continue"),
		RenderKeybinding("esc", "back"),
	}

	return m.buildShellBox(content, width, footerKeys)
}

func (m Model) renderBillableToggleView() string {
	width := m.shellWidth()

	titleBar := m.renderTitleBar()

	breadcrumb := "  " + AccentText.Render("New Time Entry") + ArrowStyle.Render(" → ") + MutedText.Render("Step 5: Billable Status")

	// Selected info
	info := ""
	if m.selectedProject != nil && m.selectedTask != nil {
		info = "  " + MutedText.Render(fmt.Sprintf("%s → %s → %s",
			m.selectedProject.Client.Name,
			m.selectedProject.Name,
			m.selectedTask.Name))
	}

	// Entry details
	var detailLines []string
	if m.newEntryNotes != "" {
		detailLines = append(detailLines, "  "+MutedText.Render("Notes: "+m.newEntryNotes))
	}
	if m.newEntryHours != "" {
		detailLines = append(detailLines, "  "+MutedText.Render("Duration: "+m.newEntryHours))
	}

	divider := "  " + RenderDividerWidth(width-4)

	// Billable toggle
	billableStatus := "  [ ] Non-billable"
	if m.newEntryBillable {
		billableStatus = "  [x] Billable"
	}

	contentLines := []string{titleBar, breadcrumb, info}
	contentLines = append(contentLines, detailLines...)
	contentLines = append(contentLines, divider, "", billableStatus)

	content := strings.Join(contentLines, "\n")

	footerKeys := []string{
		RenderKeybinding("space", "toggle"),
		RenderKeybinding("enter", "create"),
		RenderKeybinding("esc", "back"),
	}

	return m.buildShellBox(content, width, footerKeys)
}

// setStatusMessage sets a status message with a timestamp
func (m *Model) setStatusMessage(msg string) {
	m.statusMessage = msg
	m.statusMessageTime = time.Now()
}

// clearStatusMessage clears the status message
func (m *Model) clearStatusMessage() {
	m.statusMessage = ""
	m.statusMessageTime = time.Time{}
}

// renderStatusLine returns the status message styled based on its content.
// Success messages render green, errors red, warnings yellow.
func (m Model) renderStatusLine() string {
	if m.statusMessage == "" {
		return ""
	}
	style := SuccessText
	msgLower := strings.ToLower(m.statusMessage)
	if strings.Contains(msgLower, "error") ||
		strings.Contains(msgLower, "failed") ||
		strings.Contains(msgLower, "cannot") ||
		strings.Contains(msgLower, "no tasks") ||
		strings.Contains(msgLower, "invalid") {
		style = ErrorText
	} else if strings.Contains(msgLower, "locked") ||
		strings.Contains(msgLower, "loading") {
		style = WarningText
	}
	return "  " + style.Render(m.statusMessage)
}

// hasRunningTimer checks if any time entry has a running timer
func (m Model) hasRunningTimer() bool {
	for _, entry := range m.timeEntries {
		if entry.IsRunning {
			return true
		}
	}
	return false
}

// renderNewEntryModal renders the new entry form inside the shell box.
func (m Model) renderNewEntryModal() string {
	width := m.shellWidth()

	titleBar := m.renderTitleBar()

	breadcrumb := "  " + AccentText.Render("New Time Entry")

	divider := "  " + RenderDividerWidth(width-4)

	// Project field
	projectValue := "(none selected)"
	if m.selectedProject != nil {
		projectValue = fmt.Sprintf("%s → %s", m.selectedProject.Client.Name, m.selectedProject.Name)
	}

	// Task field
	taskValue := "(none selected)"
	if m.selectedTask != nil {
		taskValue = m.selectedTask.Name
	}

	// Notes field
	var notesView string
	if m.notesInput != nil && m.newEntryCurrentField == 2 {
		m.notesInput.Focus()
		notesView = m.notesInput.View()
	} else if m.notesInput != nil {
		m.notesInput.Blur()
		notesView = m.notesInput.View()
	} else {
		notesView = m.newEntryNotes
	}

	// Duration field
	var durationView string
	if m.durationInput != nil && m.newEntryCurrentField == 3 {
		m.durationInput.Focus()
		durationView = m.durationInput.View()
	} else if m.durationInput != nil {
		m.durationInput.Blur()
		durationView = m.durationInput.View()
	} else {
		durationView = m.newEntryHours
	}

	// Status message
	statusLine := m.renderStatusLine()

	contentLines := []string{
		titleBar,
		breadcrumb,
		divider,
		"",
		"  " + fieldLabel("Project:", m.newEntryCurrentField == 0) + " " + projectValue,
		"",
		"  " + fieldLabel("Task:", m.newEntryCurrentField == 1) + " " + taskValue,
		"",
		"  " + fieldLabel("Notes:", m.newEntryCurrentField == 2) + " " + notesView,
		"",
		"  " + fieldLabel("Duration:", m.newEntryCurrentField == 3) + " " + durationView,
	}
	if statusLine != "" {
		contentLines = append(contentLines, "", statusLine)
	}

	content := strings.Join(contentLines, "\n")

	footerKeys := []string{
		RenderKeybinding("tab", "next"),
	}
	if m.newEntryCurrentField <= 1 {
		footerKeys = append(footerKeys, RenderKeybinding("enter", "select"))
	}
	footerKeys = append(footerKeys,
		RenderKeybinding("ctrl+s", "save"),
		RenderKeybinding("esc", "cancel"),
	)

	return m.buildShellBox(content, width, footerKeys)
}
