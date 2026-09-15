# bzip2

A bzip2 compressor and decompressor library for Go.

Compress with `NewWriter`, decompress with `NewReader`. The compressor is implemented in this repository. `NewReader` wraps the standard library reader, so the wire format matches what `NewWriter` produces.

The writer streams compressed blocks to the underlying `io.Writer` as each block fills and again on `Close`. Check the errors from `Write` and `Close`. [API.md](API.md) covers lifecycle details and release checks.

## Performance

The default build is a pure Go encoder with an O(n log n) block sorter built on counting-sort passes. For throughput closer to the reference C implementation, build with the `libbzip2` tag. That build needs CGO, the libbz2 headers, and `-lbz2` at link time. `NewReader` is unchanged in both builds and always uses the standard library decoder.

Benchmark snapshots from Go 1.26.2, `tests/bench_test.go`:

- Pure Go, `BenchmarkWriterMultiBlock`: 116 to 119 MB/s, 4.47 MB/op, 15 allocs/op, new `Writer` each iteration.
- Pure Go, `BenchmarkWriterMultiBlockReuseDiscard`: 148 to 153 MB/s, 0 B/op, 0 allocs/op, one `Writer` reused via `Reset` after warmup.
- Pure Go, `BenchmarkWriteBlockPrepared` in `internal/enc`: 469 to 471 MB/s, 0 B/op, 0 allocs/op, prepared block encoder hot path.
- `libbzip2` build, `BenchmarkWriterMultiBlock`: 149 to 166 MB/s, 192 B/op, 3 allocs/op.
- `libbzip2` build, `BenchmarkWriterMultiBlockReuseDiscard`: 147 to 155 MB/s, 0 B/op, 0 allocs/op.

Zero allocations per op is the expected steady state when a `Writer` is reused through `Reset` with warm buffers. A new `Writer` per stream still allocates.

Build and test against libbz2:

```bash
CGO_ENABLED=1 go build -tags libbzip2 ./...
CGO_ENABLED=1 go test -tags libbzip2 ./...
```

On Linux, install the bzip2 development package (`libbz2-dev`, `bzip2-devel`, or the distro equivalent) so `bzlib.h` and `-lbz2` resolve.

## Install

```bash
go get github.com/Quad4-Software/bzip2@latest
```

For local development against a checkout, point a replace directive at it:

```go
replace github.com/Quad4-Software/bzip2 => ../bzip2
```

## Layout

| Path | Contents |
|------|----------|
| `pkg/bzip2` | Public API: `NewWriter`, `NewReader`, `Writer` |
| `internal/enc` | Encoder implementation |
| `tests` | Tests, benchmarks, fuzz targets, examples |
| `cmd/bzip2-example` | Stdin to stdout compressor |

## Usage

```go
import (
	"bytes"
	"io"

	"github.com/Quad4-Software/bzip2/pkg/bzip2"
)

func Example() {
	var buf bytes.Buffer
	w, err := bzip2.NewWriter(&buf, 9)
	if err != nil {
		panic(err)
	}
	if _, err := io.Copy(w, bytes.NewReader([]byte("hello"))); err != nil {
		panic(err)
	}
	if err := w.Close(); err != nil {
		panic(err)
	}
	out, err := io.ReadAll(bzip2.NewReader(&buf))
	_ = out
}
```

See [API.md](API.md) for the full surface.

## License

0BSD. See [LICENSE](LICENSE).
