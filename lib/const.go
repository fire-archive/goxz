// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

// Provide thin wrappers around functions and constants given by the liblzma
// library. The liblzma library by Lasse Collin is used because there are no
// available pure Go implementations that support the newer XZ file format.
//
// This wrapper library intended to be used by the lzma and xz libraries that
// exist in the same parent package. The author of this package does not ensure
// that function names, constants names, and/or function implementation remain
// stable. Thus, use this library at your own peril!
package lib

/*
#cgo LDFLAGS: -llzma
#include "lzma.h"
*/
import "C"

import "math"

// Status code constants:
const (
	OK                = C.LZMA_OK
	STREAM_END        = C.LZMA_STREAM_END
	NO_CHECK          = C.LZMA_NO_CHECK
	UNSUPPORTED_CHECK = C.LZMA_UNSUPPORTED_CHECK
	GET_CHECK         = C.LZMA_GET_CHECK
	MEM_ERROR         = C.LZMA_MEM_ERROR
	MEMLIMIT_ERROR    = C.LZMA_MEMLIMIT_ERROR
	FORMAT_ERROR      = C.LZMA_FORMAT_ERROR
	OPTIONS_ERROR     = C.LZMA_OPTIONS_ERROR
	DATA_ERROR        = C.LZMA_DATA_ERROR
	BUF_ERROR         = C.LZMA_BUF_ERROR
	PROG_ERROR        = C.LZMA_PROG_ERROR
)

// Flushing mode constants:
const (
	RUN        = C.LZMA_RUN
	SYNC_FLUSH = C.LZMA_SYNC_FLUSH
	FULL_FLUSH = C.LZMA_FULL_FLUSH
	FINISH     = C.LZMA_FINISH
)

// Encoder flag constants:
const (
	PRESET_LEVEL0 = iota
	PRESET_LEVEL1
	PRESET_LEVEL2
	PRESET_LEVEL3
	PRESET_LEVEL4
	PRESET_LEVEL5
	PRESET_LEVEL6
	PRESET_LEVEL7
	PRESET_LEVEL8
	PRESET_LEVEL9
	PRESET_DEFAULT    = C.LZMA_PRESET_DEFAULT
	PRESET_LEVEL_MASK = C.LZMA_PRESET_LEVEL_MASK
	PRESET_EXTREME    = C.LZMA_PRESET_EXTREME
)

// Decoder flag constants:
const (
	TELL_NO_CHECK          = C.LZMA_TELL_NO_CHECK
	TELL_UNSUPPORTED_CHECK = C.LZMA_TELL_UNSUPPORTED_CHECK
	TELL_ANY_CHECK         = C.LZMA_TELL_ANY_CHECK
	CONCATENATED           = C.LZMA_CONCATENATED
)

// Integrity check constants:
const (
	CHECK_NONE   = C.LZMA_CHECK_NONE
	CHECK_CRC32  = C.LZMA_CHECK_CRC32
	CHECK_CRC64  = C.LZMA_CHECK_CRC64
	CHECK_SHA256 = C.LZMA_CHECK_SHA256
)

// Index iteration mode constants:
const (
	INDEX_ITER_ANY            = C.LZMA_INDEX_ITER_ANY
	INDEX_ITER_BLOCK          = C.LZMA_INDEX_ITER_BLOCK
	INDEX_ITER_NONEMPTY_BLOCK = C.LZMA_INDEX_ITER_NONEMPTY_BLOCK
	INDEX_ITER_STREAM         = C.LZMA_INDEX_ITER_STREAM
)

// Variable length integer constants:
const (
	VLI_MAX       = C.LZMA_VLI_MAX
	VLI_BYTES_MAX = C.LZMA_VLI_BYTES_MAX
	VLI_UNKNOWN   = math.MaxUint64
)

// HACK(jtsai): LZMA_VLI_UNKNOWN is effectively -1 which overflows when Go tries
//  to cast the constant to the lzma_vli type which is unsigned. Since the
//  lzma_vli type is a uint64_t, then the VLI_UNKNOWN constant essentially gets
//  casted as MaxUint64.

// Filter identification consts:
const (
	FILTER_LZMA1    = C.LZMA_FILTER_LZMA1
	FILTER_LZMA2    = C.LZMA_FILTER_LZMA2
	FILTER_DELTA    = C.LZMA_FILTER_DELTA
	FILTER_X86      = C.LZMA_FILTER_X86
	FILTER_POWERPC  = C.LZMA_FILTER_POWERPC
	FILTER_IA64     = C.LZMA_FILTER_IA64
	FILTER_ARM      = C.LZMA_FILTER_ARM
	FILTER_ARMTHUMB = C.LZMA_FILTER_ARMTHUMB
	FILTER_SPARC    = C.LZMA_FILTER_SPARC
)

// File format constants:
var (
	HEADER_MAGIC = []byte{0xFD, '7', 'z', 'X', 'Z', 0x00}
	FOOTER_MAGIC = []byte{'Y', 'Z'}
)

const (
	STREAM_HEADER_SIZE    = C.LZMA_STREAM_HEADER_SIZE
	STREAM_FOOTER_SIZE    = C.LZMA_STREAM_HEADER_SIZE
	BACKWARD_SIZE_MAX     = C.LZMA_BACKWARD_SIZE_MAX
	BACKWARD_SIZE_MIN     = C.LZMA_BACKWARD_SIZE_MIN
	BLOCK_HEADER_SIZE_MAX = C.LZMA_BLOCK_HEADER_SIZE_MAX
	BLOCK_HEADER_SIZE_MIN = C.LZMA_BLOCK_HEADER_SIZE_MIN
)
