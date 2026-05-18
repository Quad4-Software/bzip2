// SPDX-License-Identifier: 0BSD
// Copyright (c)2026 Quad4.io

// Command bzip2-example compresses standard input to standard output in bzip2 format.
package main

import (
	"flag"
	"io"
	"log"
	"os"

	"github.com/Quad4-Software/bzip2/pkg/bzip2"
)

func main() {
	level := flag.Int("level", 9, "block size 1-9 (100k to 900k per block)")
	flag.Parse()
	if *level < 1 || *level > 9 {
		log.Fatal("level must be 1-9")
	}
	w, err := bzip2.NewWriter(os.Stdout, *level)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := io.Copy(w, os.Stdin); err != nil {
		log.Fatal(err)
	}
	if err := w.Close(); err != nil {
		log.Fatal(err)
	}
}
