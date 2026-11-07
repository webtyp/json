package json

import "webtyp.com/model"

import (
	"webtyp.com/fmt"
	"unsafe"
)

// Reader is io.Reader redeclared here, for the same reason as Writer.
type Reader interface {
	Read(p []byte) (n int, err error)
}

// Decode parses JSON into a Decodable.
// input: []byte | string | Reader.
func Decode(input any, data model.Decodable) error {
	if data == nil || data.IsNil() {
		return fmt.Err("json", "decode", "destination is nil")
	}

	var raw []byte
	switch in := input.(type) {
	case []byte:
		raw = in
	case string:
		// Avoid copy: parser is read-only, never modifies data.
		raw = unsafe.Slice(unsafe.StringData(in), len(in))
	case Reader:
		var buf []byte
		tmp := make([]byte, 4096)
		for {
			n, err := in.Read(tmp)
			if n > 0 {
				buf = append(buf, tmp[:n]...)
			}
			if err != nil {
				break
			}
		}
		raw = buf
	default:
		return fmt.Err("json", "decode", "input must be []byte, string, or json.Reader")
	}

	r := getReader()
	r.rootP = parser{data: raw}
	r.p = &r.rootP
	r.p.skipWhitespace()
	r.nextPos = 0
	r.err = nil
	defer putReader(r)

	var slice model.FielderSlice
	if s, ok := data.(interface{ FielderSlice() model.FielderSlice }); ok {
		slice = s.FielderSlice()
	} else if s, ok := data.(model.FielderSlice); ok {
		slice = s
	}

	if slice != nil {
		// Special case for root level arrays of Fielders
		if r.p.peek() != '[' {
			return fmt.Err("json", "decode", "expected array, got "+string(r.p.peek()))
		}
		r.arr = jsonArrayReader{p: r.p, start: r.p.pos}
		n := r.arr.Len()
		for i := 0; i < n; i++ {
			it := slice.Append()
			if dec, ok := it.(model.Decodable); ok {
				r.arr.Object(i, dec)
			}
		}
		return nil
	}

	if r.p.peek() != '{' {
		return fmt.Err("json", "decode", "expected object, got "+string(r.p.peek()))
	}
	r.start = r.p.pos
	data.DecodeFields(r)
	return r.err
}
