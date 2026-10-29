package json

import "webtyp.com/model"

import (
	"webtyp.com/escape"
	"webtyp.com/fmt"
)


// encodeState carries the first error of one Encode call to every nested writer.
type encodeState struct{ err error }

func (s *encodeState) fail(err error) {
	if s.err == nil {
		s.err = err
	}
}

// errArrayShort: the caller announced n elements and wrote fewer.
func errArrayShort(name string, want, got int) error {
	return fmt.Err("json", "encode", "array", name, "announced", fmt.Convert(want).String(), "got", fmt.Convert(got).String())
}

// errArrayLong: the caller wrote more elements than it announced.
func errArrayLong(name string, want int) error {
	return fmt.Err("json", "encode", "array", name, "announced", fmt.Convert(want).String(), "got more")
}

// Writer is io.Writer redeclared here. Structurally identical, so any
// io.Writer satisfies it without the caller changing a line.
type Writer interface {
	Write(p []byte) (n int, err error)
}

type jsonWriter struct {
	b     *fmt.Conv
	first bool
	st    *encodeState
	open  *jsonArrayWriter
}

func (w *jsonWriter) settle() {
	if w.open != nil && !w.open.closed {
		w.st.fail(errArrayShort(w.open.name, w.open.want, w.open.got))
		w.b.WriteByte(']')
		w.open.closed = true
	}
	w.open = nil
}

func (w *jsonWriter) maybeComma() {
	w.settle()
	if !w.first {
		w.b.WriteByte(',')
	}
	w.first = false
}

func (w *jsonWriter) writeKey(name string) {
	if name != "" {
		w.b.WriteByte('"')
		escape.JSON(w.b, name)
		w.b.WriteByte('"')
		w.b.WriteByte(':')
	}
}

func (w *jsonWriter) String(name, val string) {
	w.maybeComma()
	w.writeKey(name)
	w.b.WriteByte('"')
	escape.JSON(w.b, val)
	w.b.WriteByte('"')
}

func (w *jsonWriter) Int(name string, val int64) {
	w.maybeComma()
	w.writeKey(name)
	w.b.WriteInt(val)
}

func (w *jsonWriter) Uint(name string, val uint64) {
	w.maybeComma()
	w.writeKey(name)
	w.b.WriteString(fmt.Convert(val).String())
}

func (w *jsonWriter) Float(name string, val float64) {
	w.maybeComma()
	w.writeKey(name)
	w.b.WriteFloat(val)
}

func (w *jsonWriter) Bool(name string, val bool) {
	w.maybeComma()
	w.writeKey(name)
	if val {
		w.b.WriteString("true")
	} else {
		w.b.WriteString("false")
	}
}

func (w *jsonWriter) Bytes(name string, val []byte) {
	w.maybeComma()
	w.writeKey(name)
	w.b.WriteByte('"')
	escape.JSON(w.b, string(val))
	w.b.WriteByte('"')
}

func (w *jsonWriter) Null(name string) {
	w.maybeComma()
	w.writeKey(name)
	w.b.WriteString("null")
}

func (w *jsonWriter) Raw(name, val string) {
	w.maybeComma()
	w.writeKey(name)
	if val == "" {
		w.b.WriteString("null")
		return
	}
	w.b.WriteString(val)
}

func (w *jsonWriter) Object(name string, val model.Encodable) {
	w.maybeComma()
	w.writeKey(name)
	if val == nil || val.IsNil() {
		w.b.WriteString("null")
		return
	}
	if r, ok := val.(interface{ Raw() string }); ok {
		w.b.WriteString(r.Raw())
		return
	}

	w.b.WriteByte('{')
	iw := getWriter()
	iw.b = w.b
	iw.first = true
	iw.st = w.st
	iw.open = nil
	val.EncodeFields(iw)
	iw.settle()
	w.b.WriteByte('}')
	putWriter(iw)
}

func (w *jsonWriter) Array(name string, n int) model.ArrayWriter {
	w.maybeComma()
	w.writeKey(name)
	w.b.WriteByte('[')
	aw := &jsonArrayWriter{
		b:     w.b,
		first: true,
		name:  name,
		want:  n,
		st:    w.st,
	}
	if n <= 0 {
		w.b.WriteByte(']')
		aw.closed = true
	} else {
		w.open = aw
	}
	return aw
}

type jsonArrayWriter struct {
	b      *fmt.Conv
	first  bool
	name   string
	want   int
	got    int
	closed bool
	st     *encodeState
}

