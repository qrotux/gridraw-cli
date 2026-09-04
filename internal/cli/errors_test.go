package cli

import (
	"errors"
	"fmt"
	"testing"
)

type fakeHTTP struct{ status int }

func (e *fakeHTTP) Error() string { return fmt.Sprintf("status %d", e.status) }

func TestExitCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, 0},
		{"usage", &UsageError{Msg: "bad where"}, 2},
		{"config", &ConfigError{Msg: "no profile"}, 3},
		{"wrapped usage", fmt.Errorf("from: %w", &UsageError{Msg: "bad"}), 2},
		{"other", errors.New("boom"), 1},
	}
	for _, tc := range tests {
		if got := ExitCode(tc.err); got != tc.want {
			t.Errorf("%s: ExitCode = %d, want %d", tc.name, got, tc.want)
		}
	}
}
