// Package demo provides an in-process fake of the Harvest API v2 endpoints
// used by this application, seeded with fixture data. It backs the TUI's
// demo mode (`harvest -ui --demo`) so the app can run without credentials.
package demo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"time"

	"github.com/jc00ke/harvest/internal/harvest"
)

const dateFormat = "2006-01-02"

// server holds the mutable in-memory state behind the fake API.
type server struct {
	mu      sync.Mutex
	entries []harvest.TimeEntry
	nextID  int
}

// NewServer starts an HTTP server emulating the Harvest API v2, seeded with
// time entries on and around the given date. Callers must Close it.
func NewServer(today time.Time) *httptest.Server {
	s := &server{
		entries: seedEntries(today),
		nextID:  2000,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v2/users/me", s.handleMe)
	mux.HandleFunc("GET /v2/projects", s.handleProjects)
	mux.HandleFunc("GET /v2/task_assignments", s.handleTaskAssignments)
	mux.HandleFunc("GET /v2/time_entries", s.handleListEntries)
	mux.HandleFunc("POST /v2/time_entries", s.handleCreateEntry)
	mux.HandleFunc("PATCH /v2/time_entries/{id}", s.handleUpdateEntry)
	mux.HandleFunc("DELETE /v2/time_entries/{id}", s.handleDeleteEntry)
	mux.HandleFunc("PATCH /v2/time_entries/{id}/restart", s.handleSetRunning(true))
	mux.HandleFunc("PATCH /v2/time_entries/{id}/stop", s.handleSetRunning(false))

	return httptest.NewServer(mux)
}

var demoUser = harvest.User{
	ID:        1,
	FirstName: "Demo",
	LastName:  "User",
	Email:     "demo@example.com",
}

// Teammates whose entries only show up in team-wide views (e.g. `invoice`);
// the authenticated demo user is entryUserSelf.
var (
	entryUserSelf = harvest.TimeEntryUser{ID: 1, Name: "Demo User"}
	entryUserAlex = harvest.TimeEntryUser{ID: 2, Name: "Alex Rivera"}
	entryUserSam  = harvest.TimeEntryUser{ID: 3, Name: "Sam Chen"}
)

var demoProjects = []harvest.Project{
	{ID: 101, Name: "Website Redesign", Client: harvest.ProjectClient{ID: 11, Name: "Acme Corp"}},
	{ID: 102, Name: "Mobile App", Client: harvest.ProjectClient{ID: 12, Name: "Globex"}},
	{ID: 103, Name: "Maintenance Retainer", Client: harvest.ProjectClient{ID: 13, Name: "Initech"}},
}

var demoTaskAssignments = []harvest.TaskAssignment{
	{ID: 301, Project: harvest.TaskAssignmentProject{ID: 101, Name: "Website Redesign"}, Task: harvest.TaskAssignmentTask{ID: 201, Name: "Design"}, IsActive: true, Billable: true, HourlyRate: 150},
	{ID: 302, Project: harvest.TaskAssignmentProject{ID: 101, Name: "Website Redesign"}, Task: harvest.TaskAssignmentTask{ID: 202, Name: "Development"}, IsActive: true, Billable: true, HourlyRate: 160},
	{ID: 303, Project: harvest.TaskAssignmentProject{ID: 101, Name: "Website Redesign"}, Task: harvest.TaskAssignmentTask{ID: 203, Name: "Meetings"}, IsActive: true, Billable: false},
	{ID: 304, Project: harvest.TaskAssignmentProject{ID: 102, Name: "Mobile App"}, Task: harvest.TaskAssignmentTask{ID: 202, Name: "Development"}, IsActive: true, Billable: true, HourlyRate: 160},
	{ID: 305, Project: harvest.TaskAssignmentProject{ID: 102, Name: "Mobile App"}, Task: harvest.TaskAssignmentTask{ID: 204, Name: "QA"}, IsActive: true, Billable: true, HourlyRate: 120},
	{ID: 306, Project: harvest.TaskAssignmentProject{ID: 103, Name: "Maintenance Retainer"}, Task: harvest.TaskAssignmentTask{ID: 202, Name: "Development"}, IsActive: true, Billable: true, HourlyRate: 160},
	{ID: 307, Project: harvest.TaskAssignmentProject{ID: 103, Name: "Maintenance Retainer"}, Task: harvest.TaskAssignmentTask{ID: 203, Name: "Meetings"}, IsActive: true, Billable: false},
}

// seedEntries builds fixture time entries for today and the two days before,
// so date navigation in the TUI has data to show. Teammate entries make
// team-wide views (e.g. `invoice`) show more than the authenticated user.
func seedEntries(today time.Time) []harvest.TimeEntry {
	day := func(offset int) string { return today.AddDate(0, 0, offset).Format(dateFormat) }
	entry := func(id int, user harvest.TimeEntryUser, date string, hours float64, notes string, running bool, projectID, taskID int) harvest.TimeEntry {
		e := harvest.TimeEntry{
			ID:        id,
			SpentDate: date,
			Hours:     hours,
			Notes:     notes,
			IsRunning: running,
			User:      user,
		}
		fillNames(&e, projectID, taskID)
		return e
	}
	return []harvest.TimeEntry{
		entry(1001, entryUserSelf, day(0), 0.5, "Sprint planning", false, 101, 203),
		entry(1002, entryUserSelf, day(0), 2.25, "Homepage hero and nav implementation", false, 101, 202),
		entry(1003, entryUserSelf, day(0), 1.0, "Push notification spike", true, 102, 202),
		entry(1004, entryUserSelf, day(-1), 1.5, "Design review feedback", false, 101, 201),
		entry(1005, entryUserSelf, day(-1), 3.0, "Onboarding flow screens", false, 102, 202),
		entry(1006, entryUserSelf, day(-1), 0.75, "Dependency upgrades", false, 103, 202),
		entry(1007, entryUserSelf, day(-2), 2.0, "Release regression pass", false, 102, 204),
		entry(1008, entryUserSelf, day(-2), 1.25, "Quarterly roadmap sync", false, 103, 203),
		entry(1101, entryUserAlex, day(0), 3.0, "Checkout flow API integration", false, 101, 202),
		entry(1102, entryUserAlex, day(-1), 2.5, "Component library audit", false, 101, 201),
		entry(1103, entryUserAlex, day(-1), 1.0, "Sprint planning", false, 101, 203),
		entry(1104, entryUserAlex, day(-2), 4.0, "Offline sync engine", false, 102, 202),
		entry(1105, entryUserSam, day(0), 2.0, "Style guide illustrations", false, 101, 201),
		entry(1106, entryUserSam, day(-2), 1.5, "Cron job hardening", false, 103, 202),
	}
}

// fillNames resolves project, client, and task names from the fixtures onto
// the entry. Returns false if the project or task is unknown.
func fillNames(e *harvest.TimeEntry, projectID, taskID int) bool {
	var project *harvest.Project
	for i := range demoProjects {
		if demoProjects[i].ID == projectID {
			project = &demoProjects[i]
			break
		}
	}
	if project == nil {
		return false
	}
	var task *harvest.TaskAssignment
	for i, ta := range demoTaskAssignments {
		if ta.Project.ID == projectID && ta.Task.ID == taskID {
			task = &demoTaskAssignments[i]
			break
		}
	}
	if task == nil {
		return false
	}
	e.Project = harvest.TimeEntryProject{ID: project.ID, Name: project.Name}
	e.Client = harvest.TimeEntryClient{ID: project.Client.ID, Name: project.Client.Name}
	e.Task = harvest.TimeEntryTask{ID: taskID, Name: task.Task.Name}
	e.IsBillable = task.Billable
	if task.Billable {
		e.BillableRate = task.HourlyRate
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (s *server) handleMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, demoUser)
}

func (s *server) handleProjects(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"projects":  demoProjects,
		"next_page": nil,
	})
}

