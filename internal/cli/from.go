package cli

import (
	"bytes"
	"context"
	"fmt"

	"github.com/qrotux/gridraw-cli/internal/client"
	"github.com/qrotux/gridraw-cli/internal/render"
	"github.com/qrotux/gridraw-cli/internal/wire"
	"github.com/qrotux/gridraw-cli/internal/wql"
	"github.com/spf13/cobra"
)

func newFromCmd() *cobra.Command {
	var (
		output           string
		all, noHeader    bool
		quiet, progress  bool
		noCache, refresh bool
		nullVal          string
	)
	cmd := &cobra.Command{
		Use:   `from GRID [columns "a,b"] [where "…"] [order "a,-b"] [search "…"] [limit N] [page M]`,
		Short: "Query a grid's rows",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			t, err := parseTail(args)
			if err != nil {
				return err
			}
			api, profile, cache, err := apiFor(cmd)
			if err != nil {
				return err
			}
			format, err := render.ParseFormat(pick(output, profile.DataOutput()))
			if err != nil {
				return &UsageError{Msg: err.Error()}
			}

			mode := client.CacheDefault
			switch {
			case noCache:
				mode = client.CacheOff
			case refresh:
				mode = client.CacheRefresh
			}
			desc, err := cache.Descriptor(cmd.Context(), api, t.Grid, mode)
			if err != nil {
				return err
			}

			req, columns, err := buildRequest(t, desc)
			if err != nil {
				return err
			}

			rep := newReporter(cmd.ErrOrStderr(), quiet, progress)
			rep.pageSize = req.PageSize
			opt := render.Options{Columns: columns, NullVal: nullVal, NoHeader: noHeader}
			if all {
				return &UsageError{Msg: "--all is not available yet"}
			}
			return streamOne(cmd.Context(), cmd, api, t.Grid, req, format, opt, rep)
		},
	}
	f := cmd.Flags()
	f.StringVarP(&output, "output", "o", "", "output format: json, jsona, jsonl, yaml, yamla, csv, tsv")
	f.BoolVar(&all, "all", false, "page through the whole result")
	f.BoolVar(&noHeader, "no-header", false, "omit the csv/tsv header row")
	f.StringVar(&nullVal, "null-val", "", "text written for a null value in csv, tsv, yaml and yamla")
	f.BoolVar(&quiet, "quiet", false, "no Total and no progress on stderr")
	f.BoolVar(&progress, "progress", false, "print progress even when stderr is not a terminal")
	f.BoolVar(&noCache, "no-cache", false, "do not read or write the descriptor cache")
	f.BoolVar(&refresh, "refresh", false, "refetch the descriptor and update the cache")
	return cmd
}

func pick(flag, def string) string {
	if flag != "" {
		return flag
	}
	return def
}

// buildRequest validates the tail against the descriptor and returns the
// request plus the column order the text formats use.
func buildRequest(t tail, desc *wire.Descriptor) (wire.RowsRequest, []string, error) {
	columns, err := wql.Columns(t.Columns, desc)
	if err != nil {
		return wire.RowsRequest{}, nil, &UsageError{Msg: err.Error()}
	}
	node, err := wql.ParseWhere(t.Where)
	if err != nil {
		return wire.RowsRequest{}, nil, &UsageError{Msg: err.Error()}
	}
	groups, err := wql.DNF(node)
	if err != nil {
		return wire.RowsRequest{}, nil, &UsageError{Msg: err.Error()}
	}
	filters, err := wql.Bind(groups, desc, t.Where)
	if err != nil {
		return wire.RowsRequest{}, nil, &UsageError{Msg: err.Error()}
	}
	sort, err := wql.Order(t.Order, desc)
	if err != nil {
		return wire.RowsRequest{}, nil, &UsageError{Msg: err.Error()}
	}
	if t.Search != "" && desc.Search == nil {
		return wire.RowsRequest{}, nil, &UsageError{Msg: fmt.Sprintf("grid %q has no searchable column", desc.Name)}
	}

	pageSize := desc.PageSize
	if t.Limit > 0 {
		pageSize = t.Limit
	}
	page := t.Page
	if page == 0 {
		page = 1
	}
	// The server returns the id column alone when "columns" is empty, so an
	// absent clause asks for the descriptor's visible columns, which is also
	// what the text formats print.
	out := columns
	if out == nil {
		out = desc.VisibleKeys()
	}
	req := wire.RowsRequest{
		Columns:  out,
		Filters:  filters,
		Search:   t.Search,
		Sort:     sort,
		Page:     page,
		PageSize: pageSize,
	}
	return req, out, nil
}

// streamOne fetches a single page. With -o json the server's body is copied
// byte for byte, which is what "as is" means in the spec.
func streamOne(ctx context.Context, cmd *cobra.Command, api *client.Client, grid string,
	req wire.RowsRequest, format render.Format, opt render.Options, rep *reporter) error {
	resp, raw, err := api.Rows(ctx, grid, req)
	if err != nil {
		return err
	}
	rep.Total(resp.Total)
	if format == render.FormatJSON {
		_, err := cmd.OutOrStdout().Write(append(bytes.TrimRight(raw, "\n"), '\n'))
		return err
	}
	w, err := render.New(cmd.OutOrStdout(), format, opt)
	if err != nil {
		return &UsageError{Msg: err.Error()}
	}
	if err := w.Head(resp.Total); err != nil {
		return err
	}
	for _, row := range resp.Rows {
		if err := w.Row(row); err != nil {
			return err
		}
	}
	return w.Close()
}
