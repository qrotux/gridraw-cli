package cli

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/qrotux/gridraw-cli/internal/config"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	var (
		name, host, bearer, basic string
		infoOut, dataOut          string
		global                    bool
	)
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Create or edit a configuration profile",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			header, err := config.AuthHeader(bearer, basic)
			if err != nil {
				return &UsageError{Msg: err.Error()}
			}
			p := prompt{config.NewPrompter(cmd.InOrStdin(), cmd.ErrOrStderr())}
			userPath, localPath, err := config.Discover()
			if err != nil {
				return err
			}
			silent := flagsComplete(cmd)
			target := localPath
			if global {
				target = userPath
			}

			if name == "" {
				if silent {
					name = "default"
				} else if name, err = p.Ask("Configuration name", "default"); err != nil {
					return err
				}
			}
			// Editing a profile offers its stored values, so an empty answer
			// keeps what is there instead of resetting it to a built-in default.
			old := existingProfile(name, target)

			if !silent {
				for host == "" {
					if host, err = p.Ask("Grids URL, e.g. http://localhost:8080/api/grids", old.Host); err != nil {
						return err
					}
					if host == "" {
						fmt.Fprintln(cmd.ErrOrStderr(), "  a grids URL is required")
					}
				}
			}
			if !silent && header == "" && bearer == "" && basic == "" {
				header = old.Header
				kind, err := p.Choose("Authorization", []string{"bearer", "basic", "none"}, authKind(old))
				if err != nil {
					return err
				}
				switch kind {
				case "bearer":
					keep := keptHeader(old.Header, "Bearer ")
					tok, err := p.Ask(keepHint("Bearer token", keep), "")
					if err != nil {
						return err
					}
					if tok == "" && keep != "" {
						header = keep
					} else {
						header, _ = config.AuthHeader(tok, "")
					}
				case "basic":
					keep := keptHeader(old.Header, "Basic ")
					user, err := p.Ask("Username", basicUser(keep))
					if err != nil {
						return err
					}
					pass, err := p.Ask(keepHint("Password", keep), "")
					if err != nil {
						return err
					}
					if pass == "" && keep != "" && user == basicUser(keep) {
						header = keep
					} else {
						header, _ = config.AuthHeader("", user+":"+pass)
					}
				case "none":
					header = ""
				}
			}
			if infoOut == "" {
				if silent {
					infoOut = "yaml"
				} else if infoOut, err = p.Choose("Default info output", []string{"yaml", "json"}, old.InfoOutput()); err != nil {
					return err
				}
			}
			if dataOut == "" {
				if silent {
					dataOut = "csv"
				} else if dataOut, err = p.Choose("Default data output", []string{"csv", "tsv", "json", "jsona", "jsonl", "yaml", "yamla"}, old.DataOutput()); err != nil {
					return err
				}
			}

			if !silent && !cmd.Flags().Changed("global") {
				where, err := p.Choose(fmt.Sprintf("Write to %s or %s", localPath, userPath), []string{"local", "global"}, "local")
				if err != nil {
					return err
				}
				if where == "global" {
					target = userPath
				}
			}

			cfg, err := config.LoadFileOrNew(target)
			if err != nil {
				return err
			}
			profile := config.Profile{Host: host, Header: header, DefaultInfoOutput: infoOut, DefaultDataOutput: dataOut}
			if err := profile.Validate(name); err != nil {
				return err
			}
			cfg.Profiles[name] = profile
			if cfg.Current == "" {
				cfg.Current = name
			}
			if err := config.Save(cfg, target); err != nil {
				return err
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "Wrote profile %q to %s\n", name, target)
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&name, "name", "", "profile name")
	f.StringVar(&host, "host", "", "grids URL")
	f.StringVar(&bearer, "bearer-token", "", "Bearer token")
	f.StringVar(&basic, "basic-auth", "", "username:password for Basic auth")
	f.StringVar(&infoOut, "info-output", "", "default info output: yaml or json")
	f.StringVar(&dataOut, "data-output", "", "default data output: csv, tsv, json, jsona, jsonl, yaml, yamla")
	f.BoolVar(&global, "global", false, "write to the user configuration file")

	cmd.AddCommand(newConfigListCmd(), newConfigUseCmd(), newConfigShowCmd(), newConfigPathCmd())
	return cmd
}

