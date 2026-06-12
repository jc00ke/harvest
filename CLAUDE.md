# Claude Code Instructions

## Before Every Commit

Run `make check` and ensure it passes.

## Commit Message Format

Use conventional commits:

- `feat(scope): description` for new features
- `test(scope): description` for tests
- `fix(scope): description` for bug fixes
- `docs: description` for documentation
- `chore: description` for maintenance tasks

When implementing Harvest API calls, include the API reference URL in the commit body.
Write failing tests before implementation.

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
