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

import "unsafe"

type Options interface {
	Pointer() unsafe.Pointer
}

type Filters struct {
	filters [C.LZMA_FILTERS_MAX + 1]C.lzma_filter
	options []Options // Pointer reference to any struct to help Go's GC
	size    int
}

func (f *Filters) C() *C.lzma_filter {
	return (*C.lzma_filter)(&f.filters[0])
}

func NewFilters() *Filters {
	filters := new(Filters)
	filters.filters[0].id = C.lzma_vli(vliUnknown)
	return filters
}

func (f *Filters) Append(id int64, opts Options) {
	f.filters[f.size+1].id = C.lzma_vli(vliUnknown) // Panic if out of bounds
	f.filters[f.size].id = C.lzma_vli(id)
	f.filters[f.size].options = opts.Pointer()
	f.options = append(f.options, opts) // Memory reference for Go's GC
	f.size++
}