func (s *server) handleTaskAssignments(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"task_assignments": demoTaskAssignments,
		"next_page":        nil,
	})
}

func (s *server) handleListEntries(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from := q.Get("from")
	to := q.Get("to")
	// Optional integer filters; zero means "not filtered".
	userID, _ := strconv.Atoi(q.Get("user_id"))
	projectID, _ := strconv.Atoi(q.Get("project_id"))
	clientID, _ := strconv.Atoi(q.Get("client_id"))

	s.mu.Lock()
	defer s.mu.Unlock()
	matched := []harvest.TimeEntry{}
	for _, e := range s.entries {
		switch {
		case from != "" && e.SpentDate < from,
			to != "" && e.SpentDate > to,
			userID != 0 && e.User.ID != userID,
			projectID != 0 && e.Project.ID != projectID,
			clientID != 0 && e.Client.ID != clientID:
			continue
		}
		matched = append(matched, e)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"time_entries": matched,
		"next_page":    nil,
	})
}

func (s *server) handleCreateEntry(w http.ResponseWriter, r *http.Request) {
	var req harvest.CreateTimeEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": err.Error()})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	entry := harvest.TimeEntry{
		ID:        s.nextID,
		SpentDate: req.SpentDate,
		Hours:     req.Hours,
		Notes:     req.Notes,
		User:      entryUserSelf,
	}
	if !fillNames(&entry, req.ProjectID, req.TaskID) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "unknown project or task"})
		return
	}
	// An explicit billable flag overrides the task assignment's default.
	if req.IsBillable != nil {
		entry.IsBillable = *req.IsBillable
	}
	s.nextID++
	s.entries = append(s.entries, entry)
	writeJSON(w, http.StatusCreated, entry)
}

