// SPDX-License-Identifier: 0BSD
// Copyright (c)2026 Quad4.io

package bzip2_test

import (
	"bytes"
	"io"
	"testing"

	"git.quad4.io/Go-Libs/bzip2/pkg/bzip2"
)

func BenchmarkWriter1MiB(b *testing.B) {
	data := bytes.Repeat([]byte("abcdefghijklmnopqrstuvwxyz\n"), 1<<20/27)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		w, err := bzip2.NewWriter(&buf, 9)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := w.Write(data); err != nil {
			b.Fatal(err)
		}
		if err := w.Close(); err != nil {
			b.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, &buf)
	}
}

func BenchmarkWriterSmall(b *testing.B) {
	const s = "hello world\n"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		w, err := bzip2.NewWriter(&buf, 9)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := io.WriteString(w, s); err != nil {
			b.Fatal(err)
		}
		if err := w.Close(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWriterMultiBlock(b *testing.B) {
	data := bytes.Repeat([]byte("x"), 600000)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		w, err := bzip2.NewWriter(&buf, 9)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := w.Write(data); err != nil {
			b.Fatal(err)
		}
		if err := w.Close(); err != nil {
			b.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, &buf)
	}
}
