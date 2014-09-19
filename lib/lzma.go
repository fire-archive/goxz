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

type OptionsLZMA C.lzma_options_lzma

func (ops *OptionsLZMA) C() *C.lzma_options_lzma {
	return (*C.lzma_options_lzma)(ops)
}

func NewOptionsLZMA(preset int) (*OptionsLZMA, error) {
	opts := new(OptionsLZMA)
	if C.lzma_lzma_preset(opts.C(), C.uint32_t(preset)) != 0 {
		return nil, NewError(OPTIONS_ERROR)
	}
	return opts, nil
}

func (opts *OptionsLZMA) Pointer() unsafe.Pointer {
	return unsafe.Pointer(opts.C())
}

func (opts *OptionsLZMA) GetDictSize() uint32 {
	return uint32(opts.dict_size)
}

func (opts *OptionsLZMA) SetDictSize(size uint32) {
	opts.dict_size = C.uint32_t(size)
}
