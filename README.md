# bzip2

A small and idiomatic **bzip2 compressor and decompressor** API: compress with `NewWriter`, decompress with `NewReader`. Compression is implemented here; `NewReader` uses the standard library reader so the wire format matches `NewWriter`.

The compressor streams output to the underlying `io.Writer` per block and on `Close`; always check errors from `Write` and `Close`. See [API.md](API.md) for lifecycle and release checks.

## Install

```text
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
