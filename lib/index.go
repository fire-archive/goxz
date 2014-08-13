// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package lib

/*
#cgo LDFLAGS: -llzma
#include "lzma.h"
*/
import "C"

import "runtime"

type Index struct {
	ptr *C.lzma_index
}

func (ix *Index) C() *C.lzma_index {
	return (*C.lzma_index)(ix.ptr)
}

func NewIndex() (*Index, error) {
	index := &Index{C.lzma_index_init(nil)}
	if index.ptr == nil {
		return nil, NewError(MEM_ERROR)
	}
	runtime.SetFinalizer(index, (*Index).End)
	return index, nil
}

func (ix *Index) NewStreamEncoder() (*Stream, error) {
	strm := NewStream()
	if err := NewError(C.lzma_index_encoder(strm.C(), ix.C())); err != nil {
		return nil, err
	}
	strm.appendRef(ix) // Ensure GC doesn't free index while using it
	return strm, nil
}

func (ix *Index) NewStreamDecoder(memlimit uint64) (*Stream, error) {
	ix.End() // Index decoder will allocate a new index structure
	strm := NewStream()
	err := NewError(C.lzma_index_decoder(strm.C(), &ix.ptr, C.uint64_t(memlimit)))
	if err != nil {
		return nil, err
	}
	strm.appendRef(ix) // Ensure GC doesn't free index while using it
	return strm, err
}

func (ix *Index) Append(unpadSize, uncompSize int64) error {
	err := NewError(C.lzma_index_append(
		ix.C(), nil, C.lzma_vli(unpadSize), C.lzma_vli(uncompSize),
	))
	return err
}

func (ix *Index) StreamCount() int64 {
	return int64(C.lzma_index_stream_count(ix.C()))
}

func (ix *Index) BlockCount() int64 {
	return int64(C.lzma_index_block_count(ix.C()))
}

func (ix *Index) Size() int64 {
	return int64(C.lzma_index_size(ix.C()))
}

func (ix *Index) StreamSize() int64 {
	return int64(C.lzma_index_stream_size(ix.C()))
}

func (ix *Index) TotalSize() int64 {
	return int64(C.lzma_index_total_size(ix.C()))
}

func (ix *Index) FileSize() int64 {
	return int64(C.lzma_index_file_size(ix.C()))
}

func (ix *Index) UncompressedSize() int64 {
	return int64(C.lzma_index_uncompressed_size(ix.C()))
}

func (ix *Index) Checks() uint32 {
	return uint32(C.lzma_index_checks(ix.C()))
}

// The input flags is copied into the index data structure.
func (ix *Index) StreamFlags(flags *StreamFlags) error {
	return NewError(C.lzma_index_stream_flags(ix.C(), flags.C()))
}

func (ix *Index) StreamPadding(padding int64) error {
	return NewError(C.lzma_index_stream_padding(ix.C(), C.lzma_vli(padding)))
}

// The source index is automatically freed if Cat() is successful.
func (dest *Index) Cat(src *Index) error {
	if err := NewError(C.lzma_index_cat(dest.C(), src.C(), nil)); err != nil {
		return err
	}
	src.ptr = nil // Prevent double free
	return nil
}

// It is not required that End() be called since NewIndex() sets a Go finalizer
// on the index pointer to call lzma_index_end().
func (ix *Index) End() {
	if ix.ptr != nil {
		C.lzma_index_end(ix.C(), nil)
	}
	ix.ptr = nil
}

type IndexIter struct {
	iter   C.lzma_index_iter
	index  *Index // Pointer to index to keep reference to help GC
	Stream IndexIterStream
	Block  IndexIterBlock
}

type IndexIterStream struct {
	it *IndexIter
}

type IndexIterBlock struct {
	it *IndexIter
}

func (it *IndexIter) C() *C.lzma_index_iter {
	return (*C.lzma_index_iter)(&it.iter)
}

func (ix *Index) NewIter() *IndexIter {
	iter := new(IndexIter)
	C.lzma_index_iter_init(iter.C(), ix.C())
	iter.index = ix
	iter.Stream = IndexIterStream{iter}
	iter.Block = IndexIterBlock{iter}
	return iter
}

func (it *IndexIter) Next(mode int) bool {
	return C.lzma_index_iter_next(it.C(), C.lzma_index_iter_mode(mode)) != 0
}

func (it *IndexIter) Locate(offset int64) bool {
	return C.lzma_index_iter_locate(it.C(), C.lzma_vli(offset)) != 0
}

func (it *IndexIter) Rewind() {
	C.lzma_index_iter_rewind(it.C())
}

func (x IndexIterStream) GetFlags() *StreamFlags {
	return (*StreamFlags)(x.it.iter.stream.flags)
}

func (x IndexIterStream) GetNumber() int64 {
	return int64(x.it.iter.stream.number)
}

func (x IndexIterStream) GetBlockCount() int64 {
	return int64(x.it.iter.stream.block_count)
}

func (x IndexIterStream) GetCompressedSize() int64 {
	return int64(x.it.iter.stream.compressed_size)
}

func (x IndexIterStream) GetUncompressedSize() int64 {
	return int64(x.it.iter.stream.uncompressed_size)
}

func (x IndexIterStream) GetPadding() int64 {
	return int64(x.it.iter.stream.padding)
}

func (x IndexIterBlock) GetNumberInFile() int64 {
	return int64(x.it.iter.block.number_in_file)
}

func (x IndexIterBlock) GetNumberInStream() int64 {
	return int64(x.it.iter.block.number_in_stream)
}

func (x IndexIterBlock) GetCompressedFileOffset() int64 {
	return int64(x.it.iter.block.compressed_file_offset)
}

func (x IndexIterBlock) GetUncompressedFileOffset() int64 {
	return int64(x.it.iter.block.uncompressed_file_offset)
}

func (x IndexIterBlock) GetCompressedStreamOffset() int64 {
	return int64(x.it.iter.block.compressed_stream_offset)
}

func (x IndexIterBlock) GetUncompressedStreamOffset() int64 {
	return int64(x.it.iter.block.uncompressed_stream_offset)
}

func (x IndexIterBlock) GetUncompressedSize() int64 {
	return int64(x.it.iter.block.uncompressed_size)
}

func (x IndexIterBlock) GetUnpaddedSize() int64 {
	return int64(x.it.iter.block.unpadded_size)
}

func (x IndexIterBlock) GetTotalSize() int64 {
	return int64(x.it.iter.block.total_size)
}
