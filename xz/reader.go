// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package xz

import "io"
import "math"
import "sync"
import "bytes"
import "bitbucket.org/rawr/golib/errs"
import "bitbucket.org/rawr/golib/bufpipe"
import "bitbucket.org/rawr/goxz/lib"

const (
	readInit = iota
	readHeader
	readBlock
	readIndex
	readFooter
	readPadding
)

type Reader struct {
	rd     io.Reader
	index  *lib.Index
	closed bool

	maxWorkers int
	maxBuffer  int

	// Internal state for asynchronous operations
	idx  int64
	pipe *pipe

	// Internal state for synchronous operations
	state   int
	flags   *lib.StreamFlags
	block   *lib.Block
	stream  *lib.Stream
	rdBuf  *bufpipe.BufferPipe
	wrBuf  *bufpipe.BufferPipe

	// Communication resources between routines
	inMode, outMode int
	inPool, outPool *pipePool
	workPool        *workerPool
	errChan         chan error
	doneChan        chan bool
	syncChan        chan chan bool
	lastErr         error
}

func NewReader(rd io.Reader) (*Reader, error) {
	return NewReaderCustom(rd, BufferDefault, WorkersDefault)
}

func NewReaderCustom(rd io.Reader, maxBuffer, maxWorkers int) (_ *Reader, err error) {
	defer errs.Recover(&err)
	switch {
	case maxBuffer < BufferMin:
		maxBuffer = BufferMin
	case maxBuffer > BufferMax:
		maxBuffer = BufferMax
	}
	if maxWorkers < 0 {
		maxWorkers = WorkersMax
	}

	// Create the writer object
	r := new(Reader)
	r.rd = rd
	r.maxWorkers, r.maxBuffer = maxWorkers, maxBuffer
	if r.maxWorkers != WorkersSync { // Asynchronous operation
		// ...
		// ...
		// ...
	} else { // Synchronous operation
		r.rdBuf = bufpipe.NewBufferPipe(make([]byte, 1<<16), bufpipe.RingPoll)
		r.wrBuf = bufpipe.NewBufferPipe(make([]byte, 1<<16), bufpipe.RingPoll)
	}
	return r, nil
}

func (r *Reader) Read(data []byte) (cnt int, err error) {
	defer errs.Recover(&err)
	if r.closed {
		return 0, io.ErrClosedPipe
	}

	if r.maxWorkers != WorkersSync { // Asynchronous operation
		// ...
		// ...
		// ...
	} else { // Synchronous operation

		wr := bufpipe.NewBufferPipe(data, bufpipe.LineMono)
		for {
			switch r.state {
			case readInit, readHeader:
				pl("readHeader")
				canEOF := r.state != readInit
				header := r.readFull(lib.STREAM_HEADER_SIZE, true, canEOF)
				r.flags = lib.NewStreamFlags()
				errs.Panic(r.flags.HeaderDecode(header))
				r.state = readBlock
			case readBlock:
				c1, c2 := r.wrBuf.Pointers()
				pl("readBlock", c1, c2)
				if r.stream == nil {
					val := r.readFull(1, false, false)[0]
					if val == 0 {
						r.state = readIndex
					} else {
						r.block = lib.NewBlock(r.flags.GetCheck())
						headerSize, err := r.block.HeaderSizeDecode(val)
						errs.Panic(err)
						header := r.readFull(headerSize, true, false)
						errs.Panic(r.block.HeaderDecode(header))

						r.stream, err = r.block.NewStreamDecoder()
						errs.Panic(err)
					}
				}

				if r.stream != nil {
					r.stream.SetBuffers(r.wrBuf, r.rdBuf)
					rdCnt, _, err := r.stream.Process(lib.RUN, wr, r.rd)
					cnt += int(rdCnt)
					errs.Assert(err != io.ErrShortWrite, nil)
					errs.Panic(errs.Ignore(err, io.ErrShortWrite, streamEnd))
					if err == streamEnd {
						pl(r.block.TotalSize(), r.block.GetUncompressedSize())
						pl(int64(r.block.GetHeaderSize()) + r.stream.GetTotalIn(), r.stream.GetTotalOut())
						r.stream.End()
						r.stream = nil
					}
				}
			case readIndex:
				pl("readIndex")
				index := new(lib.Index)
				// defer index.End()
				// defer r.index.End()
				stream, err := index.NewStreamDecoder(math.MaxUint64)
				errs.Panic(err)
				stream.SetBuffers(r.wrBuf, r.rdBuf)
				_, _, err = stream.Process(lib.FINISH, nil, r.rd)
				errs.Panic(errs.Ignore(err, streamEnd))
				r.flags.SetBackwardSize(index.Size())
				index.End()

				// Compare the newly read index and generated index

				r.state = readFooter
			case readFooter:
				pl("readFooter")
				footer := r.readFull(lib.STREAM_FOOTER_SIZE, true, false)
				flags := lib.NewStreamFlags()
				errs.Panic(flags.FooterDecode(footer))
				errs.Panic(flags.Compare(r.flags))
				r.state = readPadding
			case readPadding:
				pl("readPadding")
				for ok := true; ok; {
					padding := r.readFull(4, false, true)
					if ok = bytes.Equal(padding, padZeros); ok {
						r.rdBuf.ReadMark(4)
					}
				}
				r.state = readHeader
			}
		}
	}
	return 0, nil
}

func (r *Reader) Close() error {
	if r.closed {
		return nil
	}
	r.closed = true

	if r.maxWorkers != WorkersSync { // Asynchronous operation
		// ...
		// ...
		// ...
	} else { // Synchronous operation
	}
	return nil
}

func (r *Reader) checkErr() {
	r.lastErr = <-r.errChan
}

func (r *Reader) terminate() {
	defer errs.NilRecover()
	close(r.doneChan)
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

func (r *Reader) worker(group *sync.WaitGroup, killChan chan bool) {
}

func (r *Reader) reader() {
	for {
		select {
		case retChan := <-r.syncChan:
			retChan <- true // Blocks until sender permits reader to proceed
		case _ = <-r.doneChan:
			return
		default:
		}
	}
}

func (r *Reader) readFull(size int, doMark bool, canEOF bool) []byte {
	if r.rdBuf.Length() < size {
		_, err := r.rdBuf.ReadFrom(r.rd)
		errs.Panic(errs.Ignore(err, io.ErrShortWrite))
	}

	bufLo, bufHi, _ := r.rdBuf.ReadSlices()
	rdyCnt := len(bufLo) + len(bufHi)
	errs.Assert(rdyCnt != 0 || !canEOF, io.EOF)
	errs.Assert(rdyCnt >= size, formatError)
	if doMark {
		r.rdBuf.ReadMark(size)
	}

	if len(bufLo) >= size {
		return bufLo[:size]
	}
	buf := make([]byte, 0, size)
	buf = append(buf, bufLo...)
	buf = append(buf, bufHi...)
	return buf
}

// retChan := make(chan bool)
// syncChan <- retChan // Blocks until reader agrees to sync
// SEEK_SET
// _ = <- retChan // Allow reader to proceed
