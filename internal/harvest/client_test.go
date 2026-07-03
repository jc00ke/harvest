package harvest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestValidateAuth(t *testing.T) {
	t.Run("given valid credentials when ValidateAuth called then returns user info without error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got, want := r.URL.Path, "/v2/users/me"; got != want {
				t.Errorf("path=%s, want=%s", got, want)
			}
			if got, want := r.Method, http.MethodGet; got != want {
				t.Errorf("method=%s, want=%s", got, want)
			}

			// Verify headers
			if got, want := r.Header.Get("Harvest-Account-Id"), "12345"; got != want {
				t.Errorf("Harvest-Account-Id header=%s, want=%s", got, want)
			}
			if got, want := r.Header.Get("Authorization"), "Bearer test-token"; got != want {
				t.Errorf("Authorization header=%s, want=%s", got, want)
			}
			if r.Header.Get("User-Agent") == "" {
				t.Error("expected User-Agent header to be set")
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"id":         1,
				"first_name": "Test",
				"last_name":  "User",
				"email":      "test@example.com",
			})
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		user, err := client.ValidateAuth(t.Context())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if got, want := user.ID, 1; got != want {
			t.Errorf("user.ID=%d, want=%d", got, want)
		}
		if got, want := user.FirstName, "Test"; got != want {
			t.Errorf("user.FirstName=%s, want=%s", got, want)
		}
		if got, want := user.LastName, "User"; got != want {
			t.Errorf("user.LastName=%s, want=%s", got, want)
		}
		if got, want := user.Email, "test@example.com"; got != want {
			t.Errorf("user.Email=%s, want=%s", got, want)
		}
	})

	t.Run("given invalid credentials when ValidateAuth called then returns authentication error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]any{
				"error":             "invalid_token",
				"error_description": "The access token is invalid",
			})
		}))
		defer server.Close()

		client := NewClient("12345", "invalid-token")
		client.SetBaseURL(server.URL)

		user, err := client.ValidateAuth(t.Context())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if user != nil {
			t.Errorf("expected nil user, got %v", user)
		}

		if got, want := err.Error(), "authentication failed"; !strings.Contains(got, want) {
			t.Errorf("error=%q, want substring %q", got, want)
		}
	})

	t.Run("given rate limited response when ValidateAuth called then returns rate limit error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Retry-After", "5")
			w.WriteHeader(http.StatusTooManyRequests)
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		user, err := client.ValidateAuth(t.Context())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if user != nil {
			t.Errorf("expected nil user, got %v", user)
		}

		if got, want := err.Error(), "429"; !strings.Contains(got, want) {
			t.Errorf("error=%q, want substring %q", got, want)
		}
	})

	t.Run("given malformed JSON response when ValidateAuth called then returns parse error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("{invalid json"))
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		user, err := client.ValidateAuth(t.Context())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if user != nil {
			t.Errorf("expected nil user, got %v", user)
		}

		if got, want := err.Error(), "parse"; !strings.Contains(got, want) {
			t.Errorf("error=%q, want substring %q", got, want)
		}
	})

	t.Run("given timeout when ValidateAuth called then returns network error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)
		client.SetHTTPClient(&http.Client{
			Timeout: 10 * time.Millisecond,
		})

		user, err := client.ValidateAuth(t.Context())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if user != nil {
			t.Errorf("expected nil user, got %v", user)
		}

		if got, want := err.Error(), "network"; !strings.Contains(got, want) {
			t.Errorf("error=%q, want substring %q", got, want)
		}
	})
}

func TestFetchProjects(t *testing.T) {
	t.Run("given valid response when FetchProjects called then returns projects with client data", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got, want := r.URL.Path, "/v2/projects"; got != want {
				t.Errorf("path=%s, want=%s", got, want)
			}
			if got, want := r.Method, http.MethodGet; got != want {
				t.Errorf("method=%s, want=%s", got, want)
			}
			// Verify is_active query param is set
			if got, want := r.URL.Query().Get("is_active"), "true"; got != want {
				t.Errorf("is_active query param=%s, want=%s", got, want)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"projects": []map[string]any{
					{
						"id":   1,
						"name": "API Development",
						"client": map[string]any{
							"id":   100,
							"name": "Acme Corp",
						},
					},
					{
						"id":   2,
						"name": "Mobile App",
						"client": map[string]any{
							"id":   100,
							"name": "Acme Corp",
						},
					},
					{
						"id":   3,
						"name": "Consulting",
						"client": map[string]any{
							"id":   200,
							"name": "BigCo Industries",
						},
					},
				},
				"per_page":      100,
				"total_pages":   1,
				"total_entries": 3,
				"page":          1,
			})
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		projects, err := client.FetchProjects(t.Context())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if got, want := len(projects), 3; got != want {
			t.Fatalf("len(projects)=%d, want=%d", got, want)
		}

		// Verify first project
		if got, want := projects[0].ID, 1; got != want {
			t.Errorf("project ID=%d, want=%d", got, want)
		}
		if got, want := projects[0].Name, "API Development"; got != want {
			t.Errorf("project name=%s, want=%s", got, want)
		}
		if got, want := projects[0].Client.ID, 100; got != want {
			t.Errorf("client ID=%d, want=%d", got, want)
		}
		if got, want := projects[0].Client.Name, "Acme Corp"; got != want {
			t.Errorf("client name=%s, want=%s", got, want)
		}

		// Verify third project has different client
		if got, want := projects[2].Client.ID, 200; got != want {
			t.Errorf("client ID=%d, want=%d", got, want)
		}
		if got, want := projects[2].Client.Name, "BigCo Industries"; got != want {
			t.Errorf("client name=%s, want=%s", got, want)
		}
	})

	t.Run("given empty projects response when FetchProjects called then returns empty slice", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"projects":      []any{},
				"per_page":      100,
				"total_pages":   1,
				"total_entries": 0,
				"page":          1,
			})
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		projects, err := client.FetchProjects(t.Context())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if got, want := len(projects), 0; got != want {
			t.Errorf("len(projects)=%d, want=%d", got, want)
		}
	})

	t.Run("given paginated response when FetchProjects called then fetches all pages", func(t *testing.T) {
		requestCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			page := r.URL.Query().Get("page")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			if page == "" || page == "1" {
				json.NewEncoder(w).Encode(map[string]any{
					"projects": []map[string]any{
						{"id": 1, "name": "Project 1", "client": map[string]any{"id": 1, "name": "Client 1"}},
						{"id": 2, "name": "Project 2", "client": map[string]any{"id": 1, "name": "Client 1"}},
					},
					"per_page":      2,
					"total_pages":   2,
					"total_entries": 3,
					"page":          1,
					"next_page":     2,
				})
			} else {
				json.NewEncoder(w).Encode(map[string]any{
					"projects": []map[string]any{
						{"id": 3, "name": "Project 3", "client": map[string]any{"id": 2, "name": "Client 2"}},
					},
					"per_page":      2,
					"total_pages":   2,
					"total_entries": 3,
					"page":          2,
				})
			}
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		projects, err := client.FetchProjects(t.Context())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if got, want := len(projects), 3; got != want {
			t.Errorf("len(projects)=%d, want=%d", got, want)
		}

		if got, want := requestCount, 2; got != want {
			t.Errorf("request count=%d, want=%d", got, want)
		}
	})
}

