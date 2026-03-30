// SPDX-License-Identifier: 0BSD
// Copyright (c)2026 Quad4.io

package bzip2_test

import (
	"bytes"
	"encoding/hex"
	"io"
	"testing"

	"git.quad4.io/Go-Libs/bzip2/pkg/bzip2"
)

func TestCorpusDecodeVectors(t *testing.T) {
	cases := []struct {
		name string
		hex  string
		want []byte
	}{
		{
			name: "hello_world_level9",
			hex:  "425a68393141592653594eece83600000251800010400006449080200031064c4101a7a9a580bb9431f8bb9229c28482776741b0",
			want: []byte("hello world\n"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := hex.DecodeString(tc.hex)
			if err != nil {
				t.Fatal(err)
			}
			got, err := io.ReadAll(bzip2.NewReader(bytes.NewReader(raw)))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, tc.want) {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestCorpusEncodeMatchesGolden(t *testing.T) {
	const wantHex = "425a68393141592653594eece83600000251800010400006449080200031064c4101a7a9a580bb9431f8bb9229c28482776741b0"
	want, err := hex.DecodeString(wantHex)
	if err != nil {
		t.Fatal(err)
	}
	plain := []byte("hello world\n")
	var buf bytes.Buffer
	w, err := bzip2.NewWriter(&buf, 9)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(plain); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Fatalf("compressed output mismatch stdlib golden")
	}
}
