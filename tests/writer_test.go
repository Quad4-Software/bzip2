// SPDX-License-Identifier: 0BSD
// Copyright (c)2026 Quad4.io

package bzip2_test

import (
	"bytes"
	"io"
	"testing"

	"git.quad4.io/Go-Libs/bzip2/pkg/bzip2"
)

func TestRoundtripHello(t *testing.T) {
	const want = "hello world\n"
	var buf bytes.Buffer
	w, err := bzip2.NewWriter(&buf, 9)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(w, want); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(bzip2.NewReader(&buf))
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != want {
		t.Fatalf("got %q want %q", out, want)
	}
}