// push admits one element; after the last announced one it closes the array.
func (w *jsonArrayWriter) push() bool {
	if w.closed {
		w.st.fail(errArrayLong(w.name, w.want))
		return false
	}
	w.got++
	w.maybeComma()
	return true
}

func (w *jsonArrayWriter) done() {
	if w.got == w.want {
		w.b.WriteByte(']')
		w.closed = true
	}
}

func (w *jsonArrayWriter) maybeComma() {
	if !w.first {
		w.b.WriteByte(',')
	}
	w.first = false
}

func (w *jsonArrayWriter) String(val string) {
	if !w.push() {
		return
	}
	w.b.WriteByte('"')
	escape.JSON(w.b, val)
	w.b.WriteByte('"')
	w.done()
}

func (w *jsonArrayWriter) Int(val int64) {
	if !w.push() {
		return
	}
	w.b.WriteInt(val)
	w.done()
}

func (w *jsonArrayWriter) Float(val float64) {
	if !w.push() {
		return
	}
	w.b.WriteFloat(val)
	w.done()
}

func (w *jsonArrayWriter) Bool(val bool) {
	if !w.push() {
		return
	}
	if val {
		w.b.WriteString("true")
	} else {
		w.b.WriteString("false")
	}
	w.done()
}

func (w *jsonArrayWriter) Bytes(val []byte) {
	if !w.push() {
		return
	}
	w.b.WriteByte('"')
	escape.JSON(w.b, string(val))
	w.b.WriteByte('"')
	w.done()
}

func (w *jsonArrayWriter) Object(val model.Encodable) {
	if !w.push() {
		return
	}
	if val == nil || val.IsNil() {
		w.b.WriteString("null")
		w.done()
		return
	}
	if r, ok := val.(interface{ Raw() string }); ok {
		w.b.WriteString(r.Raw())
		w.done()
		return
	}
	w.b.WriteByte('{')
	iw := getWriter()
	iw.b = w.b
	iw.first = true
	iw.st = w.st
	iw.open = nil
	val.EncodeFields(iw)
	iw.settle()
	w.b.WriteByte('}')
	putWriter(iw)
	w.done()
}

func (w *jsonArrayWriter) Close() {
	if w.closed {
		return
	}
	w.st.fail(errArrayShort(w.name, w.want, w.got))
	w.b.WriteByte(']')
	w.closed = true
}

// Encode serializes an Encodable to JSON.
// output: *[]byte | *string | Writer.
func Encode(data model.Encodable, output any) error {
	b := fmt.GetConv()
	defer b.PutConv()
	st := &encodeState{}

	if data == nil || data.IsNil() {
		b.WriteString("null")
	} else {
		w := getWriter()
		w.b = b
		w.first = true
		w.st = st
		w.open = nil

		var slice model.FielderSlice
		if s, ok := data.(interface{ FielderSlice() model.FielderSlice }); ok {
			slice = s.FielderSlice()
		} else if s, ok := data.(model.FielderSlice); ok {
			slice = s
		}

		if slice != nil {
			b.WriteByte('[')
			for i := 0; i < slice.Len(); i++ {
				if i > 0 {
					b.WriteByte(',')
				}
				if it, ok := slice.At(i).(model.Encodable); ok {
					iw := getWriter()
					iw.b = b
					iw.first = true
					iw.st = st
					iw.open = nil
					if it.IsNil() {
						b.WriteString("null")
					} else {
						b.WriteByte('{')
						it.EncodeFields(iw)
						iw.settle()
						b.WriteByte('}')
					}
					putWriter(iw)
				} else {
					b.WriteString("null")
				}
			}
			b.WriteByte(']')
		} else {
			b.WriteByte('{')
			data.EncodeFields(w)
			w.settle()
			b.WriteByte('}')
		}
		putWriter(w)
	}

	if b.GetString(fmt.BuffErr) != "" {
		return fmt.Err(b.GetString(fmt.BuffErr))
	}
	if st.err != nil {
		return st.err
	}

	switch out := output.(type) {
	case *[]byte:
		res := b.Bytes()
		cpy := make([]byte, len(res))
		copy(cpy, res)
		*out = cpy
	case *string:
		*out = b.GetString(fmt.BuffOut)
	case Writer:
		_, err := out.Write(b.Bytes())
		return err
	default:
		return fmt.Err("json", "encode", "output must be *[]byte, *string, or json.Writer")
	}
	return nil
}
