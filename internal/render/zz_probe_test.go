package render

import (
	"bytes"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func probe(t *testing.T, envelope bool, row map[string]any, cols []string) string {
	var buf bytes.Buffer
	f := FormatYAMLA
	if envelope {
		f = FormatYAML
	}
	w, err := New(&buf, f, Options{Columns: cols})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Head(nil); err != nil {
		t.Fatal(err)
	}
	if err := w.Row(row); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestProbeCases(t *testing.T) {
	cases := map[string]string{
		"lead_trail":  "\nhello\n\n",
		"quote_bs":    `he said "hi" \ back \\ done`,
		"ws_only":     "   ",
		"ws_tab":      "\t",
		"empty":       "",
		"long_single": strings.Repeat("word ", 60),
		"long_nospace": strings.Repeat("abcdefgh", 40),
		"long_quoted": "he said \"x\" " + strings.Repeat("word ", 60),
		"multi_long":  "\n" + strings.Repeat("word ", 60) + "\n",
		"cr":          "a\rb",
		"crlf":        "a\r\nb",
		"double_space": "a" + strings.Repeat("x", 70) + "  spaced  out",
		"unicode":     "héllo — ünïcode ✓",
		"tab_inline":  "a\tb",
		"leading_space": "  lead",
		"trailing_space": "trail  ",
	}
	for name, v := range cases {
		for _, env := range []bool{false, true} {
			got := probe(t, env, map[string]any{"a": v}, []string{"a"})
			var back any
			if err := yaml.Unmarshal([]byte(got), &back); err != nil {
				t.Errorf("%s env=%v: does not parse: %v\n%q", name, env, err, got)
				continue
			}
			var val any
			if env {
				m := back.(map[string]any)
				val = m["rows"].([]any)[0].(map[string]any)["a"]
			} else {
				val = back.([]any)[0].(map[string]any)["a"]
			}
			if s, ok := val.(string); !ok || s != v {
				t.Errorf("%s env=%v: round trip = %#v want %q\nout=%q", name, env, val, v, got)
			}
		}
	}
}
