// SPDX-License-Identifier: 0BSD
// Copyright (c)2026 Quad4.io

package bzip2_test

import (
	"bytes"
	"fmt"
	"io"
	"log"

	"github.com/Quad4-Software/bzip2/pkg/bzip2"
)

func ExampleNewWriter() {
	var buf bytes.Buffer
	w, err := bzip2.NewWriter(&buf, 9)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := io.WriteString(w, "hello"); err != nil {
		log.Fatal(err)
	}
	if err := w.Close(); err != nil {
		log.Fatal(err)
	}
	out, err := io.ReadAll(bzip2.NewReader(&buf))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(string(out))
	// Output: hello
}
