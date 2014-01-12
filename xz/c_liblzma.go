// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package xz

/*
#cgo LDFLAGS: -llzma
#include "lzma.h"

void initStream(lzma_stream* strm) {
	lzma_stream _strm = LZMA_STREAM_INIT;
	(*strm) = _strm;
}
*/
import "C"

import "unsafe"
import "reflect"
import "runtime"

// Provide thin wrappers around functions and constants provided by the liblzma
// library. The liblzma library by Lasse Collin is used because there are no
// available pure Go implementations that support the newer xz file format.

// Status code constants.
const lzma_OK = C.LZMA_OK
const lzma_STREAM_END = C.LZMA_STREAM_END
const lzma_NO_CHECK = C.LZMA_NO_CHECK
const lzma_UNSUPPORTED_CHECK = C.LZMA_UNSUPPORTED_CHECK
const lzma_GET_CHECK = C.LZMA_GET_CHECK
const lzma_MEM_ERROR = C.LZMA_MEM_ERROR
const lzma_MEMLIMIT_ERROR = C.LZMA_MEMLIMIT_ERROR
const lzma_FORMAT_ERROR = C.LZMA_FORMAT_ERROR
const lzma_OPTIONS_ERROR = C.LZMA_OPTIONS_ERROR
const lzma_DATA_ERROR = C.LZMA_DATA_ERROR
const lzma_BUF_ERROR = C.LZMA_BUF_ERROR
const lzma_PROG_ERROR = C.LZMA_PROG_ERROR

// Flushing mode constants.
const lzma_RUN = C.LZMA_RUN
const lzma_SYNC_FLUSH = C.LZMA_SYNC_FLUSH
const lzma_FULL_FLUSH = C.LZMA_FULL_FLUSH
const lzma_FINISH = C.LZMA_FINISH

// Encoder flag constants.
const lzma_PRESET_DEFAULT = C.LZMA_PRESET_DEFAULT
const lzma_PRESET_LEVEL_MASK = C.LZMA_PRESET_LEVEL_MASK
const lzma_PRESET_EXTREME = C.LZMA_PRESET_EXTREME

// Decoder flag constants.
const lzma_TELL_NO_CHECK = C.LZMA_TELL_NO_CHECK
const lzma_TELL_UNSUPPORTED_CHECK = C.LZMA_TELL_UNSUPPORTED_CHECK
const lzma_TELL_ANY_CHECK = C.LZMA_TELL_ANY_CHECK
const lzma_CONCATENATED = C.LZMA_CONCATENATED

// Integrity check constants.
const lzma_CHECK_NONE = C.LZMA_CHECK_NONE
const lzma_CHECK_CRC32 = C.LZMA_CHECK_CRC32
const lzma_CHECK_CRC64 = C.LZMA_CHECK_CRC64
const lzma_CHECK_SHA256 = C.LZMA_CHECK_SHA256

// Go error wrapper for underlying liblzma errors.
type lzmaErr struct {
	code int
}

func newLzmaErr(code C.lzma_ret) error {
	if code == lzma_OK {
		return nil
	} else {
		return &lzmaErr{int(code)}
	}
}

func (ze *lzmaErr) Error() string {
	name := "[UNKNOWN] Unknown error"
	switch ze.code {
	case lzma_STREAM_END:
		name = "[STREAM_END] End of stream was reached"
	case lzma_NO_CHECK:
		name = "[NO_CHECK] Input stream has no integrity check"
	case lzma_UNSUPPORTED_CHECK:
		name = "[UNSUPPORTED_CHECK] Cannot calculate the integrity check"
	case lzma_GET_CHECK:
		name = "[GET_CHECK] Integrity check type is now available"
	case lzma_MEM_ERROR:
		name = "[MEM_ERROR] Cannot allocate memory"
	case lzma_MEMLIMIT_ERROR:
		name = "[MEMLIMIT_ERROR] Memory usage limit was reached"
	case lzma_FORMAT_ERROR:
		name = "[FORMAT_ERROR] File format not recognized"
	case lzma_OPTIONS_ERROR:
		name = "[OPTIONS_ERROR] Invalid or unsupported options"
	case lzma_DATA_ERROR:
		name = "[DATA_ERROR] Data is corrupt"
	case lzma_BUF_ERROR:
		name = "[BUF_ERROR] No progress is possible"
	case lzma_PROG_ERROR:
		name = "[PROG_ERROR] Programming error"
	}
	return name
}

func lzmaErrMatch(err error, errnoMatches ...int) bool {
	if ze, ok := err.(*lzmaErr); ok {
		for _, errMatch := range errnoMatches {
			if ze.code == errMatch {
				return true
			}
		}
	}
	return false
}

func lzmaErrIgnore(err error, errnoIgns ...int) error {
	if lzmaErrMatch(err, errnoIgns...) {
		return nil
	}
	return err
}

// Go wrapper around the liblzma stream struct.
type lzmaStream C.lzma_stream

// Return a pointer to a blank version of a lzma_stream.
func newStream() *lzmaStream {
	strm := new(lzmaStream)
	C.initStream((*C.lzma_stream)(strm))
	runtime.SetFinalizer(strm, (*lzmaStream).end)
	return strm
}

// Initialize a compressor stream.
func encodeInit(preset int, check int) (*lzmaStream, error) {
	strm := newStream()
	err := newLzmaErr(C.lzma_easy_encoder(
		(*C.lzma_stream)(strm),
		C.uint32_t(preset),
		C.lzma_check(check),
	))
	if err != nil {
		strm = nil
	}
	return strm, err
}

// Initialize a decompressor stream.
func decodeInit(memLimit int64, flags int) (*lzmaStream, error) {
	strm := newStream()
	err := newLzmaErr(C.lzma_auto_decoder(
		(*C.lzma_stream)(strm),
		(C.uint64_t)(memLimit),
		(C.uint32_t)(flags),
	))
	if err != nil {
		strm = nil
	}
	return strm, err
}

// Generic lzma function to either encode or decode.
func (z *lzmaStream) code(action int) error {
	return newLzmaErr(C.lzma_code(
		(*C.lzma_stream)(z),
		(C.lzma_action(action)),
	))
}

// Clean up the lzma stream.
func (z *lzmaStream) end() {
	if z.internal != nil {
		C.lzma_end((*C.lzma_stream)(z))
	}
}

// Set a Go slice as the input buffer.
func (z *lzmaStream) setInput(buffer []byte) {
	bufPtr, bufLen := slicePtrLen(buffer)
	z.next_in = (*C.uint8_t)(unsafe.Pointer(bufPtr))
	z.avail_in = C.size_t(bufLen)
}

// Set a Go slice as the output buffer.
func (z *lzmaStream) setOutput(buffer []byte) {
	bufPtr, bufLen := slicePtrLen(buffer)
	z.next_out = (*C.uint8_t)(unsafe.Pointer(bufPtr))
	z.avail_out = C.size_t(bufLen)
}

// Split a Go slice into a C pointer and length.
func slicePtrLen(data []byte) (uintptr, int) {
	dataSlice := (*reflect.SliceHeader)(unsafe.Pointer(&data))
	return dataSlice.Data, dataSlice.Len
}
