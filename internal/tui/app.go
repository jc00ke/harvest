package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jc00ke/harvest/internal/config"
	"github.com/jc00ke/harvest/internal/harvest"
	"github.com/jc00ke/harvest/internal/state"
)

// ViewState represents the different views in the TUI application.
type ViewState int

const (
	// ViewLoading is the initial loading screen shown during startup.
	ViewLoading ViewState = iota
	// ViewList is the main list view showing time entries for the current date.
	ViewList
	// ViewSelectProject is the project selection view when creating a new time entry.
	ViewSelectProject
	// ViewSelectTask is the task selection view when creating a new time entry.
	ViewSelectTask
	// ViewNewEntry is the unified new entry form view.
	ViewNewEntry
	// ViewEditEntry is the view for editing an existing time entry.
	ViewEditEntry
	// ViewConfirmDelete is the confirmation view when deleting a time entry.
	ViewConfirmDelete
	// ViewHelp is the help overlay showing all keybindings.
	ViewHelp
	// ViewNotesInput is the view for entering notes for a new time entry.
	ViewNotesInput
	// ViewDurationInput is the view for entering duration for a new time entry.
	ViewDurationInput
	// ViewBillableToggle is the view for toggling billable status for a new time entry.
	ViewBillableToggle
	// ViewWeekSummary is the weekly summary view showing per-day and per-project totals.
	ViewWeekSummary
)

// Model represents the state of the TUI application.
type Model struct {
	// Current view state
	currentView ViewState

	// Configuration and external dependencies
	config        *config.Config
	harvestClient *harvest.Client
	appState      *state.State

	// Data
	currentDate        time.Time
	timeEntries        []harvest.TimeEntry
	projectsWithTasks  []harvest.ProjectWithTasks
	selectedEntryIndex int
	currentUser        *harvest.User

	// Weekly summary state
	weekStart   time.Time
	weekEntries []harvest.TimeEntry
	weekLoading bool

	// New entry creation state
	selectedProject      *harvest.Project
	selectedTask         *harvest.Task
	newEntryNotes        string
	newEntryHours        string
	newEntryBillable     bool
	newEntryCurrentField int // 0=project, 1=task, 2=notes, 3=duration

	// Edit entry state
	editingEntry     *harvest.TimeEntry
	editTask         *harvest.Task
	editNotes        string
	editHours        string
	editBillable     bool
	editCurrentField int // 0=task, 1=notes, 2=duration
	pendingTaskEdit  bool

	// UI state
	loading           bool
	errorMessage      string
	statusMessage     string
	statusMessageTime time.Time // Track when the status message was set
	lastFetchTime     time.Time // Track last API fetch to avoid rate limiting
	spinner           spinner.Model
	timeEntriesLoaded bool
	projectsLoaded    bool

	// List components for selection views
	projectList list.Model
	taskList    list.Model

	// Text input components
	notesInput        *textinput.Model
	durationInput     *textinput.Model
	editNotesInput    *textinput.Model
	editDurationInput *textinput.Model

	// Window dimensions
	width  int
	height int
}

// NewModel creates a new TUI model with the given configuration.
func NewModel(cfg *config.Config, client *harvest.Client, appState *state.State, user *harvest.User) Model {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(accentColor)

	return Model{
		currentView:        ViewLoading,
		config:             cfg,
		harvestClient:      client,
		appState:           appState,
		currentDate:        time.Now(),
		timeEntries:        []harvest.TimeEntry{},
		projectsWithTasks:  []harvest.ProjectWithTasks{},
		selectedEntryIndex: 0,
		currentUser:        user,
		newEntryNotes:      "",
		newEntryHours:      "",
		newEntryBillable:   true,
		editNotes:          "",
		editHours:          "",
		editBillable:       true,
		loading:            false,
		errorMessage:       "",
		statusMessage:      "",
		statusMessageTime:  time.Time{},
		spinner:            s,
		projectList:        newShellList(newProjectDelegate()),
		taskList:           newShellList(newTaskDelegate()),
		width:              80,
		height:             24,
	}
}

// Init initializes the model and returns initial commands.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		fetchTimeEntriesCmd(m.harvestClient, m.currentDate),
		fetchProjectsWithTasksCmd(m.harvestClient),
		tickCmd(), // Start the ticker for real-time updates
	)
}

