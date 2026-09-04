package config

import (
	"os"
	"path/filepath"
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
