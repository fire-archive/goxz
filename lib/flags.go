// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package lib

/*
#cgo LDFLAGS: -llzma
#include "lzma.h"
*/
import "C"

type StreamFlags C.lzma_stream_flags

func (fl *StreamFlags) C() *C.lzma_stream_flags {
	return (*C.lzma_stream_flags)(fl)
}

func NewStreamFlags() *StreamFlags {
	flags := new(StreamFlags)
	flags.version = 0
	flags.check = CHECK_NONE
	flags.backward_size = C.lzma_vli(vliUnknown)
	return flags
}

func (fl *StreamFlags) HeaderEncode(check int) ([]byte, error) {
	fl.check = C.lzma_check(check)
	buf := make([]byte, STREAM_HEADER_SIZE)
	cBufPtr, _ := CSlicePtrLen(buf)
	err := NewError(C.lzma_stream_header_encode(fl.C(), cBufPtr))
	return buf, err
}

func (fl *StreamFlags) FooterEncode(backwardSize int64) ([]byte, error) {
	fl.backward_size = C.lzma_vli(backwardSize)
	buf := make([]byte, STREAM_FOOTER_SIZE)
	cBufPtr, _ := CSlicePtrLen(buf)
	err := NewError(C.lzma_stream_footer_encode(fl.C(), cBufPtr))
	return buf, err
}

func (fl *StreamFlags) HeaderDecode(buf []byte) error {
	if len(buf) < STREAM_HEADER_SIZE {
		return NewError(BUF_ERROR)
	}
	cBufPtr, _ := CSlicePtrLen(buf)
	return NewError(C.lzma_stream_header_decode(fl.C(), cBufPtr))
}

func (fl *StreamFlags) FooterDecode(buf []byte) error {
	if len(buf) < STREAM_FOOTER_SIZE {
		return NewError(BUF_ERROR)
	}
	cBufPtr, _ := CSlicePtrLen(buf)
	return NewError(C.lzma_stream_footer_decode(fl.C(), cBufPtr))
}

func (fl1 *StreamFlags) Compare(fl2 *StreamFlags) error {
	return NewError(C.lzma_stream_flags_compare(fl1.C(), fl2.C()))
}

func (fl *StreamFlags) GetCheck() int {
	return int(fl.check)
}

func (fl *StreamFlags) SetCheck(check int) {
	fl.check = C.lzma_check(check)
}

func (fl *StreamFlags) GetBackwardSize() int64 {
	return int64(fl.backward_size)
}

func (fl *StreamFlags) SetBackwardSize(size int64) {
	fl.backward_size = C.lzma_vli(size)
}
