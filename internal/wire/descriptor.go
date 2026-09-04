// Package wire holds the gridraw protocol types. It performs no I/O, so the
// parser and the writers can share them without importing the HTTP client.
package wire

// Op is a filter operator name as it travels on the wire.
type Op string

const (
	OpEq          Op = "eq"
	OpNeq         Op = "neq"
	OpContains    Op = "contains"
	OpNotContains Op = "notContains"
	OpStarts      Op = "starts"
	OpEnds        Op = "ends"
	OpGt          Op = "gt"
	OpGte         Op = "gte"
	OpLt          Op = "lt"
	OpLte         Op = "lte"
	OpBetween     Op = "between"
	OpNotBetween  Op = "notBetween"
	OpIn          Op = "in"
	OpNotIn       Op = "notIn"
	OpIsNull      Op = "isNull"
	OpIsNotNull   Op = "isNotNull"

	OpContainsAny    Op = "containsAny"
	OpContainsAll    Op = "containsAll"
	OpContainsOnly   Op = "containsOnly"
	OpNotContainsAny Op = "notContainsAny"
	OpIsEmpty        Op = "isEmpty"
	OpIsNotEmpty     Op = "isNotEmpty"
)

// Column types as published in the descriptor.
const (
	TypeString   = "string"
	TypeUUID     = "uuid"
	TypeNumber   = "number"
	TypeDecimal  = "decimal"
	TypeDate     = "date"
	TypeTime     = "time"
	TypeDatetime = "datetime"
	TypeBoolean  = "boolean"
	TypeEnum     = "enum"
	TypeJSON     = "json"
)

// GridInfo is one entry of GET <base>/-/list.
type GridInfo struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// CatalogEntry is one entry of GET <base>/-/registry.
type CatalogEntry struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Columns     []CatalogColumn `json:"columns"`
}

// CatalogColumn is a column as the registry endpoint publishes it.
type CatalogColumn struct {
	Key         string `json:"key"`
	Title       string `json:"title"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

// Descriptor is GET <base>/{name}.
type Descriptor struct {
	Name            string   `json:"name"`
	Description     string   `json:"description,omitempty"`
	IDColumn        string   `json:"idColumn"`
	PageSize        int      `json:"pageSize"`
	PageSizeOptions []int    `json:"pageSizeOptions,omitempty"`
	DefaultSort     SortSpec `json:"defaultSort"`
	Search          *Search  `json:"search"`
	SkipTotal       bool     `json:"skipTotal,omitempty"`
	Columns         []Column `json:"columns"`
}

// Search lists the columns the quick search runs over; nil when none is searchable.
type Search struct {
	Columns []string `json:"columns"`
}

// Column is one descriptor column. On an array column Type is the element type.
type Column struct {
	Key            string  `json:"key"`
	Type           string  `json:"type"`
	Title          string  `json:"title"`
	Description    string  `json:"description,omitempty"`
	Sortable       bool    `json:"sortable"`
	DefaultVisible bool    `json:"defaultVisible"`
	Array          bool    `json:"array,omitempty"`
	Step           int     `json:"step,omitempty"`
	Filter         *Filter `json:"filter,omitempty"`
}

// Filter is the filtering contract of a column; nil on a non-filterable one.
type Filter struct {
	Operators  []OperatorSpec `json:"operators"`
	EnumValues []EnumValue    `json:"enumValues,omitempty"`
	Widget     string         `json:"widget,omitempty"`
}

// OperatorSpec is an operator the server accepts for a column, with its label.
type OperatorSpec struct {
	Op    Op     `json:"op"`
	Label string `json:"label"`
}

// EnumValue is one allowed value of an enum column.
type EnumValue struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// Column returns the column with this key, or nil.
func (d *Descriptor) Column(key string) *Column {
	for i := range d.Columns {
		if d.Columns[i].Key == key {
			return &d.Columns[i]
		}
	}
	return nil
}

// Allows reports whether the column offers this operator.
func (c *Column) Allows(op Op) bool {
	if c.Filter == nil {
		return false
	}
	for _, o := range c.Filter.Operators {
		if o.Op == op {
			return true
		}
	}
	return false
}

// VisibleKeys returns the defaultVisible columns in declaration order.
func (d *Descriptor) VisibleKeys() []string {
	var keys []string
	for _, c := range d.Columns {
		if c.DefaultVisible {
			keys = append(keys, c.Key)
		}
	}
	return keys
}
