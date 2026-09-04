package render

import (
	"bytes"
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"
)

func renderYAML(t *testing.T, f Format, opt Options, total *int64, rows ...map[string]any) string {
	t.Helper()
	var buf bytes.Buffer
	w, err := New(&buf, f, opt)
	if err != nil {
		t.Fatalf("New(%s): %v", f, err)
	}
	if err := w.Head(total); err != nil {
		t.Fatalf("Head: %v", err)
	}
	for _, row := range rows {
		if err := w.Row(row); err != nil {
			t.Fatalf("Row: %v", err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	return buf.String()
}

func TestYAMLAConcatenates(t *testing.T) {
	got := renderYAML(t, FormatYAMLA, Options{Columns: []string{"id", "count"}}, nil,
		map[string]any{"id": "a", "count": json.Number("1")},
		map[string]any{"id": "b", "count": nil})
	want := "- id: a\n  count: 1\n" +
		"- id: b\n  count: null\n"
	if got != want {
		t.Errorf("yamla =\n%q\nwant\n%q", got, want)
	}
}

func TestYAMLEnvelope(t *testing.T) {
	total := int64(2)
	got := renderYAML(t, FormatYAML, Options{Columns: []string{"id"}}, &total,
		map[string]any{"id": "a"},
		map[string]any{"id": "b"})
	want := "total: 2\nrows:\n  - id: a\n  - id: b\n"
	if got != want {
		t.Errorf("yaml =\n%q\nwant\n%q", got, want)
	}
}

func TestYAMLNoTotal(t *testing.T) {
	got := renderYAML(t, FormatYAML, Options{Columns: []string{"id"}}, nil,
		map[string]any{"id": "a"})
	want := "rows:\n  - id: a\n"
	if got != want {
		t.Errorf("yaml =\n%q\nwant\n%q", got, want)
	}
}

// Columns fixes the key order; marshalling the row as a Go map would sort it.
func TestYAMLColumnOrder(t *testing.T) {
	got := renderYAML(t, FormatYAMLA, Options{Columns: []string{"name", "age", "id"}}, nil,
		map[string]any{"id": "i", "age": json.Number("3"), "name": "x"})
	want := "- name: x\n  age: 3\n  id: i\n"
	if got != want {
		t.Errorf("yamla =\n%q\nwant\n%q", got, want)
	}
}

func TestYAMLMultilineValue(t *testing.T) {
	row := map[string]any{"a": "\nstarts blank\ntrails\n\n"}
	opt := Options{Columns: []string{"a"}}
	wantItem := `- a: "\nstarts blank\ntrails\n\n"` + "\n"
	if got := renderYAML(t, FormatYAMLA, opt, nil, row); got != wantItem {
		t.Errorf("yamla =\n%q\nwant\n%q", got, wantItem)
	}
	got := renderYAML(t, FormatYAML, opt, nil, row)
	if want := "rows:\n  " + wantItem; got != want {
		t.Errorf("yaml =\n%q\nwant\n%q", got, want)
	}
	// The envelope shifts rendered text right by two columns, which a block
	// scalar's indentation indicator would not survive.
	var back struct {
		Rows []map[string]string
	}
	if err := yaml.Unmarshal([]byte(got), &back); err != nil {
		t.Fatalf("own output does not parse: %v", err)
	}
	if len(back.Rows) != 1 || back.Rows[0]["a"] != row["a"] {
		t.Errorf("round trip = %q, want %q", back.Rows, row["a"])
	}
}

func TestYAMLNested(t *testing.T) {
	got := renderYAML(t, FormatYAML, Options{Columns: []string{"id", "meta", "tags", "none"}}, nil,
		map[string]any{
			"id":   "a",
			"meta": map[string]any{"k": json.Number("1"), "b": true},
			"tags": []any{"go", nil},
			"none": []any{},
		})
	want := "rows:\n" +
		"  - id: a\n" +
		"    meta:\n" +
		"      b: true\n" +
		"      k: 1\n" +
		"    tags:\n" +
		"      - go\n" +
		"      - null\n" +
		"    none: []\n"
	if got != want {
		t.Errorf("yaml =\n%q\nwant\n%q", got, want)
	}
}

func TestYAMLNullVal(t *testing.T) {
	got := renderYAML(t, FormatYAMLA, Options{Columns: []string{"id", "n"}, NullVal: "NULL"}, nil,
		map[string]any{"id": "a", "n": nil})
	// yaml.v3 quotes a value a YAML 1.1 reader would take for null.
	want := "- id: a\n  \"n\": \"NULL\"\n"
	if got != want {
		t.Errorf("yamla =\n%q\nwant\n%q", got, want)
	}
}

// A bare rows key would read back as null, where the json writer gives [].
func TestYAMLEmptyResult(t *testing.T) {
	total := int64(0)
	if got, want := renderYAML(t, FormatYAML, Options{Columns: []string{"id"}}, &total), "total: 0\nrows: []\n"; got != want {
		t.Errorf("yaml = %q, want %q", got, want)
	}
	if got := renderYAML(t, FormatYAMLA, Options{Columns: []string{"id"}}, nil); got != "" {
		t.Errorf("yamla = %q, want no output", got)
	}
}
