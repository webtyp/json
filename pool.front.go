//go:build wasm

package json

// WASM is single-threaded: reuse writers and readers via a lock-free stack instead of sync.Pool.
var (
	writerPool []*jsonWriter
	readerPool []*jsonReader
)

func getWriter() *jsonWriter {
	n := len(writerPool)
	if n == 0 {
		return &jsonWriter{}
	}
	w := writerPool[n-1]
	writerPool = writerPool[:n-1]
	return w
}

func putWriter(w *jsonWriter) {
	w.b = nil
	w.st = nil
	w.open = nil
	writerPool = append(writerPool, w)
}

func getReader() *jsonReader {
	n := len(readerPool)
	if n == 0 {
		return &jsonReader{}
	}
	r := readerPool[n-1]
	readerPool = readerPool[:n-1]
	return r
}

func putReader(r *jsonReader) {
	r.p = nil
	r.err = nil
	readerPool = append(readerPool, r)
}
