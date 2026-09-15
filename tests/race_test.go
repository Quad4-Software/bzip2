// SPDX-License-Identifier: 0BSD
// Copyright (c)2026 Quad4.io

package bzip2_test

import (
	"bytes"
	"io"
	"math/rand"
	"sync"
	"testing"

	"github.com/Quad4-Software/bzip2/pkg/bzip2"
)

// TestConcurrentIndependentWriters runs many goroutines each owning a private
// Writer and destination. The Writer type is documented as not safe for
// concurrent use of a single instance; separate instances must be independent.
func TestConcurrentIndependentWriters(t *testing.T) {
	const workers = 8
	const iters = 6
	var wg sync.WaitGroup
	errs := make(chan error, workers*iters)
	for g := 0; g < workers; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(g)))
			for i := 0; i < iters; i++ {
				level := 1 + (g+i)%9
				data := randBytes(rng, 1+rng.Intn(20000))
				var buf bytes.Buffer
				w, err := bzip2.NewWriter(&buf, level)
				if err != nil {
					errs <- err
					return
				}
				if _, err := w.Write(data); err != nil {
					errs <- err
					return
				}
				if err := w.Close(); err != nil {
					errs <- err
					return
				}
				got, err := io.ReadAll(bzip2.NewReader(&buf))
				if err != nil {
					errs <- err
					return
				}
				if !bytes.Equal(got, data) {
					errs <- errMismatch
					return
				}
			}
		}(g)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
}

// TestConcurrentDecoders compresses once, then decodes the same bytes from many
// goroutines. Readers over immutable input must be safe in parallel.
func TestConcurrentDecoders(t *testing.T) {
	want := bytes.Repeat([]byte("shared corpus payload\n"), 20000)
	compressed := compressBZ(t, 6, want)
	var wg sync.WaitGroup
	errs := make(chan error, 16)
	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := io.ReadAll(bzip2.NewReader(bytes.NewReader(compressed)))
			if err != nil {
				errs <- err
				return
			}
			if !bytes.Equal(got, want) {
				errs <- errMismatch
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
}

// TestConcurrentReadWriteOnSeparateBuffers interleaves encoding and decoding on
// distinct buffers to catch accidental shared mutable state in the encoder.
func TestConcurrentReadWriteOnSeparateBuffers(t *testing.T) {
	seedData := bytes.Repeat([]byte("interleave"), 5000)
	compressed := compressBZ(t, 4, seedData)
	var wg sync.WaitGroup
	errs := make(chan error, 32)
	for g := 0; g < 16; g++ {
		wg.Add(2)
		go func(g int) {
			defer wg.Done()
			data := bytes.Repeat([]byte{byte(g)}, 30000)
			if got := stdlibDecode0(data, g); !bytes.Equal(got, data) {
				errs <- errMismatch
			}
		}(g)
		go func() {
			defer wg.Done()
			got, err := io.ReadAll(bzip2.NewReader(bytes.NewReader(compressed)))
			if err != nil {
				errs <- err
				return
			}
			if !bytes.Equal(got, seedData) {
				errs <- errMismatch
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
}

var errMismatch = errMismatchT{}

type errMismatchT struct{}

func (errMismatchT) Error() string { return "roundtrip mismatch" }

// stdlibDecode0 compresses data at a level derived from seed and decodes it.
func stdlibDecode0(data []byte, seed int) []byte {
	var buf bytes.Buffer
	w, err := bzip2.NewWriter(&buf, 1+seed%9)
	if err != nil {
		return nil
	}
	if _, err := w.Write(data); err != nil {
		return nil
	}
	if err := w.Close(); err != nil {
		return nil
	}
	got, err := io.ReadAll(bzip2.NewReader(&buf))
	if err != nil {
		return nil
	}
	return got
}
