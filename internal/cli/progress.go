package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/qrotux/gridraw-cli/internal/render"
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

// resumeOptions are the flags of an interrupted run that a resumed run must
// repeat: the two config flags select the profile, hence the host and the auth
// header, and NullVal changes the text written for a null cell.
type resumeOptions struct {
	Config     string
	ConfigFile string
	NullVal    string
}

// resumeHint builds the command that continues an interrupted --all run.
func resumeHint(t tail, opt resumeOptions, format render.Format, page int) string {
	if !format.Streaming() {
		return fmt.Sprintf("the %s output cannot be appended to; rerun with -o %s to resume into the same file",
			format, appendable(format))
	}
	var b strings.Builder
	b.WriteString("gridraw from ")
	b.WriteString(shellQuote(t.Grid))
	// t.Page is deliberately dropped: the tail rejects a keyword given twice
	// and the hint supplies the page to resume from.
	for _, c := range []struct{ keyword, value string }{
		{"columns", t.Columns}, {"where", t.Where}, {"order", t.Order}, {"search", t.Search},
	} {
		if c.value != "" {
			fmt.Fprintf(&b, " %s %s", c.keyword, shellQuote(c.value))
		}
	}
	if t.Limit != 0 {
		fmt.Fprintf(&b, " limit %d", t.Limit)
	}
	for _, f := range []struct{ name, value string }{
		{"--config", opt.Config}, {"--config-file", opt.ConfigFile}, {"--null-val", opt.NullVal},
	} {
		if f.value != "" {
			fmt.Fprintf(&b, " %s %s", f.name, shellQuote(f.value))
		}
	}
	fmt.Fprintf(&b, " -o %s --all page %d", format, page)
	if format == render.FormatCSV || format == render.FormatTSV {
		b.WriteString(" --no-header")
	}
	return b.String()
}

// appendable names the sibling of a non-streaming format that a resumed run
// can append to the same file.
func appendable(format render.Format) render.Format {
	if format == render.FormatYAML {
		return render.FormatYAMLA
	}
	return render.FormatJSONL
}

// shellQuote wraps an argument in single quotes when it holds anything the
// shell would act on, so the printed command can be pasted as is.
func shellQuote(s string) string {
	if s != "" && !strings.ContainsAny(s, " \t\n\"'`$&|;<>()*?[]#~=%") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
