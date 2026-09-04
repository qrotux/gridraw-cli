package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func newCachedFixture(t *testing.T) (*Cache, *Client, *int32) {
	t.Helper()
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Write([]byte(`{"name":"users","idColumn":"id","pageSize":25,"columns":[]}`))
	}))
	t.Cleanup(srv.Close)
	return &Cache{Dir: t.TempDir(), Profile: "default", TTL: time.Minute}, New(srv.URL, "", nil), &hits
}

func TestCacheServesFreshEntry(t *testing.T) {
	cache, c, hits := newCachedFixture(t)
	ctx := context.Background()
	if _, err := cache.Descriptor(ctx, c, "users", CacheDefault); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Descriptor(ctx, c, "users", CacheDefault); err != nil {
		t.Fatal(err)
	}
	if *hits != 1 {
		t.Errorf("server hits = %d, want 1: the second call must come from the cache", *hits)
	}
}

func TestCacheExpiresAndRefreshBypasses(t *testing.T) {
	cache, c, hits := newCachedFixture(t)
	ctx := context.Background()
	cache.TTL = time.Nanosecond
	if _, err := cache.Descriptor(ctx, c, "users", CacheDefault); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Millisecond)
	if _, err := cache.Descriptor(ctx, c, "users", CacheDefault); err != nil {
		t.Fatal(err)
	}
	if *hits != 2 {
		t.Errorf("hits = %d, want 2: a stale entry must be refetched", *hits)
	}
	cache.TTL = time.Minute
	if _, err := cache.Descriptor(ctx, c, "users", CacheRefresh); err != nil {
		t.Fatal(err)
	}
	if *hits != 3 {
		t.Errorf("hits = %d, want 3: --refresh must bypass the read", *hits)
	}
}

func TestCacheOffWritesNothing(t *testing.T) {
	cache, c, _ := newCachedFixture(t)
	if _, err := cache.Descriptor(context.Background(), c, "users", CacheOff); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(filepath.Join(cache.Dir, "default"))
	if len(entries) != 0 {
		t.Errorf("cache directory holds %d entries, want 0 with --no-cache", len(entries))
	}
}

func TestCorruptEntryIsRefetched(t *testing.T) {
	cache, c, hits := newCachedFixture(t)
	path := filepath.Join(cache.Dir, "default", "users.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Descriptor(context.Background(), c, "users", CacheDefault); err != nil {
		t.Fatalf("a corrupt entry must not be fatal: %v", err)
	}
	if *hits != 1 {
		t.Errorf("hits = %d, want 1", *hits)
	}
}
