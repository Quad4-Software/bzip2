// SPDX-License-Identifier: 0BSD
// Copyright (c)2026 Quad4.io

// Package bzip2 provides bzip2 compression and decompression for byte streams.
//
// Use NewWriter to compress and NewReader to decompress. Compression is
// implemented in this module; NewReader delegates to compress/bzip2 for
// decompression, which matches the on-wire format produced by NewWriter.
//
// Writer is not safe for concurrent use by multiple goroutines. After any error
// from Write or Close, the Writer must not be used further; discard it and
// construct a new Writer if needed.
package bzip2
