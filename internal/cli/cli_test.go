package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jc00ke/harvest/internal/config"
	"github.com/jc00ke/harvest/internal/demo"
	"github.com/jc00ke/harvest/internal/harvest"
	"github.com/zalando/go-keyring"
)

// demoToday is the pivot date the demo server seeds entries around: entries
// 1001-1003 land on this day, 1004-1006 the day before, 1007-1008 two days
// before.
var demoToday = time.Date(2026, 6, 10, 9, 0, 0, 0, time.UTC)

// setupDemoCLI swaps the keyring for an in-memory mock holding demo
// credentials and points API client construction at a fresh demo server, so
// commands run end-to-end through authedClient without real credentials.
func setupDemoCLI(t *testing.T) {
	t.Helper()

	keyring.MockInit()
	if err := config.StoreCredentialsInKeyring(config.HarvestConfig{
		AccountID:   "demo",
		AccessToken: "demo",
	}); err != nil {
		t.Fatalf("store credentials: %v", err)
	}

	server := demo.NewServer(demoToday)
	t.Cleanup(server.Close)

	orig := newAPIClient
	newAPIClient = func(accountID, accessToken string) *harvest.Client {
		c := harvest.NewClient(accountID, accessToken)
		c.SetBaseURL(server.URL)
		return c
	}
	t.Cleanup(func() { newAPIClient = orig })
}

// runCLI executes the root command with args and returns its stdout.
func runCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	t.Cleanup(func() { outputJSON = false })

	root := NewRootCommand()
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs(args)
	err := root.Execute()
	return stdout.String(), err
}

func TestMeCommand(t *testing.T) {
	t.Run("given stored credentials when me runs then prints the authenticated user", func(t *testing.T) {
		setupDemoCLI(t)

		out, err := runCLI(t, "me")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		for _, want := range []string{"Demo User", "demo@example.com"} {
			if got := out; !strings.Contains(got, want) {
				t.Errorf("output=%q, want substring %q", got, want)
			}
		}
	})

	t.Run("given the json flag when me runs then emits the user as JSON", func(t *testing.T) {
		setupDemoCLI(t)

		out, err := runCLI(t, "me", "--json")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		var user harvest.User
		if err := json.Unmarshal([]byte(out), &user); err != nil {
			t.Fatalf("unmarshal: %v\noutput: %s", err, out)
		}
		if got, want := user.Email, "demo@example.com"; got != want {
			t.Errorf("user.Email=%s, want=%s", got, want)
		}
	})

	t.Run("given no stored credentials when me runs then returns a login hint error", func(t *testing.T) {
		keyring.MockInit()

		_, err := runCLI(t, "me")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if got, want := err.Error(), "harvest auth login"; !strings.Contains(got, want) {
			t.Errorf("error=%q, want substring %q", got, want)
		}
	})
}

func TestProjectsListCommand(t *testing.T) {
	t.Run("given projects with tasks when listed then prints one row per project task pair", func(t *testing.T) {
		setupDemoCLI(t)

		out, err := runCLI(t, "projects", "list")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		for _, want := range []string{"PROJECT_ID", "Acme Corp", "Website Redesign", "Design", "Maintenance Retainer"} {
			if got := out; !strings.Contains(got, want) {
				t.Errorf("output=%q, want substring %q", got, want)
			}
		}
	})

	t.Run("given the json flag when listed then emits aggregated projects sorted by client", func(t *testing.T) {
		setupDemoCLI(t)

		out, err := runCLI(t, "projects", "list", "--json")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		var projects []harvest.ProjectWithTasks
		if err := json.Unmarshal([]byte(out), &projects); err != nil {
			t.Fatalf("unmarshal: %v\noutput: %s", err, out)
		}
		if got, want := len(projects), 3; got != want {
			t.Fatalf("len(projects)=%d, want=%d", got, want)
		}
		if got, want := projects[0].Project.Client.Name, "Acme Corp"; got != want {
			t.Errorf("first client=%s, want=%s", got, want)
		}
		if got, want := len(projects[0].Tasks), 3; got != want {
			t.Errorf("first project task count=%d, want=%d", got, want)
		}
	})
}