func TestFetchTaskAssignments(t *testing.T) {
	t.Run("given valid response when FetchTaskAssignments called then returns task assignments with project and task data", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got, want := r.URL.Path, "/v2/task_assignments"; got != want {
				t.Errorf("path=%s, want=%s", got, want)
			}
			if got, want := r.Method, http.MethodGet; got != want {
				t.Errorf("method=%s, want=%s", got, want)
			}
			// Verify is_active query param is set
			if got, want := r.URL.Query().Get("is_active"), "true"; got != want {
				t.Errorf("is_active query param=%s, want=%s", got, want)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"task_assignments": []map[string]any{
					{
						"id": 1,
						"project": map[string]any{
							"id":   100,
							"name": "API Development",
						},
						"task": map[string]any{
							"id":   1000,
							"name": "Code Review",
						},
						"is_active": true,
						"billable":  true,
					},
					{
						"id": 2,
						"project": map[string]any{
							"id":   100,
							"name": "API Development",
						},
						"task": map[string]any{
							"id":   1001,
							"name": "Development",
						},
						"is_active": true,
						"billable":  true,
					},
					{
						"id": 3,
						"project": map[string]any{
							"id":   200,
							"name": "Mobile App",
						},
						"task": map[string]any{
							"id":   1002,
							"name": "Testing",
						},
						"is_active": true,
						"billable":  false,
					},
				},
				"per_page":      100,
				"total_pages":   1,
				"total_entries": 3,
				"page":          1,
			})
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		taskAssignments, err := client.FetchTaskAssignments(t.Context())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if got, want := len(taskAssignments), 3; got != want {
			t.Fatalf("len(taskAssignments)=%d, want=%d", got, want)
		}

		// Verify first task assignment
		if got, want := taskAssignments[0].ID, 1; got != want {
			t.Errorf("task assignment ID=%d, want=%d", got, want)
		}
		if got, want := taskAssignments[0].Project.ID, 100; got != want {
			t.Errorf("project ID=%d, want=%d", got, want)
		}
		if got, want := taskAssignments[0].Project.Name, "API Development"; got != want {
			t.Errorf("project name=%s, want=%s", got, want)
		}
		if got, want := taskAssignments[0].Task.ID, 1000; got != want {
			t.Errorf("task ID=%d, want=%d", got, want)
		}
		if got, want := taskAssignments[0].Task.Name, "Code Review"; got != want {
			t.Errorf("task name=%s, want=%s", got, want)
		}
	})

	t.Run("given empty task assignments response when FetchTaskAssignments called then returns empty slice", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"task_assignments": []any{},
				"per_page":         100,
				"total_pages":      1,
				"total_entries":    0,
				"page":             1,
			})
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		taskAssignments, err := client.FetchTaskAssignments(t.Context())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if got, want := len(taskAssignments), 0; got != want {
			t.Errorf("len(taskAssignments)=%d, want=%d", got, want)
		}
	})

	t.Run("given paginated response when FetchTaskAssignments called then fetches all pages", func(t *testing.T) {
		requestCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			page := r.URL.Query().Get("page")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			if page == "" || page == "1" {
				json.NewEncoder(w).Encode(map[string]any{
					"task_assignments": []map[string]any{
						{"id": 1, "project": map[string]any{"id": 1, "name": "P1"}, "task": map[string]any{"id": 1, "name": "T1"}},
						{"id": 2, "project": map[string]any{"id": 1, "name": "P1"}, "task": map[string]any{"id": 2, "name": "T2"}},
					},
					"per_page":      2,
					"total_pages":   2,
					"total_entries": 3,
					"page":          1,
					"next_page":     2,
				})
			} else {
				json.NewEncoder(w).Encode(map[string]any{
					"task_assignments": []map[string]any{
						{"id": 3, "project": map[string]any{"id": 2, "name": "P2"}, "task": map[string]any{"id": 3, "name": "T3"}},
					},
					"per_page":      2,
					"total_pages":   2,
					"total_entries": 3,
					"page":          2,
				})
			}
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		taskAssignments, err := client.FetchTaskAssignments(t.Context())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if got, want := len(taskAssignments), 3; got != want {
			t.Errorf("len(taskAssignments)=%d, want=%d", got, want)
		}

		if got, want := requestCount, 2; got != want {
			t.Errorf("request count=%d, want=%d", got, want)
		}
	})
}

