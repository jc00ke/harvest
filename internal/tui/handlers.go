// Per-view key handlers.
package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jc00ke/harvest/internal/harvest"
)

// handleListViewKeys handles key presses in the main list view.
func (m Model) handleListViewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	keys := DefaultKeyMap()

	// Check for quit first
	switch msg.String() {
	case "q":
		// Show farewell message
		if m.currentUser != nil {
			fullName := m.currentUser.FirstName + " " + m.currentUser.LastName
			return m, tea.Sequence(
				tea.Println(fmt.Sprintf("\nSee you next time, %s!", fullName)),
				tea.Quit,
			)
		}
		return m, tea.Quit
	}

	switch {
	case key.Matches(msg, keys.Up):
		if len(m.timeEntries) > 0 && m.selectedEntryIndex > 0 {
			m.selectedEntryIndex--
		}
		// Clear any status messages on navigation
		if m.statusMessage != "" {
			m.statusMessage = ""
			m.statusMessageTime = time.Time{}
		}
		return m, nil

	case key.Matches(msg, keys.Down):
		if len(m.timeEntries) > 0 && m.selectedEntryIndex < len(m.timeEntries)-1 {
			m.selectedEntryIndex++
		}
		// Clear any status messages on navigation
		if m.statusMessage != "" {
			m.statusMessage = ""
			m.statusMessageTime = time.Time{}
		}
		return m, nil

	case key.Matches(msg, keys.PrevDay):
		m.currentDate = m.currentDate.AddDate(0, 0, -1)
		m.selectedEntryIndex = 0
		m.loading = true
		m.clearStatusMessage()
		return m, fetchTimeEntriesCmd(m.harvestClient, m.currentDate)

	case key.Matches(msg, keys.NextDay):
		m.currentDate = m.currentDate.AddDate(0, 0, 1)
		m.selectedEntryIndex = 0
		m.loading = true
		m.clearStatusMessage()
		return m, fetchTimeEntriesCmd(m.harvestClient, m.currentDate)

	case key.Matches(msg, keys.Today):
		m.currentDate = time.Now()
		m.selectedEntryIndex = 0
		m.loading = true
		m.clearStatusMessage()
		return m, fetchTimeEntriesCmd(m.harvestClient, m.currentDate)

	case key.Matches(msg, keys.Week):
		m.currentView = ViewWeekSummary
		m.weekStart = mondayOf(m.currentDate)
		m.weekLoading = true
		m.clearStatusMessage()
		return m, fetchWeekEntriesCmd(m.harvestClient, m.weekStart)

	case key.Matches(msg, keys.New):
		if len(m.projectsWithTasks) > 0 {
			m.currentView = ViewNewEntry
			m.clearEditState()
			// Initialize the new entry form
			m.newEntryCurrentField = 0
			m.newEntryNotes = ""
			m.newEntryHours = "0:00"
			m.newEntryBillable = true
			m.selectedProject = nil
			m.selectedTask = nil

			// Initialize text inputs for new entry
			notesInput := textinput.New()
			notesInput.Placeholder = "Enter notes (optional)"
			notesInput.Width = 50
			m.notesInput = &notesInput

			durationInput := textinput.New()
			durationInput.SetValue("0:00")
			durationInput.Placeholder = "Enter duration (e.g., 1:30)"
			durationInput.Width = 20
			m.durationInput = &durationInput

			m.updateProjectList()
			m.setListSizes()
			return m, nil
		} else {
			m.setStatusMessage("No projects available. Please check your Harvest configuration.")
			return m, nil
		}

	case key.Matches(msg, keys.Edit):
		if len(m.timeEntries) > 0 && m.selectedEntryIndex < len(m.timeEntries) {
			selectedEntry := m.timeEntries[m.selectedEntryIndex]
			if selectedEntry.IsLocked {
				m.setStatusMessage("Cannot edit locked time entry.")
				return m, nil
			}
			if selectedEntry.IsRunning {
				m.setStatusMessage("Cannot edit running time entry. Stop the timer first.")
				return m, nil
			}
			m.editingEntry = &selectedEntry
			m.editTask = &harvest.Task{ID: selectedEntry.Task.ID, Name: selectedEntry.Task.Name}
			m.editNotes = selectedEntry.Notes
			m.editHours = formatHoursSimple(selectedEntry.Hours)
			m.editBillable = selectedEntry.IsBillable
			m.editCurrentField = 0

			// Initialize text inputs for editing
			notesInput := textinput.New()
			notesInput.SetValue(selectedEntry.Notes)
			notesInput.Placeholder = "Enter notes (optional)"
			notesInput.Width = 50
			m.editNotesInput = &notesInput

			durationInput := textinput.New()
			durationInput.SetValue(formatHoursSimple(selectedEntry.Hours))
			durationInput.Placeholder = "Enter duration (e.g., 1:30)"
			durationInput.Width = 20
			m.editDurationInput = &durationInput

			m.currentView = ViewEditEntry
			return m, nil
		}
		return m, nil

	case key.Matches(msg, keys.Delete):
		if len(m.timeEntries) > 0 && m.selectedEntryIndex < len(m.timeEntries) {
			selectedEntry := m.timeEntries[m.selectedEntryIndex]
			if selectedEntry.IsLocked {
				m.setStatusMessage("Cannot delete locked time entry.")
				return m, nil
			}
			if selectedEntry.IsRunning {
				m.setStatusMessage("Cannot delete running time entry. Stop the timer first.")
				return m, nil
			}
			m.editingEntry = &selectedEntry
			m.currentView = ViewConfirmDelete
			return m, nil
		}
		return m, nil

	case key.Matches(msg, keys.StartStop):
		if len(m.timeEntries) > 0 && m.selectedEntryIndex < len(m.timeEntries) {
			selectedEntry := m.timeEntries[m.selectedEntryIndex]
			if selectedEntry.IsLocked {
				if selectedEntry.IsRunning {
					m.setStatusMessage("Cannot stop locked time entry.")
				} else {
					m.setStatusMessage("Cannot start locked time entry.")
				}
				return m, nil
			}
			// Toggle: if running, stop it; if stopped, start it
			if selectedEntry.IsRunning {
				return m, stopTimeEntryCmd(m.harvestClient, selectedEntry.ID)
			} else {
				return m, restartTimeEntryCmd(m.harvestClient, selectedEntry.ID)
			}
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handleProjectSelectKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "esc":
		// If the list is filtering or has a filter applied, let the list handle esc
		if m.projectList.FilterState() != list.Unfiltered {
			break
		}
		// Check if we're coming from new entry form
		if m.newEntryCurrentField >= 0 && m.newEntryCurrentField <= 3 {
			// Return to new entry form
			m.currentView = ViewNewEntry
			return m, nil
		}
		// Otherwise, cancel project selection and return to main list
		m.currentView = ViewList
		m.selectedProject = nil
		m.selectedTask = nil
		return m, nil
	case "enter":
		// Get the selected project
		selected := m.projectList.SelectedItem()
		if selected != nil {
			// Skip divider items
			if _, ok := selected.(dividerItem); ok {
				// Move to next item
				m.projectList.CursorDown()
				return m, nil
			}

			if item, ok := selected.(projectItem); ok {
				m.selectedProject = &item.project

				// Find tasks for this project
				for _, pwt := range m.projectsWithTasks {
					if pwt.Project.ID == item.project.ID {
						if len(pwt.Tasks) == 0 {
							// No tasks available for this project
							m.setStatusMessage("No tasks available for this project")
							m.selectedProject = nil
							return m, nil
						}

						if len(pwt.Tasks) == 1 {
							// Only one task, skip task selection
							m.selectedTask = &pwt.Tasks[0]
							// Initialize notes input
							notesInput := textinput.New()
							notesInput.Focus()
							notesInput.Placeholder = "Enter notes (optional)"
							notesInput.Width = 50
							m.notesInput = &notesInput
							m.currentView = ViewNotesInput
						} else {
							// Multiple tasks, show task selection
							m.currentView = ViewSelectTask
							m.updateTaskList(pwt.Tasks)
						}
						break
					}
				}
			}
		}
		return m, nil
	}

	// Handle list navigation
	m.projectList, cmd = m.projectList.Update(msg)
	return m, cmd
}

