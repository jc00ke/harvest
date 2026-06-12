package cli

import (
	"bytes"
	"strings"
	"testing"
)

// executeCompletion runs `harvest completion <args>` against a fresh command
// tree and returns the generated script.
func executeCompletion(t *testing.T, args ...string) string {
	t.Helper()
	root := NewRootCommand()
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stdout)
	root.SetArgs(append([]string{"completion"}, args...))
	if err := root.Execute(); err != nil {
		t.Fatalf("completion %v: %v", args, err)
	}
	return stdout.String()
}

func TestCompletionFish(t *testing.T) {
	script := executeCompletion(t, "fish")
	if got, want := strings.Contains(script, "complete -c harvest"), true; got != want {
		t.Errorf("script registers fish completions for harvest=%v, want=%v", got, want)
	}
	if got, want := strings.Contains(script, "__complete"), true; got != want {
		t.Errorf("script calls back into harvest __complete=%v, want=%v", got, want)
	}
}

func TestCompletionBash(t *testing.T) {
	script := executeCompletion(t, "bash")
	if got, want := strings.Contains(script, "__start_harvest"), true; got != want {
		t.Errorf("script defines bash entry point __start_harvest=%v, want=%v", got, want)
	}
	if got, want := strings.Contains(script, "__complete"), true; got != want {
		t.Errorf("script calls back into harvest __complete=%v, want=%v", got, want)
	}
}

func TestCompletionZsh(t *testing.T) {
	script := executeCompletion(t, "zsh")
	if got, want := strings.HasPrefix(script, "#compdef harvest"), true; got != want {
		t.Errorf("script starts with #compdef harvest=%v, want=%v", got, want)
	}
	if got, want := strings.Contains(script, "__complete"), true; got != want {
		t.Errorf("script calls back into harvest __complete=%v, want=%v", got, want)
	}
}

func TestCompletionUnknownShellShowsUsage(t *testing.T) {
	out := executeCompletion(t, "elvish")
	if got, want := strings.Contains(out, "Available Commands"), true; got != want {
		t.Errorf("unknown shell falls back to usage help=%v, want=%v", got, want)
	}
}
