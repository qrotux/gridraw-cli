package wire

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestRowsResponseTotalIsOptional(t *testing.T) {
	var withTotal, without RowsResponse
	if err := json.Unmarshal([]byte(`{"rows":[],"total":0,"hasPrev":false,"hasNext":true}`), &withTotal); err != nil {
		t.Fatal(err)
	}
	if withTotal.Total == nil || *withTotal.Total != 0 {
		t.Errorf("Total = %v, want pointer to 0", withTotal.Total)
	}
	if !withTotal.HasNext {
		t.Error("HasNext = false, want true")
	}
	if err := json.Unmarshal([]byte(`{"rows":[],"hasPrev":false,"hasNext":false}`), &without); err != nil {
		t.Fatal(err)
	}
	if without.Total != nil {
		t.Errorf("Total = %v, want nil on a skipTotal grid", *without.Total)
	}
}

func TestRowsRequestOmitsEmptyFields(t *testing.T) {
	got, err := json.Marshal(RowsRequest{Page: 1, PageSize: 25})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"page":1,"pageSize":25}`
	if string(got) != want {
		t.Errorf("marshal = %s, want %s", got, want)
	}
}

func TestDescriptorFields(t *testing.T) {
	const src = `{"name":"users","idColumn":"id","pageSize":25,"skipTotal":true,
	  "defaultSort":{"column":"createdAt","dir":"desc"},"search":{"columns":["Email"]},
	  "columns":[{"key":"role","type":"enum","title":"Role","sortable":true,"defaultVisible":false,
	    "filter":{"operators":[{"op":"in","label":"is one of"}],
	              "enumValues":[{"value":"user","label":"User"}],"widget":"tags"}},
	    {"key":"tags","type":"string","title":"Tags","array":true},
	    {"key":"slot","type":"time","title":"Slot","step":900}]}`
	var d Descriptor
	if err := json.Unmarshal([]byte(src), &d); err != nil {
		t.Fatal(err)
	}
	if !d.SkipTotal || d.DefaultSort.Column != "createdAt" || d.Search == nil {
		t.Fatalf("descriptor = %+v", d)
	}
	role := d.Column("role")
	if role == nil || role.Filter == nil || len(role.Filter.Operators) != 1 ||
		role.Filter.Operators[0].Op != OpIn || role.Filter.Widget != "tags" {
		t.Fatalf("role = %+v", role)
	}
	if !d.Column("tags").Array || d.Column("slot").Step != 900 {
		t.Fatal("array or step lost")
	}
	if d.Column("nope") != nil {
		t.Error("Column returned a value for an unknown key")
	}
}

func TestDecodeRowsKeepsNumberLiterals(t *testing.T) {
	dec := json.NewDecoder(bytes.NewReader([]byte(`{"rows":[{"n":12345678901234567890}],"hasPrev":false,"hasNext":false}`)))
	dec.UseNumber()
	var r RowsResponse
	if err := dec.Decode(&r); err != nil {
		t.Fatal(err)
	}
	if got := r.Rows[0]["n"]; got != json.Number("12345678901234567890") {
		t.Errorf("n = %#v, want the literal preserved", got)
	}
}
