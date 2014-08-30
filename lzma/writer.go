// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package lzma

import "io"
import "bytes"
import "bitbucket.org/rawr/golib/errs"
import "bitbucket.org/rawr/goxz/lib"

type writer struct {
	*lib.Stream
	wr     io.Writer
	closed bool
}

func NewWriter(wr io.Writer) (io.WriteCloser, error) {
	return NewWriterLevel(wr, lib.PRESET_DEFAULT)
}

func NewWriterLevel(wr io.Writer, level int) (io.WriteCloser, error) {
	opts, err := lib.NewOptionsLZMA(level)
	if err != nil {
		return nil, err
	}
	stream, err := lib.NewStreamLZMAAloneEncoder(opts)
	if err != nil {
		return nil, err
	}
	return &writer{stream, wr, false}, nil
}

func (w *writer) Write(data []byte) (cnt int, err error) {
	if w.closed {
		return 0, io.ErrClosedPipe
	}

	rd := bytes.NewReader(data)
	_, rdCnt, err := w.Process(lib.RUN, w.wr, rd)
	return int(rdCnt), errs.Ignore(err, io.EOF)
}

func (w *writer) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true

	defer w.End()
	_, _, err := w.Process(lib.FINISH, w.wr, nil)
	return errs.Ignore(err, streamEnd)
}
