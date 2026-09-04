package client

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/qrotux/gridraw-cli/internal/wire"
)

// DefaultTTL is how long a cached descriptor is considered current.
const DefaultTTL = 5 * time.Minute

// Mode selects how a descriptor request uses the cache.
type Mode int

const (
	// CacheDefault reads the entry when it is fresh and writes on a miss.
	CacheDefault Mode = iota
	// CacheRefresh ignores what is stored, fetching and writing unconditionally.
	CacheRefresh
	// CacheOff neither reads nor writes the cache.
	CacheOff
)

// Cache stores descriptors under Dir/Profile/{grid}.json. The profile is part
// of the path because two profiles may serve different grids under one name.
type Cache struct {
	Dir     string
	Profile string
	TTL     time.Duration
}

// DefaultDir is $XDG_CACHE_HOME/gridraw, or ~/.cache/gridraw when that
// variable is unset or empty. os.UserCacheDir is deliberately not used: on
// darwin it answers ~/Library/Caches, which is not the promised path.
func DefaultDir() (string, error) {
	dir := os.Getenv("XDG_CACHE_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".cache")
	}
	return filepath.Join(dir, "gridraw"), nil
}

// Descriptor returns the grid descriptor, from the cache when mode allows and
// the entry is fresh. Cache failures are never fatal: a broken entry only
// costs a request.
func (c *Cache) Descriptor(ctx context.Context, api *Client, grid string, mode Mode) (*wire.Descriptor, error) {
	if mode == CacheDefault {
		if d := c.get(grid); d != nil {
			return d, nil
		}
	}
	d, raw, err := api.Descriptor(ctx, grid)
	if err != nil {
		return nil, err
	}
	if mode != CacheOff {
		c.Put(grid, raw)
	}
	return d, nil
}

// Put writes a descriptor body into the cache, ignoring failures. It is called
// on its own by `gridraw grid {id}`, which always fetches and warms the entry.
func (c *Cache) Put(grid string, raw []byte) {
	path, ok := c.path(grid)
	if !ok {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	_ = os.WriteFile(path, raw, 0o600)
}

func (c *Cache) get(grid string) *wire.Descriptor {
	path, ok := c.path(grid)
	if !ok {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil || time.Since(info.ModTime()) > c.ttl() {
		return nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var d wire.Descriptor
	if err := decode(raw, &d); err != nil {
		return nil
	}
	return &d
}

// path returns Dir/Profile/{grid}.json and reports whether both segments are
// usable as one path element. The grid name is a command-line argument and the
// profile name comes from --config, so a segment holding a separator or a ".."
// is refused rather than sanitised: the caller treats a refusal as a cache miss
// or a no-op, which costs one HTTP request and nothing else.
func (c *Cache) path(grid string) (string, bool) {
	if !safeSegment(c.Profile) || !safeSegment(grid) {
		return "", false
	}
	return filepath.Join(c.Dir, c.Profile, grid+".json"), true
}

func safeSegment(s string) bool {
	return s != "" && s != "." && s != ".." && !strings.ContainsAny(s, `/\`)
}

func (c *Cache) ttl() time.Duration {
	if c.TTL <= 0 {
		return DefaultTTL
	}
	return c.TTL
}
