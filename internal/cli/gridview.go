package cli

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/qrotux/gridraw-cli/internal/wire"
	"github.com/qrotux/gridraw-cli/internal/wql"
	"gopkg.in/yaml.v3"
)

// field is one key of an ordered object.
type field struct {
	key   string
	value any
}

// object is a mapping that keeps the order its keys were added in, for both
// JSON and YAML; a Go map would come out sorted in either.
type object []field

// add appends a key unless the value is one this view omits: an empty string,
// a false flag, a zero number or an empty collection.
func (o object) add(key string, value any) object {
	switch v := value.(type) {
	case string:
		if v == "" {
			return o
		}
	case bool:
		if !v {
			return o
		}
	case int:
		if v == 0 {
			return o
		}
	case []string:
		if len(v) == 0 {
			return o
		}
	case []any:
		if len(v) == 0 {
			return o
		}
	case object:
		if len(v) == 0 {
			return o
		}
	}
	return append(o, field{key, value})
}

// MarshalJSON writes the object with its keys in order.
func (o object) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, f := range o {
		if i > 0 {
			buf.WriteByte(',')
		}
		key, err := json.Marshal(f.key)
		if err != nil {
			return nil, err
		}
		val, err := json.Marshal(f.value)
		if err != nil {
			return nil, err
		}
		buf.Write(key)
		buf.WriteByte(':')
		buf.Write(val)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// MarshalYAML writes the object with its keys in order.
func (o object) MarshalYAML() (any, error) {
	node := &yaml.Node{Kind: yaml.MappingNode}
	for _, f := range o {
		key := &yaml.Node{Kind: yaml.ScalarNode, Value: f.key}
		val := &yaml.Node{}
		if err := val.Encode(f.value); err != nil {
			return nil, err
		}
		node.Content = append(node.Content, key, val)
	}
	return node, nil
}

// flow is a value rendered inline in YAML, so a list of operators or a set of
// enum values takes one line rather than one line per element.
type flow struct{ value any }

// MarshalJSON renders the wrapped value; JSON has no block style to opt out of.
func (f flow) MarshalJSON() ([]byte, error) { return json.Marshal(f.value) }

// MarshalYAML renders the wrapped value in flow style.
func (f flow) MarshalYAML() (any, error) {
	node := &yaml.Node{}
	if err := node.Encode(f.value); err != nil {
		return nil, err
	}
	node.Style = yaml.FlowStyle
	return node, nil
}

// gridView condenses a descriptor to what someone writing a query needs: the
// column types, the operators in the spelling the where language accepts, the
// enum values, and how a value of an awkward type is written. Everything the
// server sends for the benefit of a UI — operator labels, widgets — is dropped.
func gridView(d *wire.Descriptor) object {
	out := object{}.
		add("name", d.Name).
		add("description", d.Description).
		add("idColumn", d.IDColumn).
		add("pageSize", d.PageSize)

	if d.DefaultSort.Column != "" {
		out = out.add("defaultSort", d.DefaultSort.Column+" "+d.DefaultSort.Dir)
	}
	if d.Search != nil && len(d.Search.Columns) > 0 {
		out = out.add("search", flow{d.Search.Columns})
	}
	out = out.add("skipTotal", d.SkipTotal)

	columns := make([]any, 0, len(d.Columns))
	for i := range d.Columns {
		columns = append(columns, columnView(&d.Columns[i]))
	}
	return out.add("columns", columns)
}

func columnView(c *wire.Column) object {
	typeName := c.Type
	if c.Array {
		typeName += "[]"
	}
	out := object{}.
		add("key", c.Key).
		add("type", typeName).
		add("description", c.Description)

	if c.Filter != nil && len(c.Filter.EnumValues) > 0 {
		values := object{}
		for _, v := range c.Filter.EnumValues {
			values = append(values, field{v.Value, v.Label})
		}
		out = out.add("enum", flow{values})
	}
	out = out.add("value", wql.ValueHint(c.Type))
	// A step of one second is the default and constrains nothing.
	if c.Step > 1 {
		out = out.add("step", c.Step)
	}
	if c.Filter != nil {
		ops := make([]string, 0, len(c.Filter.Operators))
		for _, o := range c.Filter.Operators {
			ops = append(ops, wql.Spelling(o.Op))
		}
		out = out.add("filters", flow{ops})
	}
	return out.add("sortable", c.Sortable).add("visible", c.DefaultVisible)
}

// renderView writes the condensed descriptor in the information format.
func renderView(view object, format string) ([]byte, error) {
	if format == "json" {
		body, err := json.MarshalIndent(view, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("cannot render the descriptor: %w", err)
		}
		return append(body, '\n'), nil
	}
	body, err := yaml.Marshal(view)
	if err != nil {
		return nil, fmt.Errorf("cannot render the descriptor: %w", err)
	}
	return body, nil
}
