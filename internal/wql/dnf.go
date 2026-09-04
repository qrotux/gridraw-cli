package wql

import (
	"fmt"

	"github.com/qrotux/gridraw-cli/internal/wire"
)

// DNF flattens the AST into the server's disjunctive normal form: the outer
// slice is OR, each inner slice is AND. Distribution multiplies groups, so the
// group limit is enforced as the cross product grows rather than after it has
// already been built.
func DNF(n Node) ([][]Predicate, error) {
	if n == nil {
		return nil, nil
	}
	groups, err := distribute(n, true)
	if err != nil {
		return nil, err
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

// distribute returns the OR of ANDs for a node, in source order. isRoot marks
// the outermost call: only there, on the node's own last child, is a group
// count that has just tipped over the limit also the true final count: any
// nested node may still be multiplied further by an ancestor, so its overflow
// is reported as a lower bound instead.
func distribute(n Node, isRoot bool) ([][]Predicate, error) {
	switch t := n.(type) {
	case Predicate:
		return [][]Predicate{{t}}, nil
	case Or:
		var out [][]Predicate
		for i, child := range t.Nodes {
			childGroups, err := distribute(child, false)
			if err != nil {
				return nil, err
			}
			total := len(out) + len(childGroups)
			if total > wire.MaxFilterGroups {
				return nil, groupLimitError(total, isRoot && i == len(t.Nodes)-1)
			}
			out = append(out, childGroups...)
		}
		return out, nil
	case And:
		out := [][]Predicate{nil}
		for i, child := range t.Nodes {
			childGroups, err := distribute(child, false)
			if err != nil {
				return nil, err
			}
			// The next generation's size is a plain multiplication, so it is
			// known before allocating a single element of it.
			total := len(out) * len(childGroups)
			if total > wire.MaxFilterGroups {
				return nil, groupLimitError(total, isRoot && i == len(t.Nodes)-1)
			}
			next := make([][]Predicate, 0, total)
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
		return out, nil
	}
	return nil, nil
}

// groupLimitError reports a filter that expands past wire.MaxFilterGroups.
// final reports the true total when this was the last combination in the
// whole expression; otherwise more groups remained to be multiplied in, so
// only a lower bound is known and reported as such.
func groupLimitError(total int, final bool) error {
	limit := wire.MaxFilterGroups
	msg := fmt.Sprintf("the filter expands to more than %d OR groups, the server's limit", limit)
	if final {
		msg = fmt.Sprintf("the filter expands to %d OR groups, and the server accepts %d", total, limit)
	}
	return &Error{
		Pos:  -1,
		Msg:  msg,
		Hint: "each parenthesised `or` multiplies the number of groups; split the query or narrow it",
	}
}