func TestFetchTimeEntries(t *testing.T) {
	t.Run("given valid response when FetchTimeEntries called then returns time entries for date", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got, want := r.URL.Path, "/v2/time_entries"; got != want {
				t.Errorf("path=%s, want=%s", got, want)
			}
			if got, want := r.Method, http.MethodGet; got != want {
				t.Errorf("method=%s, want=%s", got, want)
			}
			// Verify from and to query params are set to the same date
			if got, want := r.URL.Query().Get("from"), "2025-01-15"; got != want {
				t.Errorf("from query param=%s, want=%s", got, want)
			}
			if got, want := r.URL.Query().Get("to"), "2025-01-15"; got != want {
				t.Errorf("to query param=%s, want=%s", got, want)
			}
			// Verify user_id is included
			if got, want := r.URL.Query().Get("user_id"), "123"; got != want {
				t.Errorf("user_id query param=%s, want=%s", got, want)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"time_entries": []map[string]any{
					{
						"id":         1,
						"spent_date": "2025-01-15",
						"hours":      1.5,
						"notes":      "Code review",
						"is_running": false,
						"is_locked":  false,
						"billable":   true,
						"client": map[string]any{
							"id":   100,
							"name": "Acme Corp",
						},
						"project": map[string]any{
							"id":   200,
							"name": "API Development",
						},
						"task": map[string]any{
							"id":   300,
							"name": "Code Review",
						},
					},
					{
						"id":         2,
						"spent_date": "2025-01-15",
						"hours":      2.0,
						"notes":      "Feature development",
						"is_running": true,
						"is_locked":  false,
						"billable":   true,
						"client": map[string]any{
							"id":   100,
							"name": "Acme Corp",
						},
						"project": map[string]any{
							"id":   201,
							"name": "Mobile App",
						},
						"task": map[string]any{
							"id":   301,
							"name": "Development",
						},
					},
					{
						"id":         3,
						"spent_date": "2025-01-15",
						"hours":      0.75,
						"notes":      "Weekly sync",
						"is_running": false,
						"is_locked":  true,
						"billable":   false,
						"client": map[string]any{
							"id":   101,
							"name": "BigCo Industries",
						},
						"project": map[string]any{
							"id":   202,
							"name": "Consulting",
						},
						"task": map[string]any{
							"id":   302,
							"name": "Meetings",
						},
					},
				},
				"per_page":      100,
				"total_pages":   1,
				"total_entries": 3,
				"page":          1,
			})
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)
		client.SetUserID(123) // Set user ID for testing

		entries, err := client.FetchTimeEntries(t.Context(), "2025-01-15", "2025-01-15")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if got, want := len(entries), 3; got != want {
			t.Fatalf("len(entries)=%d, want=%d", got, want)
		}

		// Verify first entry
		if got, want := entries[0].ID, 1; got != want {
			t.Errorf("entry ID=%d, want=%d", got, want)
		}
		if got, want := entries[0].Hours, 1.5; got != want {
			t.Errorf("hours=%f, want=%f", got, want)
		}
		if got, want := entries[0].Notes, "Code review"; got != want {
			t.Errorf("notes=%s, want=%s", got, want)
		}
		if got, want := entries[0].IsRunning, false; got != want {
			t.Errorf("IsRunning=%t, want=%t", got, want)
		}
		if got, want := entries[0].Client.Name, "Acme Corp"; got != want {
			t.Errorf("client name=%s, want=%s", got, want)
		}
		if got, want := entries[0].Project.Name, "API Development"; got != want {
			t.Errorf("project name=%s, want=%s", got, want)
		}
		if got, want := entries[0].Task.Name, "Code Review"; got != want {
			t.Errorf("task name=%s, want=%s", got, want)
		}

		// Verify second entry is running
		if got, want := entries[1].IsRunning, true; got != want {
			t.Errorf("entry 2 IsRunning=%t, want=%t", got, want)
		}

		// Verify third entry is locked
		if got, want := entries[2].IsLocked, true; got != want {
			t.Errorf("entry 3 IsLocked=%t, want=%t", got, want)
		}
	})

	t.Run("given empty time entries response when FetchTimeEntries called then returns empty slice", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"time_entries":  []any{},
				"per_page":      100,
				"total_pages":   1,
				"total_entries": 0,
				"page":          1,
			})
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)
		client.SetUserID(123) // Set user ID for testing

		entries, err := client.FetchTimeEntries(t.Context(), "2025-01-15", "2025-01-15")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if got, want := len(entries), 0; got != want {
			t.Errorf("len(entries)=%d, want=%d", got, want)
		}
	})

	t.Run("given paginated response when FetchTimeEntries called then fetches all pages", func(t *testing.T) {
		requestCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			page := r.URL.Query().Get("page")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			if page == "" || page == "1" {
				json.NewEncoder(w).Encode(map[string]any{
					"time_entries": []map[string]any{
						{"id": 1, "spent_date": "2025-01-15", "hours": 1.0, "client": map[string]any{"id": 1, "name": "C1"}, "project": map[string]any{"id": 1, "name": "P1"}, "task": map[string]any{"id": 1, "name": "T1"}},
						{"id": 2, "spent_date": "2025-01-15", "hours": 1.0, "client": map[string]any{"id": 1, "name": "C1"}, "project": map[string]any{"id": 1, "name": "P1"}, "task": map[string]any{"id": 1, "name": "T1"}},
					},
					"per_page":      2,
					"total_pages":   2,
					"total_entries": 3,
					"page":          1,
					"next_page":     2,
				})
			} else {
				json.NewEncoder(w).Encode(map[string]any{
					"time_entries": []map[string]any{
						{"id": 3, "spent_date": "2025-01-15", "hours": 1.0, "client": map[string]any{"id": 1, "name": "C1"}, "project": map[string]any{"id": 1, "name": "P1"}, "task": map[string]any{"id": 1, "name": "T1"}},
					},
					"per_page":      2,
					"total_pages":   2,
					"total_entries": 3,
					"page":          2,
				})
			}
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)
		client.SetUserID(123) // Set user ID for testing

		entries, err := client.FetchTimeEntries(t.Context(), "2025-01-15", "2025-01-15")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if got, want := len(entries), 3; got != want {
			t.Errorf("len(entries)=%d, want=%d", got, want)
		}

		if got, want := requestCount, 2; got != want {
			t.Errorf("request count=%d, want=%d", got, want)
		}
	})
}

