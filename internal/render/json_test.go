package render

import (
	"bytes"
	"encoding/json"
	"errors"
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

func TestJSONWritersDoNotEscapeHTML(t *testing.T) {
	var buf bytes.Buffer
	w, err := New(&buf, FormatJSONL, Options{Columns: []string{"note"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Head(nil); err != nil {
		t.Fatal(err)
	}
	if err := w.Row(map[string]any{"note": "a<b&c>d"}); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if got, want := buf.String(), "{\"note\":\"a<b&c>d\"}\n"; got != want {
		t.Errorf("jsonl = %q, want %q", got, want)
	}
}

// TestWriterPropagatesAWriteError pins that a failing destination — a full disk,
// a closed pipe — surfaces instead of being swallowed mid-stream.
func TestWriterPropagatesAWriteError(t *testing.T) {
	for _, format := range []Format{FormatJSON, FormatJSONA, FormatJSONL, FormatYAML, FormatYAMLA, FormatCSV, FormatTSV} {
		w, err := New(failingWriter{}, format, Options{Columns: []string{"id"}})
		if err != nil {
			t.Fatal(err)
		}
		headErr := w.Head(nil)
		rowErr := w.Row(map[string]any{"id": "a"})
		if headErr == nil && rowErr == nil {
			t.Errorf("%s: neither Head nor Row reported the write error", format)
		}
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errWriteFailed }

var errWriteFailed = errors.New("no space left on device")

// TestOrderedFallsBackToSortedKeys pins the no-columns path: without a column
// list the row still has to come out in a stable order.
func TestOrderedFallsBackToSortedKeys(t *testing.T) {
	var buf bytes.Buffer
	w, _ := New(&buf, FormatJSONL, Options{})
	_ = w.Head(nil)
	_ = w.Row(map[string]any{"b": "2", "a": "1", "c": "3"})
	if got, want := buf.String(), "{\"a\":\"1\",\"b\":\"2\",\"c\":\"3\"}\n"; got != want {
		t.Errorf("jsonl = %q, want the keys sorted", got)
	}
}

// TestRowKeysFollowTheColumnList pins both halves of the column contract: a
// column the row lacks is null, a key the list omits is dropped.
func TestRowKeysFollowTheColumnList(t *testing.T) {
	var buf bytes.Buffer
	w, _ := New(&buf, FormatJSONL, Options{Columns: []string{"id", "missing"}})
	_ = w.Head(nil)
	_ = w.Row(map[string]any{"id": "a", "extra": "dropped"})
	if got, want := buf.String(), "{\"id\":\"a\",\"missing\":null}\n"; got != want {
		t.Errorf("jsonl = %q, want %q", got, want)
	}
}
