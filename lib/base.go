// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package lib

/*
#cgo LDFLAGS: -llzma
#include "lzma.h"

void lzma_stream_init_(lzma_stream* strm) {
  lzma_stream _strm = LZMA_STREAM_INIT;
  (*strm) = _strm;
}
*/
import "C"

import "io"
import "runtime"
import "bitbucket.org/rawr/golib/bufpipe"

const chunkSize = 1 << 16

type Stream struct {
	stream C.lzma_stream
	refs   []interface{} // Pointer reference to any struct to help Go's GC
	inBuf  *bufpipe.BufferPipe
	outBuf *bufpipe.BufferPipe
}

func (z *Stream) C() *C.lzma_stream {
	return (*C.lzma_stream)(&z.stream)
}

func NewStream() *Stream {
	strm := new(Stream)
	C.lzma_stream_init_(strm.C())
	runtime.SetFinalizer(strm, (*Stream).End)
	return strm
}

func (z *Stream) appendRef(ref interface{}) {
	z.refs = append(z.refs, ref)
}

func (z *Stream) Code(action int) error {
	return NewError(C.lzma_code(z.C(), C.lzma_action(action)))
}

func (z *Stream) CodeSlice(action int, dst []byte, src []byte) (int, int, error) {
	z.SetOutput(dst)
	z.SetInput(src)
	err := z.Code(action)
	wrCnt := len(dst) - int(z.stream.avail_out)
	rdCnt := len(src) - int(z.stream.avail_in)
	return wrCnt, rdCnt, err
}

// It is not required that End() be called since NewStream() sets a Go finalizer
// on the stream pointer to call lzma_end().
func (z *Stream) End() {
	C.lzma_end(z.C())
}

func (z *Stream) SetInput(buffer []byte) {
	z.stream.next_in, z.stream.avail_in = CSlicePtrLen(buffer)
}

func (z *Stream) SetOutput(buffer []byte) {
	z.stream.next_out, z.stream.avail_out = CSlicePtrLen(buffer)
}

func (z *Stream) GetAvailIn() int64 {
	return int64(z.stream.avail_in)
}

func (z *Stream) GetTotalIn() int64 {
	return int64(z.stream.total_in)
}

func (z *Stream) GetAvailOut() int64 {
	return int64(z.stream.avail_out)
}

func (z *Stream) GetTotalOut() int64 {
	return int64(z.stream.total_out)
}

func (z *Stream) GetBuffers() (outBuf, inBuf *bufpipe.BufferPipe) {
	return z.outBuf, z.inBuf
}

func (z *Stream) SetBuffers(outBuf, inBuf *bufpipe.BufferPipe) {
	if outBuf != nil && outBuf.GetMode() != bufpipe.RingPoll {
		panic("internal buffers must be in polling mode")
	}
	if inBuf != nil && inBuf.GetMode() != bufpipe.RingPoll {
		panic("internal buffers must be in polling mode")
	}
	z.outBuf, z.inBuf = outBuf, inBuf
}

func (z *Stream) Process(code int, dst io.Writer, src io.Reader) (outCnt, inCnt int64, err error) {
	if src == nil && dst == nil {
		return
	}

	// Lazy allocate internal ring buffer.
	if src != nil && z.inBuf == nil {
		buf := make([]byte, chunkSize)
		z.inBuf = bufpipe.NewBufferPipe(buf, bufpipe.RingPoll)
	}
	if dst != nil && z.outBuf == nil {
		buf := make([]byte, chunkSize)
		z.outBuf = bufpipe.NewBufferPipe(buf, bufpipe.RingPoll)
	}

	var rdCnt, wrCnt int
	var rdErr, wrErr, xzErr error
	for {
		for rdErr == nil && src != nil && z.inBuf.Length() == 0 {
			buf, _ := z.inBuf.WriteSlice()
			rdCnt, rdErr = src.Read(buf)
			z.inBuf.WriteMark(rdCnt)
			inCnt += int64(rdCnt)
		}

		inBuf, _ := z.inBuf.ReadSlice()
		outBuf, _ := z.outBuf.WriteSlice()
		wrCnt, rdCnt, xzErr = z.CodeSlice(code, outBuf, inBuf)
		z.inBuf.ReadMark(rdCnt)
		z.outBuf.WriteMark(wrCnt)
		if (src == nil || len(inBuf) > 0) && (dst == nil || len(outBuf) > 0) {
			if xzErr == Error(BUF_ERROR) {
				xzErr = nil // Temporary error, just feed more!
			}
		}

		for wrErr == nil && dst != nil && z.outBuf.Length() > 0 {
			buf, _ := z.outBuf.ReadSlice()
			wrCnt, wrErr = dst.Write(buf)
			z.outBuf.ReadMark(wrCnt)
			outCnt += int64(wrCnt)
		}

		switch {
		case wrErr != nil:
			return outCnt, inCnt, wrErr
		case xzErr != nil && z.outBuf.Length() == 0:
			return outCnt, inCnt, xzErr
		case rdErr != nil && z.outBuf.Length() == 0 && z.inBuf.Length() == 0:
			return outCnt, inCnt, rdErr
		}
	}
}

func (z *Stream) ProcessPipe(code int, dst, src *bufpipe.BufferPipe) (outCnt, inCnt int64, err error) {
	if src == nil && dst == nil {
		return
	}

	var inBuf, outBuf []byte
	for {
		if src != nil {
			if inBuf, err = src.ReadSlice(); err != nil {
				return outCnt, inCnt, err
			}
		}

		if dst != nil {
			if outBuf, err = dst.WriteSlice(); err != nil {
				return outCnt, inCnt, err
			}
		}

		wrCnt, rdCnt, err := z.CodeSlice(code, outBuf, inBuf)
		src.ReadMark(rdCnt)
		dst.WriteMark(wrCnt)

		inCnt += int64(rdCnt)
		outCnt += int64(wrCnt)

		if err != nil {
			return outCnt, inCnt, err
		}
	}
}