// flagsComplete reports whether the flags carry a whole profile: a host and an
// authorization. Every other field then takes its documented default and no
// question is printed.
func flagsComplete(cmd *cobra.Command) bool {
	f := cmd.Flags()
	return f.Changed("host") && (f.Changed("bearer-token") || f.Changed("basic-auth"))
}

// prompt classifies the prompter's errors, which reach the user as exit codes.
type prompt struct{ *config.Prompter }

func (p prompt) Ask(question, def string) (string, error) {
	got, err := p.Prompter.Ask(question, def)
	return got, promptError(err)
}

func (p prompt) Choose(question string, options []string, def string) (string, error) {
	got, err := p.Prompter.Choose(question, options, def)
	return got, promptError(err)
}

func promptError(err error) error {
	if errors.Is(err, config.ErrNoInput) {
		return &UsageError{Msg: "no answer to read; supply --host and --bearer-token or --basic-auth to configure without questions"}
	}
	return err
}

// existingProfile is what the questions default to: the profile in the file
// about to be written, or the one the merged configuration shows, since the
// user may be editing what `gridraw config show` printed. It is the unexpanded
// form, so keeping a stored value stores the ${VAR} reference again and not
// the credential it resolved to.
func existingProfile(name, target string) config.Profile {
	if cfg, err := config.LoadFileOrNew(target); err == nil {
		if p, ok := cfg.Unexpanded(name); ok {
			return p
		}
	}
	if cfg, err := config.Load(); err == nil {
		if p, ok := cfg.Unexpanded(name); ok {
			return p
		}
	}
	return config.Profile{}
}

// authKind is the authorization the profile already uses; a profile with a
// host but no header guards nothing, which is a deliberate choice to keep.
func authKind(p config.Profile) string {
	switch {
	case strings.HasPrefix(p.Header, "Basic "):
		return "basic"
	case p.Header != "":
		return "bearer"
	case p.Host != "":
		return "none"
	}
	return "bearer"
}

func keptHeader(header, scheme string) string {
	if strings.HasPrefix(header, scheme) {
		return header
	}
	return ""
}

// keepHint says what an empty answer keeps, with the credential masked.
func keepHint(question, header string) string {
	if header == "" {
		return question
	}
	return fmt.Sprintf("%s (empty keeps %s)", question, config.MaskHeader(header))
}

func basicUser(header string) string {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(header, "Basic "))
	if err != nil {
		return ""
	}
	user, _, _ := strings.Cut(string(raw), ":")
	return user
}

func newConfigListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configuration profiles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig(cmd)
			if err != nil {
				return err
			}
			for _, name := range cfg.Names() {
				marker := " "
				if name == cfg.Current {
					marker = "*"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s %-20s %-40s %s\n", marker, name, cfg.Profiles[name].Host, cfg.Origin[name])
			}
			return nil
		},
	}
}

func newConfigUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use NAME",
		Short: "Select the current profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(cmd)
			if err != nil {
				return err
			}
			if _, _, err := cfg.Profile(args[0]); err != nil {
				return err
			}
			if len(cfg.Sources) == 0 {
				return &config.Error{Msg: "no configuration file to update; run `gridraw config` to create one"}
			}
			// current is rewritten in the nearest file that was read, so a
			// local file keeps its precedence over the user one.
			target := cfg.Sources[len(cfg.Sources)-1]
			file, err := config.LoadFileOrNew(target)
			if err != nil {
				return err
			}
			file.Current = args[0]
			if err := config.Save(file, target); err != nil {
				return err
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "Current profile is %q (%s)\n", args[0], target)
			return nil
		},
	}
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show [NAME]",
		Short: "Show a profile with its credential masked",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(cmd)
			if err != nil {
				return err
			}
			var want string
			if len(args) == 1 {
				want = args[0]
			}
			p, name, err := cfg.Profile(want)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "name: %s\nhost: %s\nheader: %s\ndefaultInfoOutput: %s\ndefaultDataOutput: %s\n",
				name, p.Host, config.MaskHeader(p.Header), p.InfoOutput(), p.DataOutput())
			return nil
		},
	}
}

func newConfigPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the configuration files that were read",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig(cmd)
			if err != nil {
				return err
			}
			if len(cfg.Sources) == 0 {
				fmt.Fprintln(cmd.ErrOrStderr(), "no configuration file found")
				return nil
			}
			for _, s := range cfg.Sources {
				fmt.Fprintln(cmd.OutOrStdout(), s)
			}
			return nil
		},
	}
}
