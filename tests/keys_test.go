package tests

import (
	"reflect"
	"testing"

	"webtyp.com/json"
)

// Keys lists an object's field names in document order, whatever their values are.
func TestKeys(t *testing.T) {
	got, err := json.Keys(` {"query":{"type":"string"}, "status":{"enum":["a","b"]},"n":1,"e\"x":[1,{"y":2}],"z":null} `)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"query", "status", "n", `e"x`, "z"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Keys = %q, want %q", got, want)
	}
}

// An empty object has no keys; anything that is not an object is an error.
func TestKeys_EmptyAndInvalid(t *testing.T) {
	if got, err := json.Keys(`{}`); err != nil || len(got) != 0 {
		t.Fatalf("Keys({}) = %q, %v", got, err)
	}
	for _, in := range []string{``, `[]`, `"a"`, `{"a"}`, `{"a":1`, `{a:1}`} {
		if _, err := json.Keys(in); err == nil {
			t.Errorf("Keys(%q): want an error", in)
		}
	}
}
