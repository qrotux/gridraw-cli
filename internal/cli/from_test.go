package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qrotux/gridraw-cli/internal/client"
	"github.com/qrotux/gridraw-cli/internal/config"
	"github.com/qrotux/gridraw-cli/internal/render"
	"github.com/qrotux/gridraw-cli/internal/wire"
	"github.com/spf13/cobra"
)

func testDescriptor() *wire.Descriptor {
	return &wire.Descriptor{
		Name:     "users",
		IDColumn: "id",
		PageSize: 25,
		Search:   &wire.Search{Columns: []string{"email"}},
		Columns: []wire.Column{
			{Key: "id", Type: wire.TypeUUID, DefaultVisible: false},
			{Key: "email", Type: wire.TypeString, DefaultVisible: true, Sortable: true,
				Filter: &wire.Filter{Operators: []wire.OperatorSpec{{Op: wire.OpEq}, {Op: wire.OpContains}}}},
			{Key: "rating", Type: wire.TypeNumber, DefaultVisible: true,
				Filter: &wire.Filter{Operators: []wire.OperatorSpec{{Op: wire.OpGte}}}},
			{Key: "secret", Type: wire.TypeString, DefaultVisible: false},
		},
	}
}

func TestBuildRequestLimitOverridesTheDescriptorPageSize(t *testing.T) {
	desc := testDescriptor()
	req, _, err := buildRequest(tail{Grid: "users"}, desc, false)
	if err != nil {
		t.Fatal(err)
	}
	if req.PageSize != 25 || req.Page != 1 {
		t.Errorf("page %d size %d, want page 1 size 25", req.Page, req.PageSize)
	}
	req, _, err = buildRequest(tail{Grid: "users", Limit: 3, Page: 4}, desc, false)
	if err != nil {
		t.Fatal(err)
	}
	if req.PageSize != 3 || req.Page != 4 {
		t.Errorf("page %d size %d, want page 4 size 3", req.Page, req.PageSize)
	}
}

// TestBuildRequestClampsADescriptorPageSize pins that a descriptor outside the
// server's own 1..100 range never turns into a request the server would reject.
func TestBuildRequestClampsADescriptorPageSize(t *testing.T) {
	for _, tc := range []struct{ have, want int }{{0, wire.MinPageSize}, {500, wire.MaxPageSize}, {25, 25}} {
		desc := testDescriptor()
		desc.PageSize = tc.have
		req, _, err := buildRequest(tail{Grid: "users"}, desc, false)
		if err != nil {
			t.Fatal(err)
		}
		if req.PageSize != tc.want {
			t.Errorf("descriptor pageSize %d became %d, want %d", tc.have, req.PageSize, tc.want)
		}
	}
}

// TestBuildRequestFallsBackToDefaultVisible pins that the request asks for the
// same columns the text formats print: the server returns the id column alone
// when the request carries no column list.
func TestBuildRequestFallsBackToDefaultVisible(t *testing.T) {
	desc := testDescriptor()
	req, order, err := buildRequest(tail{Grid: "users"}, desc, false)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"email", "rating"}
	if strings.Join(req.Columns, ",") != strings.Join(want, ",") {
		t.Errorf("request columns = %v, want %v", req.Columns, want)
	}
	if strings.Join(order, ",") != strings.Join(req.Columns, ",") {
		t.Errorf("render order %v differs from the request columns %v", order, req.Columns)
	}
	req, order, err = buildRequest(tail{Grid: "users", Columns: "id,email"}, desc, false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(req.Columns, ",") != "id,email" || strings.Join(order, ",") != "id,email" {
		t.Errorf("an explicit columns clause must win, got %v / %v", req.Columns, order)
	}
}

func TestBuildRequestSearchWithoutASearchableColumn(t *testing.T) {
	desc := testDescriptor()
	desc.Search = nil
	_, _, err := buildRequest(tail{Grid: "users", Search: "ivan"}, desc, false)
	var usage *UsageError
	if !errors.As(err, &usage) {
		t.Fatalf("err = %v, want a UsageError", err)
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
	if !strings.Contains(usage.Msg, "searchable") {
		t.Errorf("message = %q, want it to name the missing capability", usage.Msg)
	}
}

func TestBuildRequestWhereErrorIsAUsageError(t *testing.T) {
	for _, tc := range []struct{ name, where, order, columns string }{
		{name: "unknown column", where: "nosuch = 1"},
		{name: "unparseable", where: "email ="},
		{name: "bad order", order: "nosuch"},
		{name: "bad columns", columns: "nosuch"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := buildRequest(tail{Grid: "users", Where: tc.where, Order: tc.order, Columns: tc.columns}, testDescriptor(), false)
			var usage *UsageError
			if !errors.As(err, &usage) {
				t.Fatalf("err = %v, want a UsageError", err)
			}
			if ExitCode(err) != 2 {
				t.Errorf("exit code = %d, want 2", ExitCode(err))
			}
		})
	}
}

