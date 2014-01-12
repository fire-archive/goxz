// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package xz

import "io"

type Writer struct {
}

func NewWriter(w io.Writer) (*Writer, error) {
	return nil, nil
}

func (w *Writer) Write(data []byte) (n int, err error) {
	return 0, nil
}

func (w *Writer) Flush() error {
	return nil
}

func (w *Writer) Close() error {
	return nil
}
