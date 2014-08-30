// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package xz

import "fmt"
import "os"

import "io"
import "sync"
import "bytes"
import "encoding/binary"
import "crypto/sha256"
import "bitbucket.org/rawr/golib/errs"
import "bitbucket.org/rawr/golib/bufpipe"
import "bitbucket.org/rawr/goxz/lib"

var _ = fmt.Println
var _ = os.Stderr

type pipeStats struct {
	*bufpipe.BufferPipe
	idx        int64
	unpadSize  int64
	uncompSize int64
	err        error
}

func (p *pipeStats) Reset() {
	p.BufferPipe.Reset()
	p.unpadSize, p.uncompSize, p.err = 0, 0, nil
}

type blockWriter struct {
	wr      io.Writer
	flags   *lib.StreamFlags
	index   *lib.Index
	filters *lib.Filters

	maxWorkers   int
	chunkSize    int64
	maxChunkSize int64

	idx    int64
	buffer *pipeStats
	block  *lib.Block
	stream *lib.Stream
	rdBuf  *bufpipe.BufferPipe
	wrBuf  *bufpipe.BufferPipe

	group       sync.WaitGroup
	inFreeChan  chan *pipeStats
	inWorkChan  chan *pipeStats
	outFreeChan chan *pipeStats
	outWorkChan chan *pipeStats
	errChan     chan error
	doneChan    chan bool
}

func newBlockWriter(wr io.Writer, flags *lib.StreamFlags, index *lib.Index, filters *lib.Filters, chunkSize int64, maxWorkers int) *blockWriter {
	switch {
	case chunkSize < 0:
		chunkSize = ChunkStream
	case chunkSize >= 0 && chunkSize < ChunkLowest:
		chunkSize = ChunkLowest
	case chunkSize > ChunkHighest:
		chunkSize = ChunkHighest
	}
	if maxWorkers < 0 {
		maxWorkers = WorkersMax
	}

	// Make sure we support the requested check type
	switch flags.GetCheck() {
	case CheckNone, CheckCRC32, CheckCRC64, CheckSHA256:
	default:
		panic(lib.Error(lib.OPTIONS_ERROR))
	}

	blkWr := new(blockWriter)
	blkWr.wr = wr
	blkWr.flags = flags
	blkWr.index = index
	blkWr.filters = filters
	blkWr.maxWorkers = maxWorkers
	blkWr.chunkSize = chunkSize
	blkWr.maxChunkSize = blkWr.literalBlockSize(chunkSize)

	switch {
	case blkWr.maxWorkers != WorkersSync: // Asynchronous operation
		blkWr.inFreeChan = make(chan *pipeStats, chanBufSize)
		blkWr.inWorkChan = make(chan *pipeStats, chanBufSize)
		blkWr.outFreeChan = make(chan *pipeStats, chanBufSize)
		blkWr.outWorkChan = make(chan *pipeStats, chanBufSize)
		blkWr.errChan = make(chan error, 1)
		blkWr.doneChan = make(chan bool)

		blkWr.group.Add(2)
		go blkWr.monitor()
		go blkWr.writer()
	case blkWr.chunkSize != ChunkStream: // Synchronous blocks
		rdBuf := make([]byte, blkWr.chunkSize)
		wrBuf := make([]byte, blkWr.maxChunkSize)
		blkWr.rdBuf = bufpipe.NewBufferPipe(rdBuf, bufpipe.LineMono)
		blkWr.wrBuf = bufpipe.NewBufferPipe(wrBuf, bufpipe.LineMono)
	}
	if blkWr.chunkSize == ChunkStream { // Streamed mode
		blkWr.block = blkWr.blockInit(lib.VLI_UNKNOWN, lib.VLI_UNKNOWN)
		headerSize, err := blkWr.block.HeaderSize()
		errs.Panic(err)
		data := make([]byte, headerSize)
		errs.Panic(blkWr.block.HeaderEncode(data))
		_, err = blkWr.wr.Write(data)
		errs.Panic(err)

		blkWr.stream, err = blkWr.block.NewStreamEncoder()
		errs.Panic(err)
	}

	// Finalize, call end

	return blkWr
}

