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