func TestCreateTimeEntry(t *testing.T) {
	t.Run("given valid time entry data when CreateTimeEntry called then creates entry and returns it", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got, want := r.URL.Path, "/v2/time_entries"; got != want {
				t.Errorf("path=%s, want=%s", got, want)
			}
			if got, want := r.Method, http.MethodPost; got != want {
				t.Errorf("method=%s, want=%s", got, want)
			}

			// Verify Content-Type header
			if got, want := r.Header.Get("Content-Type"), "application/json"; got != want {
				t.Errorf("Content-Type header=%s, want=%s", got, want)
			}

			// Verify request body
			var reqData map[string]any
			if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
				t.Fatalf("failed to decode request body: %v", err)
			}

			wantFields := map[string]any{
				"project_id": float64(100),
				"task_id":    float64(200),
				"spent_date": "2025-01-15",
				"hours":      1.5,
				"notes":      "Code review session",
			}

			for field, want := range wantFields {
				if got := reqData[field]; got != want {
					t.Errorf("%s=%v, want=%v", field, got, want)
				}
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{
				"id":         1001,
				"spent_date": "2025-01-15",
				"hours":      1.5,
				"notes":      "Code review session",
				"is_running": false,
				"is_locked":  false,
				"billable":   true,
				"client": map[string]any{
					"id":   50,
					"name": "Test Client",
				},
				"project": map[string]any{
					"id":   100,
					"name": "Test Project",
				},
				"task": map[string]any{
					"id":   200,
					"name": "Development",
				},
			})
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		entry := CreateTimeEntryRequest{
			ProjectID: 100,
			TaskID:    200,
			SpentDate: "2025-01-15",
			Hours:     1.5,
			Notes:     "Code review session",
		}

		created, err := client.CreateTimeEntry(t.Context(), entry)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if got, want := created.ID, 1001; got != want {
			t.Errorf("created.ID=%d, want=%d", got, want)
		}
		if got, want := created.SpentDate, "2025-01-15"; got != want {
			t.Errorf("created.SpentDate=%s, want=%s", got, want)
		}
		if got, want := created.Hours, 1.5; got != want {
			t.Errorf("created.Hours=%f, want=%f", got, want)
		}
		if got, want := created.Notes, "Code review session"; got != want {
			t.Errorf("created.Notes=%s, want=%s", got, want)
		}
		if got, want := created.Project.ID, 100; got != want {
			t.Errorf("created.Project.ID=%d, want=%d", got, want)
		}
		if got, want := created.Task.ID, 200; got != want {
			t.Errorf("created.Task.ID=%d, want=%d", got, want)
		}
	})

	t.Run("given time entry with billable field when CreateTimeEntry called then creates entry with correct billable flag", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var reqData map[string]any
			if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
				t.Fatalf("failed to decode request body: %v", err)
			}

			if got, want := reqData["billable"], false; got != want {
				t.Errorf("billable=%v, want=%v", got, want)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{
				"id":         1002,
				"spent_date": "2025-01-15",
				"hours":      2.0,
				"notes":      "Non-billable work",
				"billable":   false,
				"client":     map[string]any{"id": 50, "name": "Test Client"},
				"project":    map[string]any{"id": 100, "name": "Test Project"},
				"task":       map[string]any{"id": 200, "name": "Development"},
			})
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		entry := CreateTimeEntryRequest{
			ProjectID:  100,
			TaskID:     200,
			SpentDate:  "2025-01-15",
			Hours:      2.0,
			Notes:      "Non-billable work",
			IsBillable: &[]bool{false}[0],
		}

		created, err := client.CreateTimeEntry(t.Context(), entry)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if got, want := created.IsBillable, false; got != want {
			t.Errorf("created.IsBillable=%t, want=%t", got, want)
		}
	})

	t.Run("given a request the server rejects when CreateTimeEntry called then returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"message": "The project is archived.",
			})
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		entry := CreateTimeEntryRequest{
			ProjectID: 999,
			TaskID:    200,
			SpentDate: "2025-01-15",
			Hours:     1.0,
		}

		created, err := client.CreateTimeEntry(t.Context(), entry)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if created != nil {
			t.Errorf("expected nil time entry, got %v", created)
		}

		if got, want := err.Error(), "400"; !strings.Contains(got, want) {
			t.Errorf("error=%q, want substring %q", got, want)
		}
	})

	t.Run("given unauthorized request when CreateTimeEntry called then returns auth error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]any{
				"error": "invalid_token",
			})
		}))
		defer server.Close()

		client := NewClient("12345", "invalid-token")
		client.SetBaseURL(server.URL)

		entry := CreateTimeEntryRequest{
			ProjectID: 100,
			TaskID:    200,
			SpentDate: "2025-01-15",
			Hours:     1.0,
		}

		created, err := client.CreateTimeEntry(t.Context(), entry)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if created != nil {
			t.Errorf("expected nil time entry, got %v", created)
		}

		if got, want := err.Error(), "401"; !strings.Contains(got, want) {
			t.Errorf("error=%q, want substring %q", got, want)
		}
	})
}

func TestUpdateTimeEntry(t *testing.T) {
	t.Run("given valid update data when UpdateTimeEntry called then updates entry and returns it", func(t *testing.T) {
		entryID := 1001
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got, want := r.URL.Path, fmt.Sprintf("/v2/time_entries/%d", entryID); got != want {
				t.Errorf("path=%s, want=%s", got, want)
			}
			if got, want := r.Method, http.MethodPatch; got != want {
				t.Errorf("method=%s, want=%s", got, want)
			}

			// Verify Content-Type header
			if got, want := r.Header.Get("Content-Type"), "application/json"; got != want {
				t.Errorf("Content-Type header=%s, want=%s", got, want)
			}

			// Verify request body
			var reqData map[string]any
			if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
				t.Fatalf("failed to decode request body: %v", err)
			}

			wantFields := map[string]any{
				"hours": 2.5,
				"notes": "Updated notes",
			}

			for field, want := range wantFields {
				if got := reqData[field]; got != want {
					t.Errorf("%s=%v, want=%v", field, got, want)
				}
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"id":         1001,
				"spent_date": "2025-01-15",
				"hours":      2.5,
				"notes":      "Updated notes",
				"is_running": false,
				"is_locked":  false,
				"billable":   true,
				"client": map[string]any{
					"id":   50,
					"name": "Test Client",
				},
				"project": map[string]any{
					"id":   100,
					"name": "Test Project",
				},
				"task": map[string]any{
					"id":   200,
					"name": "Development",
				},
			})
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		update := UpdateTimeEntryRequest{
			Hours: &[]float64{2.5}[0],
			Notes: &[]string{"Updated notes"}[0],
		}

		updated, err := client.UpdateTimeEntry(t.Context(), entryID, update)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if got, want := updated.ID, 1001; got != want {
			t.Errorf("updated.ID=%d, want=%d", got, want)
		}
		if got, want := updated.Hours, 2.5; got != want {
			t.Errorf("updated.Hours=%f, want=%f", got, want)
		}
		if got, want := updated.Notes, "Updated notes"; got != want {
			t.Errorf("updated.Notes=%s, want=%s", got, want)
		}
	})

	t.Run("given billable field when UpdateTimeEntry called then updates billable flag", func(t *testing.T) {
		entryID := 1001
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var reqData map[string]any
			if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
				t.Fatalf("failed to decode request body: %v", err)
			}

			if got, want := reqData["billable"], true; got != want {
				t.Errorf("billable=%v, want=%v", got, want)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"id":       1001,
				"billable": true,
				"client":   map[string]any{"id": 50, "name": "Test Client"},
				"project":  map[string]any{"id": 100, "name": "Test Project"},
				"task":     map[string]any{"id": 200, "name": "Development"},
			})
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		update := UpdateTimeEntryRequest{
			IsBillable: &[]bool{true}[0],
		}

		updated, err := client.UpdateTimeEntry(t.Context(), entryID, update)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if got, want := updated.IsBillable, true; got != want {
			t.Errorf("updated.IsBillable=%t, want=%t", got, want)
		}
	})

	t.Run("given locked time entry when UpdateTimeEntry called then returns error", func(t *testing.T) {
		entryID := 1001
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"message": "Time entry is locked and cannot be modified.",
			})
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		update := UpdateTimeEntryRequest{
			Hours: &[]float64{2.0}[0],
		}

		updated, err := client.UpdateTimeEntry(t.Context(), entryID, update)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if updated != nil {
			t.Errorf("expected nil time entry, got %v", updated)
		}

		if got, want := err.Error(), "400"; !strings.Contains(got, want) {
			t.Errorf("error=%q, want substring %q", got, want)
		}
	})

	t.Run("given nonexistent entry when UpdateTimeEntry called then returns not found error", func(t *testing.T) {
		entryID := 999999
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{
				"message": "Time entry not found.",
			})
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		update := UpdateTimeEntryRequest{
			Hours: &[]float64{2.0}[0],
		}

		updated, err := client.UpdateTimeEntry(t.Context(), entryID, update)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if updated != nil {
			t.Errorf("expected nil time entry, got %v", updated)
		}

		if got, want := err.Error(), "404"; !strings.Contains(got, want) {
			t.Errorf("error=%q, want substring %q", got, want)
		}
	})

	t.Run("given unauthorized request when UpdateTimeEntry called then returns auth error", func(t *testing.T) {
		entryID := 1001
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]any{
				"error": "invalid_token",
			})
		}))
		defer server.Close()

		client := NewClient("12345", "invalid-token")
		client.SetBaseURL(server.URL)

		update := UpdateTimeEntryRequest{
			Hours: &[]float64{2.0}[0],
		}

		updated, err := client.UpdateTimeEntry(t.Context(), entryID, update)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if updated != nil {
			t.Errorf("expected nil time entry, got %v", updated)
		}

		if got, want := err.Error(), "401"; !strings.Contains(got, want) {
			t.Errorf("error=%q, want substring %q", got, want)
		}
	})
}

