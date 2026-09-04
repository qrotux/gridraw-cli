package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qrotux/gridraw-cli/internal/client"
	"github.com/qrotux/gridraw-cli/internal/config"
	"github.com/spf13/cobra"
)

func TestInfoFormatRejectsDataFormats(t *testing.T) {
	if _, err := infoFormat("csv", "yaml"); err == nil {
		t.Fatal("want a usage error for csv on an info command")
	}
	got, err := infoFormat("", "json")
	if err != nil || got != "json" {
		t.Errorf("infoFormat(\"\", \"json\") = %q, %v; want json", got, err)
	}
	got, err = infoFormat("yaml", "json")
	if err != nil || got != "yaml" {
		t.Errorf("-o must win over the profile default, got %q", got)
	}
}

func TestJSONToYAMLKeepsNumbers(t *testing.T) {
	var v any
	dec := json.NewDecoder(strings.NewReader(`{"pageSize":25,"name":"users"}`))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		t.Fatal(err)
	}
	out, err := jsonToYAML(v)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "pageSize: 25") {
		t.Errorf("yaml = %s, want an unquoted number", out)
	}
}

// runInfo runs one information command against a stub server, in a working
// directory holding the only configuration the command can discover. It
// returns what the command printed to stdout and the path the server was
// asked for. An information command has nothing to say on stderr, and the
// helper pins that: stdout must be exactly the data a redirect would capture.
func runInfo(t *testing.T, cmd *cobra.Command, body string, args ...string) (string, string, error) {
	t.Helper()
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
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
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	err := cmd.ExecuteContext(context.Background())
	// Only on the success path: a standalone command still lets cobra report
	// a failure on stderr, which the root command silences.
	if err == nil && stderr.Len() > 0 {
		t.Errorf("stderr = %q, want nothing", stderr.String())
	}
	return stdout.String(), path, err
}

func TestInfoListPrintsYAMLByDefaultAndJSONVerbatim(t *testing.T) {
	body := `[{"id":"users","pageSize":25}]`
	stdout, path, err := runInfo(t, newListCmd(), body)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if path != "/api/grids/-/list" {
		t.Errorf("requested %q, want the list endpoint", path)
	}
	if !strings.Contains(stdout, "pageSize: 25") {
		t.Errorf("stdout = %q, want yaml with an unquoted number", stdout)
	}
	stdout, _, err = runInfo(t, newListCmd(), body, "-o", "json")
	if err != nil {
		t.Fatalf("list -o json: %v", err)
	}
	if stdout != body+"\n" {
		t.Errorf("stdout = %q, want the body byte for byte", stdout)
	}
}

func TestInfoGridWarmsTheCache(t *testing.T) {
	body := `{"name":"users","idColumn":"id","pageSize":25,"columns":[]}`
	stdout, path, err := runInfo(t, newGridCmd(), body, "users", "-o", "json")
	if err != nil {
		t.Fatalf("grid: %v", err)
	}
	if path != "/api/grids/users" {
		t.Errorf("requested %q, want the descriptor endpoint", path)
	}
	if !strings.Contains(stdout, `"name": "users"`) {
		t.Errorf("stdout = %q", stdout)
	}
	dir, err := client.DefaultDir()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "p", "users.json")); err != nil {
		t.Errorf("cache entry: %v", err)
	}
}

func TestInfoGridNoCacheDoesNotWrite(t *testing.T) {
	if _, _, err := runInfo(t, newGridCmd(), `{"id":"users"}`, "users", "--no-cache"); err != nil {
		t.Fatalf("grid --no-cache: %v", err)
	}
	dir, err := client.DefaultDir()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "p", "users.json")); !os.IsNotExist(err) {
		t.Errorf("--no-cache wrote an entry: %v", err)
	}
}

func TestInfoRejectsADataFormat(t *testing.T) {
	_, _, err := runInfo(t, newListCmd(), `[]`, "-o", "csv")
	var usage *UsageError
	if !errors.As(err, &usage) {
		t.Fatalf("err = %v, want a UsageError", err)
	}
	if !strings.Contains(usage.Msg, "yaml or json") {
		t.Errorf("message = %q, want the two allowed formats", usage.Msg)
	}
}

func TestInfoGridWithoutAnArgumentShowsTheRegistry(t *testing.T) {
	body := `[{"id":"users"}]`
	stdout, path, err := runInfo(t, newGridCmd(), body, "-o", "json")
	if err != nil {
		t.Fatalf("grid: %v", err)
	}
	if path != "/api/grids/-/registry" {
		t.Errorf("requested %q, want the registry endpoint", path)
	}
	if stdout != body+"\n" {
		t.Errorf("stdout = %q, want the body byte for byte", stdout)
	}
}

// TestInfoYAMLKeepsTheNumberLiteral pins the numbers -o yaml must not touch: an
// integer wider than int64, which float64 would round, and a float whose
// fractional zero an int conversion would drop.
func TestInfoYAMLKeepsTheNumberLiteral(t *testing.T) {
	body := `{"big":123456789012345678901234567890,"edge":9223372036854775808,"ratio":1.0}`
	stdout, _, err := runInfo(t, newListCmd(), body)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	want := "big: 123456789012345678901234567890\nedge: 9223372036854775808\nratio: 1.0\n"
	if stdout != want {
		t.Errorf("stdout = %q, want %q", stdout, want)
	}
}

func TestInfoRejectsATrailingValue(t *testing.T) {
	if _, _, err := runInfo(t, newListCmd(), `{"a":1} {"b":2}`); err == nil {
		t.Fatal("a body holding two JSON values must not print only the first")
	}
}

func TestInfoGridRawPrintsTheServerBody(t *testing.T) {
	const body = `{"name":"users","idColumn":"id","pageSize":25,"columns":[{"key":"id","type":"uuid","title":"ID","sortable":true}]}`

	raw, _, err := runInfo(t, newGridCmd(), body, "users", "--raw", "-o", "json", "--no-cache")
	if err != nil {
		t.Fatal(err)
	}
	if raw != body+"\n" {
		t.Errorf("--raw = %q, want the server body verbatim", raw)
	}
}

func TestInfoGridPrintsTheQueryView(t *testing.T) {
	const body = `{"name":"users","idColumn":"id","pageSize":25,"columns":[{"key":"id","type":"uuid","title":"ID","sortable":true,"filter":{"operators":[{"op":"in","label":"is one of"}]}}]}`

	view, _, err := runInfo(t, newGridCmd(), body, "users", "-o", "json", "--no-cache")
	if err != nil {
		t.Fatal(err)
	}
	for _, dropped := range []string{`"title"`, `"label"`, `"is one of"`} {
		if strings.Contains(view, dropped) {
			t.Errorf("the query view should drop %s:\n%s", dropped, view)
		}
	}
	if !strings.Contains(view, `"key": "id"`) || !strings.Contains(view, `"filters"`) {
		t.Errorf("the query view lost what a query needs:\n%s", view)
	}
}
