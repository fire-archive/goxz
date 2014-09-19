// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package lib

/*
#cgo LDFLAGS: -llzma
#include "lzma.h"
*/
import "C"

import "unsafe"
import "reflect"

// Split a Go slice into a pointer and length.
func SlicePtrLen(data []byte) (uintptr, int) {
	dataSlice := (*reflect.SliceHeader)(unsafe.Pointer(&data))
	return dataSlice.Data, dataSlice.Len
}

// Split a Go slice into a uint8_t pointer and size_t length.
func CSlicePtrLen(data []byte) (*C.uint8_t, C.size_t) {
	ptr, len := SlicePtrLen(data)
	return (*C.uint8_t)(unsafe.Pointer(ptr)), C.size_t(len)
}
