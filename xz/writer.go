// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package xz

import "io"
import "bitbucket.org/rawr/golib/errs"
import "bitbucket.org/rawr/goxz/lib"

type Writer struct {
	wr     io.Writer
	flags  *lib.StreamFlags
	index  *lib.Index
	blkWr  *blockWriter
	closed bool
}

func NewWriter(wr io.Writer) (*Writer, error) {
	return NewWriterLevel(wr, DefaultCompression)
}

func NewWriterLevel(wr io.Writer, level int) (*Writer, error) {
	return NewWriterCustom(wr, level, CheckDefault, ChunkDefault, WorkersDefault)
}

func NewWriterCustom(wr io.Writer, level int, checkType int, chunkSize int64, maxWorkers int) (_ *Writer, err error) {
	defer errs.Recover(&err)

	// Compose the stream header flags
	flags := lib.NewStreamFlags()
	buf, err := flags.HeaderEncode(checkType)
	errs.Panic(err)
	index, err := lib.NewIndex()
	errs.Panic(err)

	// Define the XZ filters
	opts, err := lib.NewOptionsLZMA(lib.PRESET_DEFAULT)
	errs.Panic(err)
	if chunkSize > 0 && int64(opts.GetDictSize()) > chunkSize {
		opts.SetDictSize(uint32(chunkSize))
	}
	filters := lib.NewFilters()
	filters.Append(lib.FILTER_LZMA2, opts)

	// Write the header
	_, err = wr.Write(buf)
	errs.Panic(err)

	writer := &Writer{wr, flags, index, nil, false}
	writer.blkWr = newBlockWriter(wr, flags, index, filters, chunkSize, maxWorkers)
	return writer, nil
}

func (w *Writer) Write(data []byte) (int, error) {
	if w.closed {
		return 0, io.ErrClosedPipe
	}
	return w.blkWr.Write(data)
}

func (w *Writer) Close() (err error) {
	defer errs.Recover(&err)
	if w.closed {
		return nil
	}
	w.closed = true

	// Ensure all worker routines have ended
	errs.Panic(w.blkWr.Close())

	// Write the index
	stream, err := w.index.NewStreamEncoder()
	errs.Panic(err)
	_, _, err = stream.Process(lib.FINISH, w.wr, nil)
	errs.Panic(errs.Ignore(err, streamEnd))

	// Write the footer
	buf, err := w.flags.FooterEncode(w.index.Size())
	errs.Panic(err)
	_, err = w.wr.Write(buf)
	errs.Panic(err)

	return nil
}