func (w *blockWriter) Write(data []byte) (cnt int, err error) {
	errs.Recover(&err)
	switch {
	case w.maxWorkers != WorkersSync: // Asynchronous operation
		for cnt < len(data) {
			if w.buffer == nil {
				w.buffer = <-w.inFreeChan
				w.buffer.idx = w.idx
				w.inWorkChan <- w.buffer
				w.idx++
			}

			rdCnt, err := w.buffer.Write(data[cnt:])
			cnt += rdCnt
			if err == io.ErrShortWrite {
				w.buffer.Close()
				w.buffer = nil
			}
			errs.Panic(errs.Ignore(err, io.ErrShortWrite))
		}
		return cnt, nil
	case w.chunkSize == ChunkStream: // Synchronous stream
		rd := bytes.NewReader(data)
		_, rdCnt, err := w.stream.Process(lib.RUN, w.wr, rd)
		return int(rdCnt), errs.Ignore(err, io.EOF)
	default: // Synchronous blocks
		for cnt < len(data) {
			rdCnt, err := w.rdBuf.Write(data[cnt:])
			cnt += rdCnt
			if err == io.ErrShortWrite {
				w.blockEncodeSync()
			}
			errs.Panic(errs.Ignore(err, io.ErrShortWrite))
		}
		return cnt, nil
	}
}

func (w *blockWriter) Close() (err error) {
	errs.Recover(&err)
	switch {
	case w.maxWorkers != WorkersSync: // Asynchronous operation
		if w.buffer != nil {
			w.buffer.Close()
		}
		close(w.inWorkChan)

		w.group.Wait()
		errs.Panic(<-w.errChan)
	case w.chunkSize != ChunkStream: // Synchronous blocks
		if w.rdBuf.Length() > 0 {
			w.blockEncodeSync()
		}
	}
	if w.chunkSize == ChunkStream { // Streamed mode
		defer w.stream.End()
		_, _, err := w.stream.Process(lib.FINISH, w.wr, nil)
		errs.Panic(errs.Ignore(err, streamEnd))

		unpadSize := w.block.UnpaddedSize()
		uncompSize := w.block.GetUncompressedSize()
		errs.Panic(w.index.Append(unpadSize, uncompSize))
	}
	return nil
}

/*
func (w *blockWriter) monitor(maxWorkers int, chunkSize int64) {
	defer close(w.outWorkChan)

	targetCnt := 1 // Target number of allocated resources
	var workerCnt, inBufCnt, outBufCnt int
	maxOutSize := w.bufferBound(chunkSize)

	killChan := make(chan bool, chanBufSize)
	deadChan := make(chan bool, chanBufSize)
	blockChan := make(chan bool, chanBufSize)
	for {
		for ; workerCnt < targetCnt; workerCnt++ {
			go w.worker(blockChan, killChan, deadChan)
		}
		for ; workerCnt > targetCnt; workerCnt-- {
			go func() { killChan <- true }()
		}

		for ; inBufCnt < targetCnt; inBufCnt++ {
			fmt.Println("Allocate input buffer")
			go func() {
				buf := newBuffer(make([]byte, chunkSize), true)
				w.inFreeChan <- &pipeStats{buf, 0}
			}()
		}
		for ; inBufCnt > targetCnt; inBufCnt-- {
			fmt.Println("Free input buffer")
			go func() { _ = <-w.inFreeChan }()
		}

		for ; outBufCnt < targetCnt; outBufCnt++ {
			fmt.Println("Allocate output buffer")
			go func() {
				buf := newBuffer(make([]byte, maxOutSize), false) // True if stream
				w.outFreeChan <- &pipeStats{buf, 0, 0, 0, nil}
			}()
		}
		for ; outBufCnt > targetCnt; outBufCnt-- {
			fmt.Println("Free output buffer")
			go func() { _ = <-w.outFreeChan }()
		}

		// Evaluate whether to change target count
		select {
		case _ = <-blockChan:
			// EVAL
			fmt.Println("WORKER LEVEL: ", targetCnt)
		case _ = <-deadChan:
			// DEATH
		}
	}
}
*/

