// SPDX-License-Identifier: 0BSD
// Copyright (c)2026 Quad4.io

package bzip2_test

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/Quad4-Software/bzip2/pkg/bzip2"
)

func roundtrip(t *testing.T, level int, data []byte) {
	t.Helper()
	var buf bytes.Buffer
	w, err := bzip2.NewWriter(&buf, level)
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
		t.Fatalf("len in=%d out=%d", len(data), len(got))
	}
}

func TestRealWorldHTTPLogLines(t *testing.T) {
	var b strings.Builder
	for i := range 8000 {
		b.WriteString(`127.0.0.1 - - [30/Mar/2026:12:00:00 +0000] "GET /index `)
		b.WriteString(strings.Repeat("x", i%8))
		b.WriteString(` HTTP/1.1" 200 `)
		b.WriteString(strings.Repeat("0", i%5))
		b.WriteByte('\n')
	}
	roundtrip(t, 6, []byte(b.String()))
}

func TestRealWorldJSONLines(t *testing.T) {
	type row struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		OK   bool   `json:"ok"`
	}
	var lines [][]byte
	for i := range 3000 {
		line, err := json.Marshal(row{ID: i, Name: strings.Repeat("item", 1+i%3), OK: i%2 == 0})
		if err != nil {
			t.Fatal(err)
		}
		lines = append(lines, line)
	}
	data := bytes.Join(lines, []byte{'\n'})
	roundtrip(t, 8, data)
}

func TestRealWorldSourceLikeText(t *testing.T) {
	const chunk = `package main

import "fmt"

func main() {
	fmt.Println("hello")
}
`
	data := bytes.Repeat([]byte(chunk), 400)
	roundtrip(t, 4, data)
}

func TestRealWorldSparseBinary(t *testing.T) {
	data := make([]byte, 128*1024)
	for i := range data {
		if i%4096 == 0 {
			data[i] = byte(i)
		}
	}
	roundtrip(t, 9, data)
}
