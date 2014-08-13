// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package lib

/*
#cgo LDFLAGS: -llzma
#include "lzma.h"
*/
import "C"

func NewStreamLZMAAloneEncoder(opts *OptionsLZMA) (*Stream, error) {
	strm := NewStream()
	err := NewError(C.lzma_alone_encoder(strm.C(), opts.C()))
	if err != nil {
		return nil, err
	}
	strm.appendRef(opts) // Make sure Go's GC doesn't clear away this struct
	return strm, err
}

func NewStreamLZMAAloneDecoder(memlimit uint64) (*Stream, error) {
	strm := NewStream()
	err := NewError(C.lzma_alone_decoder(strm.C(), C.uint64_t(memlimit)))
	if err != nil {
		return nil, err
	}
	return strm, err
}

func NewStreamEncoder(filters *Filters, check int) (*Stream, error) {
	strm := NewStream()
	err := NewError(C.lzma_stream_encoder(
		strm.C(), filters.C(), C.lzma_check(check),
	))
	if err != nil {
		return nil, err
	}
	strm.appendRef(filters)
	return strm, err
}

func NewStreamDecoder(memlimit uint64, flags uint32) (*Stream, error) {
	strm := NewStream()
	err := NewError(C.lzma_stream_decoder(
		strm.C(), C.uint64_t(memlimit), C.uint32_t(flags),
	))
	if err != nil {
		return nil, err
	}
	return strm, err
}
