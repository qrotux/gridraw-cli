package cli

import (
	"regexp"
	"strings"
	"testing"
)

func TestVersionStringPrefersTheStamp(t *testing.T) {
	old := version
	t.Cleanup(func() { version = old })
	version = "v1.2.3"
	if got := versionString(); got != "v1.2.3" {
		t.Errorf("versionString() = %q, want the stamped value", got)
	}
}

// TestVersionStringFallsBackToBuildInfo pins the unstamped path, which is what
// `go install` and a plain `go build` produce.
func TestVersionStringFallsBackToBuildInfo(t *testing.T) {
	old := version
	t.Cleanup(func() { version = old })
	version = ""
	got := versionString()
	if got == "" {
		t.Fatal("versionString() is empty")
	}
	// Under `go test` the main module has no version and no vcs stamp, so the
	// answer is the devel fallback; a released binary reports its module tag.
	if !regexp.MustCompile(`^(devel|unknown|v[0-9]|\(devel\))`).MatchString(got) {
		t.Errorf("versionString() = %q, want a version or the devel fallback", got)
	}
	if strings.Contains(got, "(") && !strings.Contains(got, ")") {
		t.Errorf("versionString() = %q, unbalanced revision suffix", got)
	}
}