func TestDeleteTimeEntry(t *testing.T) {
	t.Run("given existing time entry when DeleteTimeEntry called then deletes entry successfully", func(t *testing.T) {
		entryID := 1001
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got, want := r.URL.Path, fmt.Sprintf("/v2/time_entries/%d", entryID); got != want {
				t.Errorf("path=%s, want=%s", got, want)
			}
			if got, want := r.Method, http.MethodDelete; got != want {
				t.Errorf("method=%s, want=%s", got, want)
			}

			// Verify auth headers are present
			if got, want := r.Header.Get("Harvest-Account-Id"), "12345"; got != want {
				t.Errorf("Harvest-Account-Id header=%s, want=%s", got, want)
			}
			if got, want := r.Header.Get("Authorization"), "Bearer test-token"; got != want {
				t.Errorf("Authorization header=%s, want=%s", got, want)
			}

			// DELETE typically returns 200 OK with empty response body
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		err := client.DeleteTimeEntry(t.Context(), entryID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("given locked time entry when DeleteTimeEntry called then returns error", func(t *testing.T) {
		entryID := 1001
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"message": "Time entry is locked and cannot be deleted.",
			})
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		err := client.DeleteTimeEntry(t.Context(), entryID)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if got, want := err.Error(), "400"; !strings.Contains(got, want) {
			t.Errorf("error=%q, want substring %q", got, want)
		}
	})

	t.Run("given running time entry when DeleteTimeEntry called then returns error", func(t *testing.T) {
		entryID := 1001
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"message": "Cannot delete a running time entry.",
			})
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		err := client.DeleteTimeEntry(t.Context(), entryID)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if got, want := err.Error(), "400"; !strings.Contains(got, want) {
			t.Errorf("error=%q, want substring %q", got, want)
		}
	})

	t.Run("given nonexistent entry when DeleteTimeEntry called then returns not found error", func(t *testing.T) {
		entryID := 999999
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{
				"message": "Time entry not found.",
			})
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		err := client.DeleteTimeEntry(t.Context(), entryID)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if got, want := err.Error(), "404"; !strings.Contains(got, want) {
			t.Errorf("error=%q, want substring %q", got, want)
		}
	})

	t.Run("given unauthorized request when DeleteTimeEntry called then returns auth error", func(t *testing.T) {
		entryID := 1001
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]any{
				"error": "invalid_token",
			})
		}))
		defer server.Close()

		client := NewClient("12345", "invalid-token")
		client.SetBaseURL(server.URL)

		err := client.DeleteTimeEntry(t.Context(), entryID)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if got, want := err.Error(), "401"; !strings.Contains(got, want) {
			t.Errorf("error=%q, want substring %q", got, want)
		}
	})
}

func TestRestartTimeEntry(t *testing.T) {
	t.Run("given stopped time entry when RestartTimeEntry called then starts timer and returns updated entry", func(t *testing.T) {
		entryID := 1001
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got, want := r.URL.Path, fmt.Sprintf("/v2/time_entries/%d/restart", entryID); got != want {
				t.Errorf("path=%s, want=%s", got, want)
			}
			if got, want := r.Method, http.MethodPatch; got != want {
				t.Errorf("method=%s, want=%s", got, want)
			}

			// Verify auth headers are present
			if got, want := r.Header.Get("Harvest-Account-Id"), "12345"; got != want {
				t.Errorf("Harvest-Account-Id header=%s, want=%s", got, want)
			}
			if got, want := r.Header.Get("Authorization"), "Bearer test-token"; got != want {
				t.Errorf("Authorization header=%s, want=%s", got, want)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"id":         1001,
				"spent_date": "2025-01-15",
				"hours":      2.5,
				"notes":      "Development work",
				"is_running": true,
				"is_locked":  false,
				"billable":   true,
				"client": map[string]any{
					"id":   50,
					"name": "Test Client",
				},
				"project": map[string]any{
					"id":   100,
					"name": "Test Project",
				},
				"task": map[string]any{
					"id":   200,
					"name": "Development",
				},
			})
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		entry, err := client.RestartTimeEntry(t.Context(), entryID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if got, want := entry.ID, 1001; got != want {
			t.Errorf("entry.ID=%d, want=%d", got, want)
		}
		if got, want := entry.IsRunning, true; got != want {
			t.Errorf("entry.IsRunning=%t, want=%t", got, want)
		}
	})

	t.Run("given locked time entry when RestartTimeEntry called then returns error", func(t *testing.T) {
		entryID := 1001
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"message": "Time entry is locked and cannot be restarted.",
			})
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		entry, err := client.RestartTimeEntry(t.Context(), entryID)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if entry != nil {
			t.Errorf("expected nil time entry, got %v", entry)
		}

		if got, want := err.Error(), "400"; !strings.Contains(got, want) {
			t.Errorf("error=%q, want substring %q", got, want)
		}
	})

	t.Run("given nonexistent entry when RestartTimeEntry called then returns not found error", func(t *testing.T) {
		entryID := 999999
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{
				"message": "Time entry not found.",
			})
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		entry, err := client.RestartTimeEntry(t.Context(), entryID)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if entry != nil {
			t.Errorf("expected nil time entry, got %v", entry)
		}

		if got, want := err.Error(), "404"; !strings.Contains(got, want) {
			t.Errorf("error=%q, want substring %q", got, want)
		}
	})
}

