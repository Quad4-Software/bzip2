# API reference

Module: `git.quad4.io/Go-Libs/bzip2`

Import path:

```go
import "git.quad4.io/Go-Libs/bzip2/pkg/bzip2"
```

Single package for **compression** and **decompression**: use `NewWriter` to compress and `NewReader` to decompress. Compression is implemented in this module; `NewReader` wraps the standard library [`compress/bzip2`](https://pkg.go.dev/compress/bzip2) reader so the format matches what `NewWriter` emits.

## Production behavior

- **Streaming output**: `Writer` writes compressed bytes to the destination as blocks are completed and again when `Close` writes the stream trailer. Memory use scales with block size and encoder buffers, not unbounded input length.
- **Destination errors**: If `Write` to the destination returns an error, or returns a short write with no error (`n < len(p)` and `err == nil`), the `Writer` records that error and all later `Write` and `Close` calls return it until the value is discarded. A successful `Close` is required for a valid `.bz2` stream.
- **Concurrency**: `Writer` is not safe for use from multiple goroutines at once. `NewReader` follows the same rules as [`compress/bzip2`](https://pkg.go.dev/compress/bzip2).

## Functions

### `func NewWriter(w io.Writer, level int) (*Writer, error)`

Creates a compressor writing bzip2 data to `w`. `level` is **1–9** (block size roughly `100_000 * level` bytes per block). Returns `ErrNilWriter` if `w` is nil. Returns `ErrLevelRange` if `level` is invalid.

### `func NewReader(r io.Reader) io.Reader`

Returns a decompressor for bzip2 data read from `r`. Equivalent to [`compress/bzip2.NewReader`](https://pkg.go.dev/compress/bzip2#NewReader).

### `type Writer`

Implements `io.Writer`. You **must** call `Close` to finish the stream (trailer and CRC).

### `func (w *Writer) Write(p []byte) (n int, err error)`

Writes input; compressed output may be written to the destination before `Close` when a block is filled.

### `func (w *Writer) Close() error`

Flushes pending data and writes the stream end marker and combined CRC. After a successful `Close`, further `Write`/`Close` return `ErrClosed`. After a failed `Write` or `Close`, the same error is returned until the `Writer` is discarded.

## Variables (errors)

| Name | Meaning |
|------|---------|
| `ErrClosed` | `Write` or `Close` after successful `Close` |
| `ErrNilWriter` | `w` is nil in `NewWriter` |
| `ErrLevelRange` | invalid `level` in `NewWriter` |

## Tests and examples

Tests, benchmarks, fuzz targets, and examples live under [`tests/`](tests/). Run:

```text
go test ./tests/...
go test ./... 
```

(`go test ./...` includes `tests` and `internal/enc`.)

## Command-line example

`cmd/bzip2-example` compresses stdin to stdout:

```text
go run ./cmd/bzip2-example/ -level=6 < input.bin > out.bz2
```

Decompress with this library:

```go
f, _ := os.Open("out.bz2")
defer f.Close()
plain, _ := io.ReadAll(bzip2.NewReader(f))
```

## Release gate

Before tagging a release:

1. `go test ./...`
2. `go test ./tests/...` without `-short` (includes large stress test)
3. `go test -fuzz=FuzzRoundtrip -fuzztime=30s ./tests` (or longer in CI)
4. `go test -bench=. -benchmem ./tests` and compare allocs to the previous tag

Document any intentional wire-format or API change in release notes.
