// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package xz

import "io"

type Reader struct {
}

func NewReader(rd io.Reader) (*Reader, error) {
	return NewReaderCustom(rd, BufferDefault, WorkersDefault)
}

func NewReaderCustom(rd io.Reader, maxBuffer int, maxWorkers int) (*Reader, error) {
	return nil, nil
}

func (r *Reader) Read(data []byte) (n int, err error) {
	return 0, nil
}

func (r *Reader) Close() error {
	return nil
}
