// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package lzma

import "io"
import "math"
import "bitbucket.org/rawr/goxz/lib"

func NewReader(rd io.Reader) (io.ReadCloser, error) {
	stream, err := lib.NewStreamLZMAAloneDecoder(math.MaxUint64)
	if err != nil {
		return nil, err
	}
	return lib.NewStreamReader(stream, rd, make([]byte, _BUFFER_SIZE), 0), nil
}
