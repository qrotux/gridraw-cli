package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Save writes the config to path with mode 0600, creating parent directories.
// The file holds an Authorization header, so the mode is not decoration.
func Save(cfg *Config, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return &Error{Msg: "cannot create " + filepath.Dir(path), Err: err}
	}
	body, err := yaml.Marshal(cfg)
	if err != nil {
		return &Error{Msg: "cannot serialize the configuration", Err: err}
	}
	if err := os.WriteFile(path, body, 0o600); err != nil {
		return &Error{Msg: "cannot write " + path, Err: err}
	}
	return os.Chmod(path, 0o600) // an existing file keeps its old mode without this
}
