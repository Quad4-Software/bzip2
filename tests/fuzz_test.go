// SPDX-License-Identifier: 0BSD
// Copyright (c)2026 Quad4.io

package bzip2_test

import (
	"bytes"
	"io"
	"testing"

	"git.quad4.io/Go-Libs/bzip2/pkg/bzip2"
)

func FuzzRoundtrip(f *testing.F) {
	f.Add([]byte(nil))
	f.Add([]byte("a"))
	f.Add(bytes.Repeat([]byte{0x5a}, 1024))
	f.Fuzz(func(t *testing.T, data []byte) {
		var buf bytes.Buffer
		w, err := bzip2.NewWriter(&buf, 9)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(data); err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		got, err := io.ReadAll(bzip2.NewReader(&buf))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, data) {
			t.Fatal("mismatch")
		}
	})
}
