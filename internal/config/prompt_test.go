package config

import "testing"

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