func TestStopTimeEntry(t *testing.T) {
	t.Run("given running time entry when StopTimeEntry called then stops timer and returns updated entry", func(t *testing.T) {
		entryID := 1001
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got, want := r.URL.Path, fmt.Sprintf("/v2/time_entries/%d/stop", entryID); got != want {
				t.Errorf("path=%s, want=%s", got, want)
			}
			if got, want := r.Method, http.MethodPatch; got != want {
				t.Errorf("method=%s, want=%s", got, want)
			}

			// Verify auth headers are present
			if got, want := r.Header.Get("Harvest-Account-Id"), "12345"; got != want {
				t.Errorf("Harvest-Account-Id header=%s, want=%s", got, want)
			}
			if got, want := r.Header.Get("Authorization"), "Bearer test-token"; got != want {
				t.Errorf("Authorization header=%s, want=%s", got, want)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"id":         1001,
				"spent_date": "2025-01-15",
				"hours":      3.25,
				"notes":      "Development work",
				"is_running": false,
				"is_locked":  false,
				"billable":   true,
				"client": map[string]any{
					"id":   50,
					"name": "Test Client",
				},
				"project": map[string]any{
					"id":   100,
					"name": "Test Project",
				},
				"task": map[string]any{
					"id":   200,
					"name": "Development",
				},
			})
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		entry, err := client.StopTimeEntry(t.Context(), entryID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if got, want := entry.ID, 1001; got != want {
			t.Errorf("entry.ID=%d, want=%d", got, want)
		}
		if got, want := entry.IsRunning, false; got != want {
			t.Errorf("entry.IsRunning=%t, want=%t", got, want)
		}
		if got, want := entry.Hours, 3.25; got != want {
			t.Errorf("entry.Hours=%f, want=%f", got, want)
		}
	})

	t.Run("given already stopped time entry when StopTimeEntry called then returns error", func(t *testing.T) {
		entryID := 1001
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"message": "Time entry is not running.",
			})
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		entry, err := client.StopTimeEntry(t.Context(), entryID)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if entry != nil {
			t.Errorf("expected nil time entry, got %v", entry)
		}

		if got, want := err.Error(), "400"; !strings.Contains(got, want) {
			t.Errorf("error=%q, want substring %q", got, want)
		}
	})

	t.Run("given locked time entry when StopTimeEntry called then returns error", func(t *testing.T) {
		entryID := 1001
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"message": "Time entry is locked and cannot be stopped.",
			})
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		entry, err := client.StopTimeEntry(t.Context(), entryID)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if entry != nil {
			t.Errorf("expected nil time entry, got %v", entry)
		}

		if got, want := err.Error(), "400"; !strings.Contains(got, want) {
			t.Errorf("error=%q, want substring %q", got, want)
		}
	})

	t.Run("given nonexistent entry when StopTimeEntry called then returns not found error", func(t *testing.T) {
		entryID := 999999
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{
				"message": "Time entry not found.",
			})
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		entry, err := client.StopTimeEntry(t.Context(), entryID)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if entry != nil {
			t.Errorf("expected nil time entry, got %v", entry)
		}

		if got, want := err.Error(), "404"; !strings.Contains(got, want) {
			t.Errorf("error=%q, want substring %q", got, want)
		}
	})
}

func TestAggregateProjectsWithTasks(t *testing.T) {
	t.Run("given projects and task assignments when aggregated then returns projects with their tasks", func(t *testing.T) {
		projects := []Project{
			{ID: 1, Name: "API Development", Client: ProjectClient{ID: 100, Name: "Acme Corp"}},
			{ID: 2, Name: "Mobile App", Client: ProjectClient{ID: 100, Name: "Acme Corp"}},
			{ID: 3, Name: "Consulting", Client: ProjectClient{ID: 200, Name: "BigCo Industries"}},
		}
		taskAssignments := []TaskAssignment{
			{ID: 1, Project: TaskAssignmentProject{ID: 1, Name: "API Development"}, Task: TaskAssignmentTask{ID: 10, Name: "Code Review"}},
			{ID: 2, Project: TaskAssignmentProject{ID: 1, Name: "API Development"}, Task: TaskAssignmentTask{ID: 11, Name: "Development"}},
			{ID: 3, Project: TaskAssignmentProject{ID: 2, Name: "Mobile App"}, Task: TaskAssignmentTask{ID: 12, Name: "Testing"}},
			{ID: 4, Project: TaskAssignmentProject{ID: 3, Name: "Consulting"}, Task: TaskAssignmentTask{ID: 13, Name: "Meetings"}},
		}

		result := AggregateProjectsWithTasks(projects, taskAssignments)

		if got, want := len(result), 3; got != want {
			t.Fatalf("len(result)=%d, want=%d", got, want)
		}

		// Verify first project has 2 tasks
		found := false
		for _, pe := range result {
			if pe.Project.ID == 1 {
				found = true
				if got, want := len(pe.Tasks), 2; got != want {
					t.Errorf("project 1 task count=%d, want=%d", got, want)
				}
			}
		}
		if !found {
			t.Error("expected to find project 1 in results")
		}

		// Verify project 2 has 1 task
		found = false
		for _, pe := range result {
			if pe.Project.ID == 2 {
				found = true
				if got, want := len(pe.Tasks), 1; got != want {
					t.Errorf("project 2 task count=%d, want=%d", got, want)
				}
			}
		}
		if !found {
			t.Error("expected to find project 2 in results")
		}
	})

	t.Run("given projects and task assignments when aggregated then results are sorted by client then project", func(t *testing.T) {
		projects := []Project{
			{ID: 3, Name: "Zebra Project", Client: ProjectClient{ID: 200, Name: "BigCo Industries"}},
			{ID: 1, Name: "API Development", Client: ProjectClient{ID: 100, Name: "Acme Corp"}},
			{ID: 2, Name: "Mobile App", Client: ProjectClient{ID: 100, Name: "Acme Corp"}},
		}
		taskAssignments := []TaskAssignment{
			{ID: 1, Project: TaskAssignmentProject{ID: 1, Name: "API Development"}, Task: TaskAssignmentTask{ID: 10, Name: "Task"}},
			{ID: 2, Project: TaskAssignmentProject{ID: 2, Name: "Mobile App"}, Task: TaskAssignmentTask{ID: 11, Name: "Task"}},
			{ID: 3, Project: TaskAssignmentProject{ID: 3, Name: "Zebra Project"}, Task: TaskAssignmentTask{ID: 12, Name: "Task"}},
		}

		result := AggregateProjectsWithTasks(projects, taskAssignments)

		if got, want := len(result), 3; got != want {
			t.Fatalf("len(result)=%d, want=%d", got, want)
		}

		// First should be Acme Corp - API Development
		if got, want := result[0].Project.Client.Name, "Acme Corp"; got != want {
			t.Errorf("first entry client=%s, want=%s", got, want)
		}
		if got, want := result[0].Project.Name, "API Development"; got != want {
			t.Errorf("first entry project=%s, want=%s", got, want)
		}

		// Second should be Acme Corp - Mobile App
		if got, want := result[1].Project.Client.Name, "Acme Corp"; got != want {
			t.Errorf("second entry client=%s, want=%s", got, want)
		}
		if got, want := result[1].Project.Name, "Mobile App"; got != want {
			t.Errorf("second entry project=%s, want=%s", got, want)
		}

		// Third should be BigCo Industries - Zebra Project
		if got, want := result[2].Project.Client.Name, "BigCo Industries"; got != want {
			t.Errorf("third entry client=%s, want=%s", got, want)
		}
	})

	t.Run("given project with no task assignments when aggregated then project is excluded", func(t *testing.T) {
		projects := []Project{
			{ID: 1, Name: "API Development", Client: ProjectClient{ID: 100, Name: "Acme Corp"}},
			{ID: 2, Name: "Empty Project", Client: ProjectClient{ID: 100, Name: "Acme Corp"}},
		}
		taskAssignments := []TaskAssignment{
			{ID: 1, Project: TaskAssignmentProject{ID: 1, Name: "API Development"}, Task: TaskAssignmentTask{ID: 10, Name: "Task"}},
		}

		result := AggregateProjectsWithTasks(projects, taskAssignments)

		// Only project with tasks should be included
		if got, want := len(result), 1; got != want {
			t.Fatalf("len(result)=%d, want=%d", got, want)
		}
		if got, want := result[0].Project.ID, 1; got != want {
			t.Errorf("project ID=%d, want=%d", got, want)
		}
	})

	t.Run("given empty inputs when aggregated then returns empty slice", func(t *testing.T) {
		result := AggregateProjectsWithTasks([]Project{}, []TaskAssignment{})

		if got, want := len(result), 0; got != want {
			t.Errorf("len(result)=%d, want=%d", got, want)
		}
	})
}

