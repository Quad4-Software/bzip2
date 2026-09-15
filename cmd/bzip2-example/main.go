// SPDX-License-Identifier: 0BSD
// Copyright (c)2026 Quad4.io

// Command bzip2-example compresses standard input to standard output in bzip2 format.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/Quad4-Software/bzip2/pkg/bzip2"
)

// compressStream writes the bzip2-compressed form of src to dst at the given
// level. It is the whole pipeline of the command, kept separate from flag and
// exit handling so tests can drive it without spawning a process.
func compressStream(dst io.Writer, src io.Reader, level int) error {
	if level < 1 || level > 9 {
		return fmt.Errorf("level must be 1-9, got %d", level)
	}
	w, err := bzip2.NewWriter(dst, level)
	if err != nil {
		return err
	}
	if _, err := io.Copy(w, src); err != nil {
		return err
	}
	return w.Close()
}

func main() {
	level := flag.Int("level", 9, "block size 1-9 (100k to 900k per block)")
	flag.Parse()
	if err := compressStream(os.Stdout, os.Stdin, *level); err != nil {
		log.Fatal(err)
	}
}
