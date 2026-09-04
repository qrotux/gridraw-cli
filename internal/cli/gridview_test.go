package cli

import (
	"strings"
	"testing"

	"github.com/qrotux/gridraw-cli/internal/wire"
)

func viewFixture() *wire.Descriptor {
	ops := func(list ...wire.Op) *wire.Filter {
		f := &wire.Filter{}
		for _, o := range list {
			f.Operators = append(f.Operators, wire.OperatorSpec{Op: o, Label: "a label the view drops"})
		}
		return f
	}
	role := ops(wire.OpIn, wire.OpNotIn)
	role.EnumValues = []wire.EnumValue{{Value: "user", Label: "User"}, {Value: "admin", Label: "Admin"}}
	role.Widget = "tags"
	return &wire.Descriptor{
		Name: "users", Description: "Application users", IDColumn: "id", PageSize: 25,
		DefaultSort: wire.SortSpec{Column: "createdAt", Dir: "desc"},
		Search:      &wire.Search{Columns: []string{"Email"}},
		Columns: []wire.Column{
			{Key: "email", Title: "Email", Description: "Login email", Type: wire.TypeString,
				Sortable: true, DefaultVisible: true,
				Filter: ops(wire.OpEq, wire.OpContains, wire.OpNotContains)},
			{Key: "role", Title: "Role", Type: wire.TypeEnum, Sortable: true, Filter: role},
			{Key: "price", Type: wire.TypeDecimal, Filter: ops(wire.OpGte, wire.OpBetween)},
			{Key: "slot", Type: wire.TypeTime, Step: 900, Filter: ops(wire.OpEq)},
			{Key: "ticks", Type: wire.TypeTime, Step: 1, Filter: ops(wire.OpEq)},
			{Key: "tags", Type: wire.TypeString, Array: true, Filter: ops(wire.OpContainsAny, wire.OpIsEmpty)},
			{Key: "prefs", Type: wire.TypeJSON},
		},
	}
}

func renderFixture(t *testing.T, format string) string {
	t.Helper()
	body, err := renderView(gridView(viewFixture()), format)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestGridViewYAML(t *testing.T) {
	want := `name: users
description: Application users
idColumn: id
pageSize: 25
defaultSort: createdAt desc
search: [Email]
columns:
    - key: email
      title: Email
      type: string
      description: Login email
      filters: [=, contains, not contains]
      sortable: true
      visible: true
    - key: role
      title: Role
      type: enum
      enum: {user: User, admin: Admin}
      filters: [in, not in]
      sortable: true
    - key: price
      type: decimal
      value: quoted, e.g. '19.99'
      filters: ['>=', between]
    - key: slot
      type: time
      value: HH:MM:SS, quoted
      step: 900
      filters: [=]
    - key: ticks
      type: time
      value: HH:MM:SS, quoted
      filters: [=]
    - key: tags
      type: string[]
      filters: [has any, is empty]
    - key: prefs
      type: json
`
	if got := renderFixture(t, "yaml"); got != want {
		t.Errorf("view =\n%s\nwant\n%s", got, want)
	}
}

func TestGridViewKeepsFieldOrderInJSON(t *testing.T) {
	got := renderFixture(t, "json")
	// encoding/json sorts a map's keys; the object type exists to stop that,
	// and column order is the descriptor's declaration order.
	for _, pair := range [][2]string{
		{`"name"`, `"idColumn"`},
		{`"idColumn"`, `"pageSize"`},
		{`"key": "email"`, `"key": "role"`},
		{`"key": "tags"`, `"key": "prefs"`},
		{`"user": "User"`, `"admin": "Admin"`},
	} {
		if strings.Index(got, pair[0]) >= strings.Index(got, pair[1]) {
			t.Errorf("%s should come before %s", pair[0], pair[1])
		}
	}
}

func TestGridViewOmissions(t *testing.T) {
	got := renderFixture(t, "yaml")
	for _, absent := range []string{
		"a label the view drops", // operator labels
		"widget",                 // a UI hint
		"step: 1",                // the default resolution constrains nothing
		"visible: false",         // a false flag says nothing
		"sortable: false",
		"skipTotal",         // absent on a counting grid
		"filters: []",       // a json column offers none
		"value: \"\"",       // no hint for the obvious types
		"description: \"\"", // no per-column description here
	} {
		if strings.Contains(got, absent) {
			t.Errorf("the view should not contain %q:\n%s", absent, got)
		}
	}
	if !strings.Contains(got, "key: prefs\n      type: json\n") {
		t.Error("a non-filterable column should still be listed, with its type")
	}
}

func TestGridViewSkipTotal(t *testing.T) {
	d := viewFixture()
	d.SkipTotal = true
	body, err := renderView(gridView(d), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "skipTotal: true") {
		t.Error("a grid that sends no total must say so; it changes how a client pages")
	}
}
