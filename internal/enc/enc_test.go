// SPDX-License-Identifier: 0BSD
// Copyright (c)2026 Quad4.io

package enc

import (
	"bytes"
	stdbz2 "compress/bzip2"
	"io"
	"math/rand"
	"testing"
)

// naiveRLE is a deliberately simple reference implementation of the bzip2
// run-length stage used to check RLEEncoder output byte for byte.
func naiveRLE(raw []byte) (block []byte, inUse [256]bool, crc uint32) {
	c := uint32(0xffffffff)
	i := 0
	for i < len(raw) {
		ch := raw[i]
		run := 1
		for i+run < len(raw) && raw[i+run] == ch && run < 255 {
			run++
		}
		for k := 0; k < run; k++ {
			c = bzUpdateCRC(c, ch)
		}
		inUse[ch] = true
		switch run {
		case 1:
			block = append(block, ch)
		case 2:
			block = append(block, ch, ch)
		case 3:
			block = append(block, ch, ch, ch)
		default:
			block = append(block, ch, ch, ch, ch, byte(run-4))
			inUse[run-4] = true
		}
		i += run
	}
	return block, inUse, ^c
}

func TestEncodeRLEBlockVectors(t *testing.T) {
	cases := []struct {
		name  string
		raw   []byte
		block []byte
	}{
		{"empty", nil, nil},
		{"single", []byte{'a'}, []byte{'a'}},
		{"two_same", []byte{'a', 'a'}, []byte{'a', 'a'}},
		{"three_same", []byte{'a', 'a', 'a'}, []byte{'a', 'a', 'a'}},
		{"run_of_4", []byte("aaaa"), []byte{'a', 'a', 'a', 'a', 0}},
		{"run_of_5", []byte("aaaaa"), []byte{'a', 'a', 'a', 'a', 1}},
		{"run_of_255", bytes.Repeat([]byte{'a'}, 255), []byte{'a', 'a', 'a', 'a', 251}},
		// 300 = run of 255 then run of 45: two encoded pairs.
		{"run_of_300", bytes.Repeat([]byte{'a'}, 300),
			[]byte{'a', 'a', 'a', 'a', 251, 'a', 'a', 'a', 'a', 41}},
		{"mixed", []byte("abbbbc"), []byte{'a', 'b', 'b', 'b', 'b', 0, 'c'}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			block, inUse, crc := EncodeRLEBlock(tc.raw, len(tc.raw)+16)
			if !bytes.Equal(block, tc.block) {
				t.Fatalf("block got %v want %v", block, tc.block)
			}
			wantBlock, wantUse, wantCRC := naiveRLE(tc.raw)
			if !bytes.Equal(block, wantBlock) || inUse != wantUse || crc != wantCRC {
				t.Fatal("naive reference mismatch")
			}
			_ = crc
		})
	}
}

func TestEncodeRLEBlockMatchesNaive(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	for trial := 0; trial < 300; trial++ {
		n := rng.Intn(4000)
		raw := make([]byte, n)
		switch trial % 3 {
		case 0:
			for i := range raw {
				raw[i] = byte(rng.Intn(256))
			}
		case 1:
			// Runs of random lengths force the pair-encoding paths.
			for i := 0; i < n; {
				ch := byte(rng.Intn(8))
				run := 1 + rng.Intn(300)
				for j := 0; j < run && i < n; j++ {
					raw[i] = ch
					i++
				}
			}
		case 2:
			for i := range raw {
				raw[i] = byte(i)
			}
		}
		block, inUse, crc := EncodeRLEBlock(raw, n+16)
		wantBlock, wantUse, wantCRC := naiveRLE(raw)
		if !bytes.Equal(block, wantBlock) {
			t.Fatalf("trial %d n=%d: block mismatch", trial, n)
		}
		if inUse != wantUse {
			t.Fatalf("trial %d n=%d: inUse mismatch", trial, n)
		}
		if crc != wantCRC {
			t.Fatalf("trial %d n=%d: crc %#x want %#x", trial, n, crc, wantCRC)
		}
	}
}

// TestRLEBlockFullSemantics pins down AddByte behavior at capacity: with input
// where every byte differs from its predecessor the block fills one byte per
// consumed input byte plus the one pending run byte.
func TestRLEBlockFullSemantics(t *testing.T) {
	const max = 64
	var e RLEEncoder
	e.NBlockMax = max
	e.ResetStream()
	consumed := 0
	for {
		c, full := e.AddByte(byte(consumed))
		if !c {
			if !full {
				t.Fatal("not consumed but not full")
			}
			break
		}
		consumed++
		if consumed > 2*max+8 {
			t.Fatal("never reported full")
		}
	}
	if consumed != max+1 {
		t.Fatalf("consumed %d want %d", consumed, max+1)
	}
	if len(e.Block) != max {
		t.Fatalf("block len %d want %d", len(e.Block), max)
	}
	// After starting a new block the encoder accepts input again.
	e.StartBlock()
	if c, _ := e.AddByte(0x55); !c {
		t.Fatal("StartBlock did not clear the block")
	}
}

