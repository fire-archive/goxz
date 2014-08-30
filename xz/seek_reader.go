// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package xz

import "io"

type SeekReader struct {
}

func NewSeekReader(rd io.ReaderAt) (*SeekReader, error) {
	return NewSeekReaderCustom(rd, BufferDefault, WorkersDefault)
}

func NewSeekReaderCustom(rd io.ReaderAt, maxBuffer int, maxWorkers int) (*SeekReader, error) {
	return nil, nil
}

func (r *SeekReader) Read(data []byte) (n int, err error) {
	return 0, nil
}

func (r *SeekReader) ReadAt(data []byte, off int64) (n int, err error) {
	return 0, nil
}

func (r *SeekReader) Seek(offset int64, whence int) (int64, error) {
	return 0, nil
}

func (r *SeekReader) SeekPoints() []int64 {
	return nil
}

func (r *SeekReader) Close() error {
	return nil
}
