package cli

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRootRoutesPersistentFlagsToASubcommand runs the real command tree, which
// the per-command tests bypass: nothing else pins that --config-file declared
// on the root reaches `list`.
func TestRootRoutesPersistentFlagsToASubcommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"name":"users"}]`))
	}))
	defer srv.Close()

	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("HOME", dir)
	conf := filepath.Join(dir, "elsewhere.yaml")
	body := "current: named\nconfigs:\n  named:\n    host: " + srv.URL + "/api/grids\n"
	if err := os.WriteFile(conf, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	root := newRootCmd()
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"list", "--config-file", conf, "-o", "json"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("list through the root command: %v", err)
	}
	if !strings.Contains(stdout.String(), `"name":"users"`) {
		t.Errorf("stdout = %q, want the grid list", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want nothing", stderr.String())
	}
}

func TestExitCodeForANonClientNonServerStatus(t *testing.T) {
	// A 3xx the client did not follow is neither the request's fault nor a
	// server failure, so it must not claim to be a 4xx.
	if got := ExitCode(&fakeHTTP{status: 302}); got != 1 {
		t.Errorf("ExitCode(302) = %d, want 1", got)
	}
	if got := ExitCode(&fakeHTTP{status: 404}); got != 4 {
		t.Errorf("ExitCode(404) = %d, want 4", got)
	}
}

// TestRootReportsItsVersion pins the flag itself: versionString has unit tests,
// but nothing else proves the root command publishes it.
func TestRootReportsItsVersion(t *testing.T) {
	old := version
	t.Cleanup(func() { version = old })
	version = "v9.9.9"

	var stdout bytes.Buffer
	root := newRootCmd()
	root.SetOut(&stdout)
	root.SetErr(&stdout)
	root.SetArgs([]string{"--version"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("--version: %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, "v9.9.9") {
		t.Errorf("--version printed %q, want the stamped version", got)
	}
}

// TestRootRejectsAnUnknownFlagAsAUsageError pins the exit code a typo gets on a
// command that does not parse its own flags.
func TestRootRejectsAnUnknownFlagAsAUsageError(t *testing.T) {
	var out bytes.Buffer
	root := newRootCmd()
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"list", "--nosuch"})
	err := root.ExecuteContext(context.Background())
	if err == nil || ExitCode(err) != 2 {
		t.Fatalf("error = %v (exit %d), want a usage error", err, ExitCode(err))
	}
}
