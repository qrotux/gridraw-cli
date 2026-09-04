package cli

import (
	"bytes"
	"testing"
)

func total(n int64) *int64 { return &n }

func TestReporterTotalPrintsOnce(t *testing.T) {
	var buf bytes.Buffer
	r := newReporter(&buf, false, false)
	r.Total(total(8))
	r.Total(total(8))
	if buf.String() != "Total: 8\n" {
		t.Errorf("out = %q, want one count", buf.String())
	}
}

func TestReporterTotalSilentWithoutATotalOrUnderQuiet(t *testing.T) {
	var buf bytes.Buffer
	newReporter(&buf, false, false).Total(nil)
	if buf.Len() != 0 {
		t.Errorf("out = %q, want nothing when the response carries no total", buf.String())
	}
	buf.Reset()
	newReporter(&buf, true, false).Total(total(8))
	if buf.Len() != 0 {
		t.Errorf("out = %q, want nothing under --quiet", buf.String())
	}
}

// TestReporterProgressGating pins that a progress line is drawn only when
// stderr is a terminal or --progress forces it: a buffer is neither, so only
// the forced reporter writes.
func TestReporterProgressGating(t *testing.T) {
	var buf bytes.Buffer
	r := newReporter(&buf, false, false)
	r.pageSize = 25
	r.Page(1, 25, total(60))
	r.Done()
	if buf.Len() != 0 {
		t.Errorf("out = %q, want no progress off a terminal", buf.String())
	}
	buf.Reset()
	r = newReporter(&buf, false, true)
	r.pageSize = 25
	r.Page(1, 25, total(60))
	if buf.String() != "page 1/3 · 25/60 rows\n" {
		t.Errorf("out = %q, want the page ratio", buf.String())
	}
	buf.Reset()
	r = newReporter(&buf, true, true)
	r.Page(1, 25, total(60))
	if buf.Len() != 0 {
		t.Errorf("out = %q, want nothing under --quiet", buf.String())
	}
}

// TestReporterProgressWithoutATotal covers the count-free line a skipTotal
// grid gets: there is no page count to compute.
func TestReporterProgressWithoutATotal(t *testing.T) {
	var buf bytes.Buffer
	r := newReporter(&buf, false, true)
	r.pageSize = 25
	r.Page(2, 25, nil)
	if buf.String() != "page 2 · 25 rows\n" {
		t.Errorf("out = %q, want the count-free line", buf.String())
	}
}
