# JSON Benchmarks

This directory contains benchmarking tools to compare JSON against the standard library `encoding/json` when compiled to WebAssembly using TinyGo.

## Build Script

The `build.sh` script compiles the WASM binaries with different JSON implementations.

### Usage

```bash
# Build with JSON (default)
./build.sh

# Build with encoding/json (stdlib)
./build.sh stlib
```

# Run the benchmark server

```bash
# Start the local server to serve the compiled WASM
go run ./web/server.go
```

You can then open `http://localhost:8080` in a browser to see the benchmark results.

### Output

The compiled WASM binary is output to `web/public/main.wasm`.

The script will display:
- Uncompressed binary size
- Gzipped size (for realistic deployment comparison)

## Source Files

- [`clients/tinyjson/main.go`](clients/tinyjson/main.go) - Implementation using JSON
- [`clients/stdlib/main.go`](clients/stdlib/main.go) - Implementation using encoding/json (stdlib)

Both files implement the same functionality to ensure fair comparison.

## Performance Results

Last updated: 2026-11-07 (`go test -bench=. -benchmem ./tests/...`)

### Go Benchmark (`go test -bench`)

| Benchmark | webtyp/json (before) | webtyp/json (after) | encoding/json | Faster vs stdlib | Δ allocs vs stdlib |
|-----------|----------------------|---------------------|---------------|------------------|--------------------|
| Encode | 244 ns/op · 96 B/op · 2 allocs | **217 ns/op · 80 B/op · 1 allocs** | 244 ns/op · 80 B/op · 1 allocs | **+11%** | 0 |
| Decode | 483 ns/op · 125 B/op · 5 allocs | **308 ns/op · 93 B/op · 4 allocs** | 1083 ns/op · 376 B/op · 8 allocs | **+72% (3.5x)** | -4 |
| DecodeArray100 | 540,092 ns/op · 38,340 B/op · 605 allocs | **33,819 ns/op · 7,849 B/op · 203 allocs** | 88,297 ns/op · 18,952 B/op · 216 allocs | **+62% (2.6x)** | -13 |
| RoundTrip | 825 ns/op · 237 B/op · 8 allocs | **587 ns/op · 189 B/op · 6 allocs** | 1306 ns/op · 376 B/op · 8 allocs | **+55% (2.2x)** | -2 |

> Run: `go test -bench=. -benchmem ./tests/...`
>
> **2026-11-07 optimizations (`parser.go`, `decode.go`, `encode.go`, `pool.front.go`):**
> 1. **Sequential array & object cursors (`curIdx`/`curPos` in `jsonArrayReader`, `nextPos` in `jsonReader`):** Advancing sequentially across array elements and object fields is now $O(1)$ per element/field instead of rewinding to `start` on every lookup ($O(N^2)$), while falling back to `start` on out-of-order access.
> 2. **Eliminated redundant `skipObject` error allocation:** `jsonReader.Object` and `jsonArrayReader.Object` now fast-path when `innerReader` already reached the closing `}` and consume `{` before `skipObject()` on fallback, eliminating `400` spurious heap allocations per 100-item batch.
> 3. **Embedded root state & array readers/writers (`rootP`, `rootSt`, `arr`):** `jsonWriter` and `jsonReader` hold their root `encodeState`, `parser`, and active `jsonArrayWriter`/`jsonArrayReader` inline inside the pooled struct, and `pool.front.go` reuses readers/writers via a single-threaded free-list in WASM.

### WASM Binary Size

Compiled with `tinygo build -target wasm -no-debug -opt=z`.

| Implementation | Uncompressed | Gzipped |
| :--- | :--- | :--- |
| **webtyp/json** | **51 KB** | **20 KB** |
| encoding/json (stdlib) | 270 KB | 118 KB |

> Run: `./build.sh` (webtyp/json) and `./build.sh stlib` (stdlib) from this directory.

### Analysis

**webtyp/json is 83% smaller gzipped** (20 KB vs 118 KB) making it ideal for web apps where bundle size matters. By eliminating the `reflect` package, it not only significantly reduces the final WASM binary size, but also makes **encoding ~1.3x faster**, **decoding ~2.1x faster**, and **roundtripping ~1.7x faster** than the standard library.

**Use webtyp/json when:** Bundle size and raw decoding performance are critical, or running in restricted WASM environments.
**Use Stdlib when:** You need to work dynamically with arbitrary unknown schemas (like `map[string]any`), as webtyp/json is optimized strictly for predefined structs (`fmt.Encodable`/`fmt.Decodable`).

See the [main README](../README.md#benchmarks) for detailed benchmark results.
