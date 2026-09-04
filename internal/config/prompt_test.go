package config

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestAuthHeader(t *testing.T) {
	tests := []struct {
		name, bearer, basic, want string
		wantErr                   bool
	}{
		{name: "bearer", bearer: "abc", want: "Bearer abc"},
		{name: "basic", basic: "user:pass", want: "Basic dXNlcjpwYXNz"},
		{name: "basic with colon in password", basic: "user:pa:ss", want: "Basic dXNlcjpwYTpzcw=="},
		{name: "both", bearer: "abc", basic: "user:pass", wantErr: true},
		{name: "basic without colon", basic: "userpass", wantErr: true},
		{name: "neither", want: ""},
	}
	for _, tc := range tests {
		got, err := AuthHeader(tc.bearer, tc.basic)
		if tc.wantErr {
			if err == nil {
				t.Errorf("%s: want an error", tc.name)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s: header = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestMaskHeader(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"Bearer abcdefghijkl", "Bearer ****ijkl"},
		{"Basic dXNlcjpwYXNz", "Basic ****YXNz"},
		{"", ""},
		{"Bearer ab", "Bearer ****"},
	} {
		if got := MaskHeader(tc.in); got != tc.want {
			t.Errorf("MaskHeader(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestAskEmptyLineTakesTheDefault(t *testing.T) {
	p := NewPrompter(strings.NewReader("\n  \n"), io.Discard)
	for _, want := range []string{"yaml", "yaml"} {
		got, err := p.Ask("Format", want)
		if err != nil {
			t.Fatalf("Ask: %v", err)
		}
		if got != want {
			t.Errorf("Ask = %q, want %q", got, want)
		}
	}
}

// A piped or non-tty run reaches EOF at the first question; the error must be
// classifiable so the CLI can tell the user to pass flags instead.
func TestAskOnEmptyInputReportsNoInput(t *testing.T) {
	p := NewPrompter(strings.NewReader(""), io.Discard)
	if _, err := p.Ask("Host", ""); !errors.Is(err, ErrNoInput) {
		t.Errorf("Ask error = %v, want ErrNoInput", err)
	}
}

func TestChooseReAsksOnAnInvalidAnswer(t *testing.T) {
	var out strings.Builder
	p := NewPrompter(strings.NewReader("xml\njson\n"), &out)
	got, err := p.Choose("Format", []string{"yaml", "json"}, "yaml")
	if err != nil {
		t.Fatalf("Choose: %v", err)
	}
	if got != "json" {
		t.Errorf("Choose = %q, want %q", got, "json")
	}
	if !strings.Contains(out.String(), `"xml" is not one of`) {
		t.Errorf("Choose did not report the invalid answer: %q", out.String())
	}
}
