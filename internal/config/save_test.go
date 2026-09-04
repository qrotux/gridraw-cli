package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveIsPrivateAndReadsBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.yaml")
	cfg := &Config{Current: "default", Profiles: map[string]Profile{
		"default": {Host: "http://localhost:8080/api/grids", Header: "Bearer t", DefaultDataOutput: "csv"},
	}}
	if err := Save(cfg, path); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("mode = %o, want 600", perm)
	}
	back, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if back.Profiles["default"].Header != "Bearer t" || back.Current != "default" {
		t.Errorf("round trip lost data: %+v", back)
	}
}

func TestSaveTightensAnExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	write(t, path, "current: default\n")
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{Current: "default", Profiles: map[string]Profile{
		"default": {Host: "http://localhost:8080/api/grids"},
	}}
	if err := Save(cfg, path); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("mode = %o, want 600: os.WriteFile leaves an existing file's mode alone", perm)
	}
}

// A ${VAR} reference must survive a rewrite: the file holds an indirection,
// not the credential it resolves to, and a save touching one profile must not
// bake the value into any other.
func TestSaveKeepsEnvReferencesUnexpanded(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	write(t, path, "current: a\nconfigs:\n  a:\n    host: http://a/g\n    header: \"Bearer ${GRIDRAW_TEST_TOKEN}\"\n  b:\n    host: ${GRIDRAW_TEST_HOST}\n")
	t.Setenv("GRIDRAW_TEST_TOKEN", "s3cret")
	t.Setenv("GRIDRAW_TEST_HOST", "http://b/g")

	cfg, err := LoadFileOrNew(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Current = "b"
	if err := Save(cfg, path); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "${GRIDRAW_TEST_TOKEN}") || !strings.Contains(string(body), "${GRIDRAW_TEST_HOST}") {
		t.Errorf("the references were written expanded:\n%s", body)
	}
	if strings.Contains(string(body), "s3cret") {
		t.Errorf("the credential was written to disk:\n%s", body)
	}
	back, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if back.Current != "b" || back.Profiles["a"].Header != "Bearer s3cret" || back.Profiles["b"].Host != "http://b/g" {
		t.Errorf("round trip = %+v", back)
	}
}

// A profile the caller replaced is written as given: it no longer has an
// unexpanded form to fall back on.
func TestSaveWritesAReplacedProfileVerbatim(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	write(t, path, "current: a\nconfigs:\n  a:\n    host: ${GRIDRAW_TEST_HOST}\n")
	t.Setenv("GRIDRAW_TEST_HOST", "http://a/g")
	cfg, err := LoadFileOrNew(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg.SetProfile("a", Profile{Host: "http://new/g"})
	if err := Save(cfg, path); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "${GRIDRAW_TEST_HOST}") || !strings.Contains(string(body), "http://new/g") {
		t.Errorf("a replaced profile was not written as given:\n%s", body)
	}
	if strings.Contains(string(body), `header: ""`) {
		t.Errorf("an empty field was written:\n%s", body)
	}
}
