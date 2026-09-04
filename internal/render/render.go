// Package render writes row pages in the CLI's output formats. Every writer
// streams: a page is written as it arrives and nothing is kept.
package render

import (
	"fmt"
	"io"
)

// Format is an output format name.
type Format string

const (
	FormatJSON  Format = "json"
	FormatJSONA Format = "jsona"
	FormatJSONL Format = "jsonl"
	FormatYAML  Format = "yaml"
	FormatYAMLA Format = "yamla"
	FormatCSV   Format = "csv"
	FormatTSV   Format = "tsv"
)

// Options configure a writer. Columns fixes the key order of every row.
type Options struct {
	Columns  []string
	NullVal  string
	NoHeader bool
}

// Writer streams a result. Head is called once, before the first row, with the
// total when the server reported one. Close writes the suffix; it is
// deliberately NOT called when a page request fails, which leaves a JSON
// document unclosed rather than pretending the output is complete.
type Writer interface {
	Head(total *int64) error
	Row(row map[string]any) error
	Close() error
}

// New returns the writer for a format.
func New(w io.Writer, format Format, opt Options) (Writer, error) {
	switch format {
	case FormatJSON:
		return &jsonWriter{w: w, opt: opt, envelope: true}, nil
	case FormatJSONA:
		return &jsonWriter{w: w, opt: opt}, nil
	case FormatJSONL:
		return &jsonlWriter{w: w, opt: opt}, nil
	}
	return nil, fmt.Errorf("unknown output format %q", format)
}

// ParseFormat validates a format name.
func ParseFormat(s string) (Format, error) {
	switch Format(s) {
	case FormatJSON, FormatJSONA, FormatJSONL, FormatYAML, FormatYAMLA, FormatCSV, FormatTSV:
		return Format(s), nil
	}
	return "", fmt.Errorf("output %q is not one of json, jsona, jsonl, yaml, yamla, csv, tsv", s)
}

// Streaming reports whether appending another run's output to this one
// produces a valid document; it drives the resume hint on a failed page.
func (f Format) Streaming() bool {
	return f == FormatJSONL || f == FormatYAMLA || f == FormatCSV || f == FormatTSV
}
