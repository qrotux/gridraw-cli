package cli

import (
	"strings"
	"testing"

	"github.com/qrotux/gridraw-cli/internal/wire"
)

// TestWhereHelpNamesEveryOperator pins the help topic against the wire
// operators: an operator added to the protocol without a line here would ship
// undocumented.
func TestWhereHelpNamesEveryOperator(t *testing.T) {
	ops := []wire.Op{
		wire.OpEq, wire.OpNeq, wire.OpContains, wire.OpNotContains, wire.OpStarts,
		wire.OpEnds, wire.OpGt, wire.OpGte, wire.OpLt, wire.OpLte, wire.OpBetween,
		wire.OpNotBetween, wire.OpIn, wire.OpNotIn, wire.OpIsNull, wire.OpIsNotNull,
		wire.OpContainsAny, wire.OpContainsAll, wire.OpContainsOnly,
		wire.OpNotContainsAny, wire.OpIsEmpty, wire.OpIsNotEmpty,
	}
	for _, op := range ops {
		if !strings.Contains(whereHelp, string(op)) {
			t.Errorf("the where help topic never mentions the operator %s", op)
		}
	}
}

func TestWhereTopicIsAHelpTopicOnly(t *testing.T) {
	cmd := newWhereTopic()
	if cmd.Runnable() {
		t.Error("the where topic must not be runnable; it is help text, not a command")
	}
	if cmd.Long == "" {
		t.Error("the where topic has no long help to print")
	}
}
