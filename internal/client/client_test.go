package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/qrotux/gridraw-cli/internal/wire"
)

func TestEndpointPathsAndAuth(t *testing.T) {
	var gotPath, gotAuth, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth = r.URL.Path, r.Header.Get("Authorization")
		b := make([]byte, r.ContentLength)
		if r.ContentLength > 0 {
			r.Body.Read(b)
		}
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/-/list"):
			w.Write([]byte(`[{"name":"users"}]`))
		case strings.HasSuffix(r.URL.Path, "/-/registry"):
			w.Write([]byte(`[{"name":"users","columns":[{"key":"id","title":"ID","type":"uuid"}]}]`))
		case strings.HasSuffix(r.URL.Path, "/rows"):
			w.Write([]byte(`{"rows":[{"id":"a"}],"total":1,"hasPrev":false,"hasNext":false}`))
		default:
			w.Write([]byte(`{"name":"users","idColumn":"id","pageSize":25,"columns":[]}`))
		}
	}))
	defer srv.Close()

	c := New(srv.URL+"/api/grids/", "Bearer t", nil) // trailing slash on purpose
	ctx := context.Background()

	if _, err := c.Raw(ctx, "GET", "/-/list"); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/api/grids/-/list" {
		t.Errorf("list path = %q", gotPath)
	}
	if gotAuth != "Bearer t" {
		t.Errorf("auth = %q", gotAuth)
	}
	if _, err := c.Raw(ctx, "GET", "/-/registry"); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/api/grids/-/registry" {
		t.Errorf("registry path = %q", gotPath)
	}
	if _, _, err := c.Descriptor(ctx, "users"); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/api/grids/users" {
		t.Errorf("descriptor path = %q", gotPath)
	}
	resp, raw, err := c.Rows(ctx, "users", wire.RowsRequest{Page: 1, PageSize: 25})
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/api/grids/users/rows" {
		t.Errorf("rows path = %q", gotPath)
	}
	if !strings.Contains(gotBody, `"pageSize":25`) {
		t.Errorf("rows body = %q", gotBody)
	}
	if resp.Total == nil || *resp.Total != 1 || !strings.Contains(string(raw), `"hasNext":false`) {
		t.Errorf("rows response = %+v raw=%s", resp, raw)
	}
}

func TestNoAuthHeaderWhenEmpty(t *testing.T) {
	seen := true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, seen = r.Header["Authorization"]
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()
	if _, err := New(srv.URL, "", nil).Raw(context.Background(), "GET", "/-/list"); err != nil {
		t.Fatal(err)
	}
	if seen {
		t.Error("Authorization sent although the profile has none")
	}
}

func TestHTTPErrorCarriesStatusAndMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "unknown grid"})
	}))
	defer srv.Close()
	_, _, err := New(srv.URL, "", nil).Descriptor(context.Background(), "nope")
	if err == nil {
		t.Fatal("want an error")
	}
	var he *HTTPError
	if !asHTTPError(err, &he) {
		t.Fatalf("error type = %T, want *HTTPError", err)
	}
	if he.HTTPStatus() != 404 || !strings.Contains(he.Error(), "unknown grid") {
		t.Errorf("error = %v (status %d)", he, he.HTTPStatus())
	}
}

func TestServerErrorWithTextBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("proxy is on fire"))
	}))
	defer srv.Close()
	_, _, err := New(srv.URL, "", nil).Rows(context.Background(), "users", wire.RowsRequest{})
	var he *HTTPError
	if !asHTTPError(err, &he) || he.HTTPStatus() != 500 || !strings.Contains(he.Error(), "proxy is on fire") {
		t.Fatalf("error = %v", err)
	}
}

func asHTTPError(err error, target **HTTPError) bool { return errors.As(err, target) }

// TestErrorBodyTruncationKeepsValidUTF8 pins that a long error message is cut
// at a rune boundary: slicing bytes would leave half a character behind.
func TestErrorBodyTruncationKeepsValidUTF8(t *testing.T) {
	long := strings.Repeat("ы", 4000) // two bytes each, so the cap lands mid-rune
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, long)
	}))
	defer srv.Close()
	_, err := New(srv.URL, "", nil).Raw(context.Background(), "GET", "/-/list")
	var he *HTTPError
	if !asHTTPError(err, &he) {
		t.Fatalf("error = %T, want *HTTPError", err)
	}
	if !utf8.ValidString(he.Msg) {
		t.Error("the truncated message is not valid UTF-8")
	}
	if !strings.HasSuffix(he.Msg, "…") {
		t.Errorf("a truncated message should say so, got %q", he.Msg[len(he.Msg)-8:])
	}
}
