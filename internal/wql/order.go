package wql

import (
	"fmt"
	"strings"

	"github.com/qrotux/gridraw-cli/internal/wire"
)

// Order parses the order clause: "rating,-email" or "rating desc, email asc".
// A leading minus is the short form of desc.
func Order(src string, d *wire.Descriptor) ([]wire.SortSpec, error) {
	if strings.TrimSpace(src) == "" {
		return nil, nil
	}
	var out []wire.SortSpec
	seen := map[string]bool{}
	for _, part := range strings.Split(src, ",") {
		term := strings.TrimSpace(part)
		if term == "" {
			return nil, &Error{Pos: -1, Msg: fmt.Sprintf("empty sort term in %q", src)}
		}
		dir := "asc"
		if strings.HasPrefix(term, "-") {
			dir, term = "desc", strings.TrimSpace(term[1:])
			if term == "" {
				return nil, &Error{Pos: -1, Msg: fmt.Sprintf("the sort term %q names no column", strings.TrimSpace(part))}
			}
		}
		if fields := strings.Fields(term); len(fields) == 2 {
			if dir == "desc" {
				return nil, &Error{Pos: -1, Msg: fmt.Sprintf("the sort term %q gives the direction twice", strings.TrimSpace(part))}
			}
			term = fields[0]
			switch strings.ToLower(fields[1]) {
			case "asc":
				dir = "asc"
			case "desc":
				dir = "desc"
			default:
				return nil, &Error{Pos: -1, Msg: fmt.Sprintf("sort direction %q is not asc or desc", fields[1])}
			}
		} else if len(fields) > 2 {
			return nil, &Error{Pos: -1, Msg: fmt.Sprintf("cannot read the sort term %q", strings.TrimSpace(part))}
		}

		col := d.Column(term)
		if col == nil {
			return nil, &Error{Pos: -1, Msg: fmt.Sprintf("grid %q has no column %q", d.Name, term), Hint: suggest(term, d)}
		}
		if !col.Sortable {
			return nil, &Error{Pos: -1, Msg: fmt.Sprintf("column %q is not sortable", term)}
		}
		if seen[term] {
			return nil, &Error{Pos: -1, Msg: fmt.Sprintf("column %q appears twice in the sort", term)}
		}
		seen[term] = true
		out = append(out, wire.SortSpec{Column: term, Dir: dir})
	}
	if len(out) > wire.MaxSortColumns {
		return nil, &Error{Pos: -1, Msg: fmt.Sprintf("the sort has %d columns, and the server accepts %d", len(out), wire.MaxSortColumns)}
	}
	return out, nil
}

// Columns parses the columns clause and checks every key against the grid.
func Columns(src string, d *wire.Descriptor) ([]string, error) {
	if strings.TrimSpace(src) == "" {
		return nil, nil
	}
	var out []string
	seen := map[string]bool{}
	for _, part := range strings.Split(src, ",") {
		key := strings.TrimSpace(part)
		if key == "" {
			return nil, &Error{Pos: -1, Msg: fmt.Sprintf("empty column name in %q", src)}
		}
		if d.Column(key) == nil {
			return nil, &Error{Pos: -1, Msg: fmt.Sprintf("grid %q has no column %q", d.Name, key), Hint: suggest(key, d)}
		}
		if seen[key] {
			return nil, &Error{Pos: -1, Msg: fmt.Sprintf("column %q is listed twice", key)}
		}
		seen[key] = true
		out = append(out, key)
	}
	return out, nil
}
