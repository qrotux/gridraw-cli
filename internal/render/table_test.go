package render

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestCSVQuotingAndHeader(t *testing.T) {
	var buf bytes.Buffer
	w, _ := New(&buf, FormatCSV, Options{Columns: []string{"id", "note", "tags"}})
	w.Head(nil)
	w.Row(map[string]any{"id": "a", "note": `he said "hi", loudly`, "tags": []any{"go"}})
	w.Row(map[string]any{"id": "b", "note": nil, "tags": []any{}})
	w.Close()
	want := "id,note,tags\r\n" +
		`a,"he said ""hi"", loudly","[""go""]"` + "\r\n" +
		"b,,[]\r\n"
	if got := buf.String(); got != want {
		t.Errorf("csv =\n%q\nwant\n%q", got, want)
	}
}

func TestCSVNoHeaderAndNullVal(t *testing.T) {
	var buf bytes.Buffer
	w, _ := New(&buf, FormatCSV, Options{Columns: []string{"id", "n"}, NoHeader: true, NullVal: "NULL"})
	w.Head(nil)
	w.Row(map[string]any{"id": "a", "n": nil})
	w.Close()
	if got, want := buf.String(), "a,NULL\r\n"; got != want {
		t.Errorf("csv = %q, want %q", got, want)
	}
}

func TestTSVEscapes(t *testing.T) {
	var buf bytes.Buffer
	w, _ := New(&buf, FormatTSV, Options{Columns: []string{"id", "note", "n"}})
	w.Head(nil)
	w.Row(map[string]any{"id": "a", "note": "two\tcolumns\nand a line", "n": json.Number("4.10")})
	w.Close()
	want := "id\tnote\tn\r\n" + `a` + "\t" + `two\tcolumns\nand a line` + "\t4.10\r\n"
	if got := buf.String(); got != want {
		t.Errorf("tsv = %q, want %q", got, want)
	}
}
