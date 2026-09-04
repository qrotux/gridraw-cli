package cli

import (
	"errors"

	"github.com/qrotux/gridraw-cli/internal/config"
)

// UsageError is a mistake in what the user typed: an unparseable where string,
// an unknown column, an operator a column does not offer.
type UsageError struct {
	Msg string
	Err error
}

func (e *UsageError) Error() string {
	if e.Err != nil {
		return e.Msg + ": " + e.Err.Error()
	}
	return e.Msg
}
func (e *UsageError) Unwrap() error { return e.Err }

// ConfigError is a missing, unreadable or invalid configuration.
type ConfigError struct {
	Msg string
	Err error
}

func (e *ConfigError) Error() string {
	if e.Err != nil {
		return e.Msg + ": " + e.Err.Error()
	}
	return e.Msg
}
func (e *ConfigError) Unwrap() error { return e.Err }

// statusError is satisfied by client.HTTPError; declaring it here keeps the
// exit codes in one package without importing the client into every caller.
type statusError interface{ HTTPStatus() int }

// ExitCode maps an error to the process exit code documented in README.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var usage *UsageError
	if errors.As(err, &usage) {
		return 2
	}
	var cfg *ConfigError
	if errors.As(err, &cfg) {
		return 3
	}
	var cfgPkg *config.Error
	if errors.As(err, &cfgPkg) {
		return 3
	}
	var st statusError
	if errors.As(err, &st) {
		if s := st.HTTPStatus(); s >= 500 {
			return 5
		}
		return 4
	}
	return 1
}
