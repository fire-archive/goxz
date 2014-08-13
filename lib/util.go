// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package lib

/*
#cgo LDFLAGS: -llzma
#include "lzma.h"
*/
import "C"

import "io"
import "unsafe"
import "reflect"

// Split a Go slice into a pointer and length.
func SlicePtrLen(data []byte) (uintptr, int) {
	dataSlice := (*reflect.SliceHeader)(unsafe.Pointer(&data))
	return dataSlice.Data, dataSlice.Len
}

// Split a Go slice into a uint8_t pointer and size_t length.
func CSlicePtrLen(data []byte) (*C.uint8_t, C.size_t) {
	ptr, len := SlicePtrLen(data)
	return (*C.uint8_t)(unsafe.Pointer(ptr)), C.size_t(len)
}

type LinearBuffer struct {
	buf   []byte // Valid data between the ptrLo and ptrHi pointers.
	ptrLo int    // Range: 0     <= ptrLo <= ptrHi
	ptrHi int    // Range: ptrLo <= ptrHi <= len(buf)
}

// Move all data to the front of the buffer.
func (lb *LinearBuffer) Compact() ([]byte, int) {
	cnt := lb.ptrHi - lb.ptrLo
	if cnt > 0 {
		copy(lb.buf[:cnt], lb.buf[lb.ptrLo:lb.ptrHi])
	}
	return lb.buf, cnt
}

func (lb *LinearBuffer) tryReset() {
	if lb.ptrLo == len(lb.buf) && lb.ptrHi == len(lb.buf) {
		lb.ptrLo, lb.ptrHi = 0, 0
	}
}

type StreamReader struct {
	stream *Stream
	reader io.Reader
	closed bool
	end    error // Mark stream end
	LinearBuffer
}

func NewStreamReader(z *Stream, rd io.Reader, buf []byte, cnt int) *StreamReader {
	if cnt < 0 || cnt > len(buf) || len(buf) < 1 {
		panic("Invalid initialization buffer")
	}
	return &StreamReader{z, rd, false, nil, LinearBuffer{buf, 0, cnt}}
}

func (rs *StreamReader) Read(data []byte) (cnt int, err error) {
	if rs.closed {
		return 0, io.ErrClosedPipe
	}
	if len(data) == 0 {
		return 0, rs.end
	}

	rs.stream.SetOutput(data)
	for rs.stream.stream.avail_out > 0 && err == nil {
		err = rs.streamRead(OK)
	}
	cnt = len(data) - int(rs.stream.stream.avail_out)
	return cnt, rs.handleErr(cnt, err)
}

func (rs *StreamReader) Close() error {
	if rs.closed {
		return io.ErrClosedPipe
	}
	rs.closed = true

	defer rs.stream.End()
	return nil
}

func (rs *StreamReader) streamRead(code int) error {
	var zErr, rErr error
	var rAvail, rRdy, rCnt int

	// Read data into the ready buffer
	if rAvail = len(rs.buf) - rs.ptrHi; rAvail > 0 {
		rCnt, rErr = rs.reader.Read(rs.buf[rs.ptrHi:])
		rs.ptrHi += rCnt
	}

	// Process the data using the lzma stream
	if rRdy = rs.ptrHi - rs.ptrLo; rRdy > 0 {
		rs.stream.SetInput(rs.buf[rs.ptrLo:rs.ptrHi])
		zErr = rs.stream.Code(code)

		rCnt = rRdy - int(rs.stream.stream.avail_in) // Amount consumed
		rs.ptrLo += rCnt
	}

	rs.tryReset()
	return errFirst(zErr, rErr) // Stream error is more serious
}

func (rs *StreamReader) handleErr(cnt int, err error) error {
	if ErrMatch(err, STREAM_END) {
		err, rs.end = io.EOF, io.EOF // STREAM_END is this reader's true EOF.
	}
	if cnt == 0 && err == io.EOF && rs.end == nil {
		err = io.ErrUnexpectedEOF // Underlying reader EOF before STREAM_END.
	}
	if cnt > 0 && err == io.EOF {
		err = nil // So long as theres data, do not consider EOF.
	}
	return err
}

type StreamWriter struct {
	stream *Stream
	writer io.Writer
	closed bool
	LinearBuffer
}

func NewStreamWriter(z *Stream, wr io.Writer, buf []byte, cnt int) *StreamWriter {
	if cnt < 0 || cnt > len(buf) || len(buf) < 1 {
		panic("Invalid initialization buffer")
	}
	return &StreamWriter{z, wr, false, LinearBuffer{buf, 0, cnt}}
}

func (ws *StreamWriter) Write(data []byte) (cnt int, err error) {
	if ws.closed {
		return 0, io.ErrClosedPipe
	}
	if len(data) == 0 {
		return 0, nil
	}

	ws.stream.SetInput(data)
	for ws.stream.stream.avail_in > 0 && err == nil {
		err = ws.streamWrite(OK)
	}
	cnt = len(data) - int(ws.stream.stream.avail_in)
	return cnt, err
}

func (ws *StreamWriter) Close() (err error) {
	if ws.closed {
		return io.ErrClosedPipe
	}
	ws.closed = true

	defer ws.stream.End()
	for err == nil {
		err = ws.streamWrite(FINISH)
	}
	return ErrConvert(err, nil, STREAM_END) // Stream end is expected
}

func (ws *StreamWriter) streamWrite(code int) error {
	var zErr, wErr error
	var wAvail, wRdy, wCnt int

	// Process the data using the lzma stream
	if wAvail = len(ws.buf) - ws.ptrHi; wAvail > 0 {
		ws.stream.SetOutput(ws.buf[ws.ptrHi:])
		zErr = ws.stream.Code(code)

		wCnt = wAvail - int(ws.stream.stream.avail_out) // Amount consumed
		ws.ptrHi += wCnt
	}

	// Write ready bytes to the underlying transport
	if wRdy = ws.ptrHi - ws.ptrLo; wRdy > 0 {
		wCnt, wErr = ws.writer.Write(ws.buf[ws.ptrLo:ws.ptrHi])
		ws.ptrLo += wCnt
	}

	ws.tryReset()
	return errFirst(wErr, zErr) // Short-write is more serious
}
