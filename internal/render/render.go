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
	// FormatJSON is the response as the server sent it, or an envelope carrying
	// the total and the rows when paging.
	FormatJSON Format = "json"
	// FormatJSONA is a JSON array of the rows.
	FormatJSONA Format = "jsona"
	// FormatJSONL is one compact JSON object per line.
	FormatJSONL Format = "jsonl"
	// FormatYAML is the total and the rows as a YAML document.
	FormatYAML Format = "yaml"
	// FormatYAMLA is a YAML sequence of the rows, with no prologue.
	FormatYAMLA Format = "yamla"
	// FormatCSV is the rows as RFC 4180 comma-separated records.
	FormatCSV Format = "csv"
	// FormatTSV is the rows as tab-separated records with escaped controls.
	FormatTSV Format = "tsv"
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
	case FormatYAML:
		return &yamlWriter{w: w, opt: opt, envelope: true}, nil
	case FormatYAMLA:
		return &yamlWriter{w: w, opt: opt}, nil
	case FormatCSV:
		return newTableWriter(w, ',', opt), nil
	case FormatTSV:
		return newTableWriter(w, '\t', opt), nil
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
