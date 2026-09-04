package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestMergeLocalWinsWholeProfile(t *testing.T) {
	dir := t.TempDir()
	user := filepath.Join(dir, "user.yaml")
	local := filepath.Join(dir, "local.yaml")
	write(t, user, `
current: prod
configs:
  prod:
    host: https://prod/api/grids
    header: "Bearer u"
    defaultDataOutput: json
  staging:
    host: https://staging/api/grids
`)
	write(t, local, `
configs:
  prod:
    host: http://localhost:8080/api/grids
`)
	cfg, err := Merge(user, local)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Profiles["prod"].Host; got != "http://localhost:8080/api/grids" {
		t.Errorf("host = %q, want the local one", got)
	}
	if got := cfg.Profiles["prod"].Header; got != "" {
		t.Errorf("header = %q, want empty: the local profile replaces the user one whole", got)
	}
	if _, ok := cfg.Profiles["staging"]; !ok {
		t.Error("staging lost: profiles unique to the user file must survive")
	}
	if cfg.Current != "prod" {
		t.Errorf("current = %q, want prod from the user file", cfg.Current)
	}
}

func TestCurrentPrefersLocal(t *testing.T) {
	dir := t.TempDir()
	user, local := filepath.Join(dir, "u.yaml"), filepath.Join(dir, "l.yaml")
	write(t, user, "current: prod\nconfigs:\n  prod:\n    host: https://prod/g\n")
	write(t, local, "current: dev\nconfigs:\n  dev:\n    host: http://localhost/g\n")
	cfg, err := Merge(user, local)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Current != "dev" {
		t.Errorf("current = %q, want dev", cfg.Current)
	}
}

func TestCurrentDefaultsToDefault(t *testing.T) {
	dir := t.TempDir()
	local := filepath.Join(dir, "l.yaml")
	write(t, local, "configs:\n  default:\n    host: http://localhost/g\n")
	cfg, err := Merge(filepath.Join(dir, "missing.yaml"), local)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Current != "default" {
		t.Errorf("current = %q, want default", cfg.Current)
	}
}

func TestExpandEnv(t *testing.T) {
	dir := t.TempDir()
	local := filepath.Join(dir, "l.yaml")
	write(t, local, "configs:\n  default:\n    host: http://localhost/g\n    header: \"Bearer ${GRIDRAW_TEST_TOKEN}\"\n")
	t.Setenv("GRIDRAW_TEST_TOKEN", "s3cret")
	cfg, err := Merge("", local)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Profiles["default"].Header; got != "Bearer s3cret" {
		t.Errorf("header = %q, want the expanded token", got)
	}
}

func TestExpandEnvUnsetIsAnError(t *testing.T) {
	dir := t.TempDir()
	local := filepath.Join(dir, "l.yaml")
	write(t, local, "configs:\n  default:\n    host: http://localhost/g\n    header: \"Bearer ${GRIDRAW_ABSENT_VAR}\"\n")
	_, err := Merge("", local)
	if err == nil {
		t.Fatal("want an error for an unset variable")
	}
	var cerr *Error
	if !errors.As(err, &cerr) {
		t.Fatalf("error type = %T, want *config.Error", err)
	}
	if !strings.Contains(cerr.Msg, "GRIDRAW_ABSENT_VAR") {
		t.Errorf("message %q does not name the variable", cerr.Msg)
	}
}

func TestBadFormatRejected(t *testing.T) {
	dir := t.TempDir()
	local := filepath.Join(dir, "l.yaml")
	write(t, local, "configs:\n  default:\n    host: http://localhost/g\n    defaultDataOutput: xml\n")
	if _, err := Merge("", local); err == nil {
		t.Fatal("want an error for an unknown data format")
	}
}

func TestLoadFileIgnoresDiscovery(t *testing.T) {
	dir := t.TempDir()
	only := filepath.Join(dir, "only.yaml")
	write(t, only, "configs:\n  solo:\n    host: http://only/g\n")
	cfg, err := LoadFile(only)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Profiles) != 1 {
		t.Errorf("profiles = %v, want only the ones in the named file", cfg.Profiles)
	}
}
