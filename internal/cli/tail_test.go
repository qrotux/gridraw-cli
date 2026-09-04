package cli

import (
	"strings"
	"testing"
)

func TestTailFull(t *testing.T) {
	got, err := parseTail([]string{"users", "columns", "a,b", "where", "x = 1", "order", "a,-b", "search", "ivan", "limit", "50", "page", "2"})
	if err != nil {
		t.Fatal(err)
	}
	want := tail{Grid: "users", Columns: "a,b", Where: "x = 1", Order: "a,-b", Search: "ivan", Limit: 50, Page: 2}
	if got != want {
		t.Errorf("tail = %+v, want %+v", got, want)
	}
}

func TestTailAnyOrder(t *testing.T) {
	got, err := parseTail([]string{"users", "limit", "10", "where", "x = 1"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Limit != 10 || got.Where != "x = 1" {
		t.Errorf("tail = %+v", got)
	}
}

func TestTailErrors(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{}, "needs a grid name"},
		{[]string{"where", "x = 1"}, "expected a grid name"},
		{[]string{"users", "wehre", "x"}, "unexpected"},
		{[]string{"users", "where"}, "needs a value"},
		{[]string{"users", "where", "a", "where", "b"}, "more than once"},
		{[]string{"users", "limit", "x"}, "whole number"},
		{[]string{"users", "limit", "101"}, "between 1 and 100"},
		{[]string{"users", "page", "0"}, "starts at 1"},
	} {
		_, err := parseTail(tc.args)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%v: error = %v, want it to mention %q", tc.args, err, tc.want)
		}
	}
}
