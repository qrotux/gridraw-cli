package render

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestYAMLAConcatenates(t *testing.T) {
	var buf bytes.Buffer
	w, _ := New(&buf, FormatYAMLA, Options{Columns: []string{"id", "n"}})
	w.Head(nil)
	w.Row(map[string]any{"id": "a", "n": json.Number("1")})
	w.Row(map[string]any{"id": "b", "n": nil})
	w.Close()
	got := buf.String()
	if !strings.HasPrefix(got, "- ") || strings.Count(got, "- id:") != 2 {
		t.Errorf("yamla =\n%s\nwant two sequence items and no prologue", got)
	}
	if strings.Contains(got, "n: \"1\"") {
		t.Error("a number came out quoted")
	}
}

func TestYAMLEnvelope(t *testing.T) {
	var buf bytes.Buffer
	w, _ := New(&buf, FormatYAML, Options{Columns: []string{"id"}})
	total := int64(2)
	w.Head(&total)
	w.Row(map[string]any{"id": "a"})
	w.Close()
	got := buf.String()
	if !strings.HasPrefix(got, "total: 2\nrows:\n") || !strings.Contains(got, "  - id: a") {
		t.Errorf("yaml =\n%s\nwant total, rows and an indented item", got)
	}
}
