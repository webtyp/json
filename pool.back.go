//go:build !wasm

package json

import "sync"

var (
	writerPool = sync.Pool{New: func() any { return &jsonWriter{} }}
	readerPool = sync.Pool{New: func() any { return &jsonReader{} }}
)

func getWriter() *jsonWriter  { return writerPool.Get().(*jsonWriter) }
func putWriter(w *jsonWriter) { writerPool.Put(w) }

func getReader() *jsonReader  { return readerPool.Get().(*jsonReader) }
func putReader(r *jsonReader) { readerPool.Put(r) }
