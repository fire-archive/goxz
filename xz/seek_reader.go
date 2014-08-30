// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package xz

import "io"
import "bitbucket.org/rawr/goxz/lib"

type SeekReader struct {
	rd     io.ReaderAt
	Index  *lib.Index // The raw index structure. Only call accessor methods.
	closed bool
}

func NewSeekReader(rd io.ReaderAt) (*SeekReader, error) {
	return NewSeekReaderCustom(rd, BufferDefault, WorkersDefault)
}

func NewSeekReaderCustom(rd io.ReaderAt, maxBuffer int, maxWorkers int) (*SeekReader, error) {
	index, err := decodeIndex(rd)
	return &SeekReader{rd, index, false}, err
}

func (r *SeekReader) Read(data []byte) (n int, err error) {
	if r.closed {
		return 0, io.ErrClosedPipe
	}
	return 0, nil
}

func (r *SeekReader) ReadAt(data []byte, off int64) (n int, err error) {
	if r.closed {
		return 0, io.ErrClosedPipe
	}
	return 0, nil
}

func (r *SeekReader) Seek(offset int64, whence int) (int64, error) {
	if r.closed {
		return 0, io.ErrClosedPipe
	}
	return 0, nil
}

func (r *SeekReader) SeekPoints() []int64 {
	if r.closed {
		return nil
	}
	return nil
}

func (r *SeekReader) Close() error {
	if r.closed {
		return nil
	}
	r.closed = true

	defer r.Index.End()
	return nil
}
