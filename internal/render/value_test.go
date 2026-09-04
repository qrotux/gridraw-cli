package render

import (
	"encoding/json"
	"testing"
)

func TestCell(t *testing.T) {
	for _, tc := range []struct {
		in   any
		null string
		want string
	}{
		{nil, "", ""},
		{nil, "NULL", "NULL"},
		{"plain", "", "plain"},
		{json.Number("4.10"), "", "4.10"},
		{true, "", "true"},
		{[]any{"go", "cli"}, "", `["go","cli"]`},
		{map[string]any{"a": json.Number("1")}, "", `{"a":1}`},
	} {
		if got := Cell(tc.in, tc.null); got != tc.want {
			t.Errorf("Cell(%#v, %q) = %q, want %q", tc.in, tc.null, got, tc.want)
		}
	}
}

// TestCellDoesNotEscapeHTML pins that a cell's text survives as the server sent
// it: encoding/json escapes <, > and & by default, which would silently rewrite
// the value in csv, tsv and every JSON format alike.
func TestCellDoesNotEscapeHTML(t *testing.T) {
	got := Cell([]any{"a<b&c>d"}, "")
	if got != `["a<b&c>d"]` {
		t.Errorf("Cell = %s, want the text unescaped", got)
	}
}
