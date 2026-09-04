package wql

import (
	"strings"
	"testing"

	"github.com/qrotux/gridraw-cli/internal/wire"
)

func TestOrderForms(t *testing.T) {
	got, err := Order("rating,-email", testDescriptor())
	if err != nil {
		t.Fatal(err)
	}
	want := []wire.SortSpec{{Column: "rating", Dir: "asc"}, {Column: "email", Dir: "desc"}}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("order = %+v, want %+v", got, want)
	}
	got, err = Order("rating DESC, -email", testDescriptor())
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Dir != "desc" || got[1].Dir != "desc" {
		t.Errorf("order = %+v, want both desc", got)
	}
}

func TestOrderErrors(t *testing.T) {
	for _, tc := range []struct{ src, want string }{
		{"nope", "has no column"},
		{"role", "not sortable"},
		{"rating,rating", "twice"},
		{"rating sideways", "asc or desc"},
		{"rating,", "empty sort term"},
	} {
		if _, err := Order(tc.src, testDescriptor()); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%q: error = %v, want it to mention %q", tc.src, err, tc.want)
		}
	}
}

func TestColumns(t *testing.T) {
	got, err := Columns("email, rating", testDescriptor())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "email" || got[1] != "rating" {
		t.Errorf("columns = %v, want the given order preserved", got)
	}
	if _, err := Columns("emial", testDescriptor()); err == nil || !strings.Contains(err.Error(), `did you mean "email"`) {
		t.Errorf("error = %v, want a suggestion", err)
	}
	if got, err := Columns("  ", testDescriptor()); err != nil || got != nil {
		t.Errorf("empty columns = %v, %v; want nil, nil", got, err)
	}
}