func TestEntriesListCommand(t *testing.T) {
	t.Run("given entries on a date when listed then prints them as a table", func(t *testing.T) {
		setupDemoCLI(t)

		out, err := runCLI(t, "entries", "list", "--date", "2026-06-10")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		for _, want := range []string{"1001", "Sprint planning", "running", "Push notification spike"} {
			if got := out; !strings.Contains(got, want) {
				t.Errorf("output=%q, want substring %q", got, want)
			}
		}
		if got, want := out, "Design review feedback"; strings.Contains(got, want) {
			t.Errorf("output=%q, must not contain previous day's entry %q", got, want)
		}
	})

	t.Run("given the week flag when listed then includes the full seven day window", func(t *testing.T) {
		setupDemoCLI(t)

		out, err := runCLI(t, "entries", "list", "--date", "2026-06-08", "--week")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		for _, want := range []string{"Quarterly roadmap sync", "Design review feedback", "Sprint planning"} {
			if got := out; !strings.Contains(got, want) {
				t.Errorf("output=%q, want substring %q", got, want)
			}
		}
	})

	t.Run("given the json flag when listed then emits the entries as JSON", func(t *testing.T) {
		setupDemoCLI(t)

		out, err := runCLI(t, "entries", "list", "--date", "2026-06-10", "--json")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		var entries []harvest.TimeEntry
		if err := json.Unmarshal([]byte(out), &entries); err != nil {
			t.Fatalf("unmarshal: %v\noutput: %s", err, out)
		}
		if got, want := len(entries), 3; got != want {
			t.Fatalf("len(entries)=%d, want=%d", got, want)
		}
	})

	t.Run("given an invalid date when listed then returns a date format error", func(t *testing.T) {
		setupDemoCLI(t)

		_, err := runCLI(t, "entries", "list", "--date", "bogus")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if got, want := err.Error(), "expected YYYY-MM-DD"; !strings.Contains(got, want) {
			t.Errorf("error=%q, want substring %q", got, want)
		}
	})
}

func TestEntriesCreateCommand(t *testing.T) {
	t.Run("given valid flags when created then prints the new entry and it appears in the list", func(t *testing.T) {
		setupDemoCLI(t)

		out, err := runCLI(t, "entries", "create",
			"--project", "101", "--task", "202",
			"--hours", "1.5", "--notes", "CLI test entry", "--date", "2026-06-10")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		for _, want := range []string{"CLI test entry", "Website Redesign", "1:30"} {
			if got := out; !strings.Contains(got, want) {
				t.Errorf("output=%q, want substring %q", got, want)
			}
		}

		list, err := runCLI(t, "entries", "list", "--date", "2026-06-10")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got, want := list, "CLI test entry"; !strings.Contains(got, want) {
			t.Errorf("list output=%q, want substring %q", got, want)
		}
	})

	t.Run("given an unknown project when created then surfaces the API error message", func(t *testing.T) {
		setupDemoCLI(t)

		_, err := runCLI(t, "entries", "create",
			"--project", "999", "--task", "202", "--hours", "1")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if got, want := err.Error(), "unknown project or task"; !strings.Contains(got, want) {
			t.Errorf("error=%q, want substring %q", got, want)
		}
	})

	t.Run("given a missing required flag when created then returns a usage error", func(t *testing.T) {
		setupDemoCLI(t)

		_, err := runCLI(t, "entries", "create", "--task", "202", "--hours", "1")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if got, want := err.Error(), "project"; !strings.Contains(got, want) {
			t.Errorf("error=%q, want substring %q", got, want)
		}
	})
}

func TestEntriesEditCommand(t *testing.T) {
	t.Run("given changed flags when edited then only those fields are updated", func(t *testing.T) {
		setupDemoCLI(t)

		out, err := runCLI(t, "entries", "edit", "1002", "--hours", "3", "--notes", "Edited notes", "--json")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		var entry harvest.TimeEntry
		if err := json.Unmarshal([]byte(out), &entry); err != nil {
			t.Fatalf("unmarshal: %v\noutput: %s", err, out)
		}
		if got, want := entry.Hours, 3.0; got != want {
			t.Errorf("entry.Hours=%f, want=%f", got, want)
		}
		if got, want := entry.Notes, "Edited notes"; got != want {
			t.Errorf("entry.Notes=%s, want=%s", got, want)
		}
		// Untouched fields keep their seeded values.
		if got, want := entry.Project.Name, "Website Redesign"; got != want {
			t.Errorf("entry.Project.Name=%s, want=%s", got, want)
		}
	})

	t.Run("given a nonexistent entry when edited then surfaces the API error message", func(t *testing.T) {
		setupDemoCLI(t)

		_, err := runCLI(t, "entries", "edit", "9999", "--hours", "2")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if got, want := err.Error(), "time entry 9999 not found"; !strings.Contains(got, want) {
			t.Errorf("error=%q, want substring %q", got, want)
		}
	})
}

