package cli

import (
	"strings"
	"testing"

	"github.com/jc00ke/harvest/internal/buildinfo"
)

func TestVersionFlag(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "given the long flag when run then the version prints", args: []string{"--version"}},
		{name: "given the short flag when run then the version prints", args: []string{"-v"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, err := runCLI(t, tt.args...)
			if err != nil {
				t.Fatalf("harvest %v: %v", tt.args, err)
			}
			if got, want := stdout, buildinfo.String()+"\n"; got != want {
				t.Errorf("stdout=%q, want=%q", got, want)
			}
		})
	}
}

// The version flag reports without contacting the API, so it must work with
// no credentials in the keyring.
func TestVersionFlagNeedsNoCredentials(t *testing.T) {
	stdout, err := runCLI(t, "--version")
	if err != nil {
		t.Fatalf("harvest --version: %v", err)
	}
	if got, want := stdout, "harvest "; !strings.HasPrefix(got, want) {
		t.Errorf("stdout=%q, want prefix=%q", got, want)
	}
}
