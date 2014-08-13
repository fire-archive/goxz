// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package lib

/*
#cgo LDFLAGS: -llzma
#include "lzma.h"

void init_stream(lzma_stream* strm) {
  lzma_stream _strm = LZMA_STREAM_INIT;
  (*strm) = _strm;
}
*/
import "C"

import "runtime"

type Stream struct {
	stream C.lzma_stream
	refs   []interface{} // Pointer reference to any struct to help Go's GC
}

func (z *Stream) C() *C.lzma_stream {
	return (*C.lzma_stream)(&z.stream)
}

func NewStream() *Stream {
	strm := new(Stream)
	C.init_stream(strm.C())
	runtime.SetFinalizer(strm, (*Stream).End)
	return strm
}

func (z *Stream) appendRef(ref interface{}) {
	z.refs = append(z.refs, ref)
}

func (z *Stream) Code(action int) error {
	return NewError(C.lzma_code(z.C(), C.lzma_action(action)))
}

// It is not required that End() be called since NewStream() sets a Go finalizer
// on the stream pointer to call lzma_end().
func (z *Stream) End() {
	C.lzma_end(z.C())
}

func (z *Stream) SetInput(buffer []byte) {
	z.stream.next_in, z.stream.avail_in = CSlicePtrLen(buffer)
}

func (z *Stream) SetOutput(buffer []byte) {
	z.stream.next_out, z.stream.avail_out = CSlicePtrLen(buffer)
}

func (z *Stream) GetAvailIn() int64 {
	return int64(z.stream.avail_in)
}

func (z *Stream) GetTotalIn() int64 {
	return int64(z.stream.total_in)
}

func (z *Stream) GetAvailOut() int64 {
	return int64(z.stream.avail_out)
}

func (z *Stream) GetTotalOut() int64 {
	return int64(z.stream.total_out)
}