// TestRLEAddBytesConsumed checks AddBytes reports consumed counts and fullness
// across a block boundary.
func TestRLEAddBytesConsumed(t *testing.T) {
	const max = 32
	var e RLEEncoder
	e.NBlockMax = max
	e.ResetStream()
	// Cycling bytes: each consumes one input byte and emits one block byte.
	in := cycleBytes(3 * max)
	total := 0
	for total < len(in) {
		n, full := e.AddBytes(in[total:])
		total += n
		if n == 0 && !full {
			t.Fatal("no progress and not full")
		}
		if full {
			e.StartBlock()
		}
		if n == 0 {
			continue
		}
	}
	if total != len(in) {
		t.Fatalf("consumed %d want %d", total, len(in))
	}
	// A call on an empty slice consumes nothing and reports not-full.
	if n, full := e.AddBytes(nil); n != 0 || full {
		t.Fatalf("AddBytes(nil) = %d, %v", n, full)
	}
}

func TestBitWriterGrowAndResetOutput(t *testing.T) {
	var w BitWriter
	w.Grow(64)
	w.PutUChar(0x11)
	w.PutUChar(0x22)
	w.Finish()
	if len(w.Bytes()) != 2 {
		t.Fatalf("got %d bytes", len(w.Bytes()))
	}
	// ResetOutput keeps pending bits but clears drained bytes; with no pending
	// bits after Finish the next byte starts a fresh stream.
	w.ResetOutput()
	if len(w.Bytes()) != 0 {
		t.Fatal("ResetOutput left bytes")
	}
	w.Grow(1) // must not shrink or lose state
	w.PutUChar(0x33)
	w.Finish()
	if got := w.Bytes(); !bytes.Equal(got, []byte{0x33}) {
		t.Fatalf("got %x", got)
	}
}

func TestRLEResetStreamClearsState(t *testing.T) {
	var e RLEEncoder
	e.NBlockMax = 1024
	e.ResetStream()
	for i := 0; i < 10; i++ {
		e.AddByte('q')
	}
	e.ResetStream()
	if len(e.Block) != 0 || e.StateInLen != 0 || e.StateInCh != 256 {
		t.Fatal("ResetStream left state behind")
	}
	e.AddByte('z')
	e.FlushRL()
	if !bytes.Equal(e.Block, []byte{'z'}) {
		t.Fatalf("block %v", e.Block)
	}
}

func TestBitWriterMSBFirstOrder(t *testing.T) {
	var w BitWriter
	w.WriteBits(1, 1)
	w.WriteBits(1, 0)
	w.WriteBits(1, 1)
	w.WriteBits(5, 0x02) // 00010
	w.Finish()
	// Bits in order: 1 0 1 00010 -> byte 10100010 = 0xa2.
	if got := w.Bytes(); !bytes.Equal(got, []byte{0xa2}) {
		t.Fatalf("got %x want a2", got)
	}
}

func TestBitWriterByteValues(t *testing.T) {
	var w BitWriter
	w.PutUChar(0x42)
	w.PutUChar(0x5a)
	w.PutUInt32(0x01020304)
	w.Finish()
	want := []byte{0x42, 0x5a, 0x01, 0x02, 0x03, 0x04}
	if got := w.Bytes(); !bytes.Equal(got, want) {
		t.Fatalf("got %x want %x", got, want)
	}
}

// TestBitWriterUnalignedAcrossWrites: a partial byte must carry across writes
// and Finish must zero-pad the final partial byte.
func TestBitWriterUnalignedAcrossWrites(t *testing.T) {
	var w BitWriter
	w.WriteBits(3, 0x05) // 101
	w.PutUChar(0xff)
	w.WriteBits(4, 0x09) // 1001
	w.Finish()
	// bit stream: 101 11111111 1001 = 10111111 1111001 (15 bits)
	// byte0: 10111111 = 0xbf; remaining 7 bits 1111001 padded to 11110010 = 0xf2.
	want := []byte{0xbf, 0xf2}
	if got := w.Bytes(); !bytes.Equal(got, want) {
		t.Fatalf("got %x want %x", got, want)
	}
}

