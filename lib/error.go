// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package lib

/*
#cgo LDFLAGS: -llzma
#include "lzma.h"
*/
import "C"

type Error int

func NewError(code C.lzma_ret) error {
	if code == OK {
		return nil
	} else {
		return Error(code)
	}
}

func (ze Error) Error() string {
	name := "[UNKNOWN] Unknown error"
	switch ze {
	case STREAM_END:
		name = "[STREAM_END] End of stream was reached"
	case NO_CHECK:
		name = "[NO_CHECK] Input stream has no integrity check"
	case UNSUPPORTED_CHECK:
		name = "[UNSUPPORTED_CHECK] Cannot calculate the integrity check"
	case GET_CHECK:
		name = "[GET_CHECK] Integrity check type is now available"
	case MEM_ERROR:
		name = "[MEM_ERROR] Cannot allocate memory"
	case MEMLIMIT_ERROR:
		name = "[MEMLIMIT_ERROR] Memory usage limit was reached"
	case FORMAT_ERROR:
		name = "[FORMAT_ERROR] File format not recognized"
	case OPTIONS_ERROR:
		name = "[OPTIONS_ERROR] Invalid or unsupported options"
	case DATA_ERROR:
		name = "[DATA_ERROR] Data is corrupt"
	case BUF_ERROR:
		name = "[BUF_ERROR] No progress is possible"
	case PROG_ERROR:
		name = "[PROG_ERROR] Programming error"
	}
	return name
}
