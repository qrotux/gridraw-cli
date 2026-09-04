package cli

import (
	"errors"
	"fmt"
	"testing"

	"github.com/qrotux/gridraw-cli/internal/config"
)

type fakeHTTP struct{ status int }

func (e *fakeHTTP) Error() string   { return fmt.Sprintf("status %d", e.status) }
func (e *fakeHTTP) HTTPStatus() int { return e.status }

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
		{"config package", &config.Error{Msg: "no configuration found"}, 3},
		{"wrapped config package", fmt.Errorf("from: %w", &config.Error{Msg: "no profile"}), 3},
		{"other", errors.New("boom"), 1},
		{"http 4xx", &fakeHTTP{status: 404}, 4},
		{"http 5xx", &fakeHTTP{status: 500}, 5},
		{"wrapped http 4xx", fmt.Errorf("from: %w", &fakeHTTP{status: 400}), 4},
	}
	for _, tc := range tests {
		if got := ExitCode(tc.err); got != tc.want {
			t.Errorf("%s: ExitCode = %d, want %d", tc.name, got, tc.want)
		}
	}
}