func (m Model) handleTaskSelectKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "esc":
		// If the list is filtering or has a filter applied, let the list handle esc
		if m.taskList.FilterState() != list.Unfiltered {
			break
		}
		if m.editingEntry != nil {
			// Return to edit view when editing
			m.currentView = ViewEditEntry
			m.selectedProject = nil
			m.updateEditFieldFocus()
			return m, nil
		}
		// Go back to project selection
		m.currentView = ViewSelectProject
		m.selectedProject = nil
		m.selectedTask = nil
		return m, nil
	case "enter":
		// Get the selected task
		selected := m.taskList.SelectedItem()
		if selected != nil {
			if item, ok := selected.(taskItem); ok {
				if m.editingEntry != nil {
					// Set editTask and return to edit view
					m.editTask = &harvest.Task{ID: item.task.ID, Name: item.task.Name}
					m.selectedProject = nil
					m.currentView = ViewEditEntry
					m.updateEditFieldFocus()
					return m, nil
				}
				m.selectedTask = &item.task
				// Initialize notes input
				notesInput := textinput.New()
				notesInput.Focus()
				notesInput.Placeholder = "Enter notes (optional)"
				notesInput.Width = 50
				m.notesInput = &notesInput
				m.currentView = ViewNotesInput
			}
		}
		return m, nil
	}

	// Handle list navigation
	m.taskList, cmd = m.taskList.Update(msg)
	return m, cmd
}

