package wql

import "github.com/qrotux/gridraw-cli/internal/wire"

// valueShape says how many values an operator takes.
type valueShape int

const (
	shapeNone   valueShape = iota // isNull, isEmpty …
	shapeScalar                   // eq, contains, gt …
	shapeList                     // in, containsAny …
	shapeRange                    // between a and b
)

// opSpec is one row of the operator table.
type opSpec struct {
	words []string // the literal words, lowercased; a symbol is a single "word"
	op    wire.Op
	shape valueShape
}

// opTable is ordered longest-first so that "not in" wins over "in" and
// "is not null" over "is null".
var opTable = []opSpec{
	{[]string{"not", "has", "any"}, wire.OpNotContainsAny, shapeList},
	{[]string{"is", "not", "null"}, wire.OpIsNotNull, shapeNone},
	{[]string{"is", "not", "empty"}, wire.OpIsNotEmpty, shapeNone},
	{[]string{"not", "between"}, wire.OpNotBetween, shapeRange},
	{[]string{"not", "contains"}, wire.OpNotContains, shapeScalar},
	{[]string{"not", "in"}, wire.OpNotIn, shapeList},
	{[]string{"starts", "with"}, wire.OpStarts, shapeScalar},
	{[]string{"ends", "with"}, wire.OpEnds, shapeScalar},
	{[]string{"has", "any"}, wire.OpContainsAny, shapeList},
	{[]string{"has", "all"}, wire.OpContainsAll, shapeList},
	{[]string{"has", "only"}, wire.OpContainsOnly, shapeList},
	{[]string{"is", "null"}, wire.OpIsNull, shapeNone},
	{[]string{"is", "empty"}, wire.OpIsEmpty, shapeNone},
	{[]string{"between"}, wire.OpBetween, shapeRange},
	{[]string{"contains"}, wire.OpContains, shapeScalar},
	{[]string{"starts"}, wire.OpStarts, shapeScalar},
	{[]string{"ends"}, wire.OpEnds, shapeScalar},
	{[]string{"in"}, wire.OpIn, shapeList},
	{[]string{"="}, wire.OpEq, shapeScalar},
	{[]string{"!="}, wire.OpNeq, shapeScalar},
	{[]string{"<>"}, wire.OpNeq, shapeScalar},
	{[]string{">="}, wire.OpGte, shapeScalar},
	{[]string{"<="}, wire.OpLte, shapeScalar},
	{[]string{">"}, wire.OpGt, shapeScalar},
	{[]string{"<"}, wire.OpLt, shapeScalar},
	{[]string{"~"}, wire.OpContains, shapeScalar},
	{[]string{"!~"}, wire.OpNotContains, shapeScalar},
}

// shapeOf reports how many values an operator takes.
func shapeOf(op wire.Op) valueShape {
	for _, s := range opTable {
		if s.op == op {
			return s.shape
		}
	}
	return shapeScalar
}

// spellings is the form each operator is printed in, chosen as the one a user
// is most likely to type. Several operators have more than one accepted
// spelling; the round-trip test pins that every entry here parses back.
var spellings = map[wire.Op]string{
	wire.OpEq:             "=",
	wire.OpNeq:            "!=",
	wire.OpGt:             ">",
	wire.OpGte:            ">=",
	wire.OpLt:             "<",
	wire.OpLte:            "<=",
	wire.OpContains:       "contains",
	wire.OpNotContains:    "not contains",
	wire.OpStarts:         "starts with",
	wire.OpEnds:           "ends with",
	wire.OpIn:             "in",
	wire.OpNotIn:          "not in",
	wire.OpBetween:        "between",
	wire.OpNotBetween:     "not between",
	wire.OpIsNull:         "is null",
	wire.OpIsNotNull:      "is not null",
	wire.OpContainsAny:    "has any",
	wire.OpContainsAll:    "has all",
	wire.OpContainsOnly:   "has only",
	wire.OpNotContainsAny: "not has any",
	wire.OpIsEmpty:        "is empty",
	wire.OpIsNotEmpty:     "is not empty",
}

// Spelling returns how an operator is written in a where clause. An operator
// the language does not know is returned as the server spells it.
func Spelling(op wire.Op) string {
	if s, ok := spellings[op]; ok {
		return s
	}
	return string(op)
}

// ValueHint describes how a value of this column type is written, for the
// types whose form is not obvious. It is empty when the literal speaks for
// itself.
func ValueHint(colType string) string {
	switch colType {
	case wire.TypeDate:
		return "YYYY-MM-DD, quoted"
	case wire.TypeTime:
		return "HH:MM:SS, quoted"
	case wire.TypeDatetime:
		return "RFC 3339, quoted"
	case wire.TypeDecimal:
		return "quoted, e.g. '19.99'"
	case wire.TypeUUID:
		return "quoted uuid"
	}
	return ""
}
