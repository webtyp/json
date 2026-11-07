package tests

import (
	"testing"
	"webtyp.com/json"
	"webtyp.com/model"
)

type randomAccessArrayModel struct {
	First  string
	Second string
	Third  string
	Neg    string
	Len1   int
	Len2   int
}

func (m *randomAccessArrayModel) IsNil() bool { return m == nil }
func (m *randomAccessArrayModel) DecodeFields(r model.FieldReader) {
	if arr, ok := r.Array("items"); ok {
		m.Len1 = arr.Len()
		m.Len2 = arr.Len()
		m.Third = arr.String(2)
		m.First = arr.String(0)
		m.Second = arr.String(1)
		m.Neg = arr.String(-1)
	}
}

func TestParseArraySequentialAndRandomAccess(t *testing.T) {
	var m randomAccessArrayModel
	if err := json.Decode(`{"items":["a","b","c"]}`, &m); err != nil {
		t.Fatal(err)
	}
	if m.Len1 != 3 || m.Len2 != 3 || m.First != "a" || m.Second != "b" || m.Third != "c" || m.Neg != "" {
		t.Fatalf("unexpected array decode: %+v", m)
	}
}

func TestParseObjectOutOfOrderKeys(t *testing.T) {
	m := &simpleModel{}
	if err := json.Decode(`{"active":true,"age":42,"name":"Bob"}`, m); err != nil {
		t.Fatal(err)
	}
	if m.Name != "Bob" || m.Age != 42 || !m.Active {
		t.Fatalf("unexpected out-of-order decode: %+v", m)
	}
}

func TestParseArrayEmpty(t *testing.T) {
	m := &simpleModel{}
	if err := json.Decode(`{"name":[]}`, m); err != nil {
		t.Fatal(err)
	}
}

func TestParseArrayMissingBracket(t *testing.T) {
	m := &simpleModel{}
	if err := json.Decode(`{"name":[1,2}`, m); err == nil {
		t.Fatal("expected error for missing bracket")
	}
}

func TestParseArrayBadSeparator(t *testing.T) {
	m := &simpleModel{}
	if err := json.Decode(`{"name":[1;2]}`, m); err == nil {
		t.Fatal("expected error for bad separator in array")
	}
}

func TestParseValueArray(t *testing.T) {
	m := &simpleModel{}
	if err := json.Decode(`{"name":[1,2,3]}`, m); err != nil {
		t.Fatal(err)
	}
}

func TestParseValueUnknownChar(t *testing.T) {
	m := &simpleModel{}
	if err := json.Decode(`{"name":@invalid}`, m); err == nil {
		t.Fatal("expected error for unknown character")
	}
}