// Update handles messages and updates the model state.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.setListSizes()
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case spinner.TickMsg:
		if m.currentView == ViewLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case timeEntriesFetchedMsg:
		if msg.err != nil {
			m.errorMessage = "Failed to fetch time entries: " + msg.err.Error()
		} else {
			m.timeEntries = msg.entries
			m.errorMessage = ""
			m.lastFetchTime = time.Now()
		}
		m.loading = false
		m.timeEntriesLoaded = true

		// Transition from loading screen when both fetches complete
		if m.currentView == ViewLoading && m.projectsLoaded {
			m.currentView = ViewList
		}

		// If there's a running timer, continue ticking
		if m.hasRunningTimer() {
			return m, tickCmd()
		}
		return m, nil

	case tickMsg:
		// Clear status message after 3 seconds
		if m.statusMessage != "" && !m.statusMessageTime.IsZero() {
			if time.Since(m.statusMessageTime) > 3*time.Second {
				m.statusMessage = ""
				m.statusMessageTime = time.Time{}
			}
		}

		// Check if we have a running timer and it's time to refresh from API
		if m.hasRunningTimer() && m.currentView == ViewList && !m.loading {
			if time.Since(m.lastFetchTime) >= 25*time.Second {
				m.lastFetchTime = time.Now()
				return m, tea.Batch(
					fetchTimeEntriesCmd(m.harvestClient, m.currentDate),
					tickCmd(),
				)
			}
			return m, tickCmd()
		}
		// Continue ticking if we have a running timer or status message
		if m.hasRunningTimer() || m.statusMessage != "" {
			return m, tickCmd()
		}
		return m, nil

	case weekEntriesFetchedMsg:
		if msg.err != nil {
			m.errorMessage = "Failed to fetch week entries: " + msg.err.Error()
		} else {
			m.weekEntries = msg.entries
			m.errorMessage = ""
		}
		m.weekLoading = false
		return m, nil

	case projectsWithTasksFetchedMsg:
		if msg.err != nil {
			m.errorMessage = "Failed to fetch projects: " + msg.err.Error()
			m.pendingTaskEdit = false
		} else {
			m.projectsWithTasks = msg.projectsWithTasks
			m.errorMessage = ""

			// If user requested task edit while projects were loading, open it now
			if m.pendingTaskEdit && m.editingEntry != nil && m.currentView == ViewEditEntry {
				m.pendingTaskEdit = false
				if !m.openTaskSelectionForEdit() {
					m.setStatusMessage("No tasks found for this project")
				}
			}
		}
		m.projectsLoaded = true

		// Transition from loading screen when both fetches complete
		if m.currentView == ViewLoading && m.timeEntriesLoaded {
			m.currentView = ViewList
		}

		return m, nil

	case timeEntryStartedMsg:
		if msg.err != nil {
			m.setStatusMessage("Failed to start timer: " + msg.err.Error())
		} else {
			// Update the entry in our local list
			for i, entry := range m.timeEntries {
				if entry.ID == msg.entry.ID {
					m.timeEntries[i] = *msg.entry
					break
				}
			}
			m.setStatusMessage("Timer started successfully")
			// Re-fetch entries so previously running timer shows as stopped
			m.lastFetchTime = time.Now()
			return m, tea.Batch(
				fetchTimeEntriesCmd(m.harvestClient, m.currentDate),
				tickCmd(),
			)
		}
		return m, nil

	case timeEntryStoppedMsg:
		if msg.err != nil {
			m.setStatusMessage("Failed to stop timer: " + msg.err.Error())
		} else {
			// Update the entry in our local list
			for i, entry := range m.timeEntries {
				if entry.ID == msg.entry.ID {
					m.timeEntries[i] = *msg.entry
					break
				}
			}
			m.setStatusMessage("Timer stopped successfully")
		}
		return m, nil

	case timeEntryCreatedMsg:
		if msg.err != nil {
			m.setStatusMessage("Failed to create entry: " + msg.err.Error())
		} else {
			// Add the new entry to our local list
			m.timeEntries = append([]harvest.TimeEntry{*msg.entry}, m.timeEntries...)
			m.setStatusMessage("Time entry created successfully")
			// Clear new entry state and return to main list
			m.clearEditState()
			m.currentView = ViewList
		}
		return m, nil

	case timeEntryUpdatedMsg:
		if msg.err != nil {
			m.setStatusMessage("Failed to update entry: " + msg.err.Error())
		} else {
			// Update the entry in our local list
			for i, entry := range m.timeEntries {
				if entry.ID == msg.entry.ID {
					m.timeEntries[i] = *msg.entry
					break
				}
			}
			m.setStatusMessage("Time entry updated successfully")
			// Clear edit state and return to main list
			m.clearEditState()
			m.currentView = ViewList
		}
		return m, nil

	case timeEntryDeletedMsg:
		if msg.err != nil {
			m.setStatusMessage("Failed to delete entry: " + msg.err.Error())
		} else {
			// Remove the entry from our local list
			newEntries := []harvest.TimeEntry{}
			for _, entry := range m.timeEntries {
				if entry.ID != msg.entryID {
					newEntries = append(newEntries, entry)
				}
			}
			m.timeEntries = newEntries

			// Adjust selected index if necessary
			if m.selectedEntryIndex >= len(m.timeEntries) && m.selectedEntryIndex > 0 {
				m.selectedEntryIndex--
			}

			m.setStatusMessage("Time entry deleted successfully")
			// Clear edit state and return to main list
			m.clearEditState()
			m.currentView = ViewList
		}
		return m, nil

	default:
		return m, nil
	}
}

