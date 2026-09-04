package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/qrotux/gridraw-cli/internal/wire"
)

// tail is the parsed keyword tail of `gridraw from`.
type tail struct {
	Grid    string
	Columns string
	Where   string
	Order   string
	Search  string
	Limit   int // 0 when absent
	Page    int // 0 when absent
}

var tailKeywords = []string{"columns", "where", "order", "search", "limit", "page"}

// parseTail reads the positional keyword arguments cobra left after the flags.
func parseTail(args []string) (tail, error) {
	var t tail
	if len(args) == 0 {
		return t, &UsageError{Msg: "from needs a grid name: gridraw from GRID [columns \"…\"] [where \"…\"] …"}
	}
	t.Grid = args[0]
	if isKeyword(t.Grid) {
		return t, &UsageError{Msg: fmt.Sprintf("expected a grid name, got the keyword %q", t.Grid)}
	}
	seen := map[string]bool{}
	for i := 1; i < len(args); {
		word := strings.ToLower(args[i])
		if !isKeyword(word) {
			return t, &UsageError{Msg: fmt.Sprintf("unexpected %q; expected one of %s", args[i], strings.Join(tailKeywords, ", "))}
		}
		if seen[word] {
			return t, &UsageError{Msg: fmt.Sprintf("%s is given more than once", word)}
		}
		seen[word] = true
		if i+1 >= len(args) {
			return t, &UsageError{Msg: fmt.Sprintf("%s needs a value", word)}
		}
		value := args[i+1]
		switch word {
		case "columns":
			t.Columns = value
		case "where":
			t.Where = value
		case "order":
			t.Order = value
		case "search":
			t.Search = value
		case "limit":
			n, err := strconv.Atoi(value)
			if err != nil {
				return t, &UsageError{Msg: fmt.Sprintf("limit must be a whole number, got %q", value)}
			}
			if n < wire.MinPageSize || n > wire.MaxPageSize {
				return t, &UsageError{Msg: fmt.Sprintf("limit must be between %d and %d, got %d", wire.MinPageSize, wire.MaxPageSize, n)}
			}
			t.Limit = n
		case "page":
			n, err := strconv.Atoi(value)
			if err != nil {
				return t, &UsageError{Msg: fmt.Sprintf("page must be a whole number, got %q", value)}
			}
			if n < 1 {
				return t, &UsageError{Msg: fmt.Sprintf("page starts at 1, got %d", n)}
			}
			t.Page = n
		}
		i += 2
	}
	return t, nil
}

// isKeyword reports whether s matches one of the from tail keywords, case-insensitively.
func isKeyword(s string) bool {
	for _, k := range tailKeywords {
		if strings.EqualFold(s, k) {
			return true
		}
	}
	return false
}
