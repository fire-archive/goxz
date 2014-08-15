// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package lzma

import "io"

import "bitbucket.org/rawr/goxz/lib"

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
	return lib.NewStreamWriter(stream, wr, lib.NewBuffer(bufSize)), nil
}
