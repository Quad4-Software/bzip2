// SPDX-License-Identifier: 0BSD
// Copyright (c)2026 Quad4.io

package bzip2

import (
	"errors"
	"io"

	"git.quad4.io/Go-Libs/bzip2/internal/enc"
)

var (
	// ErrClosed is returned when Write or Close is called after Close has completed successfully.
	ErrClosed = errors.New("bzip2: writer closed")
	// ErrLevelRange is returned when NewWriter is called with a level outside 1-9.
	ErrLevelRange = errors.New("bzip2: level must be between 1 and 9")
	// ErrNilWriter is returned when NewWriter is called with a nil destination writer.
	ErrNilWriter = errors.New("bzip2: nil io.Writer")
)

// Writer compresses input to the bzip2 format and writes it to the destination given to [NewWriter].
// Call [Writer.Close] to flush the stream trailer and combined CRC. The zero value must not be used;
// always construct with NewWriter.
//
// Writer is not safe for concurrent use. After any error from Write or Close, the Writer must not be used further.
type Writer struct {
	dst         io.Writer
	level       int
	bw          enc.BitWriter
	rle         enc.RLEEncoder
	scratch     enc.Scratch
	combinedCRC uint32
	blockNo     int
	closed      bool
	abortErr    error
}

// NewWriter returns a new [Writer] that compresses data to w. level must be between 1 and 9
// (block size is roughly 100_000×level bytes per block). If w is nil, NewWriter returns [ErrNilWriter];
// if level is invalid, it returns [ErrLevelRange].
func NewWriter(w io.Writer, level int) (*Writer, error) {
	if w == nil {
		return nil, ErrNilWriter
	}
	if level < 1 || level > 9 {
		return nil, ErrLevelRange
	}
	nblockMax := 100000*level - 19
	zw := &Writer{
		dst:   w,
		level: level,
	}
	zw.rle.NBlockMax = nblockMax
	zw.rle.Block = make([]byte, 0, nblockMax)
	zw.rle.ResetStream()
	zw.bw.Grow(nblockMax * 2)
	return zw, nil
}

// Write compresses p and appends the compressed form to the underlying writer. It implements [io.Writer].
// After [Writer.Close], Write returns [ErrClosed]. After any other error, Write returns that error.
func (w *Writer) Write(p []byte) (int, error) {
	if w.closed {
		return 0, ErrClosed
	}
	if w.abortErr != nil {
		return 0, w.abortErr
	}
	for i := range p {
		for {
			consumed, full := w.rle.AddByte(p[i])
			if !consumed {
				if err := w.flushBlock(); err != nil {
					w.abortErr = err
					return i, err
				}
				w.rle.StartBlock()
				continue
			}
			if full {
				if err := w.flushBlock(); err != nil {
					w.abortErr = err
					return i + 1, err
				}
				w.rle.StartBlock()
			}
			break
		}
	}
	return len(p), nil
}

func (w *Writer) flushBlock() error {
	block := w.rle.Block
	if len(block) == 0 {
		return nil
	}
	inUse := w.rle.InUse
	crc := w.rle.FinalCRC()
	if w.blockNo == 0 {
		enc.WriteStreamHeader(&w.bw, w.level)
	}
	enc.WriteBlock(&w.bw, block, inUse, crc, &w.scratch)
	w.combinedCRC = enc.CombineCRC(w.combinedCRC, crc)
	w.blockNo++
	return w.flushBitWriter()
}

func (w *Writer) flushBitWriter() error {
	buf := w.bw.Bytes()
	if len(buf) == 0 {
		return nil
	}
	if err := writeFull(w.dst, buf); err != nil {
		return err
	}
	w.bw.ResetOutput()
	return nil
}

func writeFull(dst io.Writer, p []byte) error {
	for len(p) > 0 {
		n, err := dst.Write(p)
		if n < 0 || len(p) < n {
			return errors.New("bzip2: invalid Write result")
		}
		p = p[n:]
		if err != nil {
			return err
		}
		if n == 0 && len(p) > 0 {
			return io.ErrShortWrite
		}
	}
	return nil
}

// Close flushes any pending data, writes the stream trailer and combined CRC, and releases resources.
// Close must be called to produce a valid bzip2 stream. A second Close returns [ErrClosed].
func (w *Writer) Close() error {
	if w.closed {
		return ErrClosed
	}
	if w.abortErr != nil {
		return w.abortErr
	}
	w.rle.FlushRL()
	if len(w.rle.Block) > 0 {
		if err := w.flushBlock(); err != nil {
			w.abortErr = err
			return err
		}
	} else if w.blockNo == 0 {
		enc.WriteStreamHeader(&w.bw, w.level)
	}
	enc.WriteStreamTrailer(&w.bw, w.combinedCRC)
	if err := w.flushBitWriter(); err != nil {
		w.abortErr = err
		return err
	}
	w.closed = true
	return nil
}
