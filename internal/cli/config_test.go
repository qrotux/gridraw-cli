package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qrotux/gridraw-cli/internal/config"
)

// runConfig runs `gridraw config` in an empty working directory with its own
// user config path, and returns what it wrote to stderr.
func runConfig(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "xdg"))
	cmd := newConfigCmd()
	var stderr bytes.Buffer
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetOut(&stderr)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stderr.String(), err
}

func TestConfigWithHostAndAuthAsksNothing(t *testing.T) {
	stderr, err := runConfig(t, "", "--host=http://h/api/grids", "--bearer-token=t")
	if err != nil {
		t.Fatalf("config: %v", err)
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
	cmd := newConfigCmd()
	var stderr bytes.Buffer
	// Every answer is an empty line: the stored profile must survive intact.
	cmd.SetIn(strings.NewReader(strings.Repeat("\n", 8)))
	cmd.SetOut(&stderr)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--name=p"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("config: %v\n%s", err, stderr.String())
	}
	cfg, err := config.LoadFile(config.LocalFileName)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	want := config.Profile{Host: "http://kept/api/grids", Header: "Basic dXNlcjpwYXNz", DefaultInfoOutput: "json", DefaultDataOutput: "tsv"}
	if cfg.Profiles["p"] != want {
		t.Errorf("profile = %+v, want %+v", cfg.Profiles["p"], want)
	}
	if strings.Contains(stderr.String(), "dXNlcjpwYXNz") {
		t.Errorf("the credential was printed in full: %q", stderr.String())
	}
}

func TestConfigWithoutFlagsOrInputIsAUsageError(t *testing.T) {
	_, err := runConfig(t, "")
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
	cmd := newConfigCmd()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"list"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("list: %v", err)
	}
	want := filepath.Join(dir, config.LocalFileName)
	if !strings.Contains(stdout.String(), want) {
		t.Errorf("list = %q, want it to name %s", stdout.String(), want)
	}
}
