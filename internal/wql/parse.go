package wql

import (
	"strings"

	"github.com/qrotux/gridraw-cli/internal/wire"
)

// Node is a where AST node: either a Predicate or a boolean combination.
type Node interface{ isNode() }

// And is a conjunction of nodes.
type And struct{ Nodes []Node }

// Or is a disjunction of nodes.
type Or struct{ Nodes []Node }

// Predicate is one comparison, with values still untyped by the descriptor.
type Predicate struct {
	Field  string
	Op     wire.Op
	Values []any // 0 for shapeNone, 1 for shapeScalar, 2 for shapeRange, n for shapeList
	Pos    int
}

func (And) isNode()       {}
func (Or) isNode()        {}
func (Predicate) isNode() {}

// ParseWhere parses a where string. An empty string yields a nil Node.
func ParseWhere(src string) (Node, error) {
	if strings.TrimSpace(src) == "" {
		return nil, nil
	}
	lx := &lexer{src: src}
	toks, err := lx.all()
	if err != nil {
		return nil, err
	}
	p := &parser{src: src, toks: toks}
	node, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if p.peek().kind != tokEOF {
		return nil, errAt(src, p.peek().pos, "unexpected %q", p.peek().text)
	}
	return node, nil
}

type parser struct {
	src  string
	toks []token
	i    int
}

func (p *parser) peek() token    { return p.toks[p.i] }
func (p *parser) advance() token { t := p.toks[p.i]; p.i++; return t }

func (p *parser) atWord(word string) bool {
	t := p.peek()
	return t.kind == tokIdent && strings.EqualFold(t.text, word)
}

func (p *parser) atSymbol(s string) bool {
	t := p.peek()
	return t.kind == tokSymbol && t.text == s
}

func (p *parser) parseOr() (Node, error) {
	nodes := []Node{}
	for {
		n, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
		if !p.atWord("or") {
			break
		}
		p.advance()
	}
	if len(nodes) == 1 {
		return nodes[0], nil
	}
	return Or{Nodes: nodes}, nil
}

func (p *parser) parseAnd() (Node, error) {
	nodes := []Node{}
	for {
		n, err := p.parseFactor()
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
		if !p.atWord("and") {
			break
		}
		p.advance()
	}
	if len(nodes) == 1 {
		return nodes[0], nil
	}
	return And{Nodes: nodes}, nil
}

func (p *parser) parseFactor() (Node, error) {
	if p.atWord("not") && p.i+1 < len(p.toks) && p.toks[p.i+1].kind == tokSymbol && p.toks[p.i+1].text == "(" {
		return nil, &Error{
			Msg:    "`not` in front of a group is not supported",
			Pos:    p.peek().pos,
			Source: p.src,
			Hint:   "use a negative operator instead: !=, not contains, not in, not between, is not null",
		}
	}
	if p.atSymbol("(") {
		p.advance()
		n, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if !p.atSymbol(")") {
			return nil, errAt(p.src, p.peek().pos, "expected `)`")
		}
		p.advance()
		return n, nil
	}
	return p.parsePredicate()
}

func (p *parser) parsePredicate() (Node, error) {
	field := p.peek()
	if field.kind != tokIdent {
		return nil, errAt(p.src, field.pos, "expected a column name, got %q", field.text)
	}
	p.advance()
	spec, err := p.matchOperator()
	if err != nil {
		return nil, err
	}
	pred := Predicate{Field: field.text, Op: spec.op, Pos: field.pos}
	switch spec.shape {
	case shapeNone:
	case shapeScalar:
		v, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		pred.Values = []any{v}
	case shapeRange:
		lo, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		if !p.atWord("and") {
			return nil, errAt(p.src, p.peek().pos, "expected `and` between the two bounds of %s", spec.op)
		}
		p.advance()
		hi, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		pred.Values = []any{lo, hi}
	case shapeList:
		vals, err := p.parseList()
		if err != nil {
			return nil, err
		}
		pred.Values = vals
	}
	return pred, nil
}

// matchOperator consumes the longest matching operator, symbol or word form.
func (p *parser) matchOperator() (opSpec, error) {
	for _, spec := range opTable {
		if p.i+len(spec.words) > len(p.toks) {
			continue
		}
		ok := true
		for j, w := range spec.words {
			t := p.toks[p.i+j]
			if isSymbolWord(w) {
				ok = t.kind == tokSymbol && t.text == w
			} else {
				ok = t.kind == tokIdent && strings.EqualFold(t.text, w)
			}
			if !ok {
				break
			}
		}
		if ok {
			p.i += len(spec.words)
			return spec, nil
		}
	}
	t := p.peek()
	return opSpec{}, errAt(p.src, t.pos, "expected an operator, got %q", t.text)
}

func isSymbolWord(w string) bool { return !isIdentStart(w[0]) }

func (p *parser) parseValue() (any, error) {
	t := p.peek()
	switch {
	case t.kind == tokNumber, t.kind == tokString:
		p.advance()
		return t.val, nil
	case t.kind == tokIdent && strings.EqualFold(t.text, "true"):
		p.advance()
		return true, nil
	case t.kind == tokIdent && strings.EqualFold(t.text, "false"):
		p.advance()
		return false, nil
	}
	return nil, errAt(p.src, t.pos, "expected a value, got %q", t.text)
}

func (p *parser) parseList() ([]any, error) {
	if !p.atSymbol("(") {
		return nil, errAt(p.src, p.peek().pos, "expected `(` to start a list")
	}
	open := p.advance().pos
	var vals []any
	for {
		v, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		vals = append(vals, v)
		if p.atSymbol(",") {
			p.advance()
			continue
		}
		break
	}
	if !p.atSymbol(")") {
		return nil, errAt(p.src, p.peek().pos, "expected `,` or `)`")
	}
	p.advance()
	if len(vals) == 0 {
		return nil, errAt(p.src, open, "the list is empty")
	}
	return vals, nil
}
