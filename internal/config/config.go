// Package config reads, merges and writes gridraw CLI configuration.
package config

import "fmt"

// Profile is one named configuration.
type Profile struct {
	Host              string `yaml:"host"`
	Header            string `yaml:"header"`
	DefaultInfoOutput string `yaml:"defaultInfoOutput"`
	DefaultDataOutput string `yaml:"defaultDataOutput"`
}

// Config is the file format: named profiles plus the selected one.
type Config struct {
	Current  string             `yaml:"current"`
	Profiles map[string]Profile `yaml:"configs"`

	// Sources lists the files this Config was merged from, nearest last.
	Sources []string `yaml:"-"`
}

// Error is a configuration problem; the CLI maps it to exit code 3.
type Error struct {
	Msg string
	Err error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return e.Msg + ": " + e.Err.Error()
	}
	return e.Msg
}
func (e *Error) Unwrap() error { return e.Err }

var (
	infoFormats = []string{"yaml", "json"}
	dataFormats = []string{"csv", "tsv", "json", "jsona", "jsonl", "yaml", "yamla"}
)

// Validate checks the output formats and the presence of a host.
func (p Profile) Validate(name string) error {
	if p.Host == "" {
		return &Error{Msg: fmt.Sprintf("profile %q has no host", name)}
	}
	if p.DefaultInfoOutput != "" && !contains(infoFormats, p.DefaultInfoOutput) {
		return &Error{Msg: fmt.Sprintf("profile %q: defaultInfoOutput %q is not one of yaml, json", name, p.DefaultInfoOutput)}
	}
	if p.DefaultDataOutput != "" && !contains(dataFormats, p.DefaultDataOutput) {
		return &Error{Msg: fmt.Sprintf("profile %q: defaultDataOutput %q is not one of csv, tsv, json, jsona, jsonl, yaml, yamla", name, p.DefaultDataOutput)}
	}
	return nil
}

// InfoOutput is the profile's info format with its default applied.
func (p Profile) InfoOutput() string {
	if p.DefaultInfoOutput == "" {
		return "yaml"
	}
	return p.DefaultInfoOutput
}

// DataOutput is the profile's data format with its default applied.
func (p Profile) DataOutput() string {
	if p.DefaultDataOutput == "" {
		return "csv"
	}
	return p.DefaultDataOutput
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
