// Package buildinfo reports the version of the running harvest binary.
//
// Release builds stamp Version, Commit, and Date via -ldflags (goreleaser
// does this). Builds made with `go install` or `go build` leave them empty,
// so the values fall back to the module version and VCS metadata that the Go
// toolchain embeds in the binary.
package buildinfo

import (
	"runtime/debug"
	"strings"
)

// Stamps set at link time by goreleaser:
//
//	-ldflags "-X github.com/jc00ke/harvest/internal/buildinfo.Version=..."
var (
	Version string
	Commit  string
	Date    string
)

// devVersion is reported when neither ldflags nor the embedded build info
// name a version, e.g. `go build` in a checkout with no tags.
const devVersion = "dev"

// String returns a one-line human-readable version, e.g.
// "harvest 1.2.3 (abcdef1, 2026-08-28T12:00:00Z)". The commit and date are
// omitted when unknown.
func String() string {
	info, _ := debug.ReadBuildInfo()
	version, commit, date := resolve(info)

	s := "harvest " + version
	details := make([]string, 0, 2)
	if commit != "" {
		details = append(details, commit)
	}
	if date != "" {
		details = append(details, date)
	}
	if len(details) > 0 {
		s += " (" + strings.Join(details, ", ") + ")"
	}
	return s
}

// resolve picks the version, commit, and date to report, preferring the
// ldflags stamps and falling back to info. info may be nil, which the
// toolchain reports when the binary carries no build information.
func resolve(info *debug.BuildInfo) (version, commit, date string) {
	version, commit, date = Version, Commit, Date

	if info != nil {
		if version == "" && isRelease(info.Main.Version) {
			version = info.Main.Version
		}
		if commit == "" {
			commit = vcsCommit(info)
		}
		if date == "" {
			date = setting(info, "vcs.time")
		}
	}

	if version == "" {
		version = devVersion
	}
	return version, commit, date
}

// isRelease reports whether a module version names an actual tagged release.
// It rejects "(devel)" (an unversioned build) and Go pseudo-versions, whose
// embedded commit and timestamp would duplicate the VCS details.
func isRelease(version string) bool {
	if version == "" || version == "(devel)" {
		return false
	}
	// Pseudo-versions look like v0.5.2-0.20260713144244-916d049b8522, and
	// carry a "+dirty" build suffix when built from a modified tree.
	if plus := strings.Index(version, "+"); plus != -1 {
		version = version[:plus]
	}
	dash := strings.LastIndex(version, "-")
	return dash == -1 || !isHex(version[dash+1:])
}

// isHex reports whether s is a non-empty lowercase hex string, as in the
// commit suffix of a pseudo-version.
func isHex(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

// vcsCommit returns the revision the binary was built from, marked "-dirty"
// when the working tree had uncommitted changes.
func vcsCommit(info *debug.BuildInfo) string {
	commit := setting(info, "vcs.revision")
	if commit != "" && setting(info, "vcs.modified") == "true" {
		commit += "-dirty"
	}
	return commit
}

// setting returns the value of a build setting, or "" if it is absent.
func setting(info *debug.BuildInfo, key string) string {
	for _, s := range info.Settings {
		if s.Key == key {
			return s.Value
		}
	}
	return ""
}
