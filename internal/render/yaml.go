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
	wrote    bool
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
	// The line is left unterminated: the first row closes it, and Close turns
	// an empty result into "rows: []" rather than a bare key, which would read
	// back as null.
	_, err := io.WriteString(y.w, "rows:")
	return err
}

func (y *yamlWriter) Row(row map[string]any) error {
	item, err := rowNode(row, y.opt)
	if err != nil {
		return err
	}
	body, err := yaml.Marshal(&yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{item}})
	if err != nil {
		return err
	}
	if y.envelope {
		if !y.wrote {
			if _, err := io.WriteString(y.w, "\n"); err != nil {
				return err
			}
		}
		body = shift(body, "  ")
	}
	y.wrote = true
	_, err = y.w.Write(body)
	return err
}

func (y *yamlWriter) Close() error {
	if !y.envelope || y.wrote {
		return nil
	}
	_, err := io.WriteString(y.w, " []\n")
	return err
}

// rowNode builds the row as an explicit mapping so the key order is the one
// Options.Columns fixes; marshalling a Go map would sort the keys instead.
func rowNode(row map[string]any, opt Options) (*yaml.Node, error) {
	m := &yaml.Node{Kind: yaml.MappingNode}
	for _, kv := range ordered(row, opt.Columns) {
		v := kv.Value
		if v == nil && opt.NullVal != "" {
			v = opt.NullVal
		}
		val, err := yamlNode(v)
		if err != nil {
			return nil, err
		}
		key, err := yamlNode(kv.Key)
		if err != nil {
			return nil, err
		}
		m.Content = append(m.Content, key, val)
	}
	return m, nil
}

// yamlNode renders a value as a node tree, turning a json.Number into a real
// scalar so it is not quoted.
func yamlNode(v any) (*yaml.Node, error) {
	switch t := v.(type) {
	case string:
		// A multi-line value is double quoted by hand: block style would
		// carry an indentation indicator the envelope's uniform shift does
		// not rewrite, and Node.Encode drops the leading line break of a
		// value that starts with one. Single-line values go through Encode
		// below, which quotes those a YAML 1.1 reader would read as scalars
		// of another type.
		if strings.ContainsAny(t, "\n\r") {
			return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: t, Style: yaml.DoubleQuotedStyle}, nil
		}
	case json.Number:
		if i, err := t.Int64(); err == nil {
			v = i
		} else if f, err := t.Float64(); err == nil {
			v = f
		} else {
			v = t.String()
		}
	case map[string]any:
		m := &yaml.Node{Kind: yaml.MappingNode}
		for _, kv := range sortedPairs(t) {
			val, err := yamlNode(kv.Value)
			if err != nil {
				return nil, err
			}
			key, err := yamlNode(kv.Key)
			if err != nil {
				return nil, err
			}
			m.Content = append(m.Content, key, val)
		}
		return m, nil
	case []any:
		s := &yaml.Node{Kind: yaml.SequenceNode}
		for _, e := range t {
			n, err := yamlNode(e)
			if err != nil {
				return nil, err
			}
			s.Content = append(s.Content, n)
		}
		return s, nil
	}
	var n yaml.Node
	if err := n.Encode(v); err != nil {
		return nil, err
	}
	return &n, nil
}

func shift(body []byte, pad string) []byte {
	var sb strings.Builder
	for _, line := range strings.Split(strings.TrimSuffix(string(body), "\n"), "\n") {
		sb.WriteString(pad)
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	return []byte(sb.String())
}
