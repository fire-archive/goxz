// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package xz

import "io"

type Reader struct {
}

func NewReader(r io.Reader) (*Reader, error) {
	return nil, nil
}

func (r *Reader) Read(data []byte) (n int, err error) {
	return 0, nil
}

func (r *Reader) Close() error {
	return nil
}
