package render

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

// yamlWriter serves yamla (a bare sequence) and yaml (the sequence under a
// rows key, with the total above it). Both stream: a YAML sequence needs no
// prologue, so items are emitted as they arrive.
type yamlWriter struct {
	w        io.Writer
	opt      Options
	envelope bool
	indent   string
}

func (y *yamlWriter) Head(total *int64) error {
	if !y.envelope {
		return nil
	}
	if total != nil {
		if _, err := fmt.Fprintf(y.w, "total: %d\n", *total); err != nil {
			return err
		}
	}
	y.indent = "  "
	_, err := io.WriteString(y.w, "rows:\n")
	return err
}

func (y *yamlWriter) Row(row map[string]any) error {
	node := make(map[string]any, len(row))
	for _, kv := range ordered(row, y.opt.Columns) {
		if kv.Value == nil && y.opt.NullVal != "" {
			node[kv.Key] = y.opt.NullVal
			continue
		}
		node[kv.Key] = yamlValue(kv.Value)
	}
	body, err := yaml.Marshal([]any{node})
	if err != nil {
		return err
	}
	if y.indent != "" {
		body = []byte(y.indent + strings.ReplaceAll(strings.TrimRight(string(body), "\n"), "\n", "\n"+y.indent) + "\n")
	}
	_, err = y.w.Write(body)
	return err
}

func (y *yamlWriter) Close() error { return nil }

// yamlValue turns a json.Number into a real scalar so it is not quoted.
func yamlValue(v any) any {
	switch t := v.(type) {
	case json.Number:
		if i, err := t.Int64(); err == nil {
			return i
		}
		if f, err := t.Float64(); err == nil {
			return f
		}
		return t.String()
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[k] = yamlValue(val)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			out[i] = yamlValue(val)
		}
		return out
	}
	return v
}
