---
PLAN: "fix(encode): an array closes itself after the n elements it announced; a wrong count is an error, not broken JSON"
EXECUTOR: jules
REVIEWER: none
STATUS: running
SESSION: 3143793184284194820
---

> Read AGENTS.md at the repo root before any change.

# PLAN — arrays that cannot be left open

## Why

`model.FieldWriter.Array(name, n)` announces `n` elements. `jsvalue` (preallocates `n`) and
`binary` (writes `n` first) already rely on it; **only this package ignores `n`** and needs the
caller to remember `ArrayWriter.Close()`. Two repos (`webtyp/view`, `webtyp/access`) forgot it and
every save of the generic screens sent `{"records":[{…}}` — invalid JSON, a 400 on every save, and
no test saw it. This plan makes the JSON writer close the array by itself after the `n`-th
element, and turns a wrong count into an `Encode` error. It is step 1 of 3 of the wave
(`mjosefa-cms/docs/MVP_DEPLOYMENT_ROADMAP.md`, M2f): step 2 deletes the `Close()` calls in every
repo; step 3 removes `Close()` from `model.ArrayWriter`. **In this plan `Close()` stays and becomes
harmless**, so no consumer breaks.

## Design gate (api-design)

1. **Prior art.** `encoding/json` encodes whole values (no open/close); `jsoniter.Stream` and
   `easyjson`'s `jwriter` need explicit `WriteArrayStart/End` (the forgettable design we have);
   protobuf's packed fields and CBOR definite-length arrays write the count first, so the end is
   implied — the model `binary` and `jsvalue` already follow. JSON has no count on the wire, but
   the writer knows `n`, so it can imply the end too.
2. **Names.** No new exported name.
3. **Ledger.** Concepts −1 (nobody has to know about `Close`), lines at the call site −1 per
   array (after step 2), ways to do it: unchanged now, −1 after step 3.
4. **Where.** Here: the JSON writer is the only implementation that needs the end marker.
5. **Deletes.** The array-writer pool on the server (`arrayWriterPool`): a pooled writer could be
   reused while a caller still holds it and later calls `Close()` on it.

## Stage 1 — `encode.go`

Shared error state for one `Encode` call:

```go
// encodeState carries the first error of one Encode call to every nested writer.
type encodeState struct{ err error }

func (s *encodeState) fail(err error) {
	if s.err == nil {
		s.err = err
	}
}
```

- `jsonWriter` gets two fields: `st *encodeState` and `open *jsonArrayWriter` (the array it handed
  out that has not received all its elements yet).
- Every place that creates a nested `jsonWriter` (`jsonWriter.Object`, `jsonArrayWriter.Object`,
  the slice loop and the object branch in `Encode`) sets `iw.st = <parent's st>` and `iw.open = nil`.
  `Encode` creates `st := &encodeState{}` for the root writer.
- New method `func (w *jsonWriter) settle()`: if `w.open != nil && !w.open.closed`, record
  `errArrayShort` (below) with the array's name, `n` and count, and write `]` so the output stays
  balanced; then `w.open = nil`. Call `w.settle()` at the start of `maybeComma` (before a new field)
  and right before every `'}'` written after `EncodeFields(iw)` (the three places above).
- `jsonWriter.Array(name, n)`: `w.maybeComma()`, key, `[`. Then `aw := &jsonArrayWriter{b: w.b,
  first: true, name: name, want: n, st: w.st}`. If `n <= 0`: write `]`, set `aw.closed = true`.
  Else `w.open = aw`. Return `aw`.
- `jsonArrayWriter` fields: `b *fmt.Conv`, `first bool`, `name string`, `want, got int`,
  `closed bool`, `st *encodeState`. Every element method (`String`, `Int`, `Float`, `Bool`,
  `Bytes`, `Object`) starts with `if !w.push() { return }` where:

  ```go
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
  ```

  and ends with `w.done()`: `if w.got == w.want { w.b.WriteByte(']'); w.closed = true }`. For
  `Object`, call `w.done()` on **every** path: after the `null` of a nil value, after `Raw()`, and
  after the element's `'}'` (a `null` element counts as one element).

Sweep done before writing this plan: all 60 `Array(` calls in `webtyp` and `veltylabs` pass
`len(x)` of the slice they then loop over without skipping, so no current caller changes output.
- `jsonArrayWriter.Close()`: if `w.closed` → return (no-op: the array already ended). Else record
  `errArrayShort`, write `]`, `w.closed = true`.
