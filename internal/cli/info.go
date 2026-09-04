package cli

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// infoFormat resolves -o against the profile default and rejects data formats.
func infoFormat(flag, def string) (string, error) {
	f := flag
	if f == "" {
		f = def
	}
	if f != "yaml" && f != "json" {
		return "", &UsageError{Msg: fmt.Sprintf("output %q is not available here; an information command prints yaml or json", f)}
	}
	return f, nil
}

// jsonToYAML converts a decoded JSON value to YAML. json.Number is emitted
// unquoted so a page size stays a number.
func jsonToYAML(v any) ([]byte, error) {
	return yaml.Marshal(numbersToScalars(v))
}

// numbersToScalars replaces every json.Number with a plain scalar node holding
// the literal from the response: converting to int64 or float64 first would
// round an integer wider than int64 and would drop the fractional zero of
// `1.0`. An untagged plain node is emitted unquoted, so it stays a number.
func numbersToScalars(v any) any {
	switch t := v.(type) {
	case json.Number:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: t.String()}
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[k] = numbersToScalars(val)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			out[i] = numbersToScalars(val)
		}
		return out
	}
	return v
}

// writeInfo prints a raw JSON body in the requested information format.
func writeInfo(cmd *cobra.Command, raw []byte, format string) error {
	if format == "json" {
		_, err := cmd.OutOrStdout().Write(append(bytes.TrimRight(raw, "\n"), '\n'))
		return err
	}
	var v any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		return fmt.Errorf("cannot decode the response: %w", err)
	}
	if dec.More() {
		return fmt.Errorf("the response holds more than one JSON value")
	}
	out, err := jsonToYAML(v)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(out)
	return err
}

func newListCmd() *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List the grids the server publishes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, profile, _, err := apiFor(cmd)
			if err != nil {
				return err
			}
			format, err := infoFormat(output, profile.InfoOutput())
			if err != nil {
				return err
			}
			raw, err := api.Raw(cmd.Context(), "GET", "/-/list")
			if err != nil {
				return err
			}
			return writeInfo(cmd, raw, format)
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "", "output format: yaml or json")
	return cmd
}

func newGridCmd() *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "grid [NAME]",
		Short: "Show the grid registry, or one grid's descriptor",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, profile, cache, err := apiFor(cmd)
			if err != nil {
				return err
			}
			format, err := infoFormat(output, profile.InfoOutput())
			if err != nil {
				return err
			}
			if len(args) == 0 {
				raw, err := api.Raw(cmd.Context(), "GET", "/-/registry")
				if err != nil {
					return err
				}
				return writeInfo(cmd, raw, format)
			}
			// Always a live fetch: the command reports the server's state. The
			// body is then cached, since the request has been paid for anyway.
			desc, raw, err := api.Descriptor(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if noCache, _ := cmd.Flags().GetBool("no-cache"); !noCache {
				cache.Put(args[0], raw)
			}
			if rawOut, _ := cmd.Flags().GetBool("raw"); rawOut {
				return writeInfo(cmd, raw, format)
			}
			body, err := renderView(gridView(desc), format)
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(body)
			return err
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "", "output format: yaml or json")
	cmd.Flags().Bool("no-cache", false, "do not write the fetched descriptor into the cache")
	cmd.Flags().Bool("raw", false, "print the server's descriptor unchanged instead of the query view")
	return cmd
}
