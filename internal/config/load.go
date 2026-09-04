package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// LocalFileName is the config file looked up in the working directory.
const LocalFileName = ".gridraw.yaml"

var envRef = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// UserPath is $XDG_CONFIG_HOME/gridraw/config.yaml, or ~/.config/gridraw/config.yaml
// when that variable is unset or empty. os.UserConfigDir is deliberately not used:
// on darwin it answers ~/Library/Application Support, which is not the promised path.
func UserPath() (string, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", &Error{Msg: "cannot locate the home directory", Err: err}
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "gridraw", "config.yaml"), nil
}

// Discover returns the user path and the local path, either of which may not exist.
func Discover() (userPath, localPath string, err error) {
	userPath, err = UserPath()
	if err != nil {
		return "", "", err
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", "", &Error{Msg: "cannot read the working directory", Err: err}
	}
	return userPath, filepath.Join(wd, LocalFileName), nil
}

// Load merges the discovered user and local files.
func Load() (*Config, error) {
	user, local, err := Discover()
	if err != nil {
		return nil, err
	}
	return Merge(user, local)
}

// LoadFile reads exactly one file, cancelling discovery. A missing file is an error.
func LoadFile(path string) (*Config, error) {
	cfg, found, err := readFile(path)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, &Error{Msg: fmt.Sprintf("configuration file %s does not exist", path)}
	}
	if cfg.Current == "" {
		cfg.Current = "default"
	}
	cfg.Sources = []string{path}
	cfg.Origin = originOf(cfg, path)
	return cfg, validate(cfg)
}

// LoadFileOrNew reads path, or returns an empty Config when it does not exist.
func LoadFileOrNew(path string) (*Config, error) {
	cfg, found, err := readFile(path)
	if err != nil {
		return nil, err
	}
	if !found {
		return &Config{Profiles: map[string]Profile{}, Origin: map[string]string{}, unexpanded: map[string]Profile{}}, nil
	}
	cfg.Sources = []string{path}
	cfg.Origin = originOf(cfg, path)
	return cfg, validate(cfg)
}

// Merge reads both files and lets the local one win profile by profile. A
// missing file contributes nothing; both missing yields an empty Config.
func Merge(userPath, localPath string) (*Config, error) {
	out := &Config{Profiles: map[string]Profile{}, Origin: map[string]string{}, unexpanded: map[string]Profile{}}
	for _, path := range []string{userPath, localPath} {
		if path == "" {
			continue
		}
		cfg, found, err := readFile(path)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		out.Sources = append(out.Sources, path)
		for name, p := range cfg.Profiles {
			out.Profiles[name] = p // a same-named profile is replaced whole
			out.Origin[name] = path
			delete(out.unexpanded, name)
			if raw, ok := cfg.unexpanded[name]; ok {
				out.unexpanded[name] = raw
			}
		}
		if cfg.Current != "" {
			out.Current = cfg.Current
		}
	}
	if out.Current == "" {
		out.Current = "default"
	}
	return out, validate(out)
}

func originOf(cfg *Config, path string) map[string]string {
	origin := make(map[string]string, len(cfg.Profiles))
	for name := range cfg.Profiles {
		origin[name] = path
	}
	return origin
}

func readFile(path string) (*Config, bool, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, &Error{Msg: "cannot read " + path, Err: err}
	}
	var unexpanded Config
	if err := yaml.Unmarshal(raw, &unexpanded); err != nil {
		return nil, false, &Error{Msg: "cannot parse " + path, Err: err}
	}
	expanded, err := expand(string(raw), path)
	if err != nil {
		return nil, false, err
	}
	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, false, &Error{Msg: "cannot parse " + path, Err: err}
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	cfg.unexpanded = unexpanded.Profiles
	return &cfg, true, nil
}

// expand substitutes ${VAR} from the environment. An unset variable is an
// error rather than an empty string: a silently empty Authorization header
// would fail much later, as a 401.
func expand(src, path string) (string, error) {
	var missing []string
	out := envRef.ReplaceAllStringFunc(src, func(ref string) string {
		name := envRef.FindStringSubmatch(ref)[1]
		val, ok := os.LookupEnv(name)
		if !ok {
			missing = append(missing, name)
			return ref
		}
		return val
	})
	if len(missing) > 0 {
		return "", &Error{Msg: fmt.Sprintf("%s references unset environment variable(s): %s", path, strings.Join(missing, ", "))}
	}
	return out, nil
}

func validate(cfg *Config) error {
	for name, p := range cfg.Profiles {
		if err := p.Validate(name); err != nil {
			return err
		}
	}
	return nil
}

// Profile returns the named profile, or the current one when name is empty.
func (c *Config) Profile(name string) (Profile, string, error) {
	if name == "" {
		name = c.Current
	}
	p, ok := c.Profiles[name]
	if !ok {
		if len(c.Profiles) == 0 {
			return Profile{}, name, &Error{Msg: "no configuration found; run `gridraw config` to create one"}
		}
		return Profile{}, name, &Error{Msg: fmt.Sprintf("no configuration named %q; have %s", name, strings.Join(c.Names(), ", "))}
	}
	return p, name, nil
}

// Names lists the profile names in sorted order.
func (c *Config) Names() []string {
	names := make([]string, 0, len(c.Profiles))
	for n := range c.Profiles {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
