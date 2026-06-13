// Bubble Tea messages and the async commands that produce them.
package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jc00ke/harvest/internal/harvest"
)

// Messages for handling async operations
type timeEntriesFetchedMsg struct {
	entries []harvest.TimeEntry
	err     error
}

// tickMsg is sent periodically to update running timers
type tickMsg time.Time

type projectsWithTasksFetchedMsg struct {
	projectsWithTasks []harvest.ProjectWithTasks
	err               error
}

type timeEntryStartedMsg struct {
	entry *harvest.TimeEntry
	err   error
}

type timeEntryStoppedMsg struct {
	entry *harvest.TimeEntry
	err   error
}

type timeEntryCreatedMsg struct {
	entry *harvest.TimeEntry
	err   error
}

type timeEntryUpdatedMsg struct {
	entry *harvest.TimeEntry
	err   error
}

type timeEntryDeletedMsg struct {
	entryID int
	err     error
}

// Commands for fetching data
func fetchTimeEntriesCmd(client *harvest.Client, date time.Time) tea.Cmd {
	return func() tea.Msg {
		dateStr := date.Format("2006-01-02")
		entries, err := client.FetchTimeEntries(context.Background(), dateStr, dateStr)
		return timeEntriesFetchedMsg{entries: entries, err: err}
	}
}

func fetchProjectsWithTasksCmd(client *harvest.Client) tea.Cmd {
	return func() tea.Msg {
		// Fetch projects and task assignments, then aggregate them
		projects, err := client.FetchProjects(context.Background())
		if err != nil {
			return projectsWithTasksFetchedMsg{err: err}
		}

		taskAssignments, err := client.FetchTaskAssignments(context.Background())
		if err != nil {
			return projectsWithTasksFetchedMsg{err: err}
		}

		projectsWithTasks := harvest.AggregateProjectsWithTasks(projects, taskAssignments)
		return projectsWithTasksFetchedMsg{projectsWithTasks: projectsWithTasks}
	}
}

func restartTimeEntryCmd(client *harvest.Client, entryID int) tea.Cmd {
	return func() tea.Msg {
		entry, err := client.RestartTimeEntry(context.Background(), entryID)
		return timeEntryStartedMsg{entry: entry, err: err}
	}
}

func stopTimeEntryCmd(client *harvest.Client, entryID int) tea.Cmd {
	return func() tea.Msg {
		entry, err := client.StopTimeEntry(context.Background(), entryID)
		return timeEntryStoppedMsg{entry: entry, err: err}
	}
}

func deleteTimeEntryCmd(client *harvest.Client, entryID int) tea.Cmd {
	return func() tea.Msg {
		err := client.DeleteTimeEntry(context.Background(), entryID)
		return timeEntryDeletedMsg{entryID: entryID, err: err}
	}
}

// tickCmd returns a command that sends a tick message after a delay
func tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// createTimeEntry creates a new time entry and returns a command
func (m Model) createTimeEntry() tea.Cmd {
	if m.selectedProject == nil || m.selectedTask == nil {
		return nil
	}

	// Parse duration
	hours, err := parseDuration(m.newEntryHours)
	if err != nil {
		return nil
	}

	request := harvest.CreateTimeEntryRequest{
		ProjectID: m.selectedProject.ID,
		TaskID:    m.selectedTask.ID,
		SpentDate: m.currentDate.Format("2006-01-02"),
		Hours:     hours,
		Notes:     m.newEntryNotes,
	}

	return func() tea.Msg {
		entry, err := m.harvestClient.CreateTimeEntry(context.Background(), request)
		if err != nil {
			return timeEntryCreatedMsg{err: err}
		}

		// Update recents
		m.appState.AddRecent(
			m.selectedProject.Client.ID,
			m.selectedProject.ID,
			m.selectedTask.ID,
		)
		if saveErr := m.appState.Save(); saveErr != nil {
			// Log error but don't fail the operation
			fmt.Printf("Failed to save recents: %v\n", saveErr)
		}

		return timeEntryCreatedMsg{entry: entry}
	}
}

// updateTimeEntry updates an existing time entry and returns a command
func (m Model) updateTimeEntry() tea.Cmd {
	if m.editingEntry == nil {
		return nil
	}

	// Validate duration
	hours, err := parseDuration(m.editHours)
	if err != nil {
		// Return an error message
		return func() tea.Msg {
			return timeEntryUpdatedMsg{err: fmt.Errorf("Invalid duration format. Use HH:MM (e.g., 1:30)")}
		}
	}

	request := harvest.UpdateTimeEntryRequest{
		Hours: &hours,
		Notes: &m.editNotes,
	}

	// Include TaskID if the task was changed
	if m.editTask != nil && m.editTask.ID != m.editingEntry.Task.ID {
		request.TaskID = &m.editTask.ID
	}

	entryID := m.editingEntry.ID

	return func() tea.Msg {
		entry, err := m.harvestClient.UpdateTimeEntry(context.Background(), entryID, request)
		if err != nil {
			return timeEntryUpdatedMsg{err: err}
		}

		return timeEntryUpdatedMsg{entry: entry}
	}
}