func TestAPIErrorsIncludeResponseBody(t *testing.T) {
	t.Run("given JSON message body when CreateTimeEntry fails then error includes message", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(map[string]any{
				"message": "Notes can't be blank",
			})
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		_, err := client.CreateTimeEntry(t.Context(), CreateTimeEntryRequest{ProjectID: 1, TaskID: 2, SpentDate: "2026-06-12", Hours: 1.5})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if got, want := err.Error(), "failed to create time entry with status 422: Notes can't be blank"; got != want {
			t.Errorf("error=%q, want=%q", got, want)
		}
	})

	t.Run("given error_description body when ValidateAuth fails then error includes description", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]any{
				"error":             "invalid_token",
				"error_description": "The access token is invalid",
			})
		}))
		defer server.Close()

		client := NewClient("12345", "invalid-token")
		client.SetBaseURL(server.URL)

		_, err := client.ValidateAuth(t.Context())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if got, want := err.Error(), "authentication failed with status 401: The access token is invalid"; got != want {
			t.Errorf("error=%q, want=%q", got, want)
		}
	})

	t.Run("given non-JSON body when DeleteTimeEntry fails then error includes raw body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
			fmt.Fprint(w, "upstream timeout")
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		err := client.DeleteTimeEntry(t.Context(), 123)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if got, want := err.Error(), "failed to delete time entry with status 502: upstream timeout"; got != want {
			t.Errorf("error=%q, want=%q", got, want)
		}
	})

	t.Run("given empty body when StopTimeEntry fails then error only includes status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnprocessableEntity)
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		_, err := client.StopTimeEntry(t.Context(), 123)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if got, want := err.Error(), "failed to stop time entry with status 422"; got != want {
			t.Errorf("error=%q, want=%q", got, want)
		}
	})

	t.Run("given JSON message body when FetchProjects fails then error includes message", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]any{
				"message": "The object you requested was not found",
			})
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		_, err := client.FetchProjects(t.Context())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if got, want := err.Error(), "failed to fetch projects with status 403: The object you requested was not found"; got != want {
			t.Errorf("error=%q, want=%q", got, want)
		}
	})
}

func TestContextCancellation(t *testing.T) {
	t.Run("given a cancelled context when FetchProjects called then returns context error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		_, err := client.FetchProjects(ctx)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if got, want := err, context.Canceled; !errors.Is(got, want) {
			t.Errorf("error=%v, want %v", got, want)
		}
	})

	t.Run("given an expired context deadline when CreateTimeEntry called then returns deadline error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusCreated)
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
		defer cancel()

		_, err := client.CreateTimeEntry(ctx, CreateTimeEntryRequest{ProjectID: 1, TaskID: 2, SpentDate: "2026-06-12"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if got, want := err, context.DeadlineExceeded; !errors.Is(got, want) {
			t.Errorf("error=%v, want %v", got, want)
		}
	})
}

// failIfCalled returns a test server that fails the test if any request
// reaches it, for asserting that validation rejects requests client-side.
func failIfCalled(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request to %s %s; validation should have failed first", r.Method, r.URL.Path)
	}))
	t.Cleanup(server.Close)
	return server
}

func TestCreateTimeEntryValidation(t *testing.T) {
	valid := CreateTimeEntryRequest{ProjectID: 100, TaskID: 200, SpentDate: "2026-06-12", Hours: 1.5}

	cases := []struct {
		name    string
		mutate  func(*CreateTimeEntryRequest)
		wantErr string
	}{
		{"given a missing project ID then it is rejected", func(r *CreateTimeEntryRequest) { r.ProjectID = 0 }, "project_id is required"},
		{"given a missing task ID then it is rejected", func(r *CreateTimeEntryRequest) { r.TaskID = 0 }, "task_id is required"},
		{"given a missing spent date then it is rejected", func(r *CreateTimeEntryRequest) { r.SpentDate = "" }, "spent_date is required"},
		{"given a malformed spent date then it is rejected", func(r *CreateTimeEntryRequest) { r.SpentDate = "06/12/2026" }, "expected YYYY-MM-DD"},
		{"given negative hours then it is rejected", func(r *CreateTimeEntryRequest) { r.Hours = -1 }, "hours must not be negative"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := failIfCalled(t)
			client := NewClient("12345", "test-token")
			client.SetBaseURL(server.URL)

			req := valid
			tc.mutate(&req)

			_, err := client.CreateTimeEntry(t.Context(), req)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if got, want := err.Error(), tc.wantErr; !strings.Contains(got, want) {
				t.Errorf("error=%q, want substring %q", got, want)
			}
		})
	}
}

func TestUpdateTimeEntryValidation(t *testing.T) {
	intPtr := func(v int) *int { return &v }
	floatPtr := func(v float64) *float64 { return &v }
	strPtr := func(v string) *string { return &v }

	cases := []struct {
		name    string
		req     UpdateTimeEntryRequest
		wantErr string
	}{
		{"given a zero project ID then it is rejected", UpdateTimeEntryRequest{ProjectID: intPtr(0)}, "project_id must be positive"},
		{"given a zero task ID then it is rejected", UpdateTimeEntryRequest{TaskID: intPtr(0)}, "task_id must be positive"},
		{"given a malformed spent date then it is rejected", UpdateTimeEntryRequest{SpentDate: strPtr("June 12")}, "expected YYYY-MM-DD"},
		{"given negative hours then it is rejected", UpdateTimeEntryRequest{Hours: floatPtr(-0.5)}, "hours must not be negative"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := failIfCalled(t)
			client := NewClient("12345", "test-token")
			client.SetBaseURL(server.URL)

			_, err := client.UpdateTimeEntry(t.Context(), 1001, tc.req)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if got, want := err.Error(), tc.wantErr; !strings.Contains(got, want) {
				t.Errorf("error=%q, want substring %q", got, want)
			}
		})
	}
}

