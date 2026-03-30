// SPDX-License-Identifier: 0BSD
// Copyright (c)2026 Quad4.io

package bzip2_test

import (
	"bytes"
	"testing"

	"git.quad4.io/Go-Libs/bzip2/pkg/bzip2"
)

func TestAllocsPerRunSmall(t *testing.T) {
	n := testing.AllocsPerRun(100, func() {
		var buf bytes.Buffer
		w, err := bzip2.NewWriter(&buf, 9)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte("hello")); err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
	})
	if n > 50 {
		t.Fatalf("allocs per run: %f", n)
	}
}
