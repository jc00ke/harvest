// List items and list plumbing for the project and task pickers.
package tui

import (
	"sort"

	"github.com/charmbracelet/bubbles/list"
	"github.com/jc00ke/harvest/internal/harvest"
)

// projectItem represents a project in the selection list.
type projectItem struct {
	project harvest.Project
	client  harvest.ProjectClient
}

func (i projectItem) FilterValue() string {
	return i.project.Name + " " + i.client.Name
}

func (i projectItem) Title() string {
	clientName := truncateString(i.client.Name, 25)
	projectName := truncateString(i.project.Name, 35)
	return clientName + " → " + projectName
}

func (i projectItem) Description() string {
	return ""
}

// dividerItem represents a divider in the selection list.
type dividerItem struct{}

func (i dividerItem) FilterValue() string {
	return ""
}

func (i dividerItem) Title() string {
	return "─────────────────────────────────────"
}

func (i dividerItem) Description() string {
	return ""
}

// taskItem represents a task in the selection list.

type taskItem struct {
	task harvest.Task
}

func (i taskItem) FilterValue() string {
	return i.task.Name
}

func (i taskItem) Title() string {
	return truncateString(i.task.Name, 50)
}

func (i taskItem) Description() string {
	return ""
}

// newProjectDelegate creates a new delegate for project list items.
func newProjectDelegate() list.DefaultDelegate {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	return delegate
}

// newTaskDelegate creates a new delegate for task list items.
func newTaskDelegate() list.DefaultDelegate {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	return delegate
}

// setListSizes updates the project and task list dimensions based on the shell width and window height.
func (m *Model) setListSizes() {
	contentW := m.shellWidth() - 4
	contentH := m.height - 7
	if contentH < 5 {
		contentH = 5
	}
	m.projectList.SetSize(contentW, contentH)
	m.taskList.SetSize(contentW, contentH)
}

// newShellList creates a list.Model with title and status bar disabled.
func newShellList(delegate list.DefaultDelegate) list.Model {
	l := list.New([]list.Item{}, delegate, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	return l
}

// updateProjectList updates the project list with current projects and recents.
func (m *Model) updateProjectList() {
	var items []list.Item
	recentsAdded := 0

	// Add recents section first
	if len(m.appState.Recents) > 0 {
		for _, recent := range m.appState.Recents {
			// Find the matching project and client
			for _, pwt := range m.projectsWithTasks {
				if pwt.Project.ID == recent.ProjectID && pwt.Project.Client.ID == recent.ClientID {
					items = append(items, projectItem{
						project: pwt.Project,
						client:  pwt.Project.Client,
					})
					recentsAdded++
					break
				}
			}
		}

		// Add divider after recents only if we actually added any
		if recentsAdded > 0 {
			items = append(items, dividerItem{})
		}
	}

	// Add all projects sorted by client then project name
	allProjects := make([]harvest.ProjectWithTasks, len(m.projectsWithTasks))
	copy(allProjects, m.projectsWithTasks)

	sort.Slice(allProjects, func(i, j int) bool {
		if allProjects[i].Project.Client.Name != allProjects[j].Project.Client.Name {
			return allProjects[i].Project.Client.Name < allProjects[j].Project.Client.Name
		}
		return allProjects[i].Project.Name < allProjects[j].Project.Name
	})

	// Add all projects to items (including those in recents)
	for _, pwt := range allProjects {
		items = append(items, projectItem{
			project: pwt.Project,
			client:  pwt.Project.Client,
		})
	}

	m.projectList.SetItems(items)
}

// updateTaskList updates the task list with tasks from the selected project.
func (m *Model) updateTaskList(tasks []harvest.Task) {
	var items []list.Item
	for _, task := range tasks {
		items = append(items, taskItem{task: task})
	}
	m.taskList.SetItems(items)
}
