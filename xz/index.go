// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package xz

import "io"
import "bytes"
import "bitbucket.org/rawr/golib/errs"
import "bitbucket.org/rawr/golib/ioutil"
import "bitbucket.org/rawr/goxz/lib"

func decodeIndex(rd io.ReaderAt) (index *lib.Index, err error) {
	// Implementation based on pixz/reader.c
	defer errs.Recover(&err)

	for pos := getSize(rd); pos > 0; {
		var preIndex *lib.Index
		preIndex, pos = prevStreamIndex(rd, pos)
		if index != nil {
			errs.Panic(preIndex.Cat(index))
		}
		index = preIndex
	}
	return index, nil
}

func getSize(rd io.ReaderAt) int64 {
	// Get size from seek if available
	if rds, ok := rd.(io.Seeker); ok {
		pos, err := ioutil.SeekerSize(rds)
		errs.Panic(err)
		return pos
	}

	pos, err := ioutil.ReaderAtSize(rd)
	errs.Panic(err)
	return pos
}

func prevStreamIndex(rd io.ReaderAt, pos int64) (*lib.Index, int64) {
	// Implementation based on pixz/reader.c
	padding, pos := unrollPadding(rd, pos)
	ftrFlags, pos := unrollFooter(rd, pos)
	index, pos := unrollIndex(rd, pos, ftrFlags.GetBackwardSize())
	pos -= index.TotalSize() // Skip the blocks
	hdrFlags, pos := unrollHeader(rd, pos)

	// Ensure header and footer flags match
	errs.Panic(hdrFlags.Compare(ftrFlags))

	// Store the flags and padding
	errs.Panic(index.StreamFlags(ftrFlags))
	errs.Panic(index.StreamPadding(padding))
	return index, pos
}

func unrollPadding(rd io.ReaderAt, pos int64) (padding int64, _ int64) {
	padAlias := int64(len(padZeros))
	buf := make([]byte, padAlias)
	for ok := true; ok; {
		_, err := rd.ReadAt(buf, pos-padding-padAlias)
		errs.Panic(errs.Convert(err, io.ErrUnexpectedEOF, io.EOF))

		if ok = bytes.Equal(buf, padZeros); ok {
			padding += padAlias
		}
	}
	return padding, (pos - padding)
}

func unrollFooter(rd io.ReaderAt, pos int64) (*lib.StreamFlags, int64) {
	buf := make([]byte, lib.STREAM_FOOTER_SIZE)
	_, err := rd.ReadAt(buf, pos-lib.STREAM_FOOTER_SIZE)
	errs.Panic(errs.Convert(err, io.ErrUnexpectedEOF, io.EOF))

	flags := lib.NewStreamFlags()
	errs.Panic(flags.FooterDecode(buf))
	return flags, (pos - lib.STREAM_FOOTER_SIZE)
}

func unrollIndex(rd io.ReaderAt, pos int64, size int64) (*lib.Index, int64) {
	// Create a stream decoder for the index.
	index := new(lib.Index)
	stream, err := index.NewStreamDecoder(maxMemory)
	errs.Panic(err)
	rds := io.NewSectionReader(rd, pos-size, size)

	// Keep reading the index until EOF.
	_, _, err = stream.Process(lib.FINISH, nil, rds)
	errs.Panic(err)

	// Check that all input was consumed.
	errs.Assert(stream.GetTotalIn() == size, formatError)
	return index, (pos - size)
}

func unrollHeader(rd io.ReaderAt, pos int64) (*lib.StreamFlags, int64) {
	buf := make([]byte, lib.STREAM_HEADER_SIZE)
	_, err := rd.ReadAt(buf, pos-lib.STREAM_HEADER_SIZE)
	errs.Panic(errs.Convert(err, io.ErrUnexpectedEOF, io.EOF))

	flags := lib.NewStreamFlags()
	errs.Panic(flags.HeaderDecode(buf))
	return flags, (pos - lib.STREAM_HEADER_SIZE)
}
