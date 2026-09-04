package cli

import (
	"strings"
	"testing"

	"github.com/spf13/pflag"
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
		{[]string{""}, "needs a grid name"},
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

// TestSplitArgs pins the split that lets `order -id` through: pflag would read
// it as a shorthand cluster, so the keyword takes its value first.
func TestSplitArgs(t *testing.T) {
	flags := pflag.NewFlagSet("from", pflag.ContinueOnError)
	flags.StringP("output", "o", "", "")
	flags.String("null-val", "", "")
	flags.BoolP("all", "a", false, "") // a boolean shorthand, so -ao csv is a cluster
	for _, tc := range []struct {
		args       []string
		tail, flag string
	}{
		{[]string{"users", "order", "-id"}, "users order -id", ""},
		{[]string{"users", "-o", "csv", "order", "-id", "--all"}, "users order -id", "-o csv --all"},
		{[]string{"users", "-ocsv", "limit", "5"}, "users limit 5", "-ocsv"},
		{[]string{"users", "--null-val=-", "where", "a = 1"}, "users where a = 1", "--null-val=-"},
		{[]string{"users", "--", "order", "-id"}, "users order -id", ""},
		{[]string{"users", "--nosuch", "limit", "5"}, "users limit 5", "--nosuch"},
		{[]string{"users", "-ao", "csv", "limit", "5"}, "users limit 5", "-ao csv"},
	} {
		tailArgs, flagArgs := splitArgs(tc.args, flags)
		if got := strings.Join(tailArgs, " "); got != tc.tail {
			t.Errorf("%v: tail = %q, want %q", tc.args, got, tc.tail)
		}
		if got := strings.Join(flagArgs, " "); got != tc.flag {
			t.Errorf("%v: flags = %q, want %q", tc.args, got, tc.flag)
		}
	}
}
