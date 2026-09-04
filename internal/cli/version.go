package cli

import (
	"runtime/debug"
)

// version is stamped at build time with
// -ldflags "-X github.com/qrotux/gridraw-cli/internal/cli.version=v1.2.3".
// Left empty, the build's own metadata answers instead.
var version string

// shortRevision is how much of a commit hash the version line carries.
const shortRevision = 12

// versionString reports the build: the stamped version, or what the toolchain
// recorded — a module version for `go install`, the commit for a build from a
// checkout, and nothing useful for a build stripped of both.
func versionString() string {
	if version != "" {
		return version
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	// A module version, pseudo-version included, already carries the commit and
	// the dirty marker, so only the devel fallback needs them appended.
	if v := info.Main.Version; v != "" && v != "(devel)" {
		return v
	}
	var revision string
	var dirty bool
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
			if len(revision) > shortRevision {
				revision = revision[:shortRevision]
			}
		case "vcs.modified":
			dirty = setting.Value == "true"
		}
	}
	if revision == "" {
		return "devel"
	}
	if dirty {
		revision += ", dirty"
	}
	return "devel (" + revision + ")"
}
