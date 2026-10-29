//go:build wasm

package json

// WASM is single-threaded: allocate directly instead of sync.Pool.

func getWriter() *jsonWriter  { return &jsonWriter{} }
func putWriter(_ *jsonWriter) {}

func getReader() *jsonReader  { return &jsonReader{} }
func putReader(_ *jsonReader) {}
