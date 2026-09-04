package wql

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/qrotux/gridraw-cli/internal/wire"
)

// Bind validates the groups against the descriptor and converts every value to
// the shape the server accepts.
func Bind(groups [][]Predicate, d *wire.Descriptor, src string) ([][]wire.Clause, error) {
	if groups == nil {
		return nil, nil
	}
	out := make([][]wire.Clause, 0, len(groups))
	for _, g := range groups {
		clauses := make([]wire.Clause, 0, len(g))
		for _, p := range g {
			c, err := bindOne(p, d, src)
			if err != nil {
				return nil, err
			}
			clauses = append(clauses, c)
		}
		out = append(out, clauses)
	}
	return out, nil
}

func bindOne(p Predicate, d *wire.Descriptor, src string) (wire.Clause, error) {
	col := d.Column(p.Field)
	if col == nil {
		return wire.Clause{}, &Error{
			Msg:    fmt.Sprintf("grid %q has no column %q", d.Name, p.Field),
			Pos:    p.Pos,
			Source: src,
			Hint:   suggest(p.Field, d),
		}
	}
	if col.Filter == nil {
		return wire.Clause{}, &Error{
			Msg:    fmt.Sprintf("column %q cannot be filtered", p.Field),
			Pos:    p.Pos,
			Source: src,
		}
	}
	if !col.Allows(p.Op) {
		return wire.Clause{}, &Error{
			Msg:    fmt.Sprintf("column %q of type %s does not offer the operator %s", p.Field, columnType(col), p.Op),
			Pos:    p.Pos,
			Source: src,
			Hint:   "it offers " + strings.Join(operatorNames(col), ", "),
		}
	}

	clause := wire.Clause{Field: p.Field, Op: p.Op}
	switch shapeOf(p.Op) {
	case shapeNone:
		if len(p.Values) != 0 {
			return wire.Clause{}, errAt(src, p.Pos, "operator %s takes no value", p.Op)
		}
	case shapeScalar:
		v, err := convert(p.Values[0], col, p, src)
		if err != nil {
			return wire.Clause{}, err
		}
		clause.Value = v
	case shapeRange:
		lo, err := convert(p.Values[0], col, p, src)
		if err != nil {
			return wire.Clause{}, err
		}
		hi, err := convert(p.Values[1], col, p, src)
		if err != nil {
			return wire.Clause{}, err
		}
		clause.Value = []any{lo, hi}
	case shapeList:
		vals := make([]any, 0, len(p.Values))
		for _, raw := range p.Values {
			v, err := convert(raw, col, p, src)
			if err != nil {
				return wire.Clause{}, err
			}
			vals = append(vals, v)
		}
		clause.Value = vals
	}
	return clause, nil
}

// convert types one literal for the column. The single silent coercion is a
// JSON number written for a decimal column: the server rejects numbers there,
// and quoting money by hand is the mistake everyone makes.
func convert(v any, col *wire.Column, p Predicate, src string) (any, error) {
	switch col.Type {
	case wire.TypeNumber:
		n, ok := v.(json.Number)
		if !ok {
			return nil, typeErr(v, col, p, src, "a number, written without quotes")
		}
		return n, nil
	case wire.TypeDecimal:
		switch t := v.(type) {
		case string:
			return t, nil
		case json.Number:
			return t.String(), nil // the coercion, documented in README
		}
		return nil, typeErr(v, col, p, src, "a decimal, e.g. '19.99'")
	case wire.TypeBoolean:
		b, ok := v.(bool)
		if !ok {
			return nil, typeErr(v, col, p, src, "true or false")
		}
		return b, nil
	case wire.TypeUUID:
		s, ok := v.(string)
		if !ok {
			return nil, typeErr(v, col, p, src, "a quoted uuid")
		}
		return strings.ToLower(s), nil
	case wire.TypeEnum:
		s, ok := v.(string)
		if !ok {
			return nil, typeErr(v, col, p, src, "a quoted string")
		}
		return s, checkEnum(s, col, p, src)
	case wire.TypeString, wire.TypeDate, wire.TypeTime, wire.TypeDatetime:
		s, ok := v.(string)
		if !ok {
			return nil, typeErr(v, col, p, src, "a quoted string")
		}
		return s, nil
	default:
		return nil, errAt(src, p.Pos, "column %q has the type %s, which this version does not know how to filter", col.Key, columnType(col))
	}
}

// checkEnum rejects a value the descriptor does not publish. A descriptor may
// omit enumValues, and then any string goes.
func checkEnum(s string, col *wire.Column, p Predicate, src string) error {
	allowed := enumValues(col)
	if len(allowed) == 0 {
		return nil
	}
	for _, v := range allowed {
		if v == s {
			return nil
		}
	}
	hint := "allowed values: " + strings.Join(allowed, ", ")
	if best := closest(s, allowed); best != "" {
		hint = fmt.Sprintf("did you mean %q?", best)
	}
	return &Error{
		Msg:    fmt.Sprintf("column %q has no value %q", col.Key, s),
		Pos:    p.Pos,
		Source: src,
		Hint:   hint,
	}
}

func enumValues(col *wire.Column) []string {
	if col.Filter == nil {
		return nil
	}
	out := make([]string, 0, len(col.Filter.EnumValues))
	for _, e := range col.Filter.EnumValues {
		out = append(out, e.Value)
	}
	return out
}

func typeErr(v any, col *wire.Column, p Predicate, src, want string) error {
	return &Error{
		Msg:    fmt.Sprintf("column %q is of type %s, so %s is not a valid value", col.Key, columnType(col), literal(v)),
		Pos:    p.Pos,
		Source: src,
		Hint:   "write " + want,
	}
}

func literal(v any) string {
	switch t := v.(type) {
	case string:
		return fmt.Sprintf("%q", t)
	case json.Number:
		return t.String()
	case bool:
		return fmt.Sprintf("%t", t)
	}
	return fmt.Sprintf("%v", v)
}

func columnType(col *wire.Column) string {
	if col.Array {
		return col.Type + "[]"
	}
	return col.Type
}

func operatorNames(col *wire.Column) []string {
	out := make([]string, 0, len(col.Filter.Operators))
	for _, o := range col.Filter.Operators {
		// Spelled the way a where clause takes them, so the hint can be copied
		// into the query that failed.
		out = append(out, Spelling(o.Op))
	}
	return out
}

// suggest offers the closest column key when the typo is small.
func suggest(field string, d *wire.Descriptor) string {
	best := closest(field, columnKeys(d))
	if best == "" {
		return "known columns: " + strings.Join(columnKeys(d), ", ")
	}
	return fmt.Sprintf("did you mean %q?", best)
}

// closest returns the candidate a plausible typo away from word, or "". The
// budget is relative to the two lengths: on an absolute one, every two-letter
// word would "mean" every short key.
func closest(word string, candidates []string) string {
	best, bestDist := "", 0
	for _, c := range candidates {
		dist := editDistance(strings.ToLower(word), strings.ToLower(c))
		if dist >= 3 || 3*dist > len(word)+len(c) {
			continue
		}
		if best == "" || dist < bestDist {
			best, bestDist = c, dist
		}
	}
	return best
}

func columnKeys(d *wire.Descriptor) []string {
	out := make([]string, 0, len(d.Columns))
	for _, c := range d.Columns {
		out = append(out, c.Key)
	}
	return out
}

// editDistance is the Levenshtein distance, used only for the suggestion.
func editDistance(a, b string) int {
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(cur[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}