func (m Model) handleEditViewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "esc":
		// Return to main list, clearing all edit and entry state
		m.currentView = ViewList
		m.clearEditState()
		return m, nil

	case "tab":
		// Move to next field
		m.editCurrentField = (m.editCurrentField + 1) % 3
		m.updateEditFieldFocus()
		return m, nil

	case "shift+tab":
		// Move to previous field
		m.editCurrentField = (m.editCurrentField - 1 + 3) % 3
		m.updateEditFieldFocus()
		return m, nil

	case "enter":
		if m.editCurrentField == 0 {
			// Open task selection for the current project
			if m.editingEntry != nil {
				if len(m.projectsWithTasks) == 0 {
					m.pendingTaskEdit = true
					m.setStatusMessage("Loading tasks...")
					return m, fetchProjectsWithTasksCmd(m.harvestClient)
				}
				if !m.openTaskSelectionForEdit() {
					m.setStatusMessage("No tasks found for this project")
				}
			}
		}
		return m, nil

	case "ctrl+s":
		// Save changes
		return m, m.updateTimeEntry()

	default:
		// Pass to the appropriate input field if it's focused
		if m.editCurrentField == 1 && m.editNotesInput != nil {
			*m.editNotesInput, cmd = m.editNotesInput.Update(msg)
			m.editNotes = m.editNotesInput.Value()
		} else if m.editCurrentField == 2 && m.editDurationInput != nil {
			*m.editDurationInput, cmd = m.editDurationInput.Update(msg)
			m.editHours = m.editDurationInput.Value()
		}
	}

	return m, cmd
}

// updateEditFieldFocus updates text input focus based on the current edit field.
func (m *Model) updateEditFieldFocus() {
	if m.editNotesInput != nil {
		if m.editCurrentField == 1 {
			m.editNotesInput.Focus()
		} else {
			m.editNotesInput.Blur()
		}
	}
	if m.editDurationInput != nil {
		if m.editCurrentField == 2 {
			m.editDurationInput.Focus()
		} else {
			m.editDurationInput.Blur()
		}
	}
}

// openTaskSelectionForEdit finds the editing entry's project tasks and switches to task selection.
// Returns true if the transition succeeded, false if the project was not found.
func (m *Model) openTaskSelectionForEdit() bool {
	for _, pwt := range m.projectsWithTasks {
		if pwt.Project.ID == m.editingEntry.Project.ID {
			m.selectedProject = &pwt.Project
			m.updateTaskList(pwt.Tasks)
			if m.editNotesInput != nil {
				m.editNotesInput.Blur()
			}
			if m.editDurationInput != nil {
				m.editDurationInput.Blur()
			}
			m.currentView = ViewSelectTask
			return true
		}
	}
	return false
}

func (m Model) handleConfirmDeleteKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "n":
		// Cancel deletion and return to main list
		m.currentView = ViewList
		m.editingEntry = nil
		return m, nil
	case "y":
		// Confirm deletion
		if m.editingEntry != nil {
			return m, deleteTimeEntryCmd(m.harvestClient, m.editingEntry.ID)
		}
		return m, nil
	}
	return m, nil
}

func (m Model) handleHelpViewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "?":
		// Return to main list
		m.currentView = ViewList
		return m, nil
	}
	return m, nil
}

func (m Model) handleNotesInputKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "esc":
		// Cancel and return to main list
		m.currentView = ViewList
		m.selectedProject = nil
		m.selectedTask = nil
		m.notesInput = nil
		return m, nil
	case "enter":
		// Store notes and move to duration input
		if m.notesInput != nil {
			m.newEntryNotes = m.notesInput.Value()
		}
		// Initialize duration input
		durationInput := textinput.New()
		durationInput.Focus()
		durationInput.Placeholder = "Enter duration (e.g., 1:30)"
		durationInput.Width = 20
		m.durationInput = &durationInput
		m.currentView = ViewDurationInput
		return m, nil
	}

	// Pass other messages to the text input
	if m.notesInput != nil {
		*m.notesInput, cmd = m.notesInput.Update(msg)
	}
	return m, cmd
}

