package render

import (
	"bytes"
	"fmt"
	"io"
)

// jsonWriter serves both jsona (a bare array) and json under --all (the same
// array inside an envelope carrying the total).
type jsonWriter struct {
	w        io.Writer
	opt      Options
	envelope bool
	started  bool
}

func (j *jsonWriter) Head(total *int64) error {
	if j.envelope {
		if total != nil {
			if _, err := fmt.Fprintf(j.w, `{"total":%d,"rows":[`, *total); err != nil {
				return err
			}
			return nil
		}
		_, err := io.WriteString(j.w, `{"rows":[`)
		return err
	}
	_, err := io.WriteString(j.w, "[")
	return err
}

func (j *jsonWriter) Row(row map[string]any) error {
	if j.started {
		if _, err := io.WriteString(j.w, ","); err != nil {
			return err
		}
	}
	j.started = true
	return writeObject(j.w, row, j.opt.Columns)
}

func (j *jsonWriter) Close() error {
	suffix := "]\n"
	if j.envelope {
		suffix = "]}\n"
	}
	_, err := io.WriteString(j.w, suffix)
	return err
}

// jsonlWriter writes one compact object per line, with no prologue at all.
type jsonlWriter struct {
	w   io.Writer
	opt Options
}

func (j *jsonlWriter) Head(*int64) error { return nil }

func (j *jsonlWriter) Row(row map[string]any) error {
	if err := writeObject(j.w, row, j.opt.Columns); err != nil {
		return err
	}
	_, err := io.WriteString(j.w, "\n")
	return err
}

func (j *jsonlWriter) Close() error { return nil }

// writeObject emits the row with its keys in column order. encoding/json sorts
// map keys, so the object is assembled by hand.
func writeObject(w io.Writer, row map[string]any, columns []string) error {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, kv := range ordered(row, columns) {
		if i > 0 {
			buf.WriteByte(',')
		}
		key, err := compactJSON(kv.Key)
		if err != nil {
			return err
		}
		buf.Write(key)
		buf.WriteByte(':')
		val, err := compactJSON(kv.Value)
		if err != nil {
			return err
		}
		buf.Write(val)
	}
	buf.WriteByte('}')
	_, err := w.Write(buf.Bytes())
	return err
}
