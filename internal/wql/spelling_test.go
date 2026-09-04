package wql

import (
	"testing"

	"github.com/qrotux/gridraw-cli/internal/wire"
)

// TestSpellingsRoundTrip pins the contract that makes `gridraw grid` usable:
// every operator the CLI prints must parse back to the operator it names, so a
// spelling copied out of a descriptor listing works verbatim in a where clause.
func TestSpellingsRoundTrip(t *testing.T) {
	values := map[valueShape]string{
		shapeNone:   "",
		shapeScalar: " 1",
		shapeList:   " (1)",
		shapeRange:  " 1 and 2",
	}
	seen := map[wire.Op]bool{}
	for _, spec := range opTable {
		if seen[spec.op] {
			continue
		}
		seen[spec.op] = true
		written := Spelling(spec.op)
		if written == string(spec.op) && spellings[spec.op] == "" {
			t.Errorf("operator %s has no spelling", spec.op)
			continue
		}
		node, err := ParseWhere("col " + written + values[spec.shape])
		if err != nil {
			t.Errorf("%s: printed as %q, which does not parse: %v", spec.op, written, err)
			continue
		}
		pred, ok := node.(Predicate)
		if !ok {
			t.Errorf("%s: %q parsed to %T, want a Predicate", spec.op, written, node)
			continue
		}
		if pred.Op != spec.op {
			t.Errorf("%s: printed as %q, which parses back as %s", spec.op, written, pred.Op)
		}
	}
}

func TestValueHint(t *testing.T) {
	if ValueHint(wire.TypeString) != "" {
		t.Error("a string needs no hint")
	}
	if ValueHint(wire.TypeDecimal) == "" || ValueHint(wire.TypeDate) == "" {
		t.Error("decimal and date are exactly the types a hint is for")
	}
}
