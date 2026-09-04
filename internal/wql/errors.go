// Package wql parses the gridraw CLI query language: the where and order
// clauses of `gridraw from`.
package wql

import (
	"fmt"
	"unicode/utf8"
)

// Error is a mistake in what the user typed. Pos is a byte offset into the
// source string, or -1 when the error is about the whole clause.
type Error struct {
	Msg    string
	Pos    int
	Source string
	Hint   string
}

func (e *Error) Error() string {
	out := e.Msg
	if e.Pos >= 0 && e.Source != "" {
		out = fmt.Sprintf("%s at position %d: %s", e.Msg, e.Pos, caret(e.Source, e.Pos))
	}
	if e.Hint != "" {
		out += "\n" + e.Hint
	}
	return out
}

// caret quotes the source with a marker under the offending rune. The marker
// is padded by rune count, not by Pos, so a multi-byte rune earlier in the
// source does not push it out of column.
func caret(src string, pos int) string {
	if pos > len(src) {
		pos = len(src)
	}
	return fmt.Sprintf("%s\n%*s^", src, utf8.RuneCountInString(src[:pos]), "")
}

func errAt(src string, pos int, format string, args ...any) *Error {
	return &Error{Msg: fmt.Sprintf(format, args...), Pos: pos, Source: src}
}
