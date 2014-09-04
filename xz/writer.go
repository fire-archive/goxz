// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package xz

import "io"
import "sync"
import "bytes"
import "time"
import "runtime"
import "bitbucket.org/rawr/golib/errs"
import "bitbucket.org/rawr/golib/bufpipe"
import "bitbucket.org/rawr/goxz/lib"

type sizeStats struct {
	unpadSize  int64
	uncompSize int64
}

type Writer struct {
	wr      io.Writer
	flags   *lib.StreamFlags
	index   *lib.Index
	filters *lib.Filters
	closed  bool

	maxWorkers int
	chunkSize  int
	maxSize    int

	// Internal state for asynchronous operations
	idx  int64
	pipe *pipe

	// Internal state for synchronous operations
	block  *lib.Block
	stream *lib.Stream
	rdBuf  *bufpipe.BufferPipe
	wrBuf  *bufpipe.BufferPipe

	// Communication resources between routines
	inSize, outSize int
	inMode, outMode int
	inPool, outPool *pipePool
	workPool        *workerPool
	errChan         chan error
	doneChan        chan bool
	lastErr         error
}

func NewWriter(wr io.Writer) (*Writer, error) {
	return NewWriterLevel(wr, DefaultCompression)
}

func NewWriterLevel(wr io.Writer, level int) (*Writer, error) {
	return NewWriterCustom(wr, level, CheckDefault, ChunkDefault, WorkersDefault)
}

func NewWriterCustom(wr io.Writer, level, checkType, chunkSize, maxWorkers int) (_ *Writer, err error) {
	defer errs.Recover(&err)
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
	maxSize := int(literalBlockSize(checkType, int64(chunkSize)))

	// Compose the stream header flags
	flags := lib.NewStreamFlags()
	buf, err := flags.HeaderEncode(checkType)
	errs.Panic(err)
	index, err := lib.NewIndex()
	errs.Panic(err)

	// Define the XZ filters
	opts, err := lib.NewOptionsLZMA(lib.PRESET_DEFAULT)
	errs.Panic(err)
	if chunkSize > 0 && int(opts.GetDictSize()) > chunkSize {
		opts.SetDictSize(uint32(chunkSize))
	}
	filters := lib.NewFilters()
	filters.Append(lib.FILTER_LZMA2, opts)

	// Write the header
	_, err = wr.Write(buf)
	errs.Panic(err)

	// Create the writer object
	w := new(Writer)
	w.wr = wr
	w.flags, w.index, w.filters = flags, index, filters
	w.maxWorkers, w.chunkSize, w.maxSize = maxWorkers, chunkSize, maxSize
	switch {
	case w.maxWorkers != WorkersSync: // Asynchronous operation
		w.inSize, w.outSize = int(w.chunkSize), int(w.maxSize)
		w.inMode, w.outMode = bufpipe.LineDual, bufpipe.LineMono
		w.inPool, w.outPool = newPipePool(), newPipePool()
		w.workPool = newWorkerPool(w.worker, w.outPool.close)
		w.errChan = make(chan error, 1)
		w.doneChan = make(chan bool)

		go w.monitor()
		go w.writer()

		// Make sure to shut everything down
		runtime.SetFinalizer(w, (*Writer).terminate)
	case w.chunkSize != ChunkStream: // Synchronous blocks
		rdBuf := make([]byte, w.chunkSize)
		wrBuf := make([]byte, w.maxSize)
		w.rdBuf = bufpipe.NewBufferPipe(rdBuf, bufpipe.LineMono)
		w.wrBuf = bufpipe.NewBufferPipe(wrBuf, bufpipe.LineMono)
	}
	if w.chunkSize == ChunkStream { // Streamed mode
		w.inMode, w.outMode = bufpipe.RingBlock, bufpipe.RingBlock
		w.block = w.blockInit(lib.VLI_UNKNOWN, lib.VLI_UNKNOWN)
		headerSize, err := w.block.HeaderSize()
		errs.Panic(err)
		data := make([]byte, headerSize)
		errs.Panic(w.block.HeaderEncode(data))
		_, err = w.wr.Write(data)
		errs.Panic(err)

		w.stream, err = w.block.NewStreamEncoder()
		errs.Panic(err)
	}
	return w, nil
}

