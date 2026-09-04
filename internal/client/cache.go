package client

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/qrotux/gridraw-cli/internal/wire"
)

// DefaultTTL is how long a cached descriptor is considered current.
const DefaultTTL = 5 * time.Minute

// Mode selects how a descriptor request uses the cache.
type Mode int

const (
	CacheDefault Mode = iota // read when fresh, write on a miss
	CacheRefresh             // ignore what is stored, fetch and write
	CacheOff                 // neither read nor write
)

// Cache stores descriptors under Dir/Profile/{grid}.json. The profile is part
// of the path because two profiles may serve different grids under one name.
type Cache struct {
	Dir     string
	Profile string
	TTL     time.Duration
}

// DefaultDir is ~/.cache/gridraw.
func DefaultDir() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
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
	path := c.path(grid)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	_ = os.WriteFile(path, raw, 0o600)
}

func (c *Cache) get(grid string) *wire.Descriptor {
	path := c.path(grid)
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

func (c *Cache) path(grid string) string {
	// A grid name is a path segment on the wire, so it cannot contain a
	// separator; keeping the name verbatim makes the cache inspectable.
	return filepath.Join(c.Dir, c.Profile, grid+".json")
}

func (c *Cache) ttl() time.Duration {
	if c.TTL <= 0 {
		return DefaultTTL
	}
	return c.TTL
}