func (w *blockWriter) monitor() {
	defer w.group.Done()
	// defer group.Wait() // Wait for all workers to die

}

func (w *blockWriter) worker(group *sync.WaitGroup) {
	defer group.Done()

	for inBuf := range w.inWorkChan {
		outBuf := <-w.outFreeChan
		outBuf.idx = inBuf.idx
		w.outWorkChan <- outBuf

		func() {
			defer errs.Recover(&outBuf.err)
			wrBuf, rdBuf := inBuf.BufferPipe, outBuf.BufferPipe
			if w.chunkSize != ChunkStream { // Blocking mode
				unpadSize, uncompSize := w.blockEncode(wrBuf, rdBuf)
				outBuf.unpadSize, outBuf.uncompSize = unpadSize, uncompSize
			} else { // Streamed mode
				_, _, err := w.stream.ProcessPipe(lib.RUN, wrBuf, rdBuf)
				errs.Panic(errs.Ignore(err, io.EOF))
			}
		}()

		outBuf.Close()
		inBuf.Reset()
		w.inFreeChan <- inBuf
	}
}

func (w *blockWriter) writer() {
	defer w.group.Done()
	defer func() {
		var err error
		if errs.Recover(&err); err != nil {
			w.errChan <- err
		}
		close(w.errChan)
		close(w.doneChan)
	}()

	var index int64
	buffers := make(map[int64]*pipeStats)
	for outBuf := range w.outWorkChan {
		buffers[outBuf.idx] = outBuf
		for buffers[index] != nil {
			buf := buffers[index]

			// Write data until buffer is closed
			_, err := buf.WriteTo(w.wr)
			errs.Panic(err)

			// Error and statistic values will be set before buffer is closed
			errs.Panic(buf.err)
			if buf.unpadSize != 0 && buf.uncompSize != 0 {
				errs.Panic(w.index.Append(buf.unpadSize, buf.uncompSize))
			}

			buf.Reset()
			w.outFreeChan <- buf
			delete(buffers, index)
			index++
		}
	}

	// Sanity check (no error raised thus far)
	if len(buffers) > 0 {
		panic(lib.Error(lib.DATA_ERROR))
	}
}

// Initialize a new block with the given sizes.
func (w *blockWriter) blockInit(compSize int64, uncompSize int64) *lib.Block {
	return lib.NewBlockCustom(w.flags.GetCheck(), w.filters, compSize, uncompSize)
}

// Encode a single block synchronously.
func (w *blockWriter) blockEncodeSync() {
	w.rdBuf.Close()
	defer w.rdBuf.Reset()
	unpadSize, uncompSize := w.blockEncode(w.wrBuf, w.rdBuf)
	errs.Panic(w.index.Append(unpadSize, uncompSize))

	w.wrBuf.Close()
	defer w.wrBuf.Reset()
	data, _ := w.wrBuf.ReadSlice()
	_, err := w.wr.Write(data)
	errs.Panic(err)
}

