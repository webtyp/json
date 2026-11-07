package tests

import (
    stdjson "encoding/json"
    "strings"
    "testing"
    "webtyp.com/json"
    "webtyp.com/model"
)

var benchJSONStr = `{"name":"Alice","email":"alice@example.com","age":30,"score":9.5}`

type benchUserBatch struct {
    Items []benchUser `json:"items"`
}

func (b *benchUserBatch) IsNil() bool { return b == nil }
func (b *benchUserBatch) DecodeFields(r model.FieldReader) {
    if arr, ok := r.Array("items"); ok {
        n := arr.Len()
        b.Items = make([]benchUser, n)
        for i := 0; i < n; i++ {
            arr.Object(i, &b.Items[i])
        }
    }
}

var benchArray100JSON = func() string {
    items := make([]string, 100)
    for i := range items {
        items[i] = benchJSONStr
    }
    return `{"items":[` + strings.Join(items, ",") + `]}`
}()

func BenchmarkDecode_webtyp(b *testing.B) {
    b.ReportAllocs()
    for i := 0; i < b.N; i++ {
        u := &benchUser{}
        if err := json.Decode(benchJSONStr, u); err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkDecode_stdlib(b *testing.B) {
    b.ReportAllocs()
    for i := 0; i < b.N; i++ {
        var u benchUser
        if err := stdjson.Unmarshal([]byte(benchJSONStr), &u); err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkDecodeArray100_webtyp(b *testing.B) {
    b.ReportAllocs()
    for i := 0; i < b.N; i++ {
        var batch benchUserBatch
        if err := json.Decode(benchArray100JSON, &batch); err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkDecodeArray100_stdlib(b *testing.B) {
    raw := []byte(benchArray100JSON)
    b.ReportAllocs()
    for i := 0; i < b.N; i++ {
        var batch benchUserBatch
        if err := stdjson.Unmarshal(raw, &batch); err != nil {
            b.Fatal(err)
        }
    }
}
