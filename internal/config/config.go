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
	// substitution, so Save can put a reference back instead of the value it
	// resolved to.
	unexpanded map[string]Profile
}

// Unexpanded returns the named profile in the form its file spells it, ${VAR}
// references included. It is what an interactive edit must offer as the value
// an empty answer keeps: accepting the offer then stores the reference again
// rather than the credential behind it.
func (c *Config) Unexpanded(name string) (Profile, bool) {
	if p, ok := c.unexpanded[name]; ok {
		return p, true
	}
	p, ok := c.Profiles[name]
	return p, ok
}

// forFile is the config as it goes back to disk. A profile that still expands
// to what it was loaded with is written in the form its file spelled it; one a
// caller has changed no longer matches and is written as it stands. The
// comparison is what makes the guarantee hold: assigning into Profiles is as
// safe as any accessor would be.
func (c *Config) forFile() *Config {
	out := &Config{Current: c.Current, Profiles: make(map[string]Profile, len(c.Profiles))}
	for name, p := range c.Profiles {
		if raw, ok := c.unexpanded[name]; ok && expandProfile(raw) == p {
			p = raw
		}
		out.Profiles[name] = p
	}
	return out
}

// expandProfile substitutes ${VAR} in every field, leaving an unset variable
// as its reference: only forFile's comparison reads the result, and a miss
// there just means the profile is written as it stands.
func expandProfile(p Profile) Profile {
	p.Host = expandLenient(p.Host)
	p.Header = expandLenient(p.Header)
	p.DefaultInfoOutput = expandLenient(p.DefaultInfoOutput)
	p.DefaultDataOutput = expandLenient(p.DefaultDataOutput)
	return p
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