// findEntry returns the index of the entry with the given path ID, or an
// error suitable for the response. Callers must hold s.mu.
func (s *server) findEntry(r *http.Request) (int, error) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		return 0, fmt.Errorf("invalid id %q", r.PathValue("id"))
	}
	for i := range s.entries {
		if s.entries[i].ID == id {
			return i, nil
		}
	}
	return 0, fmt.Errorf("time entry %d not found", id)
}

func (s *server) handleUpdateEntry(w http.ResponseWriter, r *http.Request) {
	var req harvest.UpdateTimeEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": err.Error()})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	i, err := s.findEntry(r)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": err.Error()})
		return
	}
	entry := &s.entries[i]
	if req.SpentDate != nil {
		entry.SpentDate = *req.SpentDate
	}
	if req.Hours != nil {
		entry.Hours = *req.Hours
	}
	if req.Notes != nil {
		entry.Notes = *req.Notes
	}
	if req.IsBillable != nil {
		entry.IsBillable = *req.IsBillable
	}
	if req.ProjectID != nil || req.TaskID != nil {
		projectID, taskID := entry.Project.ID, entry.Task.ID
		if req.ProjectID != nil {
			projectID = *req.ProjectID
		}
		if req.TaskID != nil {
			taskID = *req.TaskID
		}
		if !fillNames(entry, projectID, taskID) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "unknown project or task"})
			return
		}
	}
	writeJSON(w, http.StatusOK, *entry)
}

func (s *server) handleDeleteEntry(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i, err := s.findEntry(r)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": err.Error()})
		return
	}
	s.entries = append(s.entries[:i], s.entries[i+1:]...)
	writeJSON(w, http.StatusOK, map[string]string{})
}

// handleSetRunning returns a handler that starts or stops an entry's timer.
func (s *server) handleSetRunning(running bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		i, err := s.findEntry(r)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": err.Error()})
			return
		}
		// Mirror Harvest: only one timer runs at a time.
		if running {
			for j := range s.entries {
				s.entries[j].IsRunning = false
			}
		}
		s.entries[i].IsRunning = running
		writeJSON(w, http.StatusOK, s.entries[i])
	}
}
