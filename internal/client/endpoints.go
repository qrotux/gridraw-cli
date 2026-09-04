package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/qrotux/gridraw-cli/internal/wire"
)

// List returns GET <base>/-/list.
func (c *Client) List(ctx context.Context) ([]wire.GridInfo, error) {
	raw, err := c.do(ctx, "GET", "/-/list", nil)
	if err != nil {
		return nil, err
	}
	var out []wire.GridInfo
	return out, decode(raw, &out)
}

// Registry returns GET <base>/-/registry.
func (c *Client) Registry(ctx context.Context) ([]wire.CatalogEntry, error) {
	raw, err := c.do(ctx, "GET", "/-/registry", nil)
	if err != nil {
		return nil, err
	}
	var out []wire.CatalogEntry
	return out, decode(raw, &out)
}

// Descriptor returns GET <base>/{grid} both decoded and raw; the raw bytes are
// what `gridraw grid {id} -o json` prints and what the cache stores.
func (c *Client) Descriptor(ctx context.Context, grid string) (*wire.Descriptor, []byte, error) {
	raw, err := c.do(ctx, "GET", "/"+url.PathEscape(grid), nil)
	if err != nil {
		return nil, nil, err
	}
	var d wire.Descriptor
	if err := decode(raw, &d); err != nil {
		return nil, nil, err
	}
	return &d, raw, nil
}

// Rows returns POST <base>/{grid}/rows both decoded and raw.
func (c *Client) Rows(ctx context.Context, grid string, req wire.RowsRequest) (*wire.RowsResponse, []byte, error) {
	raw, err := c.do(ctx, "POST", "/"+url.PathEscape(grid)+"/rows", req)
	if err != nil {
		return nil, nil, err
	}
	var resp wire.RowsResponse
	if err := decode(raw, &resp); err != nil {
		return nil, nil, err
	}
	return &resp, raw, nil
}

// decode keeps number literals intact: a number column holding an integer
// larger than float64's exact range must not come back in exponent form.
func decode(raw []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("cannot decode the response: %w", err)
	}
	return nil
}
