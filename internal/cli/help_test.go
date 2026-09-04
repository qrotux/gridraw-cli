package cli

import (
	"strings"
	"testing"

	"github.com/qrotux/gridraw-cli/internal/wql"
)

// operatorSection is the part of the help topic that lists the operators; the
// rest is prose in which a short word like "in" occurs by accident.
func operatorSection(t *testing.T) string {
	t.Helper()
	start := strings.Index(whereHelp, "OPERATORS")
	end := strings.Index(whereHelp, "LITERALS")
	if start < 0 || end < 0 || end < start {
		t.Fatal("the help topic has lost its OPERATORS section")
	}
	return whereHelp[start:end]
}

// TestWhereHelpListsEverySpelling pins the topic against the language itself:
// an operator the parser learns without a line here would ship undocumented.
func TestWhereHelpListsEverySpelling(t *testing.T) {
	section := operatorSection(t)
	for op, written := range wql.Spellings() {
		// A spelling is documented only as a table row, which starts a line.
		if !strings.Contains(section, "\n  "+written) {
			t.Errorf("the operator table has no row for %s (%q)", op, written)
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
