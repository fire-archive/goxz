// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// Copyright 2014, Joe Tsai. All rights reserved.
// license that can be found in the LICENSE.md file.

package lib

/*
#cgo LDFLAGS: -llzma
#include "lzma.h"
*/
import "C"

func CheckIsSupported(check int) bool {
	return C.lzma_check_is_supported(C.lzma_check(check)) != 0
}

func CheckSize(check int) int {
	return int(C.lzma_check_size(C.lzma_check(check)))
}

func CRC32(buf []byte, crc uint32) uint32 {
	ptr, size := CSlicePtrLen(buf)
	return uint32(C.lzma_crc32(ptr, size, C.uint32_t(crc)))
}

func CRC64(buf []byte, crc uint64) uint64 {
	ptr, size := CSlicePtrLen(buf)
	return uint64(C.lzma_crc64(ptr, size, C.uint64_t(crc)))
}

func (z *Stream) GetCheck() int {
	return int(C.lzma_get_check(z.C()))
}