func TestBuildRequestOmitsAbsentClauses(t *testing.T) {
	req, _, err := buildRequest(tail{Grid: "users"}, testDescriptor(), false)
	if err != nil {
		t.Fatal(err)
	}
	if req.Filters != nil || req.Sort != nil || req.Search != "" {
		t.Errorf("absent clauses must stay empty, got %+v", req)
	}
}

// runFrom runs the from command against a stub rows server and returns what it
// wrote to stdout and to stderr separately: stdout must carry data only.
func runFrom(t *testing.T, desc, rows string, args ...string) (string, string, error) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/rows") {
			_, _ = io.WriteString(w, rows)
			return
		}
		_, _ = io.WriteString(w, desc)
	}))
	t.Cleanup(srv.Close)
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "xdg"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(dir, "cache"))
	t.Setenv("HOME", dir)
	conf := "current: p\nconfigs:\n  p:\n    host: " + srv.URL + "/api/grids\n"
	if err := os.WriteFile(config.LocalFileName, []byte(conf), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	root := &cobra.Command{Use: "gridraw", SilenceUsage: true, SilenceErrors: true}
	root.PersistentFlags().String("config", "", "")
	root.PersistentFlags().String("config-file", "", "")
	root.AddCommand(newFromCmd())
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs(append([]string{"from"}, args...))
	err := root.ExecuteContext(context.Background())
	return stdout.String(), stderr.String(), err
}

const stubDescriptor = `{"name":"users","idColumn":"id","pageSize":25,"search":null,"columns":[
	{"key":"id","type":"uuid","defaultVisible":false},
	{"key":"email","type":"string","defaultVisible":true}]}`

func TestFromPrintsTotalOnStderrAndDataOnStdout(t *testing.T) {
	rows := `{"rows":[{"id":"a","email":"eve@example.com"}],"total":8,"hasPrev":false,"hasNext":true}`
	stdout, stderr, err := runFrom(t, stubDescriptor, rows, "users", "-o", "csv")
	if err != nil {
		t.Fatal(err)
	}
	if stdout != "email\r\neve@example.com\r\n" {
		t.Errorf("stdout = %q, want the header and the row only", stdout)
	}
	if stderr != "Total: 8\n" {
		t.Errorf("stderr = %q, want the count only", stderr)
	}
}

func TestFromQuietPrintsNoTotal(t *testing.T) {
	rows := `{"rows":[],"total":8,"hasPrev":false,"hasNext":false}`
	stdout, stderr, err := runFrom(t, stubDescriptor, rows, "users", "-o", "csv", "--quiet")
	if err != nil {
		t.Fatal(err)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want nothing under --quiet", stderr)
	}
	if stdout != "email\r\n" {
		t.Errorf("stdout = %q", stdout)
	}
}

// TestFromWithoutATotalPrintsNone covers a grid that sets skipTotal: the
// response carries no total, so there is nothing to report.
func TestFromWithoutATotalPrintsNone(t *testing.T) {
	rows := `{"rows":[{"email":"eve@example.com"}],"hasPrev":false,"hasNext":false}`
	_, stderr, err := runFrom(t, stubDescriptor, rows, "users", "-o", "csv")
	if err != nil {
		t.Fatal(err)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want nothing when the response carries no total", stderr)
	}
}

func TestFromJSONPrintsTheBodyByteForByte(t *testing.T) {
	rows := `{"rows":[{"id":"a","email":"eve@example.com"}],"total":1,"hasPrev":false,"hasNext":false}`
	stdout, _, err := runFrom(t, stubDescriptor, rows, "users", "-o", "json")
	if err != nil {
		t.Fatal(err)
	}
	if stdout != rows+"\n" {
		t.Errorf("stdout = %q, want the body byte for byte", stdout)
	}
}

func TestFromWhereErrorWritesNothingToStdout(t *testing.T) {
	rows := `{"rows":[],"total":0,"hasPrev":false,"hasNext":false}`
	stdout, _, err := runFrom(t, stubDescriptor, rows, "users", "where", "nosuch = 1")
	if ExitCode(err) != 2 {
		t.Fatalf("err = %v, exit %d; want a usage error", err, ExitCode(err))
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want nothing before a usage error", stdout)
	}
}

