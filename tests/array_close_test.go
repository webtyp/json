package tests

import (
	"strings"
	"testing"

	"webtyp.com/json"
	"webtyp.com/model"
)

type encodableFunc func(w model.FieldWriter)

func (f encodableFunc) EncodeFields(w model.FieldWriter) { f(w) }
func (f encodableFunc) IsNil() bool                      { return false }

func assertEncode(t *testing.T, e model.Encodable, expected string) {
	t.Helper()
	var out string
	err := json.Encode(e, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != expected {
		t.Errorf("expected %q, got %q", expected, out)
	}
}

func assertEncodeError(t *testing.T, e model.Encodable, wantErrorContent string) {
	t.Helper()
	var out string
	err := json.Encode(e, &out)
	if err == nil {
		t.Fatalf("expected error containing %q, got none (output: %q)", wantErrorContent, out)
	}
	if !strings.Contains(err.Error(), wantErrorContent) {
		t.Errorf("expected error containing %q, got %q", wantErrorContent, err.Error())
	}
}

func TestArray_ClosesItselfAfterN(t *testing.T) {
	assertEncode(t, encodableFunc(func(w model.FieldWriter) {
		a := w.Array("ids", 2)
		a.Int(1)
		a.Int(2)
		w.Int("x", 3)
	}), `{"ids":[1,2],"x":3}`)
}

func TestArray_EmptyIsClosed(t *testing.T) {
	assertEncode(t, encodableFunc(func(w model.FieldWriter) {
		w.Array("ids", 0)
		w.Int("x", 3)
	}), `{"ids":[],"x":3}`)
}

func TestArray_LegacyCloseIsHarmless(t *testing.T) {
	assertEncode(t, encodableFunc(func(w model.FieldWriter) {
		a := w.Array("ids", 2)
		a.Int(1)
		a.Int(2)
		if closer, ok := a.(interface{ Close() }); ok {
			closer.Close()
		}
		w.Int("x", 3)
	}), `{"ids":[1,2],"x":3}`)
}

func TestArray_ObjectsWithNestedArrays(t *testing.T) {
	assertEncode(t, encodableFunc(func(w model.FieldWriter) {
		a := w.Array("records", 2)

		a.Object(encodableFunc(func(w model.FieldWriter) {
			w.String("id", "r1")
			w.Array("tags", 1).String("t")
		}))

		a.Object(encodableFunc(func(w model.FieldWriter) {
			w.String("id", "r2")
			w.Array("tags", 1).String("t")
		}))
	}), `{"records":[{"id":"r1","tags":["t"]},{"id":"r2","tags":["t"]}]}`)
}

func TestArray_FewerThanAnnouncedFails(t *testing.T) {
	assertEncodeError(t, encodableFunc(func(w model.FieldWriter) {
		a := w.Array("ids", 2)
		a.Int(1)
		w.Int("x", 3)
	}), "ids")
}

func TestArray_FewerAtObjectEndFails(t *testing.T) {
	assertEncodeError(t, encodableFunc(func(w model.FieldWriter) {
		a := w.Array("ids", 2)
		a.Int(1)
	}), "ids")
}

func TestArray_MoreThanAnnouncedFails(t *testing.T) {
	assertEncodeError(t, encodableFunc(func(w model.FieldWriter) {
		a := w.Array("ids", 1)
		a.Int(1)
		a.Int(2)
	}), "ids")
}

type decodableFunc func(r model.FieldReader)

func (f decodableFunc) DecodeFields(r model.FieldReader) { f(r) }
func (f decodableFunc) IsNil() bool                      { return false }

func TestArray_RoundTrip(t *testing.T) {
	var out string
	err := json.Encode(encodableFunc(func(w model.FieldWriter) {
		a := w.Array("records", 2)

		a.Object(encodableFunc(func(w model.FieldWriter) {
			w.String("id", "r1")
			w.Array("tags", 1).String("t")
		}))

		a.Object(encodableFunc(func(w model.FieldWriter) {
			w.String("id", "r2")
			w.Array("tags", 1).String("t")
		}))
	}), &out)

	if err != nil {
		t.Fatalf("unexpected encode error: %v", err)
	}

	count := 0
	err = json.Decode(out, decodableFunc(func(r model.FieldReader) {
		if arr, ok := r.Array("records"); ok {
			count = arr.Len()
		}
	}))

	if err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 records, got %d", count)
	}
}
