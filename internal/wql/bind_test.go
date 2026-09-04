package wql

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/qrotux/gridraw-cli/internal/wire"
)

func testDescriptor() *wire.Descriptor {
	all := func(ops ...wire.Op) *wire.Filter {
		f := &wire.Filter{}
		for _, o := range ops {
			f.Operators = append(f.Operators, wire.OperatorSpec{Op: o, Label: string(o)})
		}
		return f
	}
	return &wire.Descriptor{
		Name: "users", IDColumn: "id", PageSize: 25,
		Columns: []wire.Column{
			{Key: "email", Type: wire.TypeString, DefaultVisible: true, Sortable: true,
				Filter: all(wire.OpEq, wire.OpContains, wire.OpNotContains)},
			{Key: "rating", Type: wire.TypeNumber, DefaultVisible: true, Sortable: true,
				Filter: all(wire.OpEq, wire.OpGte, wire.OpBetween)},
			{Key: "balance", Type: wire.TypeDecimal, DefaultVisible: true,
				Filter: all(wire.OpGte, wire.OpBetween)},
			{Key: "isBanned", Type: wire.TypeBoolean, Filter: all(wire.OpEq)},
			{Key: "role", Type: wire.TypeEnum, Filter: all(wire.OpIn, wire.OpNotIn)},
			{Key: "id", Type: wire.TypeUUID, Filter: all(wire.OpEq, wire.OpIn)},
			{Key: "lastSeenAt", Type: wire.TypeDatetime, Filter: all(wire.OpGte, wire.OpIsNull)},
			{Key: "tags", Type: wire.TypeString, Array: true,
				Filter: all(wire.OpContainsAny, wire.OpIsEmpty)},
			{Key: "prefs", Type: wire.TypeJSON},
		},
	}
}

func bind(t *testing.T, src string) ([][]wire.Clause, error) {
	t.Helper()
	node, err := ParseWhere(src)
	if err != nil {
		return nil, err
	}
	groups, err := DNF(node)
	if err != nil {
		return nil, err
	}
	return Bind(groups, testDescriptor(), src)
}

func TestBindValueShapes(t *testing.T) {
	got, err := bind(t, "rating between 1 and 5 and lastSeenAt is null and role in ('admin','mod') and tags has any ('go')")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || len(got[0]) != 4 {
		t.Fatalf("groups = %#v", got)
	}
	raw, _ := json.Marshal(got[0])
	want := `[{"field":"rating","op":"between","value":[1,5]},` +
		`{"field":"lastSeenAt","op":"isNull"},` +
		`{"field":"role","op":"in","value":["admin","mod"]},` +
		`{"field":"tags","op":"containsAny","value":["go"]}]`
	if string(raw) != want {
		t.Errorf("clauses = %s\nwant     %s", raw, want)
	}
}

func TestBindDecimalCoercion(t *testing.T) {
	got, err := bind(t, "balance >= 19.99")
	if err != nil {
		t.Fatal(err)
	}
	if v := got[0][0].Value; v != "19.99" {
		t.Errorf("value = %#v (%T), want the string \"19.99\"", v, v)
	}
	got, err = bind(t, "balance between 1 and '2.50'")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(got[0][0].Value)
	if string(raw) != `["1","2.50"]` {
		t.Errorf("range = %s, want both bounds as strings", raw)
	}
}

func TestBindUUIDLowercased(t *testing.T) {
	got, err := bind(t, "id = 'AABBCCDD-0000-0000-0000-000000000000'")
	if err != nil {
		t.Fatal(err)
	}
	if got[0][0].Value != "aabbccdd-0000-0000-0000-000000000000" {
		t.Errorf("value = %v, want it lowercased", got[0][0].Value)
	}
}

func TestBindErrors(t *testing.T) {
	for _, tc := range []struct{ src, want string }{
		{"emial = 'x'", `did you mean "email"`},
		{"email >= 'x'", "does not offer the operator gte"},
		{"prefs = 'x'", "cannot be filtered"},
		{"rating = 'four'", "is of type number"},
		{"email = 4", "is of type string"},
		{"isBanned = 'yes'", "is of type boolean"},
		{"balance >= true", "is of type decimal"},
	} {
		_, err := bind(t, tc.src)
		if err == nil {
			t.Errorf("%q: want an error", tc.src)
			continue
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%q: error = %q, want it to mention %q", tc.src, err, tc.want)
		}
	}
}
