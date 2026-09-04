package wql

import (
	"strings"
	"testing"
)

// fields renders groups as "ab|ac" so a test reads at a glance.
func fields(groups [][]Predicate) string {
	var out []string
	for _, g := range groups {
		var s strings.Builder
		for _, p := range g {
			s.WriteString(p.Field)
		}
		out = append(out, s.String())
	}
	return strings.Join(out, "|")
}

func TestDNFDistributes(t *testing.T) {
	for _, tc := range []struct{ src, want string }{
		{"a = 1", "a"},
		{"a = 1 and b = 2", "ab"},
		{"a = 1 or b = 2", "a|b"},
		{"a = 1 and (b = 2 or c = 3)", "ab|ac"},
		{"(a = 1 or b = 2) and (c = 3 or d = 4)", "ac|ad|bc|bd"},
		{"a = 1 and b = 2 and c = 3", "abc"},
		{"a = 1 or (b = 2 and (c = 3 or d = 4))", "a|bc|bd"},
	} {
		node, err := ParseWhere(tc.src)
		if err != nil {
			t.Fatalf("%s: %v", tc.src, err)
		}
		groups, err := DNF(node)
		if err != nil {
			t.Fatalf("%s: %v", tc.src, err)
		}
		if got := fields(groups); got != tc.want {
			t.Errorf("%s: groups = %s, want %s", tc.src, got, tc.want)
		}
	}
}

func TestDNFNilIsNil(t *testing.T) {
	groups, err := DNF(nil)
	if err != nil || groups != nil {
		t.Errorf("DNF(nil) = %v, %v; want nil, nil", groups, err)
	}
}

func TestDNFGroupLimit(t *testing.T) {
	// Four two-way ors multiply to 16 groups, over the limit of 10.
	src := "(a = 1 or a = 2) and (b = 1 or b = 2) and (c = 1 or c = 2) and (d = 1 or d = 2)"
	node, err := ParseWhere(src)
	if err != nil {
		t.Fatal(err)
	}
	_, err = DNF(node)
	if err == nil {
		t.Fatal("want an error over the group limit")
	}
	if !strings.Contains(err.Error(), "16") || !strings.Contains(err.Error(), "10") {
		t.Errorf("error = %q, want it to name both counts", err)
	}
}

func TestDNFClauseLimit(t *testing.T) {
	var parts []string
	for i := 0; i < 21; i++ {
		parts = append(parts, "a = 1")
	}
	node, err := ParseWhere(strings.Join(parts, " and "))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DNF(node); err == nil || !strings.Contains(err.Error(), "20") {
		t.Fatalf("error = %v, want it to name the clause limit", err)
	}
}
