// Package config reads, merges and writes gridraw CLI configuration.
package config

import "fmt"

// Profile is one named configuration.
type Profile struct {
	Host              string `yaml:"host"`
	Header            string `yaml:"header,omitempty"`
	DefaultInfoOutput string `yaml:"defaultInfoOutput,omitempty"`
	DefaultDataOutput string `yaml:"defaultDataOutput,omitempty"`
}

// Config is the file format: named profiles plus the selected one.
type Config struct {
	Current  string             `yaml:"current"`
	Profiles map[string]Profile `yaml:"configs"`

	// Sources lists the files this Config was merged from, nearest last.
	Sources []string `yaml:"-"`

	// Origin maps each profile name to the file the merge took it from.
	Origin map[string]string `yaml:"-"`

	// unexpanded holds each profile as its file spells it, before ${VAR}
	// substitution. Save writes these, so a rewrite never replaces a
	// reference with the credential it resolved to.
	unexpanded map[string]Profile
}

// SetProfile stores a profile under name, dropping the unexpanded form Save
// would otherwise have written for it.
func (c *Config) SetProfile(name string, p Profile) {
	if c.Profiles == nil {
		c.Profiles = map[string]Profile{}
	}
	c.Profiles[name] = p
	delete(c.unexpanded, name)
}

// forFile is the config as it goes back to disk: every profile still holding
// an unexpanded form is written in that form.
func (c *Config) forFile() *Config {
	out := &Config{Current: c.Current, Profiles: make(map[string]Profile, len(c.Profiles))}
	for name, p := range c.Profiles {
		if raw, ok := c.unexpanded[name]; ok {
			p = raw
		}
		out.Profiles[name] = p
	}
	return out
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
