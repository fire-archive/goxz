// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package xz

import "io"

type Reader struct {
	rd     io.Reader
	closed bool

	maxWorkers int
	maxBuffer  int

	// Communication resources between routines
	inSize, outSize int
	inMode, outMode int
	inPool, outPool *pipePool
	workPool        *workerPool
	errChan         chan error
	doneChan        chan bool
	lastErr         error
}

func NewReader(rd io.Reader) (*Reader, error) {
	return NewReaderCustom(rd, BufferDefault, WorkersDefault)
}

func NewReaderCustom(rd io.Reader, maxBuffer, maxWorkers int) (*Reader, error) {
	return nil, nil
}

func (r *Reader) Read(data []byte) (n int, err error) {
	return 0, nil
}

func (r *Reader) Close() error {
	return nil
}

func (r *Reader) monitor() {
	defer r.inPool.terminate()
	defer r.outPool.terminate()
	defer r.workPool.terminate()
	setCapacity := func(resCnt int) {
		r.inPool.setCapacity(resCnt)
		r.outPool.setCapacity(resCnt)
		r.workPool.setCapacity(resCnt)
	}
	setCapacity(1) // Must have at least 1 worker to proceed

	_ = <-r.doneChan
}

func (r *Reader) reader() {
}
