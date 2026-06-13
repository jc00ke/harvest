# Claude Code Instructions

## Before Every Commit

Run `mise run check` and ensure it passes.
Prefer the mise tasks over direct commands. For instance

```bash
# prefer
mise run lint
# over
go vet ./...
```

## Commit Message Format

Use conventional commits:

- `feat(scope): description` for new features
- `test(scope): description` for tests
- `fix(scope): description` for bug fixes
- `docs: description` for documentation
- `chore: description` for maintenance tasks

When implementing Harvest API calls, include the API reference URL in the commit body.
Write failing tests before implementation.

## Code Conventions

- `harvest.Client` API methods take `ctx context.Context` as the first
  parameter. CLI commands pass `cmd.Context()`; TUI commands pass
  `context.Background()`.
- Build non-2xx API errors with `apiError` (internal/harvest/client.go) so
  the response body's message is surfaced.
- In internal/tui, put code in the file matching its concern: key handlers
  in handlers.go, per-view renderers in views.go, messages and async
  commands in commands.go, list items in items.go.

## Test Style

Write tests in the `if got, want` style.
Reference this blog post if needed: [https://mtlynch.io/if-got-want-improve-go-tests/].

```go
// do this
if got, want := GetUser(), "dummyUser"; got != want {
  t.Errorf("username=%s, want=%s", got, want)
}

// not this
username := GetUser()
if username != "dummyUser" {
  t.Errorf("unexpected username: got %s, want: %s", username, "dummyUser")
}
```

Name subtests in the form "given X when Y then Z". For substring checks use
`if got, want := err.Error(), "..."; !strings.Contains(got, want)`. Plain
`if err != nil { t.Fatalf(...) }` error checks stay as-is.

## Testing Conventions

- Never touch the real OS keyring in tests: call `keyring.MockInit()` first.
- CLI command tests run end-to-end against internal/demo's server via the
  `newAPIClient` seam — reuse `setupDemoCLI` in internal/cli/cli_test.go.
- Table output is pinned by golden files; after an intentional format
  change, regenerate with `go test ./internal/cli -update`.

## CI

- Pin GitHub Actions to commit SHAs with the version in a trailing comment;
  Dependabot keeps the pins current.
