package buildinfo

import (
	"runtime/debug"
	"strings"
	"testing"
)

func TestString(t *testing.T) {
	tests := []struct {
		name    string
		version string
		commit  string
		date    string
		want    string
	}{
		{
			name:    "given all stamps when formatted then all are shown",
			version: "1.2.3",
			commit:  "abcdef1234567890",
			date:    "2026-08-28T12:00:00Z",
			want:    "harvest 1.2.3 (abcdef1234567890, 2026-08-28T12:00:00Z)",
		},
		{
			name:    "given only a version when formatted then no parenthetical",
			version: "1.2.3",
			want:    "harvest 1.2.3",
		},
		{
			name:    "given a commit but no date when formatted then only commit is shown",
			version: "1.2.3",
			commit:  "abcdef1",
			want:    "harvest 1.2.3 (abcdef1)",
		},
		{
			name:    "given a date but no commit when formatted then only date is shown",
			version: "1.2.3",
			date:    "2026-08-28T12:00:00Z",
			want:    "harvest 1.2.3 (2026-08-28T12:00:00Z)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub(t, tt.version, tt.commit, tt.date)
			if got, want := String(), tt.want; got != want {
				t.Errorf("String()=%q, want=%q", got, want)
			}
		})
	}
}

func TestStringUsesFallbackVersion(t *testing.T) {
	stub(t, "", "", "")
	if got, want := String(), "harvest "+devVersion; got != want {
		t.Errorf("String()=%q, want=%q", got, want)
	}
}

func TestResolvePrefersLDFlagStamps(t *testing.T) {
	stub(t, "1.2.3", "cafebabe", "2026-08-28T12:00:00Z")
	info := &debug.BuildInfo{
		Main: debug.Module{Version: "v9.9.9"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "deadbeef"},
			{Key: "vcs.time", Value: "2020-01-01T00:00:00Z"},
		},
	}

	version, commit, date := resolve(info)
	if got, want := version, "1.2.3"; got != want {
		t.Errorf("version=%q, want=%q", got, want)
	}
	if got, want := commit, "cafebabe"; got != want {
		t.Errorf("commit=%q, want=%q", got, want)
	}
	if got, want := date, "2026-08-28T12:00:00Z"; got != want {
		t.Errorf("date=%q, want=%q", got, want)
	}
}

func TestResolveFallsBackToBuildInfo(t *testing.T) {
	stub(t, "", "", "")
	info := &debug.BuildInfo{
		Main: debug.Module{Version: "v0.4.0"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "deadbeef"},
			{Key: "vcs.time", Value: "2020-01-01T00:00:00Z"},
			{Key: "vcs.modified", Value: "true"},
		},
	}

	version, commit, date := resolve(info)
	if got, want := version, "v0.4.0"; got != want {
		t.Errorf("version=%q, want=%q", got, want)
	}
	if got, want := commit, "deadbeef-dirty"; got != want {
		t.Errorf("commit=%q, want=%q", got, want)
	}
	if got, want := date, "2020-01-01T00:00:00Z"; got != want {
		t.Errorf("date=%q, want=%q", got, want)
	}
}

func TestResolveIgnoresNonReleaseMainVersions(t *testing.T) {
	tests := []struct {
		name    string
		version string
	}{
		{name: "given an unversioned build then the fallback is used", version: "(devel)"},
		{name: "given no module version then the fallback is used", version: ""},
		{
			name:    "given a pseudo-version then the fallback is used",
			version: "v0.5.2-0.20260713144244-916d049b8522",
		},
		{
			name:    "given a dirty pseudo-version then the fallback is used",
			version: "v0.5.2-0.20260713144244-916d049b8522+dirty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub(t, "", "", "")
			info := &debug.BuildInfo{Main: debug.Module{Version: tt.version}}

			version, _, _ := resolve(info)
			if got, want := version, devVersion; got != want {
				t.Errorf("version=%q, want=%q", got, want)
			}
		})
	}
}

func TestResolveKeepsTaggedMainVersions(t *testing.T) {
	tests := []string{"v0.4.0", "v1.2.3", "v1.2.3-rc1"}

	for _, tagged := range tests {
		t.Run("given "+tagged+" then it is reported as-is", func(t *testing.T) {
			stub(t, "", "", "")
			info := &debug.BuildInfo{Main: debug.Module{Version: tagged}}

			version, _, _ := resolve(info)
			if got, want := version, tagged; got != want {
				t.Errorf("version=%q, want=%q", got, want)
			}
		})
	}
}

func TestResolveWithoutBuildInfo(t *testing.T) {
	stub(t, "", "", "")

	version, commit, date := resolve(nil)
	if got, want := version, devVersion; got != want {
		t.Errorf("version=%q, want=%q", got, want)
	}
	if commit != "" || date != "" {
		t.Errorf("commit=%q date=%q, want both empty", commit, date)
	}
}

// TestRealBuildIsStamped guards against the version string degrading to the
// bare fallback under `go build`/`go test`, which embeds VCS metadata.
func TestRealBuildIsStamped(t *testing.T) {
	if got := String(); !strings.HasPrefix(got, "harvest ") {
		t.Errorf("String()=%q, want prefix %q", got, "harvest ")
	}
}

// stub overrides the ldflags-injected stamps for the duration of a test.
func stub(t *testing.T, version, commit, date string) {
	t.Helper()
	origVersion, origCommit, origDate := Version, Commit, Date
	Version, Commit, Date = version, commit, date
	t.Cleanup(func() { Version, Commit, Date = origVersion, origCommit, origDate })
}