// View renders the current view.
func (m Model) View() string {
	switch m.currentView {
	case ViewLoading:
		return m.renderLoadingView()
	case ViewList:
		return m.renderStyledListView()
	case ViewSelectProject:
		return m.renderProjectSelectView()
	case ViewSelectTask:
		return m.renderTaskSelectView()
	case ViewNewEntry:
		return m.renderNewEntryModal()
	case ViewEditEntry:
		return m.renderEditView()
	case ViewConfirmDelete:
		return m.renderConfirmDeleteView()
	case ViewHelp:
		return m.renderHelpView()
	case ViewNotesInput:
		return m.renderNotesInputView()
	case ViewDurationInput:
		return m.renderDurationInputView()
	case ViewBillableToggle:
		return m.renderBillableToggleView()
	case ViewWeekSummary:
		return m.renderWeekSummaryView()
	default:
		return "Unknown view"
	}
}

// handleKeyPress processes key presses for the current view.
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	previousView := m.currentView

	// Ctrl+C always quits, even during loading
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}

	// Block all other input during loading
	if m.currentView == ViewLoading {
		return m, nil
	}

	// Global keybindings that work in all views
	switch msg.String() {
	case "?":
		if m.currentView == ViewHelp {
			m.currentView = ViewList
		} else {
			m.currentView = ViewHelp
		}
		m.clearStatusMessage()
		return m, nil
	}

	// View-specific keybindings
	var result tea.Model
	var cmd tea.Cmd

	switch m.currentView {
	case ViewList:
		result, cmd = m.handleListViewKeys(msg)
	case ViewSelectProject:
		result, cmd = m.handleProjectSelectKeys(msg)
	case ViewSelectTask:
		result, cmd = m.handleTaskSelectKeys(msg)
	case ViewNewEntry:
		result, cmd = m.handleNewEntryKeys(msg)
	case ViewEditEntry:
		result, cmd = m.handleEditViewKeys(msg)
	case ViewConfirmDelete:
		result, cmd = m.handleConfirmDeleteKeys(msg)
	case ViewHelp:
		result, cmd = m.handleHelpViewKeys(msg)
	case ViewNotesInput:
		result, cmd = m.handleNotesInputKeys(msg)
	case ViewDurationInput:
		result, cmd = m.handleDurationInputKeys(msg)
	case ViewBillableToggle:
		result, cmd = m.handleBillableToggleKeys(msg)
	case ViewWeekSummary:
		result, cmd = m.handleWeekSummaryKeys(msg)
	default:
		return m, nil
	}

	// Clear status message when transitioning between views
	if resultModel, ok := result.(Model); ok && resultModel.currentView != previousView {
		resultModel.clearStatusMessage()
		return resultModel, cmd
	}

	return result, cmd
}

// clearEditState resets the editing and new entry state.
func (m *Model) clearEditState() {
	m.selectedProject = nil
	m.selectedTask = nil
	m.newEntryNotes = ""
	m.newEntryHours = ""
	m.newEntryBillable = true
	m.notesInput = nil
	m.durationInput = nil
	m.editingEntry = nil
	m.editTask = nil
	m.editNotes = ""
	m.editHours = ""
	m.editBillable = true
	m.editNotesInput = nil
	m.editDurationInput = nil
	m.editCurrentField = 0
	m.pendingTaskEdit = false
}

// formatHoursSimple formats hours as HH:MM format (hours zero-padded to two
// digits) so durations have a constant width and align cleanly in the UI.
func formatHoursSimple(hours float64) string {
	h := int(hours)
	m := int((hours - float64(h)) * 60)
	return fmt.Sprintf("%02d:%02d", h, m)
}

// truncateString truncates a string to the given max length, adding "..." if truncated.
func truncateString(s string, maxLen int) string {
	if maxLen <= 3 || len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// parseDuration parses a duration string in HH:MM format and returns hours as a float64.
func parseDuration(durationStr string) (float64, error) {
	durationStr = strings.TrimSpace(durationStr)
	if durationStr == "" {
		return 0, fmt.Errorf("duration cannot be empty")
	}

	parts := strings.Split(durationStr, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid duration format. Use HH:MM (e.g., 1:30)")
	}

	hours, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid duration format. Use HH:MM (e.g., 1:30)")
	}

	minutes, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("invalid duration format. Use HH:MM (e.g., 1:30)")
	}

	if hours < 0 || minutes < 0 || minutes >= 60 {
		return 0, fmt.Errorf("invalid duration format. Use HH:MM (e.g., 1:30)")
	}

	return float64(hours) + float64(minutes)/60.0, nil
}
