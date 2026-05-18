// SPDX-License-Identifier: 0BSD
// Copyright (c)2026 Quad4.io

package bzip2_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/Quad4-Software/bzip2/pkg/bzip2"
)

func TestDecodeTruncatedFails(t *testing.T) {
	var buf bytes.Buffer
	w, err := bzip2.NewWriter(&buf, 9)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(w, "truncated stream test data\n"); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	full := buf.Bytes()
	for cut := 1; cut < len(full) && cut < 40; cut += 7 {
		_, err := io.ReadAll(bzip2.NewReader(bytes.NewReader(full[:cut])))
		if err == nil {
			t.Fatalf("cut=%d: expected error", cut)
		}
	}
}