func TestBitWriterResetForNewStream(t *testing.T) {
	var w BitWriter
	w.WriteBits(3, 0x05)
	w.Finish()
	w.ResetForNewStream()
	if len(w.Bytes()) != 0 {
		t.Fatal("reset left output")
	}
	w.WriteBits(4, 0x0a)
	w.Finish()
	if got := w.Bytes(); !bytes.Equal(got, []byte{0xa0}) {
		t.Fatalf("got %x want a0", got)
	}
}

// rotLess compares two cyclic rotations of block lexicographically.
func rotLess(block []byte, a, b int) bool {
	n := len(block)
	for k := 0; k < n; k++ {
		ca, cb := block[(a+k)%n], block[(b+k)%n]
		if ca != cb {
			return ca < cb
		}
	}
	return false
}

// inverseBWT reconstructs the original block from the last column and the index
// of the original rotation in the sorted matrix.
func inverseBWT(last []byte, origPtr int) []byte {
	n := len(last)
	var base [256]int
	for _, c := range last {
		base[c]++
	}
	sum := 0
	for c := 0; c < 256; c++ {
		base[c], sum = sum, sum+base[c]
	}
	lf := make([]int, n)
	var occ [256]int
	for i, c := range last {
		lf[i] = base[c] + occ[c]
		occ[c]++
	}
	out := make([]byte, n)
	j := origPtr
	for i := n - 1; i >= 0; i-- {
		out[i] = last[j]
		j = lf[j]
	}
	return out
}

// TestBuildCyclicSuffixArray verifies the suffix array is a sorted permutation
// of cyclic rotations and that (last column, origPtr) inverts to the input.
func TestBuildCyclicSuffixArray(t *testing.T) {
	rng := rand.New(rand.NewSource(5))
	inputs := [][]byte{
		[]byte("banana"),
		[]byte("x"),
		[]byte("ababababab"),
		[]byte("mississippi"),
		bytes.Repeat([]byte{'q'}, 64),
		cycleBytes(256),
	}
	for i := 0; i < 60; i++ {
		raw := make([]byte, 1+rng.Intn(200))
		for j := range raw {
			raw[j] = byte(rng.Intn(256))
		}
		inputs = append(inputs, raw)
	}
	for _, block := range inputs {
		var sc Scratch
		sc.PrepareEncoderAux()
		sa, origPtr := buildCyclicSuffixArray(block, &sc)
		n := len(block)
		if len(sa) != n {
			t.Fatalf("n=%d: sa len %d", n, len(sa))
		}
		seen := make([]bool, n)
		for _, v := range sa {
			if v < 0 || int(v) >= n || seen[v] {
				t.Fatalf("n=%d: sa not a permutation", n)
			}
			seen[v] = true
		}
		for i := 1; i < n; i++ {
			if rotLess(block, int(sa[i]), int(sa[i-1])) {
				t.Fatalf("n=%d: rotations out of order at %d", n, i)
			}
		}
		if int(sa[origPtr]) != 0 {
			t.Fatalf("n=%d: origPtr=%d but sa[origPtr]=%d", n, origPtr, sa[origPtr])
		}
		last := make([]byte, n)
		for i, v := range sa {
			last[i] = block[(int(v)+n-1)%n]
		}
		if got := inverseBWT(last, origPtr); !bytes.Equal(got, block) {
			t.Fatalf("n=%d: inverse BWT mismatch", n)
		}
	}
}

func cycleBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i)
	}
	return b
}

func TestMakeUnseqToSeq(t *testing.T) {
	var inUse [256]bool
	inUse[0x10] = true
	inUse[0x00] = true
	inUse[0xff] = true
	m, n := makeUnseqToSeq(inUse)
	if n != 3 {
		t.Fatalf("nInUse %d want 3", n)
	}
	if m[0x00] != 0 || m[0x10] != 1 || m[0xff] != 2 {
		t.Fatalf("mapping wrong: %v %v %v", m[0x00], m[0x10], m[0xff])
	}
}

func TestMTFIndexAndMove(t *testing.T) {
	yy := []byte{10, 20, 30, 40}
	if j := mtfIndexAndMove(yy, 30); j != 2 {
		t.Fatalf("index %d want 2", j)
	}
	if !bytes.Equal(yy, []byte{30, 10, 20, 40}) {
		t.Fatalf("yy %v", yy)
	}
	if j := mtfIndexAndMove(yy, 30); j != 0 {
		t.Fatalf("front index %d want 0", j)
	}
	if j := mtfIndexAndMove(yy, 40); j != 3 {
		t.Fatalf("index %d want 3", j)
	}
	if !bytes.Equal(yy, []byte{40, 30, 10, 20}) {
		t.Fatalf("yy %v", yy)
	}
}