func (m Model) handleDurationInputKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "esc":
		// Go back to notes input
		m.currentView = ViewNotesInput
		m.durationInput = nil
		return m, nil
	case "enter":
		// Validate and store duration
		if m.durationInput != nil {
			duration := m.durationInput.Value()
			if duration == "" {
				duration = "0:00"
			}
			// Validate duration format
			if _, err := parseDuration(duration); err != nil {
				m.setStatusMessage("Invalid duration format. Use HH:MM (e.g., 1:30)")
				return m, nil
			}
			m.newEntryHours = duration
			m.newEntryBillable = true // Default to billable
			m.currentView = ViewBillableToggle
		}
		return m, nil
	}

	// Pass other messages to the text input
	if m.durationInput != nil {
		*m.durationInput, cmd = m.durationInput.Update(msg)
	}
	return m, cmd
}

func (m Model) handleBillableToggleKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Go back to duration input
		m.currentView = ViewDurationInput
		return m, nil
	case "tab", " ", "b":
		// Toggle billable status
		m.newEntryBillable = !m.newEntryBillable
		return m, nil
	case "enter":
		// Create the time entry
		return m, m.createTimeEntry()
	}
	return m, nil
}

// handleNewEntryKeys handles key presses in the new entry form
func (m Model) handleNewEntryKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "esc":
		// Cancel and return to main list
		m.currentView = ViewList
		m.clearEditState()
		return m, nil

	case "tab":
		// Move to next field
		m.newEntryCurrentField = (m.newEntryCurrentField + 1) % 4
		// Update focus for text inputs
		if m.newEntryCurrentField == 2 && m.notesInput != nil {
			m.notesInput.Focus()
			if m.durationInput != nil {
				m.durationInput.Blur()
			}
		} else if m.newEntryCurrentField == 3 && m.durationInput != nil {
			m.durationInput.Focus()
			if m.notesInput != nil {
				m.notesInput.Blur()
			}
		} else {
			// Not on a text field, blur both
			if m.notesInput != nil {
				m.notesInput.Blur()
			}
			if m.durationInput != nil {
				m.durationInput.Blur()
			}
		}
		return m, nil

	case "shift+tab":
		// Move to previous field
		m.newEntryCurrentField = (m.newEntryCurrentField - 1 + 4) % 4
		// Update focus for text inputs
		if m.newEntryCurrentField == 2 && m.notesInput != nil {
			m.notesInput.Focus()
			if m.durationInput != nil {
				m.durationInput.Blur()
			}
		} else if m.newEntryCurrentField == 3 && m.durationInput != nil {
			m.durationInput.Focus()
			if m.notesInput != nil {
				m.notesInput.Blur()
			}
		} else {
			// Not on a text field, blur both
			if m.notesInput != nil {
				m.notesInput.Blur()
			}
			if m.durationInput != nil {
				m.durationInput.Blur()
			}
		}
		return m, nil

	case "enter":
		// Handle enter based on current field
		switch m.newEntryCurrentField {
		case 0: // Project field
			// Open project selection
			m.currentView = ViewSelectProject
			m.updateProjectList()
			return m, nil
		case 1: // Task field
			if m.selectedProject != nil {
				// Find tasks for selected project
				for _, pwt := range m.projectsWithTasks {
					if pwt.Project.ID == m.selectedProject.ID {
						if len(pwt.Tasks) > 0 {
							m.currentView = ViewSelectTask
							m.updateTaskList(pwt.Tasks)
						}
						break
					}
				}
			}
			return m, nil
		}
		return m, nil

	case "ctrl+s":
		// Save entry
		// Validate required fields
		if m.selectedProject == nil || m.selectedTask == nil {
			m.setStatusMessage("Please select a project and task")
			return m, nil
		}

		// Store current values from inputs
		if m.notesInput != nil {
			m.newEntryNotes = m.notesInput.Value()
		}
		if m.durationInput != nil {
			m.newEntryHours = m.durationInput.Value()
		}

		// Validate duration
		if _, err := parseDuration(m.newEntryHours); err != nil {
			m.setStatusMessage("Invalid duration format. Use HH:MM (e.g., 1:30)")
			return m, nil
		}

		return m, m.createTimeEntry()

	default:
		// Pass to text inputs if focused
		if m.newEntryCurrentField == 2 && m.notesInput != nil {
			*m.notesInput, cmd = m.notesInput.Update(msg)
			m.newEntryNotes = m.notesInput.Value()
		} else if m.newEntryCurrentField == 3 && m.durationInput != nil {
			*m.durationInput, cmd = m.durationInput.Update(msg)
			m.newEntryHours = m.durationInput.Value()
		}
	}

	return m, cmd
}