func TestFetchTeamTimeEntries(t *testing.T) {
	teamEntriesResponse := func(entries []map[string]any) map[string]any {
		return map[string]any{
			"time_entries":  entries,
			"per_page":      100,
			"total_pages":   1,
			"total_entries": len(entries),
			"page":          1,
		}
	}

	t.Run("given a project filter when FetchTeamTimeEntries called then requests project_id without user_id and decodes users", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got, want := r.URL.Path, "/v2/time_entries"; got != want {
				t.Errorf("path=%s, want=%s", got, want)
			}
			if got, want := r.URL.Query().Get("from"), "2026-06-01"; got != want {
				t.Errorf("from query param=%s, want=%s", got, want)
			}
			if got, want := r.URL.Query().Get("to"), "2026-06-30"; got != want {
				t.Errorf("to query param=%s, want=%s", got, want)
			}
			if got, want := r.URL.Query().Get("project_id"), "101"; got != want {
				t.Errorf("project_id query param=%s, want=%s", got, want)
			}
			if got, want := r.URL.Query().Has("user_id"), false; got != want {
				t.Errorf("user_id present=%t, want=%t", got, want)
			}
			if got, want := r.URL.Query().Has("client_id"), false; got != want {
				t.Errorf("client_id present=%t, want=%t", got, want)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(teamEntriesResponse([]map[string]any{
				{
					"id":         1,
					"spent_date": "2026-06-15",
					"hours":      1.5,
					"billable":   true,
					"user":       map[string]any{"id": 7, "name": "Alex Rivera"},
					"client":     map[string]any{"id": 11, "name": "Acme Corp"},
					"project":    map[string]any{"id": 101, "name": "Website Redesign"},
					"task":       map[string]any{"id": 202, "name": "Development"},
				},
			}))
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)
		client.SetUserID(123)

		entries, err := client.FetchTeamTimeEntries(t.Context(), "2026-06-01", "2026-06-30", TeamTimeEntriesFilter{ProjectID: 101})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got, want := len(entries), 1; got != want {
			t.Fatalf("len(entries)=%d, want=%d", got, want)
		}
		if got, want := entries[0].User.ID, 7; got != want {
			t.Errorf("user ID=%d, want=%d", got, want)
		}
		if got, want := entries[0].User.Name, "Alex Rivera"; got != want {
			t.Errorf("user name=%s, want=%s", got, want)
		}
	})

	t.Run("given billable rates when FetchTeamTimeEntries called then decodes them treating null as zero", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(teamEntriesResponse([]map[string]any{
				{
					"id": 1, "spent_date": "2026-06-15", "hours": 2.0, "billable": true,
					"billable_rate": 160.0,
					"user":          map[string]any{"id": 7, "name": "Alex Rivera"},
				},
				{
					"id": 2, "spent_date": "2026-06-15", "hours": 1.0, "billable": false,
					"billable_rate": nil,
					"user":          map[string]any{"id": 7, "name": "Alex Rivera"},
				},
			}))
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		entries, err := client.FetchTeamTimeEntries(t.Context(), "2026-06-01", "2026-06-30", TeamTimeEntriesFilter{ProjectID: 101})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got, want := len(entries), 2; got != want {
			t.Fatalf("len(entries)=%d, want=%d", got, want)
		}
		if got, want := entries[0].BillableRate, 160.0; got != want {
			t.Errorf("billable rate=%f, want=%f", got, want)
		}
		if got, want := entries[1].BillableRate, 0.0; got != want {
			t.Errorf("null billable rate=%f, want=%f", got, want)
		}
	})

	t.Run("given a client filter when FetchTeamTimeEntries called then requests client_id without user_id", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got, want := r.URL.Query().Get("client_id"), "11"; got != want {
				t.Errorf("client_id query param=%s, want=%s", got, want)
			}
			if got, want := r.URL.Query().Has("project_id"), false; got != want {
				t.Errorf("project_id present=%t, want=%t", got, want)
			}
			if got, want := r.URL.Query().Has("user_id"), false; got != want {
				t.Errorf("user_id present=%t, want=%t", got, want)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(teamEntriesResponse(nil))
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		entries, err := client.FetchTeamTimeEntries(t.Context(), "2026-06-01", "2026-06-30", TeamTimeEntriesFilter{ClientID: 11})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got, want := len(entries), 0; got != want {
			t.Errorf("len(entries)=%d, want=%d", got, want)
		}
	})

	t.Run("given a paginated response when FetchTeamTimeEntries called then fetches all pages", func(t *testing.T) {
		requestCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if r.URL.Query().Get("page") == "2" {
				json.NewEncoder(w).Encode(teamEntriesResponse([]map[string]any{
					{"id": 2, "spent_date": "2026-06-16", "hours": 2.0, "user": map[string]any{"id": 8, "name": "Sam Chen"}},
				}))
				return
			}
			json.NewEncoder(w).Encode(map[string]any{
				"time_entries": []map[string]any{
					{"id": 1, "spent_date": "2026-06-15", "hours": 1.0, "user": map[string]any{"id": 7, "name": "Alex Rivera"}},
				},
				"per_page":      1,
				"total_pages":   2,
				"total_entries": 2,
				"page":          1,
				"next_page":     2,
			})
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		entries, err := client.FetchTeamTimeEntries(t.Context(), "2026-06-01", "2026-06-30", TeamTimeEntriesFilter{ProjectID: 101})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got, want := len(entries), 2; got != want {
			t.Errorf("len(entries)=%d, want=%d", got, want)
		}
		if got, want := requestCount, 2; got != want {
			t.Errorf("request count=%d, want=%d", got, want)
		}
	})

	t.Run("given an error response when FetchTeamTimeEntries called then surfaces the API message", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"message": "insufficient permissions"})
		}))
		defer server.Close()

		client := NewClient("12345", "test-token")
		client.SetBaseURL(server.URL)

		_, err := client.FetchTeamTimeEntries(t.Context(), "2026-06-01", "2026-06-30", TeamTimeEntriesFilter{ProjectID: 101})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if got, want := err.Error(), "insufficient permissions"; !strings.Contains(got, want) {
			t.Errorf("error=%q, want substring %q", got, want)
		}
	})
}
