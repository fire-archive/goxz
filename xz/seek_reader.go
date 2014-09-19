// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package xz

import "io"
import "bitbucket.org/rawr/golib/errs"
import "bitbucket.org/rawr/golib/ioutil"
import "bitbucket.org/rawr/goxz/lib"

type SeekReader struct {
	rda    io.ReaderAt
	rds    io.ReadSeeker
	Index  *lib.Index // The raw index structure. Only call accessor methods.
	closed bool
}

// If the ReaderAt provided also satisfies the the ReadSeeker interface, then
// the native Read and Seek methods will be used.
func NewSeekReader(rd io.ReaderAt) (*SeekReader, error) {
	return NewSeekReaderCustom(rd, BufferDefault, WorkersDefault)
}

// If the ReaderAt provided also satisfies the the ReadSeeker interface, then
// the native Read and Seek methods will be used.
func NewSeekReaderCustom(rd io.ReaderAt, maxBuffer int, maxWorkers int) (_ *SeekReader, err error) {
	defer errs.Recover(&err)

	var size int64
	var rds io.ReadSeeker
	if _, ok := rd.(io.ReadSeeker); ok {
		rds = rd.(io.ReadSeeker)
		size, err = ioutil.SeekerSize(rds)
	} else {
		size, err = ioutil.ReaderAtSize(rd)
		rds = ioutil.NewSectionReader(rd, 0, size)
	}
	errs.Panic(err)

	index, err := decodeIndex(rd, size)
	errs.Panic(err)

	return &SeekReader{rd, rds, index, false}, nil
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
