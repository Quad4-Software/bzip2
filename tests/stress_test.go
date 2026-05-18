// SPDX-License-Identifier: 0BSD
// Copyright (c)2026 Quad4.io

package bzip2_test

import (
	"bytes"
	"io"
	"runtime"
	"testing"

	"github.com/Quad4-Software/bzip2/pkg/bzip2"
)

func TestStressLargeMultiBlock(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	const size = 32 << 20
	pattern := bytes.Repeat([]byte("abcdefghij\n"), size/11+1)
	data := pattern[:size]

	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

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

	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	heapGrowth := int64(after.HeapAlloc) - int64(before.HeapAlloc)
	if heapGrowth > int64(size)*4 {
		t.Fatalf("heap growth %d too large vs input %d", heapGrowth, size)
	}

	got, err := io.ReadAll(bzip2.NewReader(&buf))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("len got=%d want=%d", len(got), len(data))
	}
}
