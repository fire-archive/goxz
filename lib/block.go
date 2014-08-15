// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package lib

/*
#cgo LDFLAGS: -llzma
#include "lzma.h"

uint32_t lzma_block_header_size_decode_(uint8_t size) {
  return lzma_block_header_size_decode(size);
}
*/
import "C"

type Block struct {
	block   C.lzma_block
	filters *Filters // Reference to prevent Go's GC from freeing the memory
}

func (b *Block) C() *C.lzma_block {
	return (*C.lzma_block)(&b.block)
}

func NewBlock(check int) *Block {
	filters := NewFilters()
	return NewBlockCustom(check, filters, VLI_UNKNOWN, VLI_UNKNOWN)
}

func NewBlockCustom(check int, filters *Filters, compressSize int64, uncompressSize int64) *Block {
	blk := new(Block)
	blk.block.version = 0
	blk.SetCheck(check)
	blk.SetFilters(filters)
	blk.SetCompressedSize(compressSize)
	blk.SetUncompressedSize(uncompressSize)
	return blk
}

func (b *Block) NewStreamEncoder() (*Stream, error) {
	strm := NewStream()
	if err := NewError(C.lzma_block_encoder(strm.C(), b.C())); err != nil {
		return nil, err
	}
	strm.appendRef(b) // Ensure GC doesn't free the block while using it
	return strm, nil
}

func (b *Block) NewStreamDecoder() (*Stream, error) {
	strm := NewStream()
	if err := NewError(C.lzma_block_decoder(strm.C(), b.C())); err != nil {
		return nil, err
	}
	strm.appendRef(b) // Ensure GC doesn't free the block while using it
	return strm, nil
}

func (b *Block) HeaderEncode(data []byte) error {
	if len(data) < int(b.block.header_size) {
		panic("Input header not large enough")
	}
	ptr, _ := CSlicePtrLen(data)
	return NewError(C.lzma_block_header_encode(b.C(), ptr))
}

func (b *Block) HeaderDecode(data []byte) error {
	if len(data) < int(b.block.header_size) {
		panic("Output header not large enough")
	}
	ptr, _ := CSlicePtrLen(data)
	return NewError(C.lzma_block_header_decode(b.C(), nil, ptr))
}

func (b *Block) HeaderSize() (int, error) {
	err := NewError(C.lzma_block_header_size(b.C()))
	return int(b.block.header_size), err
}

func (b *Block) HeaderSizeDecode(val byte) (size int, err error) {
	size = int(C.lzma_block_header_size_decode_(C.uint8_t(val)))
	if size < BLOCK_HEADER_SIZE_MIN || size > BLOCK_HEADER_SIZE_MAX {
		return size, Error(DATA_ERROR)
	}
	b.block.header_size = C.uint32_t(size)
	return
}

func (b *Block) CompressedSize(unpadSize int64) (int64, error) {
	err := NewError(C.lzma_block_compressed_size(b.C(), C.lzma_vli(unpadSize)))
	return int64(b.block.compressed_size), err
}

func (b *Block) UnpaddedSize() int64 {
	return int64(C.lzma_block_unpadded_size(b.C()))
}

func (b *Block) TotalSize() int64 {
	return int64(C.lzma_block_total_size(b.C()))
}

func (b *Block) BufferBound(uncompressSize int64) int64 {
	return int64(C.lzma_block_buffer_bound(C.size_t(uncompressSize)))
}

func (b *Block) GetVersion() int {
	return int(b.block.version)
}

func (b *Block) GetHeaderSize() int {
	return int(b.block.header_size)
}

func (b *Block) GetCheck() int {
	return int(b.block.check)
}

func (b *Block) SetCheck(check int) {
	b.block.check = C.lzma_check(check)
}

func (b *Block) GetCompressedSize() int64 {
	return int64(b.block.compressed_size)
}

func (b *Block) SetCompressedSize(size int64) {
	b.block.compressed_size = C.lzma_vli(size)
}

func (b *Block) GetUncompressedSize() int64 {
	return int64(b.block.uncompressed_size)
}

func (b *Block) SetUncompressedSize(size int64) {
	b.block.uncompressed_size = C.lzma_vli(size)
}

func (b *Block) GetFilters() *Filters {
	return b.filters
}

func (b *Block) SetFilters(filters *Filters) {
	b.block.filters = filters.C()
	b.filters = filters
}

func (b *Block) GetRawCheck() []byte {
	var buf [C.LZMA_CHECK_SIZE_MAX]byte
	size := CheckSize(int(b.block.check))
	if size < 0 || size > C.LZMA_CHECK_SIZE_MAX {
		return buf[:0]
	}
	for idx := 0; idx < size; idx++ {
		buf[idx] = byte(b.block.raw_check[idx])
	}
	return buf[:size]
}
