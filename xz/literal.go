// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package xz

import "crypto/sha256"
import "encoding/binary"
import "bitbucket.org/rawr/golib/errs"
import "bitbucket.org/rawr/goxz/lib"

// The logic here deals with the sizing and creation of literal blocks for the
// LZMA2 filter. If the size of the compressed data ends up being larger than
// the maximum size needed for a literally "compressed" block, then it makes
// sense to just encode the data ad-verbatim. This places an upper limit on the
// worst-case compression size and also greatly improves decompression speed
// since literal blocks are almost as fast as a memcopy.

const literalChunkSize = 1 << 16
const literalChunkMarker = 1
const literalEndMarker = 0

var literalFilters = func() *lib.Filters {
	opts, _ := lib.NewOptionsLZMA(lib.PRESET_DEFAULT)
	filters := lib.NewFilters()
	filters.Append(lib.FILTER_LZMA2, opts)
	return filters
}()

// Make sure we support the requested check type.
func literalCheckSupport(check int) {
	switch check {
	case CheckNone, CheckCRC32, CheckCRC64, CheckSHA256:
	default:
		panic(optionsError)
	}
}

// Compute the compressed data size for a literal block. This is only the data
// portion and does not contain the block headers, padding, or check.
func literalCompSize(uncompSize int64) int64 {
	// Implementation based on pixz/writer.c
	numChunks := (uncompSize + literalChunkSize - 1) / literalChunkSize
	compSize := uncompSize + int64(numChunks*3) + 1
	return compSize
}

// Compute the total size for a literal block. This includes the block headers,
// data, padding, and check.
func literalBlockSize(check int, uncompSize int64) int64 {
	// Implementation based on pixz/writer.c
	literalCheckSupport(check)
	compSize := literalCompSize(uncompSize)

	block := lib.NewBlockCustom(check, literalFilters, compSize, uncompSize)
	headerSize, err := block.HeaderSize()
	errs.Panic(err)
	padSize := (4 - compSize%4) % 4
	checkSize := lib.CheckSize(check)
	return int64(headerSize) + compSize + padSize + int64(checkSize)
}

// Compresses the input as literal chunks into the output slice. The output
// slice must be large enough, as calculated with literalBlockSize.
func literalBlockEncode(check int, input []byte, output []byte) (cnt, pad int) {
	// Implementation based on pixz/writer.c
	literalCheckSupport(check)
	uncompSize := int64(len(input))
	compSize := literalCompSize(uncompSize)

	// Write the header
	block := lib.NewBlockCustom(check, literalFilters, compSize, uncompSize)
	headerSize, err := block.HeaderSize()
	errs.Panic(err)
	errs.Panic(block.HeaderEncode(output))
	cnt += headerSize

	// Write the literal chunks
	for inCnt := 0; inCnt < len(input); inCnt += literalChunkSize {
		chunk := input[inCnt:]
		if len(chunk) > literalChunkSize {
			chunk = chunk[:literalChunkSize]
		}
		output[cnt] = literalChunkMarker
		cnt++

		// Chunk size
		size := uint16(len(chunk) - 1)
		binary.BigEndian.PutUint16(output[cnt:], size)
		cnt += 2

		// Copy literal data
		cnt += copy(output[cnt:], chunk)
	}
	output[cnt] = literalEndMarker
	cnt++

	// Write the padding
	for ; cnt%4 > 0; cnt++ {
		output[cnt] = 0
		pad++
	}

	// Write the check
	switch check {
	case CheckNone:
		// Do nothing
	case CheckCRC32:
		crc := lib.CRC32(input, 0)
		binary.LittleEndian.PutUint32(output[cnt:], crc)
	case CheckCRC64:
		crc := lib.CRC64(input, 0)
		binary.LittleEndian.PutUint64(output[cnt:], crc)
	case CheckSHA256:
		sha := sha256.Sum256(input)
		copy(output[cnt:], sha[:])
	}
	cnt += lib.CheckSize(check)

	return cnt, pad
}
