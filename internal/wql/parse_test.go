package wql

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/qrotux/gridraw-cli/internal/wire"
)

func TestParsePrecedence(t *testing.T) {
	got, err := ParseWhere("a = 1 and b = 2 or c = 3")
	if err != nil {
		t.Fatal(err)
	}
	or, ok := got.(Or)
	if !ok || len(or.Nodes) != 2 {
		t.Fatalf("root = %#v, want an Or of two", got)
	}
	if _, ok := or.Nodes[0].(And); !ok {
		t.Errorf("left = %#v, want an And: `and` binds tighter", or.Nodes[0])
	}
}

func TestParseParenthesesRegroup(t *testing.T) {
	got, err := ParseWhere("a = 1 and (b = 2 or c = 3)")
	if err != nil {
		t.Fatal(err)
	}
	and, ok := got.(And)
	if !ok || len(and.Nodes) != 2 {
		t.Fatalf("root = %#v, want an And of two", got)
	}
	if _, ok := and.Nodes[1].(Or); !ok {
		t.Errorf("right = %#v, want an Or", and.Nodes[1])
	}
}

func TestParseOperators(t *testing.T) {
	tests := []struct {
		src    string
		op     wire.Op
		values []any
	}{
		{"a = 1", wire.OpEq, []any{json.Number("1")}},
		{"a != 'x'", wire.OpNeq, []any{"x"}},
		{"a <> 'x'", wire.OpNeq, []any{"x"}},
		{"a > 1", wire.OpGt, []any{json.Number("1")}},
		{"a >= 1", wire.OpGte, []any{json.Number("1")}},
		{"a < 1", wire.OpLt, []any{json.Number("1")}},
		{"a <= 1", wire.OpLte, []any{json.Number("1")}},
		{"a ~ 'x'", wire.OpContains, []any{"x"}},
		{"a contains 'x'", wire.OpContains, []any{"x"}},
		{"a !~ 'x'", wire.OpNotContains, []any{"x"}},
		{"a not contains 'x'", wire.OpNotContains, []any{"x"}},
		{"a starts 'x'", wire.OpStarts, []any{"x"}},
		{"a starts with 'x'", wire.OpStarts, []any{"x"}},
		{"a ends with 'x'", wire.OpEnds, []any{"x"}},
		{"a in ('x','y')", wire.OpIn, []any{"x", "y"}},
		{"a not in ('x')", wire.OpNotIn, []any{"x"}},
		{"a between 1 and 2", wire.OpBetween, []any{json.Number("1"), json.Number("2")}},
		{"a not between 1 and 2", wire.OpNotBetween, []any{json.Number("1"), json.Number("2")}},
		{"a is null", wire.OpIsNull, nil},
		{"a is not null", wire.OpIsNotNull, nil},
		{"a has any ('x')", wire.OpContainsAny, []any{"x"}},
		{"a has all ('x')", wire.OpContainsAll, []any{"x"}},
		{"a has only ('x')", wire.OpContainsOnly, []any{"x"}},
		{"a not has any ('x')", wire.OpNotContainsAny, []any{"x"}},
		{"a is empty", wire.OpIsEmpty, nil},
		{"a is not empty", wire.OpIsNotEmpty, nil},
		{"a = TRUE", wire.OpEq, []any{true}},
		{"a = false", wire.OpEq, []any{false}},
		{"a = `x`", wire.OpEq, []any{"x"}},
		{"a = \"x\"", wire.OpEq, []any{"x"}},
		{"a = 'it\\'s'", wire.OpEq, []any{"it's"}},
		{"a = -1.5", wire.OpEq, []any{json.Number("-1.5")}},
	}
	for _, tc := range tests {
		node, err := ParseWhere(tc.src)
		if err != nil {
			t.Errorf("%s: %v", tc.src, err)
			continue
		}
		pred, ok := node.(Predicate)
		if !ok {
			t.Errorf("%s: node = %#v, want a Predicate", tc.src, node)
			continue
		}
		if pred.Op != tc.op || !reflect.DeepEqual(pred.Values, tc.values) {
			t.Errorf("%s: op = %s values = %#v, want %s %#v", tc.src, pred.Op, pred.Values, tc.op, tc.values)
		}
	}
}

func TestBetweenDoesNotSwallowTheConjunction(t *testing.T) {
	node, err := ParseWhere("a between 1 and 2 and b = 3")
	if err != nil {
		t.Fatal(err)
	}
	and, ok := node.(And)
	if !ok || len(and.Nodes) != 2 {
		t.Fatalf("node = %#v, want an And of two", node)
	}
}

func TestParseErrors(t *testing.T) {
	for _, tc := range []struct{ src, want string }{
		{"not (a = 1)", "not"},
		{"a = 1 and", "expected a column name"},
		{"(a = 1", "expected `)`"},
		{"a in ()", "expected a value"},
		{"a 'x'", "expected an operator"},
		{"a = 'x", "unterminated string"},
		{"a between 1 2", "expected `and`"},
	} {
		_, err := ParseWhere(tc.src)
		if err == nil {
			t.Errorf("%q: want an error", tc.src)
			continue
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%q: error = %q, want it to mention %q", tc.src, err, tc.want)
		}
	}
}

func TestParseEmpty(t *testing.T) {
	node, err := ParseWhere("   ")
	if err != nil || node != nil {
		t.Errorf("ParseWhere(\"\") = %#v, %v; want nil, nil", node, err)
	}
}
