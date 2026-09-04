package render

import (
	"io"
	"strings"
)

// tableWriter serves csv and tsv. csv follows RFC 4180: fields holding the
// delimiter, a quote or a line break are quoted and inner quotes doubled, and
// records end with CRLF. tsv cannot quote, so it escapes instead.
type tableWriter struct {
	w   io.Writer
	sep rune
	opt Options
}

func newTableWriter(w io.Writer, sep rune, opt Options) *tableWriter {
	return &tableWriter{w: w, sep: sep, opt: opt}
}

func (t *tableWriter) Head(*int64) error {
	if t.opt.NoHeader || len(t.opt.Columns) == 0 {
		return nil
	}
	return t.writeRecord(t.opt.Columns)
}

func (t *tableWriter) Row(row map[string]any) error {
	pairs := ordered(row, t.opt.Columns)
	fields := make([]string, 0, len(pairs))
	for _, kv := range pairs {
		fields = append(fields, Cell(kv.Value, t.opt.NullVal))
	}
	return t.writeRecord(fields)
}

func (t *tableWriter) Close() error { return nil }

func (t *tableWriter) writeRecord(fields []string) error {
	var sb strings.Builder
	for i, f := range fields {
		if i > 0 {
			sb.WriteRune(t.sep)
		}
		if t.sep == '\t' {
			sb.WriteString(escapeTSV(f))
			continue
		}
		sb.WriteString(quoteCSV(f))
	}
	sb.WriteString("\r\n")
	_, err := io.WriteString(t.w, sb.String())
	return err
}

func quoteCSV(s string) string {
	if !strings.ContainsAny(s, ",\"\r\n") {
		return s
	}
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func escapeTSV(s string) string {
	r := strings.NewReplacer("\\", `\\`, "\t", `\t`, "\n", `\n`, "\r", `\r`)
	return r.Replace(s)
}