- Delete `getArrayWriter`/`putArrayWriter` from `pool.back.go` and `pool.front.go` and
  `arrayWriterPool`; arrays are allocated with `&jsonArrayWriter{…}`.
- At the end of `Encode`, after the existing `BuffErr` check: `if st.err != nil { return st.err }`.

Errors (in `encode.go`, named, no literal in the logic):

```go
// errArrayShort: the caller announced n elements and wrote fewer.
func errArrayShort(name string, want, got int) error {
	return fmt.Err("json", "encode", "array", name, "announced", want, "got", got)
}

// errArrayLong: the caller wrote more elements than it announced.
func errArrayLong(name string, want int) error {
	return fmt.Err("json", "encode", "array", name, "announced", want, "got more")
}
```

Use the same word style as the existing `fmt.Err("json", "encode", …)` call in `Encode`. If
`fmt.Err` does not accept `int` arguments, pass `fmt.Convert(want).String()`.

## Stage 2 — `tests/array_close_test.go` (new)

Hand-written `Encodable` helpers in the test file (no ormc), each with `EncodeFields(w
model.FieldWriter)` and `IsNil() bool { return false }`. Assert the **exact** output string of
`json.Encode(v, &s)`, and for the error cases that `err != nil` and `err.Error()` contains the
field name.

| Test | `EncodeFields` writes | Expected |
|---|---|---|
| `TestArray_ClosesItselfAfterN` | `a := w.Array("ids", 2); a.Int(1); a.Int(2); w.Int("x", 3)` (no `Close`) | `{"ids":[1,2],"x":3}` |
| `TestArray_EmptyIsClosed` | `w.Array("ids", 0); w.Int("x", 3)` | `{"ids":[],"x":3}` |
| `TestArray_LegacyCloseIsHarmless` | as the first, plus `a.Close()` after `a.Int(2)` | `{"ids":[1,2],"x":3}` |
| `TestArray_ObjectsWithNestedArrays` | `a := w.Array("records", 2)`, two `a.Object(rec)` where `rec` writes `w.String("id","r1")` and `w.Array("tags",1).String("t")` | `{"records":[{"id":"r1","tags":["t"]},{"id":"r2","tags":["t"]}]}` |
| `TestArray_FewerThanAnnouncedFails` | `a := w.Array("ids", 2); a.Int(1); w.Int("x", 3)` | error containing `ids` |
| `TestArray_FewerAtObjectEndFails` | `a := w.Array("ids", 2); a.Int(1)` (nothing after) | error containing `ids` |
| `TestArray_MoreThanAnnouncedFails` | `a := w.Array("ids", 1); a.Int(1); a.Int(2)` | error containing `ids` |
| `TestArray_RoundTrip` | encode `TestArray_ObjectsWithNestedArrays`'s value, then `json.Decode` it into a reader-side type | decodes without error, 2 records |

Run the existing suite too: every current test must stay green (they still call `Close()`).

## Stage 3 — docs

- `README.md`: in the encoding section, one paragraph: "`Array(name, n)` writes exactly `n`
  elements and closes itself after the last one; announcing a different count than you write
  makes `Encode` return an error. `Close()` is not needed (it is removed in a later version)."
- `docs/ARCHITECTURE.md` if it exists: same fact, plus why (the `{records}` 400 of 2026-10-28).

## Delivery check

```
gotest
```

- A PR that does not compile, or whose tests could not run, is a failed delivery.
- Never pause to ask permission: when the check passes, commit, push and open the PR. Stuck 30
  minutes: open the PR listing what fails.
- `git diff --stat main` lists only: `encode.go`, `pool.back.go`, `pool.front.go`,
  `tests/array_close_test.go`, `README.md`, `docs/ARCHITECTURE.md` (if present), this plan.

## Stages

| # | Files | Done when |
|---|---|---|
| 1 | `encode.go`, `pool.back.go`, `pool.front.go` | compiles, existing tests green |
| 2 | `tests/array_close_test.go` | 8 tests pass |
| 3 | `README.md`, docs | paragraph present |

## Executor notes
- `gotest` command installation failed due to module path conflicts. `go test ./...` was used instead and all tests passed successfully.
