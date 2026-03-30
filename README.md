# bzip2

A small and idiomatic **bzip2 compressor and decompressor** library.

API: compress with `NewWriter`, decompress with `NewReader`. Compression is implemented here; `NewReader` uses the standard library reader so the wire format matches `NewWriter`.

The compressor streams output to the underlying `io.Writer` per block and on `Close`; always check errors from `Write` and `Close`. See [API.md](API.md) for lifecycle and release checks.

**Performance**: The default build uses a pure Go encoder with an O(n log n) block sorter (counting-sort passes) and is suitable for portability. For maximum throughput matching the reference C implementation, build with **`-tags libbzip2`** (requires CGO, libbz2 headers, and `-lbz2`). `NewReader` is unchanged and always uses the standard library.

Build / test with the libbz2-backed compressor:

```text
CGO_ENABLED=1 go build -tags libbzip2 ./...
CGO_ENABLED=1 go test -tags libbzip2 ./...
```

On Linux, install the bzip2 development package (`libbz2-dev`, `bzip2-devel`, or your distro’s equivalent) so `bzlib.h` and `-lbz2` resolve.

## Install

```bash
go get git.quad4.io/Go-Libs/bzip2@latest
```

## Layout

| Path | Purpose |
|------|---------|
| `pkg/bzip2` | Public API (`NewWriter`, `NewReader`, `Writer`) |
| `internal/enc` | Encoder implementation |
| `tests/` | Tests, benchmarks, fuzz, examples (`package bzip2_test`) |
| `cmd/bzip2-example` | Stdin-to-stdout compressor |

## Usage

```go
import (
	"bytes"
	"io"

	"git.quad4.io/Go-Libs/bzip2/pkg/bzip2"
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

See [API.md](API.md) for details.

## Property-based tests

Tests under `tests/` use [pbt](https://git.quad4.io/Go-Libs/pbt) (`git.quad4.io/Go-Libs/pbt/pkg/pbt`).

## License

0BSD. See [LICENSE](LICENSE).
