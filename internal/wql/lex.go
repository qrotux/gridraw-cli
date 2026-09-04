package wql

import (
	"encoding/json"
	"strings"
	"unicode"
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
	for l.pos < len(l.src) && unicode.IsSpace(rune(l.src[l.pos])) {
		l.pos++
	}
	if l.pos >= len(l.src) {
		return token{kind: tokEOF, pos: l.pos}, nil
	}
	start := l.pos
	c := l.src[l.pos]
	switch {
	case c == '\'' || c == '`' || c == '"':
		return l.lexString(c)
	case c == '(' || c == ')' || c == ',':
		l.pos++
		return token{kind: tokSymbol, text: string(c), pos: start}, nil
	case strings.ContainsRune("=<>!~", rune(c)):
		return l.lexSymbol()
	case c == '-' || c == '+' || (c >= '0' && c <= '9'):
		return l.lexNumber()
	case isIdentStart(c):
		for l.pos < len(l.src) && isIdentPart(l.src[l.pos]) {
			l.pos++
		}
		return token{kind: tokIdent, text: l.src[start:l.pos], pos: start}, nil
	}
	return token{}, errAt(l.src, start, "unexpected character %q", string(c))
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
			sb.WriteByte(l.src[l.pos+1])
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

func isIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isIdentPart(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}
