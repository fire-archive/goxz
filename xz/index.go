// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package xz

import "io"
import "os"
import "math"

import "bitbucket.org/rawr/goxz/lib"

func decodeIndex(rd io.ReadSeeker) (index *lib.Index, err error) {
	defer errRecover(&err)

	pos, err := rd.Seek(0, os.SEEK_END)
	errPanic(err)

	var preIndex *lib.Index
	for pos > 0 {
		preIndex, pos = prevIndex(rd, pos)

		if index != nil {
			errPanic(preIndex.Cat(index))
		}
		index = preIndex
	}

	return index, nil
}

func prevIndex(rd io.ReadSeeker, pos int64) (index *lib.Index, newPos int64) {
	var err error

	padding := unrollStreamPadding(rd)
	flags := unrollStreamFooter(rd)
	index = unrollStreamIndex(rd, flags.GetBackwardSize())

	// Store the flags and padding
	errPanic(index.StreamFlags(flags))
	errPanic(index.StreamPadding(padding))

	// Seek to the end of the previous stream
	pos -= index.FileSize()
	_, err = rd.Seek(pos, os.SEEK_SET)
	errPanic(err)

	return index, pos
}

func unrollStreamPadding(rd io.ReadSeeker) (padding int64) {
	var err error
	buf := [4]byte{}
	for ok := true; ok; {
		_, err = rd.Seek(-int64(len(buf)), os.SEEK_CUR)
		errPanic(err)
		_, err = io.ReadFull(rd, buf[:])
		errPanic(err)

		if ok = buf[0] == 0 && buf[1] == 0 && buf[2] == 0 && buf[3] == 0; ok {
			_, err = rd.Seek(-4, os.SEEK_CUR)
			errPanic(err)
			padding += 4
		}
	}
	return padding
}

func unrollStreamFooter(rd io.ReadSeeker) (flags *lib.StreamFlags) {
	var err error
	buf := [lib.STREAM_HEADER_SIZE]byte{}
	_, err = rd.Seek(-lib.STREAM_HEADER_SIZE, os.SEEK_CUR)
	errPanic(err)
	_, err = io.ReadFull(rd, buf[:])
	errPanic(err)
	_, err = rd.Seek(-lib.STREAM_HEADER_SIZE, os.SEEK_CUR)
	errPanic(err)

	flags = lib.NewStreamFlags()
	errPanic(flags.FooterDecode(buf[:]))
	return flags
}

func unrollStreamIndex(rd io.ReadSeeker, size int64) (index *lib.Index) {
	var err error
	_, err = rd.Seek(-size, os.SEEK_CUR)
	errPanic(err)

	// Create a stream decoder for the index.
	index = new(lib.Index)
	stream, err := index.NewStreamDecoder(math.MaxUint64)
	errPanic(err)
	rdLimit := io.LimitReader(rd, size)
	rdStream := lib.NewStreamReader(stream, rdLimit, make([]byte, 1<<12), 0)

	// Keep reading the index until EOF.
	for err == nil {
		_, err = rdStream.Read([]byte{0})
		errPanic(errIgnore(err, io.EOF))
	}
	errPanic(rdStream.Close())

	// Check that all input was consumed.
	if stream.GetTotalIn() != size {
		errPanic(lib.NewError(lib.FORMAT_ERROR))
	}

	_, err = rd.Seek(-size, os.SEEK_CUR)
	errPanic(err)
	return index
}
