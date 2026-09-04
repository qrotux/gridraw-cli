package wire

// SortSpec is one sort term; Dir is "asc" or "desc".
type SortSpec struct {
	Column string `json:"column"`
	Dir    string `json:"dir"`
}

// Clause is one filter condition. Value is absent for isNull, isNotNull,
// isEmpty and isNotEmpty.
type Clause struct {
	Field string `json:"field"`
	Op    Op     `json:"op"`
	Value any    `json:"value,omitempty"`
}

// RowsRequest is the body of POST <base>/{name}/rows. Filters is disjunctive
// normal form: the outer slice is OR, each inner slice is AND.
type RowsRequest struct {
	Columns  []string   `json:"columns,omitempty"`
	Filters  [][]Clause `json:"filters,omitempty"`
	Search   string     `json:"search,omitempty"`
	Sort     []SortSpec `json:"sort,omitempty"`
	Page     int        `json:"page,omitempty"`
	PageSize int        `json:"pageSize,omitempty"`
}

// RowsResponse is the rows page. Total is nil on a grid that sets SkipTotal.
type RowsResponse struct {
	Rows    []map[string]any `json:"rows"`
	Total   *int64           `json:"total"`
	HasPrev bool             `json:"hasPrev"`
	HasNext bool             `json:"hasNext"`
}

// Server limits, mirrored from ../gridraw-go/request.go so a bad request is
// caught before it is sent. They change here when they change there.
const (
	MaxFilterGroups    = 10
	MaxClausesPerGroup = 20
	MaxSortColumns     = 16
	MinPageSize        = 1
	MaxPageSize        = 100
)