func TestFromUnknownOutputFormatIsAUsageError(t *testing.T) {
	_, _, err := runFrom(t, stubDescriptor, `{"rows":[]}`, "users", "-o", "nope")
	if ExitCode(err) != 2 {
		t.Fatalf("err = %v, exit %d; want a usage error", err, ExitCode(err))
	}
}

func decodeBody(t *testing.T, r *http.Request, v any) {
	t.Helper()
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		t.Fatalf("cannot decode the request body: %v", err)
	}
}

// pagedServer serves `pages` pages of one row each and fails on failAt (0 = never).
func pagedServer(t *testing.T, pages, failAt int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req wire.RowsRequest
		decodeBody(t, r, &req)
		if req.Page == failAt {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error":"query failed"}`)
			return
		}
		total := int64(pages)
		fmt.Fprintf(w, `{"rows":[{"id":"row%d"}],"total":%d,"hasPrev":%t,"hasNext":%t}`,
			req.Page, total, req.Page > 1, req.Page < pages)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestAllPagesUntilHasNext(t *testing.T) {
	srv := pagedServer(t, 3, 0)
	var out, errOut bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	rep := newReporter(&errOut, true, false)
	api := client.New(srv.URL, "", nil)
	req := wire.RowsRequest{Page: 1, PageSize: 100}
	err := streamAll(context.Background(), cmd, api, "users", req, render.FormatJSONL,
		render.Options{Columns: []string{"id"}}, rep)
	if err != nil {
		t.Fatal(err)
	}
	want := "{\"id\":\"row1\"}\n{\"id\":\"row2\"}\n{\"id\":\"row3\"}\n"
	if got := out.String(); got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestAllLeavesTheStreamUnclosedOnFailure(t *testing.T) {
	srv := pagedServer(t, 5, 3)
	var out, errOut bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	rep := newReporter(&errOut, true, false)
	err := streamAll(context.Background(), cmd, client.New(srv.URL, "", nil), "users",
		wire.RowsRequest{Page: 1, PageSize: 100}, render.FormatJSONA,
		render.Options{Columns: []string{"id"}}, rep)
	if err == nil {
		t.Fatal("want the server error")
	}
	if ExitCode(err) != 5 {
		t.Errorf("exit code = %d, want 5", ExitCode(err))
	}
	if strings.HasSuffix(out.String(), "]") {
		t.Errorf("stdout = %q, want it left unclosed", out.String())
	}
	if !strings.Contains(err.Error(), "page 3") {
		t.Errorf("error = %q, want it to name the page", err)
	}
}

func TestResumeHint(t *testing.T) {
	got := resumeHint([]string{"users", "where", "x = 1"}, render.FormatCSV, "csv", 37)
	if !strings.Contains(got, "--all page 37") || !strings.Contains(got, "--no-header") {
		t.Errorf("hint = %q, want the page and --no-header", got)
	}
	// The tail rejects "page" twice, so a run that already named a page must
	// not have that clause copied into the hint.
	got = resumeHint([]string{"users", "page", "2", "limit", "50"}, render.FormatJSONL, "jsonl", 9)
	if got != "gridraw from users limit 50 -o jsonl --all page 9" {
		t.Errorf("hint = %q, want the original page clause replaced", got)
	}
	got = resumeHint([]string{"users"}, render.FormatJSONA, "jsona", 4)
	if !strings.Contains(got, "jsonl") {
		t.Errorf("hint = %q, want it to point at jsonl", got)
	}
}

// TestBuildRequestAllDefaultsToTheLargestPage pins that --all asks for the
// server's maximum page when no limit clause is given: fewer requests for the
// same rows. An explicit limit still wins.
func TestBuildRequestAllDefaultsToTheLargestPage(t *testing.T) {
	desc := testDescriptor()
	req, _, err := buildRequest(tail{Grid: "users"}, desc, true)
	if err != nil {
		t.Fatal(err)
	}
	if req.PageSize != wire.MaxPageSize {
		t.Errorf("page size = %d, want %d", req.PageSize, wire.MaxPageSize)
	}
	req, _, err = buildRequest(tail{Grid: "users", Limit: 7}, desc, true)
	if err != nil {
		t.Fatal(err)
	}
	if req.PageSize != 7 {
		t.Errorf("page size = %d, want the limit clause to win", req.PageSize)
	}
}