func TestHbAssignCodesCanonical(t *testing.T) {
	length := []uint8{1, 2, 3, 3}
	code := make([]int32, 4)
	hbAssignCodes(code, length, 1, 3, 4)
	want := []int32{0b0, 0b10, 0b110, 0b111}
	for i := range want {
		if code[i] != want[i] {
			t.Fatalf("code[%d]=%b want %b", i, code[i], want[i])
		}
	}

	// Uniform lengths produce sequential codes.
	length = []uint8{2, 2, 2, 2}
	hbAssignCodes(code, length, 2, 2, 4)
	for i := range code {
		if code[i] != int32(i) {
			t.Fatalf("uniform code[%d]=%d want %d", i, code[i], i)
		}
	}
}

// TestHbAssignCodesPrefixFree: codes produced for a valid length set must be
// prefix-free (no code is a prefix of a longer code).
func TestHbAssignCodesPrefixFree(t *testing.T) {
	length := []uint8{2, 2, 3, 3, 3, 3, 4, 4, 4, 4}
	code := make([]int32, len(length))
	hbAssignCodes(code, length, 2, 4, len(length))
	for i := range code {
		for j := range code {
			if i == j || length[j] >= length[i] {
				continue
			}
			shift := uint(length[i] - length[j])
			if code[i]>>shift == code[j] {
				t.Fatalf("code %b(len %d) has prefix %b(len %d)", code[i], length[i], code[j], length[j])
			}
		}
	}
}

func TestCombineCRC(t *testing.T) {
	if got := CombineCRC(0, 0xdeadbeef); got != 0xdeadbeef {
		t.Fatalf("identity: %#x", got)
	}
	prev := uint32(0x80000001)
	want := uint32(0x00000003) ^ 0x1234
	if got := CombineCRC(prev, 0x1234); got != want {
		t.Fatalf("rotate-xor: got %#x want %#x", got, want)
	}
}

// TestWriteBlockStreamDecodes assembles whole bzip2 streams by hand through the
// internal block writer and checks them with the standard library decoder,
// covering header, block, and trailer emission end to end inside enc.
func TestWriteBlockStreamDecodes(t *testing.T) {
	stdDecode := func(stream []byte) []byte {
		out, err := io.ReadAll(stdbz2.NewReader(bytes.NewReader(stream)))
		if err != nil {
			t.Fatalf("stdlib decode: %v", err)
		}
		return out
	}
	nblockMax := 100000*9 - 19

	// Single block.
	raw := []byte("internal stream assembly check")
	var bw BitWriter
	var sc Scratch
	sc.PrepareEncoderAux()
	block, inUse, crc := EncodeRLEBlock(raw, nblockMax)
	WriteStreamHeader(&bw, 9)
	WriteBlock(&bw, block, inUse, crc, &sc)
	WriteStreamTrailer(&bw, CombineCRC(0, crc))
	if got := stdDecode(bw.Bytes()); !bytes.Equal(got, raw) {
		t.Fatalf("single block: got %q", got)
	}

	// Two blocks with a combined CRC.
	raw1 := bytes.Repeat([]byte("first block "), 3000)
	raw2 := bytes.Repeat([]byte("second block "), 2000)
	bw.ResetForNewStream()
	b1, u1, c1 := EncodeRLEBlock(raw1, nblockMax)
	b2, u2, c2 := EncodeRLEBlock(raw2, nblockMax)
	WriteStreamHeader(&bw, 9)
	WriteBlock(&bw, b1, u1, c1, &sc)
	WriteBlock(&bw, b2, u2, c2, &sc)
	WriteStreamTrailer(&bw, CombineCRC(CombineCRC(0, c1), c2))
	want := append(append([]byte(nil), raw1...), raw2...)
	if got := stdDecode(bw.Bytes()); !bytes.Equal(got, want) {
		t.Fatal("two-block stream mismatch")
	}

	// Empty stream: header then trailer with combined CRC zero.
	bw.ResetForNewStream()
	WriteStreamHeader(&bw, 9)
	WriteStreamTrailer(&bw, 0)
	if got := stdDecode(bw.Bytes()); len(got) != 0 {
		t.Fatalf("empty stream decoded to %d bytes", len(got))
	}
}

func TestEncodedBitBufferCap(t *testing.T) {
	if EncodedBitBufferCap(0) != 0 || EncodedBitBufferCap(-5) != 0 {
		t.Fatal("nonpositive input must give 0")
	}
	if EncodedBitBufferCap(1000) <= 1000 {
		t.Fatal("capacity should exceed input size")
	}
}
