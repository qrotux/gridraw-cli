package wql

import (
	"fmt"

	"github.com/qrotux/gridraw-cli/internal/wire"
)

// DNF flattens the AST into the server's disjunctive normal form: the outer
// slice is OR, each inner slice is AND. Distribution multiplies groups, so the
// wire limits are checked here rather than after the request is refused.
func DNF(n Node) ([][]Predicate, error) {
	if n == nil {
		return nil, nil
	}
	groups := distribute(n)
	if len(groups) > wire.MaxFilterGroups {
		return nil, &Error{
			Pos:  -1,
			Msg:  fmt.Sprintf("the filter expands to %d OR groups, and the server accepts %d", len(groups), wire.MaxFilterGroups),
			Hint: "each parenthesised `or` multiplies the number of groups; split the query or narrow it",
		}
	}
	for i, g := range groups {
		if len(g) > wire.MaxClausesPerGroup {
			return nil, &Error{
				Pos: -1,
				Msg: fmt.Sprintf("OR group %d holds %d conditions, and the server accepts %d per group", i+1, len(g), wire.MaxClausesPerGroup),
			}
		}
	}
	return groups, nil
}

// distribute returns the OR of ANDs for a node, in source order.
func distribute(n Node) [][]Predicate {
	switch t := n.(type) {
	case Predicate:
		return [][]Predicate{{t}}
	case Or:
		var out [][]Predicate
		for _, child := range t.Nodes {
			out = append(out, distribute(child)...)
		}
		return out
	case And:
		out := [][]Predicate{nil}
		for _, child := range t.Nodes {
			childGroups := distribute(child)
			next := make([][]Predicate, 0, len(out)*len(childGroups))
			for _, left := range out {
				for _, right := range childGroups {
					merged := make([]Predicate, 0, len(left)+len(right))
					merged = append(merged, left...)
					merged = append(merged, right...)
					next = append(next, merged)
				}
			}
			out = next
		}
		return out
	}
	return nil
}
