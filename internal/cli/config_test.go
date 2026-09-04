package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qrotux/gridraw-cli/internal/config"
)

// runConfig runs the config command tree in an empty working directory with
// its own user config path, and returns stdout and stderr separately: the
// questions and the confirmations belong on stderr, the listings on stdout.
func runConfig(t *testing.T, stdin string, args ...string) (string, string, error) {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "xdg"))
	return runConfigHere(t, stdin, args...)
}

// runConfigHere is runConfig without the fresh directory, for a test that has
// written a configuration file first.
func runConfigHere(t *testing.T, stdin string, args ...string) (string, string, error) {
	t.Helper()
	cmd := newConfigCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

func TestConfigWithHostAndAuthAsksNothing(t *testing.T) {
	stdout, stderr, err := runConfig(t, "", "--host=http://h/api/grids", "--bearer-token=t")
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want nothing: the confirmation belongs on stderr", stdout)
	}
	// A question is printed without a trailing newline, so any question would
	// show up before the confirmation on the first line.
	if lines := strings.Split(strings.TrimSpace(stderr), "\n"); len(lines) != 1 || !strings.HasPrefix(lines[0], "Wrote profile") {
		t.Errorf("a fully flagged run printed %q, want only the confirmation", stderr)
	}
	cfg, err := config.LoadFile(config.LocalFileName)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	p, ok := cfg.Profiles["default"]
	if !ok {
		t.Fatalf("profiles = %v, want a profile named default", cfg.Names())
	}
	if p.Host != "http://h/api/grids" || p.Header != "Bearer t" {
		t.Errorf("profile = %+v", p)
	}
	if p.InfoOutput() != "yaml" || p.DataOutput() != "csv" {
		t.Errorf("outputs = %s/%s, want yaml/csv", p.InfoOutput(), p.DataOutput())
	}
}

func TestConfigEditKeepsTheStoredValues(t *testing.T) {
	stored := "current: p\nconfigs:\n  p:\n    host: http://kept/api/grids\n    header: Basic dXNlcjpwYXNz\n    defaultInfoOutput: json\n    defaultDataOutput: tsv\n"
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "xdg"))
	if err := os.WriteFile(config.LocalFileName, []byte(stored), 0o600); err != nil {
		t.Fatal(err)
	}
	// Every answer is an empty line: the stored profile must survive intact.
	stdout, stderr, err := runConfigHere(t, strings.Repeat("\n", 8), "--name=p")
	if err != nil {
		t.Fatalf("config: %v\n%s", err, stderr)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want nothing: the questions belong on stderr", stdout)
	}
	cfg, err2 := config.LoadFile(config.LocalFileName)
	if err2 != nil {
		t.Fatalf("load: %v", err2)
	}
	want := config.Profile{Host: "http://kept/api/grids", Header: "Basic dXNlcjpwYXNz", DefaultInfoOutput: "json", DefaultDataOutput: "tsv"}
	if cfg.Profiles["p"] != want {
		t.Errorf("profile = %+v, want %+v", cfg.Profiles["p"], want)
	}
	if strings.Contains(stderr, "dXNlcjpwYXNz") {
		t.Errorf("the credential was printed in full: %q", stderr)
	}
}

func TestConfigWithoutFlagsOrInputIsAUsageError(t *testing.T) {
	_, _, err := runConfig(t, "")
	if err == nil {
		t.Fatal("want an error")
	}
	if got := ExitCode(err); got != 2 {
		t.Errorf("ExitCode = %d, want 2: %v", got, err)
	}
}

func TestConfigListNamesTheSourceFile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "xdg"))
	if err := os.WriteFile(config.LocalFileName, []byte("current: p\nconfigs:\n  p:\n    host: http://h/api/grids\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, err := runConfigHere(t, "", "list")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	want := filepath.Join(dir, config.LocalFileName)
	if !strings.Contains(stdout, want) {
		t.Errorf("list = %q, want it to name %s", stdout, want)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want nothing: the listing is data", stderr)
	}
}

// config use rewrites current in the nearest file that was read and leaves the
// profiles alone; the confirmation is stderr, not data.
func TestConfigUseSelectsTheProfile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "xdg"))
	stored := "current: a\nconfigs:\n  a:\n    host: http://a/g\n  b:\n    host: http://b/g\n"
	if err := os.WriteFile(config.LocalFileName, []byte(stored), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, err := runConfigHere(t, "", "use", "b")
	if err != nil {
		t.Fatalf("use: %v", err)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want nothing: the confirmation belongs on stderr", stdout)
	}
	if !strings.Contains(stderr, `"b"`) {
		t.Errorf("stderr = %q, want it to name the selected profile", stderr)
	}
	cfg, err := config.LoadFile(config.LocalFileName)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Current != "b" || len(cfg.Profiles) != 2 || cfg.Profiles["a"].Host != "http://a/g" {
		t.Errorf("config = %+v, want current b and both profiles intact", cfg)
	}
}

func TestConfigUseRejectsAnUnknownProfile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "xdg"))
	if err := os.WriteFile(config.LocalFileName, []byte("current: a\nconfigs:\n  a:\n    host: http://a/g\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err := runConfigHere(t, "", "use", "nope")
	if err == nil {
		t.Fatal("want an error")
	}
	if got := ExitCode(err); got != 3 {
		t.Errorf("ExitCode = %d, want 3: %v", got, err)
	}
}
