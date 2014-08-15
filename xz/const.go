// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

// Package xz implements reading and writing files in the XZ file format.
package xz

import "bitbucket.org/rawr/goxz/lib"

// These constants are copied from the lib package, so that code that imports
// "goxz/xz" does not also have to import "goxz/lib".
const (
	Extreme            = lib.PRESET_EXTREME
	BestSpeed          = lib.PRESET_LEVEL0
	BestCompression    = lib.PRESET_LEVEL9
	DefaultCompression = lib.PRESET_DEFAULT
)

// Chunksize configuration constants.
const (
	ChunkStream  = -1      // Disables chunking, single chunk per stream
	ChunkDefault = 1 << 20 // Defaults to 1 MiB chunks
)
