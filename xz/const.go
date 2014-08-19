// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

// Package xz implements reading and writing files in the XZ file format.
package xz

import "errors"

import "bitbucket.org/rawr/goxz/lib"

const kiloByte = 1 << 10
const megaByte = 1 << 20
const gigaByte = 1 << 30

// The available compression mode constants.
//
// These constants are copied from the lib package, so that code that imports
// "goxz/xz" does not also have to import "goxz/lib". Extreme flag can be
// ORed with any of the other options to try and improve compression ratio at
// the cost of CPU cycles.
const (
	Extreme            = lib.PRESET_EXTREME
	BestSpeed          = lib.PRESET_LEVEL0
	BestCompression    = lib.PRESET_LEVEL9
	DefaultCompression = lib.PRESET_DEFAULT
)

// The available checksum mode constants.
//
// These constants are copied from the lib package, so that code that imports
// "goxz/xz" does not also have to import "goxz/lib". The CheckNone flag
// disables the placement of checksums; it's usage is heavily discouraged.
const (
	CheckNone    = lib.CHECK_NONE
	CheckCRC32   = lib.CHECK_CRC32
	CheckCRC64   = lib.CHECK_CRC64
	CheckSHA256  = lib.CHECK_SHA256
	CheckDefault = CheckCRC64
)

// Chunksize encoding configuration constants.
//
// This controls the size of the individually compressed chunks. If the
// ChunkStream constant is used, then a single stream with a single block of
// arbitrary size will be outputted. Additionally, using ChunkStream forces the
// encoder to be non-parallelized. On the other hand, any positive chunk size
// allows the encoder to speed-up compression by operating on each block in
// parallel. The ChunkDefault value was chosen as a compromise between fair
// memory usage, high compression ratio, and good random access properties.
const (
	ChunkStream  = -1           // Disables chunking, single chunk per stream
	ChunkDefault = 1 * megaByte // Defaults to 1 MiB chunks
)

// Maximum decoding buffer per routine configuration constants.
//
// This controls the maximum amount of buffer space that may be used by each
// decompression routine. In general, this buffer should be large enough to
// hold both the compressed data and uncompressed data for a chunk. The value
// for MaxBufferDefault was chosen to be able handle larger chunk sizes. If a
// chunk that is too large to handle during decoding, then the decoder will fall
// back to single-routine streaming mode. This loses benefits of parallelism,
// but ensures that progress is made.
//
// Note that this only controls the input/output buffers before using liblzma
// codec. The library itself may use memory orders of magnitude more than the
// allocated buffer size. The latest liblzma implementation gives estimates
// where the codec could use up to 10x more memory than the dictionary size.
// For files generated by this package, the dictionary size is usually the size
// of each chunk.
const (
	MaxBufferLowest  = 4 * kiloByte
	MaxBufferHighest = 1 * gigaByte
	MaxBufferDefault = 64 * ChunkDefault
)

// Maximum number of worker routines constants.
//
// The logic in this library tries to dynamically allocate the right number of
// workers. There is no point in trying to parallelize the work more if the
// calling program or underyling writer does not consume the output fast enough.
const (
	MaxWorkersLowest  = 1  // Use only one routine
	MaxWorkersHighest = -1 // Use as many routines as needed
	MaxWorkersDefault = 8  // Dynamically scales up to this many workers
)

// Errors conditions due to unsupported check types.
//
// These warnings may be returned when first creating the decoding object or
// also during decoding of the stream itself. These errors are not fatal, and
// decompression may continue. However, a user should be aware that the decoder
// provides no guaruntees that the data is valid.
var (
	WarnNoCheck      = errors.New("XZ: No checksum in file")
	WarnUnknownCheck = errors.New("XZ: Unsupported checksum in file")
)

// Padding between streams is always a multiple of four zero bytes.
var padZeros = []byte{0, 0, 0, 0}
