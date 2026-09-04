package render

import (
	"encoding/json"
	"fmt"
	"sort"
)

// Cell renders a row value for a text format. A nested value becomes compact
// JSON, which is what makes csv usable on array and json columns.
func Cell(v any, nullVal string) string {
	switch t := v.(type) {
	case nil:
		return nullVal
	case string:
		return t
	case json.Number:
		return t.String()
	case bool:
		return fmt.Sprintf("%t", t)
	case float64:
		return fmt.Sprintf("%v", t)
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(raw)
}

// ordered rebuilds a row in the writer's column order so every line of the
// output has the same keys in the same sequence, whatever the server sent.
func ordered(row map[string]any, columns []string) []keyValue {
	if len(columns) == 0 {
		return sortedPairs(row)
	}
	out := make([]keyValue, 0, len(columns))
	for _, k := range columns {
		out = append(out, keyValue{k, row[k]})
	}
	return out
}

type keyValue struct {
	Key   string
	Value any
}

// sortedPairs keeps the output deterministic where the server's key order is not.
func sortedPairs(row map[string]any) []keyValue {
	keys := make([]string, 0, len(row))
	for k := range row {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]keyValue, 0, len(keys))
	for _, k := range keys {
		out = append(out, keyValue{k, row[k]})
	}
	return out
}
