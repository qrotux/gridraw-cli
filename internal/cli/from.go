package cli

import (
	"bytes"
	"context"
	"errors"
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
		// The tail is split from the flags by splitArgs, not by pflag.
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.InheritedFlags() // merges --config and --config-file into cmd.Flags()
			tailArgs, flagArgs := splitArgs(args, cmd.Flags())
			if err := cmd.Flags().Parse(flagArgs); err != nil {
				return &UsageError{Msg: err.Error()}
			}
			if help, _ := cmd.Flags().GetBool("help"); help {
				return cmd.Help()
			}
			t, err := parseTail(tailArgs)
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

			req, columns, err := buildRequest(t, desc, all)
			if err != nil {
				return err
			}

			rep := newReporter(cmd.ErrOrStderr(), quiet, progress)
			rep.pageSize = req.PageSize
			opt := render.Options{Columns: columns, NullVal: nullVal, NoHeader: noHeader}
			if all {
				err := streamAll(cmd.Context(), cmd, api, t.Grid, req, format, opt, rep)
				var pe *pageError
				if errors.As(err, &pe) {
					conf, _ := cmd.Flags().GetString("config")
					confFile, _ := cmd.Flags().GetString("config-file")
					pe.Hint = resumeHint(t, resumeOptions{Config: conf, ConfigFile: confFile, NullVal: nullVal}, format, pe.Page)
				}
				return err
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
func buildRequest(t tail, desc *wire.Descriptor, all bool) (wire.RowsRequest, []string, error) {
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

	// limit is bounds-checked while parsing the tail; the descriptor's own
	// page size is not ours to trust, so clamp it rather than send a request
	// the server would reject.
	pageSize := t.Limit
	switch {
	case pageSize != 0:
	case all:
		// Under --all the descriptor's page size only costs requests.
		pageSize = wire.MaxPageSize
	default:
		pageSize = min(max(desc.PageSize, wire.MinPageSize), wire.MaxPageSize)
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
	// The format came through render.ParseFormat, so New cannot fail here.
	w, _ := render.New(cmd.OutOrStdout(), format, opt)
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

// streamAll pages until the server reports no next page. Head is written once,
// Close only after the last page: an interrupted run must leave an unclosed
// document rather than a valid but truncated one.
func streamAll(ctx context.Context, cmd *cobra.Command, api *client.Client, grid string,
	req wire.RowsRequest, format render.Format, opt render.Options, rep *reporter) error {
	w, err := render.New(cmd.OutOrStdout(), format, opt)
	if err != nil {
		return &UsageError{Msg: err.Error()}
	}
	var written int64
	headed := false
	page := req.Page
	for {
		req.Page = page
		resp, _, err := api.Rows(ctx, grid, req)
		if err != nil {
			rep.Done()
			return &pageError{Page: page, Err: err}
		}
		if !headed {
			headed = true
			rep.Total(resp.Total)
			if err := w.Head(resp.Total); err != nil {
				rep.Done()
				return err
			}
		}
		for _, row := range resp.Rows {
			if err := w.Row(row); err != nil {
				rep.Done()
				return err
			}
			written++
		}
		rep.Page(page, written, resp.Total)
		if !resp.HasNext {
			break
		}
		page++
	}
	rep.Done()
	return w.Close()
}

// pageError names the page whose request failed. Hint is filled in by the
// command, which knows the tail and the flags the resume command has to carry.
type pageError struct {
	Page int
	Hint string
	Err  error
}

func (e *pageError) Error() string { return fmt.Sprintf("page %d: %s", e.Page, e.Err) }
func (e *pageError) Unwrap() error { return e.Err }
