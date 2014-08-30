// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

// Package lzma implements reading and writing files in the LZMA_ALONE format.
// This file format is deprecated. As such, consider using the XZ format.
package lzma

import "math"
import "bitbucket.org/rawr/goxz/lib"

// These constants are copied from the lib package, so that code that imports
// "goxz/lzma" does not also have to import "goxz/lib".
const (
	Extreme            = lib.PRESET_EXTREME
	BestSpeed          = lib.PRESET_LEVEL0
	BestCompression    = lib.PRESET_LEVEL9
	DefaultCompression = lib.PRESET_DEFAULT
)

// Use the maximum amount of memory necessary.
const maxMemory = math.MaxUint64

// Expected error condition when a stream ends.
var streamEnd = lib.Error(lib.STREAM_END)
