package cli

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

// reporter writes the counts and the progress line on stderr. stdout carries
// data only, so nothing here ever touches it.
type reporter struct {
	out      io.Writer
	quiet    bool
	show     bool // draw progress at all: stderr is a terminal, or --progress
	live     bool // redraw one line instead of printing one per page
	pageSize int  // set by from.go; the progress line derives the page count from it
	printed  bool
}

// newReporter decides whether progress is drawn: on a terminal by default,
// always with --progress, never with --quiet.
func newReporter(out io.Writer, quiet, force bool) *reporter {
	tty := false
	if f, ok := out.(*os.File); ok {
		tty = term.IsTerminal(int(f.Fd()))
	}
	return &reporter{out: out, quiet: quiet, show: tty || force, live: tty && !force}
}

// Total prints the count once, after the first page.
func (r *reporter) Total(total *int64) {
	if r.quiet || total == nil || r.printed {
		return
	}
	r.printed = true
	fmt.Fprintf(r.out, "Total: %d\n", *total)
}

// Page reports progress after a page has been written.
func (r *reporter) Page(page int, rows int64, total *int64) {
	if r.quiet || !r.show {
		return
	}
	line := fmt.Sprintf("page %d · %d rows", page, rows)
	if total != nil && r.pageSize > 0 {
		pages := (*total + int64(r.pageSize) - 1) / int64(r.pageSize)
		line = fmt.Sprintf("page %d/%d · %d/%d rows", page, pages, rows, *total)
	}
	if r.live {
		fmt.Fprintf(r.out, "\r\033[K%s", line)
		return
	}
	fmt.Fprintln(r.out, line)
}

// Done clears the redrawn line so the shell prompt starts clean.
func (r *reporter) Done() {
	if r.live && !r.quiet {
		fmt.Fprint(r.out, "\r\033[K")
	}
}
