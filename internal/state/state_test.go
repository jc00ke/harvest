package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStateLoading(t *testing.T) {
	t.Run("given an existing state file when loaded then returns state with correct recents", func(t *testing.T) {
		tempDir := t.TempDir()
		originalHome := os.Getenv("HOME")
		t.Cleanup(func() { os.Setenv("HOME", originalHome) })

		os.Setenv("HOME", tempDir)

		stateDir := filepath.Join(tempDir, ".config", "harvest")
		err := os.MkdirAll(stateDir, 0755)
		if err != nil {
			t.Fatal(err)
		}

		statePath := filepath.Join(stateDir, "state.json")
		stateContent := `{
  "recents": [
    {"client_id": 123, "project_id": 456, "task_id": 789},
    {"client_id": 124, "project_id": 457, "task_id": 790}
  ]
}`
		err = os.WriteFile(statePath, []byte(stateContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		state, err := Load()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if got, want := len(state.Recents), 2; got != want {
			t.Errorf("len(state.Recents)=%d, want=%d", got, want)
		}

		if got, want := state.Recents[0], (RecentEntry{ClientID: 123, ProjectID: 456, TaskID: 789}); got != want {
			t.Errorf("first recent=%+v, want=%+v", got, want)
		}

		if got, want := state.Recents[1], (RecentEntry{ClientID: 124, ProjectID: 457, TaskID: 790}); got != want {
			t.Errorf("second recent=%+v, want=%+v", got, want)
		}
	})

	t.Run("given missing state file when loaded then returns empty state", func(t *testing.T) {
		tempDir := t.TempDir()
		originalHome := os.Getenv("HOME")
		t.Cleanup(func() { os.Setenv("HOME", originalHome) })

		os.Setenv("HOME", tempDir)

		state, err := Load()
		if err != nil {
			t.Fatalf("expected no error for missing state file, got %v", err)
		}

		if state == nil {
			t.Fatal("expected state to be initialized, got nil")
		}

		if got, want := len(state.Recents), 0; got != want {
			t.Errorf("len(state.Recents)=%d, want=%d", got, want)
		}
	})

	t.Run("given malformed state file when loaded then returns parse error", func(t *testing.T) {
		tempDir := t.TempDir()
		originalHome := os.Getenv("HOME")
		t.Cleanup(func() { os.Setenv("HOME", originalHome) })

		os.Setenv("HOME", tempDir)

		stateDir := filepath.Join(tempDir, ".config", "harvest")
		err := os.MkdirAll(stateDir, 0755)
		if err != nil {
			t.Fatal(err)
		}

		statePath := filepath.Join(stateDir, "state.json")
		malformedContent := `{"recents": [invalid json`
		err = os.WriteFile(statePath, []byte(malformedContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		_, err = Load()
		if err == nil {
			t.Fatal("expected error for malformed state file")
		}

		if got, want := err.Error(), "could not parse state file:"; !strings.HasPrefix(got, want) {
			t.Errorf("error=%q, want prefix %q", got, want)
		}
	})

	t.Run("given state file with null recents when loaded then returns empty recents list", func(t *testing.T) {
		tempDir := t.TempDir()
		originalHome := os.Getenv("HOME")
		t.Cleanup(func() { os.Setenv("HOME", originalHome) })

		os.Setenv("HOME", tempDir)

		stateDir := filepath.Join(tempDir, ".config", "harvest")
		err := os.MkdirAll(stateDir, 0755)
		if err != nil {
			t.Fatal(err)
		}

		statePath := filepath.Join(stateDir, "state.json")
		nullContent := `{"recents": null}`
		err = os.WriteFile(statePath, []byte(nullContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		state, err := Load()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if got, want := len(state.Recents), 0; got != want {
			t.Errorf("len(state.Recents)=%d, want=%d", got, want)
		}
	})
}

func TestStateSaving(t *testing.T) {
	t.Run("given a state with recents when saved then creates correct JSON file", func(t *testing.T) {
		tempDir := t.TempDir()
		originalHome := os.Getenv("HOME")
		t.Cleanup(func() { os.Setenv("HOME", originalHome) })

		os.Setenv("HOME", tempDir)

		state := &State{
			Recents: []RecentEntry{
				{ClientID: 123, ProjectID: 456, TaskID: 789},
				{ClientID: 124, ProjectID: 457, TaskID: 790},
			},
		}

		err := state.Save()
		if err != nil {
			t.Fatalf("expected no error saving state, got %v", err)
		}

		statePath := filepath.Join(tempDir, ".config", "harvest", "state.json")
		data, err := os.ReadFile(statePath)
		if err != nil {
			t.Fatalf("expected state file to exist, got error: %v", err)
		}

		var savedState State
		err = json.Unmarshal(data, &savedState)
		if err != nil {
			t.Fatalf("expected valid JSON, got error: %v", err)
		}

		if got, want := len(savedState.Recents), 2; got != want {
			t.Errorf("len(savedState.Recents)=%d, want=%d", got, want)
		}

		if got, want := savedState.Recents[0], (RecentEntry{ClientID: 123, ProjectID: 456, TaskID: 789}); got != want {
			t.Errorf("first recent=%+v, want=%+v", got, want)
		}
	})

	t.Run("given empty state when saved then creates file with empty recents array", func(t *testing.T) {
		tempDir := t.TempDir()
		originalHome := os.Getenv("HOME")
		t.Cleanup(func() { os.Setenv("HOME", originalHome) })

		os.Setenv("HOME", tempDir)

		state := &State{Recents: []RecentEntry{}}

		err := state.Save()
		if err != nil {
			t.Fatalf("expected no error saving empty state, got %v", err)
		}

		statePath := filepath.Join(tempDir, ".config", "harvest", "state.json")
		data, err := os.ReadFile(statePath)
		if err != nil {
			t.Fatalf("expected state file to exist, got error: %v", err)
		}

		var savedState State
		err = json.Unmarshal(data, &savedState)
		if err != nil {
			t.Fatalf("expected valid JSON, got error: %v", err)
		}

		if got, want := len(savedState.Recents), 0; got != want {
			t.Errorf("len(savedState.Recents)=%d, want=%d", got, want)
		}
	})

	t.Run("given state when saved then creates state directory if it does not exist", func(t *testing.T) {
		tempDir := t.TempDir()
		originalHome := os.Getenv("HOME")
		t.Cleanup(func() { os.Setenv("HOME", originalHome) })

		os.Setenv("HOME", tempDir)

		state := &State{Recents: []RecentEntry{}}

		err := state.Save()
		if err != nil {
			t.Fatalf("expected no error saving to non-existent directory, got %v", err)
		}

		stateDir := filepath.Join(tempDir, ".config", "harvest")
		if _, err := os.Stat(stateDir); os.IsNotExist(err) {
			t.Error("expected state directory to be created")
		}

		statePath := filepath.Join(stateDir, "state.json")
		if _, err := os.Stat(statePath); os.IsNotExist(err) {
			t.Error("expected state file to be created")
		}
	})
}

func TestRecentsManagement(t *testing.T) {
	t.Run("given empty state when recent added then becomes first in list", func(t *testing.T) {
		state := &State{Recents: []RecentEntry{}}

		state.AddRecent(123, 456, 789)

		if got, want := len(state.Recents), 1; got != want {
			t.Errorf("len(state.Recents)=%d, want=%d", got, want)
		}

		if got, want := state.Recents[0], (RecentEntry{ClientID: 123, ProjectID: 456, TaskID: 789}); got != want {
			t.Errorf("recent=%+v, want=%+v", got, want)
		}
	})

	t.Run("given state with recents when duplicate added then moves to top and removes duplicate", func(t *testing.T) {
		state := &State{
			Recents: []RecentEntry{
				{ClientID: 123, ProjectID: 456, TaskID: 789},
				{ClientID: 124, ProjectID: 457, TaskID: 790},
				{ClientID: 125, ProjectID: 458, TaskID: 791},
			},
		}

		state.AddRecent(124, 457, 790)

		if got, want := len(state.Recents), 3; got != want {
			t.Errorf("len(state.Recents)=%d, want=%d", got, want)
		}

		if got, want := state.Recents[0], (RecentEntry{ClientID: 124, ProjectID: 457, TaskID: 790}); got != want {
			t.Errorf("first recent=%+v, want=%+v", got, want)
		}

		for i := 1; i < len(state.Recents); i++ {
			if state.Recents[i] == (RecentEntry{ClientID: 124, ProjectID: 457, TaskID: 790}) {
				t.Error("found duplicate entry that should have been removed")
			}
		}
	})

	t.Run("given state with 3 recents when new recent added then keeps only 3 most recent", func(t *testing.T) {
		state := &State{
			Recents: []RecentEntry{
				{ClientID: 123, ProjectID: 456, TaskID: 789},
				{ClientID: 124, ProjectID: 457, TaskID: 790},
				{ClientID: 125, ProjectID: 458, TaskID: 791},
			},
		}

		state.AddRecent(126, 459, 792)

		if got, want := len(state.Recents), 3; got != want {
			t.Errorf("len(state.Recents)=%d, want=%d", got, want)
		}

		if got, want := state.Recents[0], (RecentEntry{ClientID: 126, ProjectID: 459, TaskID: 792}); got != want {
			t.Errorf("first recent=%+v, want=%+v", got, want)
		}

		for _, recent := range state.Recents {
			if recent == (RecentEntry{ClientID: 125, ProjectID: 458, TaskID: 791}) {
				t.Error("oldest entry should have been removed when adding 4th recent")
			}
		}
	})

	t.Run("given state with recents when new recent added then places at top", func(t *testing.T) {
		state := &State{
			Recents: []RecentEntry{
				{ClientID: 123, ProjectID: 456, TaskID: 789},
				{ClientID: 124, ProjectID: 457, TaskID: 790},
			},
		}

		state.AddRecent(125, 458, 791)

		if got, want := len(state.Recents), 3; got != want {
			t.Errorf("len(state.Recents)=%d, want=%d", got, want)
		}

		if got, want := state.Recents[0], (RecentEntry{ClientID: 125, ProjectID: 458, TaskID: 791}); got != want {
			t.Errorf("first recent=%+v, want=%+v", got, want)
		}

		if got, want := state.Recents[1], (RecentEntry{ClientID: 123, ProjectID: 456, TaskID: 789}); got != want {
			t.Errorf("second recent=%+v, want=%+v", got, want)
		}
	})
}
