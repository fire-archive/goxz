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

// HACK(jtsai): LZMA_VLI_UNKNOWN is effectively -1 which overflows when Go tries
//  to cast the constant to the lzma_vli type which is unsigned. Do the casting
//  in C and then use the value in Go.
const lzma_vli _LZMA_VLI_UNKNOWN = LZMA_VLI_UNKNOWN;
*/
import "C"

// Status code constants.
const OK = C.LZMA_OK
const STREAM_END = C.LZMA_STREAM_END
const NO_CHECK = C.LZMA_NO_CHECK
const UNSUPPORTED_CHECK = C.LZMA_UNSUPPORTED_CHECK
const GET_CHECK = C.LZMA_GET_CHECK
const MEM_ERROR = C.LZMA_MEM_ERROR
const MEMLIMIT_ERROR = C.LZMA_MEMLIMIT_ERROR
const FORMAT_ERROR = C.LZMA_FORMAT_ERROR
const OPTIONS_ERROR = C.LZMA_OPTIONS_ERROR
const DATA_ERROR = C.LZMA_DATA_ERROR
const BUF_ERROR = C.LZMA_BUF_ERROR
const PROG_ERROR = C.LZMA_PROG_ERROR

// Flushing mode constants.
const RUN = C.LZMA_RUN
const SYNC_FLUSH = C.LZMA_SYNC_FLUSH
const FULL_FLUSH = C.LZMA_FULL_FLUSH
const FINISH = C.LZMA_FINISH

// Encoder flag constants.
const PRESET_DEFAULT = C.LZMA_PRESET_DEFAULT
const PRESET_LEVEL_MASK = C.LZMA_PRESET_LEVEL_MASK
const PRESET_EXTREME = C.LZMA_PRESET_EXTREME

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
)

// Decoder flag constants.
const TELL_NO_CHECK = C.LZMA_TELL_NO_CHECK
const TELL_UNSUPPORTED_CHECK = C.LZMA_TELL_UNSUPPORTED_CHECK
const TELL_ANY_CHECK = C.LZMA_TELL_ANY_CHECK
const CONCATENATED = C.LZMA_CONCATENATED

// Integrity check constants.
const CHECK_NONE = C.LZMA_CHECK_NONE
const CHECK_CRC32 = C.LZMA_CHECK_CRC32
const CHECK_CRC64 = C.LZMA_CHECK_CRC64
const CHECK_SHA256 = C.LZMA_CHECK_SHA256

// Index iteration mode constants.
const INDEX_ITER_ANY = C.LZMA_INDEX_ITER_ANY
const INDEX_ITER_BLOCK = C.LZMA_INDEX_ITER_BLOCK
const INDEX_ITER_NONEMPTY_BLOCK = C.LZMA_INDEX_ITER_NONEMPTY_BLOCK
const INDEX_ITER_STREAM = C.LZMA_INDEX_ITER_STREAM

// Variable length integer constants.
var VLI_UNKNOWN = C._LZMA_VLI_UNKNOWN

const VLI_MAX = C.LZMA_VLI_MAX
const VLI_BYTES_MAX = C.LZMA_VLI_BYTES_MAX

// File format constants.
var HEADER_MAGIC = []byte{0xFD, '7', 'z', 'X', 'Z', 0x00}
var FOOTER_MAGIC = []byte{'Y', 'Z'}
const STREAM_HEADER_SIZE = C.LZMA_STREAM_HEADER_SIZE
const STREAM_FOOTER_SIZE = C.LZMA_STREAM_HEADER_SIZE
const BACKWARD_SIZE_MAX = C.LZMA_BACKWARD_SIZE_MAX
const BACKWARD_SIZE_MIN = C.LZMA_BACKWARD_SIZE_MIN
const BLOCK_HEADER_SIZE_MAX = C.LZMA_BLOCK_HEADER_SIZE_MAX
const BLOCK_HEADER_SIZE_MIN = C.LZMA_BLOCK_HEADER_SIZE_MIN