// Encode a single block using the provided read and write pipes.
// The write pipe, must be provided in the LineMono mode.
func (w *blockWriter) blockEncode(wrBuf, rdBuf *bufpipe.BufferPipe) (unpadSize, uncompSize int64) {
	if wrBuf.GetMode() != bufpipe.LineMono {
		panic("invalid output pipe mode")
	}

	// Create the block
	block := w.blockInit(w.chunkSize, w.maxChunkSize)
	headerSize, err := block.HeaderSize()
	errs.Panic(err)
	wrBuf.WriteMark(headerSize)

	stream, err := block.NewStreamEncoder()
	errs.Panic(err)

	// Try normal compression
	doLiteral := false
	_, _, err = stream.ProcessPipe(lib.RUN, wrBuf, rdBuf)
	doLiteral = doLiteral || errs.Match(err, io.ErrShortWrite)
	errs.Panic(errs.Ignore(err, io.ErrShortWrite, io.EOF))
	_, _, err = stream.ProcessPipe(lib.FINISH, wrBuf, nil)
	doLiteral = doLiteral || errs.Match(err, io.ErrShortWrite)
	errs.Panic(errs.Ignore(err, io.ErrShortWrite, streamEnd))

	if doLiteral { // Encode literal block if data is incompressible
		_, rdSize := rdBuf.GetOffsets()
		input, output := rdBuf.GetBuffer(), wrBuf.GetBuffer()
		cnt, pad := w.literalBlockEncode(input[:rdSize], output)
		unpadSize, uncompSize = int64(cnt-pad), int64(len(input))
		wrBuf.Reset()
		wrBuf.WriteMark(cnt)
	} else { // Otherwise, write the normal header
		data := wrBuf.GetBuffer()[:headerSize]
		errs.Panic(block.HeaderEncode(data))
		unpadSize = block.UnpaddedSize()
		uncompSize = block.GetUncompressedSize()
	}
	return unpadSize, uncompSize
}

// Compute the compressed data size for a literal block. This is only the data
// portion and does not contain the block headers, padding, or check.
func (w *blockWriter) literalCompSize(uncompSize int64) int64 {
	// Implementation based on pixz/writer.c
	numChunks := (uncompSize + lzmaChunkSize - 1) / lzmaChunkSize
	compSize := uncompSize + int64(numChunks*3) + 1
	return compSize
}

// Compute the total size for a literal block. This includes the block headers,
// data, padding, and check.
func (w *blockWriter) literalBlockSize(uncompSize int64) int64 {
	// Implementation based on pixz/writer.c
	compSize := w.literalCompSize(uncompSize)

	check := w.flags.GetCheck()
	block := lib.NewBlockCustom(check, w.filters, compSize, uncompSize)
	headerSize, err := block.HeaderSize()
	errs.Panic(err)
	padSize := (4 - compSize%4) % 4
	checkSize := lib.CheckSize(check)
	return int64(headerSize) + compSize + padSize + int64(checkSize)
}

// Compresses the input as literal chunks into the output slice. The output
// slice must be large enough, as calculated with literalBlockSize().
func (w *blockWriter) literalBlockEncode(input []byte, output []byte) (cnt, pad int) {
	// Implementation based on pixz/writer.c
	uncompSize := int64(len(input))
	compSize := w.literalCompSize(uncompSize)

	// Write the header
	check := w.flags.GetCheck()
	block := lib.NewBlockCustom(check, w.filters, compSize, uncompSize)
	headerSize, err := block.HeaderSize()
	errs.Panic(err)
	errs.Panic(block.HeaderEncode(output))
	cnt += headerSize

	// Write the literal chunks
	for inCnt := 0; inCnt < len(input); inCnt += lzmaChunkSize {
		chunk := input[inCnt:]
		if len(chunk) > lzmaChunkSize {
			chunk = chunk[:lzmaChunkSize]
		}
		output[cnt] = 1 // Literal chunk marker
		cnt++

		// Chunk size
		size := uint16(len(chunk) - 1)
		binary.BigEndian.PutUint16(output[cnt:], size)
		cnt += 2

		// Copy literal data
		cnt += copy(output[cnt:], chunk)
	}
	output[cnt] = 0 // End-of-chunk marker
	cnt++

	// Write the padding
	for ; cnt%4 > 0; cnt++ {
		output[cnt] = 0
		pad++
	}

	// Write the check
	switch check {
	case CheckNone:
		// Do nothing
	case CheckCRC32:
		crc := lib.CRC32(input, 0)
		binary.LittleEndian.PutUint32(output[cnt:], crc)
	case CheckCRC64:
		crc := lib.CRC64(input, 0)
		binary.LittleEndian.PutUint64(output[cnt:], crc)
	case CheckSHA256:
		sha := sha256.Sum256(input)
		copy(output[cnt:], sha[:])
	}
	cnt += lib.CheckSize(check)

	return cnt, pad
}