func TestEntriesStartStopCommands(t *testing.T) {
	t.Run("given a stopped entry when started then the timer is running", func(t *testing.T) {
		setupDemoCLI(t)

		out, err := runCLI(t, "entries", "start", "1001", "--json")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		var entry harvest.TimeEntry
		if err := json.Unmarshal([]byte(out), &entry); err != nil {
			t.Fatalf("unmarshal: %v\noutput: %s", err, out)
		}
		if got, want := entry.IsRunning, true; got != want {
			t.Errorf("entry.IsRunning=%t, want=%t", got, want)
		}
	})

	t.Run("given a running entry when stopped then the timer is no longer running", func(t *testing.T) {
		setupDemoCLI(t)

		out, err := runCLI(t, "entries", "stop", "1003", "--json")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		var entry harvest.TimeEntry
		if err := json.Unmarshal([]byte(out), &entry); err != nil {
			t.Fatalf("unmarshal: %v\noutput: %s", err, out)
		}
		if got, want := entry.IsRunning, false; got != want {
			t.Errorf("entry.IsRunning=%t, want=%t", got, want)
		}
	})

	t.Run("given a non-integer entry ID when started then returns a parse error", func(t *testing.T) {
		setupDemoCLI(t)

		_, err := runCLI(t, "entries", "start", "abc")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if got, want := err.Error(), "invalid entry ID"; !strings.Contains(got, want) {
			t.Errorf("error=%q, want substring %q", got, want)
		}
	})
}

func TestEntriesDeleteCommand(t *testing.T) {
	t.Run("given an existing entry when deleted then confirms and removes it from the list", func(t *testing.T) {
		setupDemoCLI(t)

		out, err := runCLI(t, "entries", "delete", "1001")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got, want := out, "Deleted time entry 1001"; !strings.Contains(got, want) {
			t.Errorf("output=%q, want substring %q", got, want)
		}

		list, err := runCLI(t, "entries", "list", "--date", "2026-06-10")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got, want := list, "Sprint planning"; strings.Contains(got, want) {
			t.Errorf("list output=%q, must not contain deleted entry %q", got, want)
		}
	})

	t.Run("given a nonexistent entry when deleted then surfaces the API error message", func(t *testing.T) {
		setupDemoCLI(t)

		_, err := runCLI(t, "entries", "delete", "9999")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if got, want := err.Error(), "time entry 9999 not found"; !strings.Contains(got, want) {
			t.Errorf("error=%q, want substring %q", got, want)
		}
	})
}

func TestAuthCommands(t *testing.T) {
	t.Run("given valid flag credentials when login runs then validates and stores them", func(t *testing.T) {
		keyring.MockInit()
		server := demo.NewServer(demoToday)
		t.Cleanup(server.Close)
		orig := newAPIClient
		newAPIClient = func(accountID, accessToken string) *harvest.Client {
			c := harvest.NewClient(accountID, accessToken)
			c.SetBaseURL(server.URL)
			return c
		}
		t.Cleanup(func() { newAPIClient = orig })

		out, err := runCLI(t, "auth", "login", "--account-id", "demo", "--access-token", "demo")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got, want := out, "Logged in as Demo User"; !strings.Contains(got, want) {
			t.Errorf("output=%q, want substring %q", got, want)
		}

		cfg, err := config.LoadFromKeyring()
		if err != nil {
			t.Fatalf("expected credentials in keyring, got %v", err)
		}
		if got, want := cfg.Harvest.AccountID, "demo"; got != want {
			t.Errorf("stored account ID=%s, want=%s", got, want)
		}
	})

	t.Run("given stored credentials when status runs then reports the keyring source", func(t *testing.T) {
		setupDemoCLI(t)

		out, err := runCLI(t, "auth", "status")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		for _, want := range []string{"keyring", "demo"} {
			if got := out; !strings.Contains(got, want) {
				t.Errorf("output=%q, want substring %q", got, want)
			}
		}
	})

	t.Run("given no stored credentials when status runs then prints a login hint", func(t *testing.T) {
		keyring.MockInit()

		out, err := runCLI(t, "auth", "status")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got, want := out, "No credentials found"; !strings.Contains(got, want) {
			t.Errorf("output=%q, want substring %q", got, want)
		}
	})

	t.Run("given stored credentials when logout runs then removes them from the keyring", func(t *testing.T) {
		setupDemoCLI(t)

		out, err := runCLI(t, "auth", "logout")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got, want := out, "Removed credentials from keyring."; !strings.Contains(got, want) {
			t.Errorf("output=%q, want substring %q", got, want)
		}

		if _, err := config.LoadFromKeyring(); err == nil {
			t.Error("expected keyring load to fail after logout, got nil error")
		}
	})

	t.Run("given no stored credentials when logout runs then reports nothing was stored", func(t *testing.T) {
		keyring.MockInit()

		out, err := runCLI(t, "auth", "logout")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got, want := out, "No credentials were stored in the keyring."; !strings.Contains(got, want) {
			t.Errorf("output=%q, want substring %q", got, want)
		}
	})
}
