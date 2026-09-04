package wql

import (
	"encoding/json"
	"strings"
	"unicode"
	"unicode/utf8"
)

type tokenKind int

const (
	tokEOF tokenKind = iota
	tokIdent
	tokNumber
	tokString
	tokSymbol // = != <> > >= < <= ~ !~ ( ) ,
)

type token struct {
	kind tokenKind
	text string
	val  any // json.Number for tokNumber, string for tokString
	pos  int
}

type lexer struct {
	src string
	pos int
}

func (l *lexer) all() ([]token, error) {
	var out []token
	for {
		t, err := l.next()
		if err != nil {
			return nil, err
		}
		out = append(out, t)
		if t.kind == tokEOF {
			return out, nil
		}
	}
}

func (l *lexer) next() (token, error) {
	for l.pos < len(l.src) {
		r, size := utf8.DecodeRuneInString(l.src[l.pos:])
		if !unicode.IsSpace(r) {
			break
		}
		l.pos += size
	}
	if l.pos >= len(l.src) {
		return token{kind: tokEOF, pos: l.pos}, nil
	}
	start := l.pos
	c := l.src[l.pos]
	r, size := utf8.DecodeRuneInString(l.src[l.pos:])
	switch {
	case c == '\'' || c == '`' || c == '"':
		return l.lexString(c)
	case c == '(' || c == ')' || c == ',':
		l.pos++
		return token{kind: tokSymbol, text: string(c), pos: start}, nil
	case strings.ContainsRune("=<>!~", r):
		return l.lexSymbol()
	case c == '-' || c == '+' || (c >= '0' && c <= '9'):
		return l.lexNumber()
	case isIdentStart(r):
		l.pos += size
		for l.pos < len(l.src) {
			r, size := utf8.DecodeRuneInString(l.src[l.pos:])
			if !isIdentPart(r) {
				break
			}
			l.pos += size
		}
		return token{kind: tokIdent, text: l.src[start:l.pos], pos: start}, nil
	}
	return token{}, errAt(l.src, start, "unexpected character %q", string(r))
}

func (l *lexer) lexSymbol() (token, error) {
	start := l.pos
	two := ""
	if l.pos+1 < len(l.src) {
		two = l.src[l.pos : l.pos+2]
	}
	switch two {
	case "!=", "<>", ">=", "<=", "!~":
		l.pos += 2
		return token{kind: tokSymbol, text: two, pos: start}, nil
	}
	one := string(l.src[l.pos])
	switch one {
	case "=", ">", "<", "~":
		l.pos++
		return token{kind: tokSymbol, text: one, pos: start}, nil
	}
	return token{}, errAt(l.src, start, "unexpected character %q", one)
}

func (l *lexer) lexNumber() (token, error) {
	start := l.pos
	l.pos++ // sign or first digit
	for l.pos < len(l.src) && (l.src[l.pos] == '.' || l.src[l.pos] == 'e' || l.src[l.pos] == 'E' ||
		l.src[l.pos] == '+' || l.src[l.pos] == '-' || (l.src[l.pos] >= '0' && l.src[l.pos] <= '9')) {
		l.pos++
	}
	text := l.src[start:l.pos]
	if !json.Valid([]byte(text)) {
		return token{}, errAt(l.src, start, "%q is not a number", text)
	}
	return token{kind: tokNumber, text: text, val: json.Number(text), pos: start}, nil
}

func (l *lexer) lexString(quote byte) (token, error) {
	start := l.pos
	l.pos++
	var sb strings.Builder
	for l.pos < len(l.src) {
		c := l.src[l.pos]
		switch c {
		case '\\':
			if l.pos+1 >= len(l.src) {
				return token{}, errAt(l.src, l.pos, "trailing backslash")
			}
			// A closed escape set, so a Windows path such as 'C:\new' is
			// reported rather than silently read as "C:new".
			esc := l.src[l.pos+1]
			if esc != '\'' && esc != '`' && esc != '"' && esc != '\\' {
				return token{}, errAt(l.src, l.pos, "unknown escape \\%c; the escapes are \\' \\` \\\" and \\\\", esc)
			}
			sb.WriteByte(esc)
			l.pos += 2
			continue
		case quote:
			l.pos++
			return token{kind: tokString, text: sb.String(), val: sb.String(), pos: start}, nil
		}
		sb.WriteByte(c)
		l.pos++
	}
	return token{}, errAt(l.src, start, "unterminated string")
}

// isIdentStart accepts any letter, not just ASCII: a column key is whatever
// string the descriptor publishes and the language has no quoting escape for
// identifiers.
func isIdentStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func isIdentPart(r rune) bool {
	return isIdentStart(r) || unicode.IsDigit(r)
}