func (w *Writer) Write(data []byte) (cnt int, err error) {
	defer errs.Recover(&err)
	if w.closed {
		return 0, io.ErrClosedPipe
	}
	switch {
	case w.maxWorkers != WorkersSync: // Asynchronous operation
		for cnt < len(data) {
			if w.pipe == nil {
				w.pipe = w.inPool.getWriter(w.inSize, w.inMode, w.idx, nil)
				if w.pipe == nil {
					break
				}
				w.idx++
			}

			rdCnt, err := w.pipe.Write(data[cnt:])
			cnt += rdCnt
			if err == io.ErrShortWrite {
				w.pipe.Close()
				w.pipe = nil
			} else if err != nil {
				break
			}
		}
		if cnt != len(data) {
			w.checkErr() // Bigger problem downstream
			errs.Panic(io.ErrShortWrite)
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

func (w *Writer) Close() (err error) {
	defer errs.Recover(&err)
	if w.closed {
		return nil
	}
	w.closed = true
	defer w.index.End()

	// Finish off any intermediate operations
	switch {
	case w.maxWorkers != WorkersSync: // Asynchronous operation
		if w.pipe != nil {
			w.pipe.Close()
			w.pipe = nil
		}
		w.inPool.close()
		w.checkErr()
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

	// Write the index
	stream, err := w.index.NewStreamEncoder()
	errs.Panic(err)
	_, _, err = stream.Process(lib.FINISH, w.wr, nil)
	errs.Panic(errs.Ignore(err, streamEnd))

	// Write the footer
	buf, err := w.flags.FooterEncode(w.index.Size())
	errs.Panic(err)
	_, err = w.wr.Write(buf)
	errs.Panic(err)

	return nil
}

func (w *Writer) checkErr() {
	errs.Panic(w.lastErr)
	w.lastErr = <-w.errChan
	errs.Panic(w.lastErr)
}

func (w *Writer) terminate() {
	defer errs.NilRecover()
	close(w.doneChan)
}

// Routine responsible for monitor progress and determining the optimal number
// of workers to allocate.
func (w *Writer) monitor() {
	defer w.inPool.terminate()
	defer w.outPool.terminate()
	defer w.workPool.terminate()
	setCapacity := func(resCnt int) {
		w.inPool.setCapacity(resCnt)
		w.outPool.setCapacity(resCnt)
		w.workPool.setCapacity(resCnt)
	}
	setCapacity(1) // Must have at least 1 worker to proceed

	if w.chunkSize != ChunkStream { // Asynchronous blocks
		// TODO(jtsai): Allow for dynamic scaling
		if w.maxWorkers == WorkersMax {
			setCapacity(runtime.NumCPU())
		} else {
			setCapacity(w.maxWorkers)
		}
	}
	_ = <-w.doneChan
}

// Routine responsible for actually compressing data. The number of worker
// routines spawned is monitored by the workerPool class.
func (w *Writer) worker(group *sync.WaitGroup, killChan chan bool) {
	defer group.Done()

	for inBuf := (*pipe)(nil); w.inPool.iterReader(&inBuf); {
		sizes := new(sizeStats)
		outBuf := w.outPool.getWriter(w.outSize, w.outMode, inBuf.idx, sizes)
		if outBuf != nil {
			var err error
			func() {
				defer errs.Recover(&err)
				wrBuf, rdBuf := outBuf.BufferPipe, inBuf.BufferPipe
				if w.chunkSize != ChunkStream { // Blocking mode
					unpadSize, uncompSize := w.blockEncode(wrBuf, rdBuf)
					sizes.unpadSize, sizes.uncompSize = unpadSize, uncompSize
				} else { // Streamed mode
					_, _, err := w.stream.ProcessPipe(lib.RUN, wrBuf, rdBuf)
					errs.Panic(errs.Ignore(err, io.EOF))
				}
			}()
			outBuf.CloseWithError(err)
		}
		w.inPool.restore(inBuf)

		select {
		case _ = <-w.doneChan:
			return
		case _ = <-killChan:
			return
		default:
		}
	}
}

// Routine responsible for only writing data to the underlying writer. When
// completed blocks come in asynchronously, they may be out-of-order. The logic
// here serializes the data before writing them.
func (w *Writer) writer() {
	buffers := make(map[int64]*pipe)
	var err error
	defer func() {
		for _, buf := range buffers {
			w.outPool.restore(buf)
		}
		w.errChan <- err
		close(w.errChan)
		w.terminate()
	}()
	defer errs.Recover(&err)

	var index int64
	for outBuf := (*pipe)(nil); w.outPool.iterReader(&outBuf); {
		buffers[outBuf.idx] = outBuf
		for buffers[index] != nil {
			buf := buffers[index]

			// Write data until buffer is closed
			_, err := buf.WriteTo(w.wr)
			errs.Panic(err)

			// Statistic values will be set before buffer is closed
			if sizes, ok := buf.data.(*sizeStats); ok && sizes.unpadSize != 0 {
				errs.Panic(w.index.Append(sizes.unpadSize, sizes.uncompSize))
			}

			w.outPool.restore(buf)
			delete(buffers, index)
			index++
		}
	}

	// Sanity check (no error raised thus far)
	if len(buffers) > 0 {
		panic(dataError)
	}
}

// Initialize a new block with the given sizes.
func (w *Writer) blockInit(compSize int64, uncompSize int64) *lib.Block {
	return lib.NewBlockCustom(w.flags.GetCheck(), w.filters, compSize, uncompSize)
}

// Encode a single block synchronously.
func (w *Writer) blockEncodeSync() {
	w.rdBuf.Close()
	defer w.rdBuf.Reset()
	unpadSize, uncompSize := w.blockEncode(w.wrBuf, w.rdBuf)
	errs.Panic(w.index.Append(unpadSize, uncompSize))

	w.wrBuf.Close()
	defer w.wrBuf.Reset()
	data, _, _ := w.wrBuf.ReadSlices()
	_, err := w.wr.Write(data)
	errs.Panic(err)
}

// Encode a single block using the provided read and write pipes.
// The write pipe, must be provided in the LineMono mode.
func (w *Writer) blockEncode(wrBuf, rdBuf *bufpipe.BufferPipe) (unpadSize, uncompSize int64) {
	if wrBuf.Mode() != bufpipe.LineMono {
		panic("invalid output pipe mode")
	}

	// Create the block
	block := w.blockInit(int64(w.chunkSize), int64(w.maxSize))
	headerSize, err := block.HeaderSize()
	errs.Panic(err)
	wrBuf.WriteMark(headerSize)

	stream, err := block.NewStreamEncoder()
	defer stream.End()
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
		_, rdSize := rdBuf.Pointers()
		input, output := rdBuf.Buffer(), wrBuf.Buffer()
		check := w.flags.GetCheck()
		cnt, pad := literalBlockEncode(check, input[:rdSize], output)
		unpadSize, uncompSize = int64(cnt-pad), int64(len(input))
		wrBuf.Rollback() // Valid for LineMono pipes
		wrBuf.WriteMark(cnt)
	} else { // Otherwise, write the normal header
		data := wrBuf.Buffer()[:headerSize]
		errs.Panic(block.HeaderEncode(data))
		unpadSize = block.UnpaddedSize()
		uncompSize = block.GetUncompressedSize()
	}
	return unpadSize, uncompSize
}
