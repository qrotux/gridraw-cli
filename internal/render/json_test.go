package render

import (
	"bytes"
	"encoding/json"
	"testing"
)

func rows() []map[string]any {
	return []map[string]any{
		{"id": "a", "n": json.Number("1"), "tags": []any{"go", "cli"}},
		{"id": "b", "n": nil, "tags": []any{}},
	}
}

func run(t *testing.T, format Format, opt Options, total *int64, close bool) string {
	t.Helper()
	var buf bytes.Buffer
	w, err := New(&buf, format, opt)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Head(total); err != nil {
		t.Fatal(err)
	}
	for _, r := range rows() {
		if err := w.Row(r); err != nil {
			t.Fatal(err)
		}
	}
	if close {
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
	}
	return buf.String()
}

func TestJSONAArray(t *testing.T) {
	total := int64(2)
	got := run(t, FormatJSONA, Options{Columns: []string{"id", "n", "tags"}}, &total, true)
	want := `[{"id":"a","n":1,"tags":["go","cli"]},{"id":"b","n":null,"tags":[]}]` + "\n"
	if got != want {
		t.Errorf("jsona = %s\nwant     %s", got, want)
	}
}

func TestJSONEnvelope(t *testing.T) {
	total := int64(2)
	got := run(t, FormatJSON, Options{Columns: []string{"id"}}, &total, true)
	if want := `{"total":2,"rows":[{"id":"a"},{"id":"b"}]}` + "\n"; got != want {
		t.Errorf("json = %s\nwant    %s", got, want)
	}
	got = run(t, FormatJSON, Options{Columns: []string{"id"}}, nil, true)
	if want := `{"rows":[{"id":"a"},{"id":"b"}]}` + "\n"; got != want {
		t.Errorf("json without total = %s\nwant %s", got, want)
	}
}

func TestJSONLLines(t *testing.T) {
	got := run(t, FormatJSONL, Options{Columns: []string{"id", "n"}}, nil, true)
	want := "{\"id\":\"a\",\"n\":1}\n{\"id\":\"b\",\"n\":null}\n"
	if got != want {
		t.Errorf("jsonl = %q, want %q", got, want)
	}
}

func TestUnclosedStreamStaysUnclosed(t *testing.T) {
	got := run(t, FormatJSONA, Options{Columns: []string{"id"}}, nil, false)
	if want := `[{"id":"a"},{"id":"b"}`; got != want {
		t.Errorf("interrupted jsona = %s, want %s (no closing bracket)", got, want)
	}
}

func TestEmptyResult(t *testing.T) {
	var buf bytes.Buffer
	w, _ := New(&buf, FormatJSONA, Options{})
	total := int64(0)
	w.Head(&total)
	w.Close()
	if got := buf.String(); got != "[]\n" {
		t.Errorf("empty jsona = %q, want \"[]\\n\"", got)
	}
}
