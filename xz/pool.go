// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package xz

import "sync"
import "bitbucket.org/rawr/golib/bufpipe"

type pipe struct {
	*bufpipe.BufferPipe
	idx  int64
	data interface{}
}

type pipeSet map[int64]*pipe

func (m pipeSet) remove(p *pipe) (ok bool) {
	if p != nil {
		_, ok = m[p.idx]
		delete(m, p.idx)
	}
	return ok
}

func (m pipeSet) pop() (p *pipe) {
	for _, p = range m {
		delete(m, p.idx)
		break
	}
	return p
}

func (m pipeSet) push(p *pipe) {
	if p != nil {
		m[p.idx] = p
	}
}

func (m pipeSet) close() {
	for _, p := range m {
		p.Close()
	}
}

type pipePool struct {
	len    int // Number of buffers currently allocated
	cap    int // Number of buffers the pool may have
	closed bool

	free   pipeSet // Pipes that are unused
	ready  pipeSet // Pipes with a producer, and no consumer
	active pipeSet // Pipes with both a producer and consumer

	mutex  sync.Mutex
	wrCond sync.Cond
	rdCond sync.Cond

	waitIdx   int64
	wrWaitCnt int
	rdWaitCnt int
}

func newPipePool() *pipePool {
	pp := new(pipePool)
	pp.free = make(map[int64]*pipe)
	pp.ready = make(map[int64]*pipe)
	pp.active = make(map[int64]*pipe)
	pp.wrCond.L = &pp.mutex
	pp.rdCond.L = &pp.mutex
	pp.waitIdx = -1
	return pp
}

func (pp *pipePool) getWriter(idx int64, size, mode int, data interface{}) (p *pipe) {
	pp.mutex.Lock()
	defer pp.mutex.Unlock()

	if idx < 0 {
		panic("identifier index must be non-negative")
	}

	for !pp.closed && idx != pp.waitIdx && len(pp.free) == 0 && pp.len >= pp.cap {
		pp.wrWaitCnt++
		pp.wrCond.Wait()
		pp.wrWaitCnt--
	}
	if pp.closed {
		return
	}

	p = pp.free.pop()
	if p == nil { // Free set is empty, allocate new one
		p = new(pipe)
		p.BufferPipe = bufpipe.NewBufferPipe(make([]byte, size), mode)
		pp.len++
	} else if p.Capacity() != size || p.Mode() != mode {
		if cap(p.Buffer()) >= size { // Existing capacity is large enough
			p.BufferPipe = bufpipe.NewBufferPipe(p.Buffer()[:size], mode)
		} else { // Allocate new slice for BufferPipe
			p.BufferPipe = bufpipe.NewBufferPipe(make([]byte, size), mode)
		}
	} else {
		p.Reset()
	}
	p.idx, p.data = idx, data
	if _, ok := pp.ready[idx]; ok {
		panic("pipe with this index already exists")
	}
	pp.ready.push(p)
	pp.rdCond.Signal()
	return p
}

// Get a pipe reader from the pool. If the provided index is negative, then any
// reader in the pool will be popped.
//
// Waiting on a specific index ensures that a getWriter call for that index
// will succeed. This prevents a deadlock where all buffers are allocated and
// the writer for the next index in a sequence cannot proceed since no resource
// slots are available. Specifying an index allows a getWriter call to proceed
// even if it means violating the capacity limit temporarily.
func (pp *pipePool) getReader(idx int64) (p *pipe) {
	pp.mutex.Lock()
	defer pp.mutex.Unlock()

	var doWait func() bool
	if idx >= 0 {
		if pp.waitIdx >= 0 {
			panic("already waiting on an index identifier")
		}
		pp.waitIdx = idx
		pp.wrCond.Broadcast()
		doWait = func() bool { return pp.ready[idx] == nil } // Specific pipe
	} else {
		doWait = func() bool { return len(pp.ready) == 0 } // Any pipe
	}

	for !pp.closed && doWait() {
		pp.rdWaitCnt++
		pp.rdCond.Wait()
		pp.rdWaitCnt--
	}

	if idx >= 0 {
		p = pp.ready[idx]
		pp.ready.remove(p)
		pp.waitIdx = -1
	} else {
		p = pp.ready.pop()
	}
	pp.active.push(p)
	return p
}

func (pp *pipePool) iterReader(ptr **pipe) bool {
	*ptr = pp.getReader(-1)
	return *ptr != nil
}

func (pp *pipePool) iterReaderIdx(ptr **pipe, idx *int64) bool {
	*ptr = pp.getReader(*idx)
	*idx++
	return *ptr != nil
}

func (pp *pipePool) restore(p *pipe) {
	pp.mutex.Lock()
	defer pp.mutex.Unlock()
	if !pp.active.remove(p) {
		panic("pipe is not active in this pool")
	}

	p.Close()
	if pp.cap >= pp.len {
		pp.free.push(p)
		pp.wrCond.Signal()
	} else {
		pp.len--
	}
}

func (pp *pipePool) setCapacity(cnt int) {
	pp.mutex.Lock()
	defer pp.mutex.Unlock()

	pp.cap = cnt
	pp.wrCond.Broadcast()
}

func (pp *pipePool) getStats() (int, int) {
	pp.mutex.Lock()
	defer pp.mutex.Unlock()
	return pp.wrWaitCnt, pp.rdWaitCnt
}

// Closes down all pipes, but does not close down the pool itself. This is used
// as a way to tell all workers to discard their work, return their pipes to the
// pool, and restart processing. This is useful during seeking where work being
// done asynchronously is now stale and should be discarded.
func (pp *pipePool) interrupt() {
	pp.mutex.Lock()
	defer pp.mutex.Unlock()
	pp.ready.close()
	pp.active.close()
}

// Close the pool. Clients can no longer get any more writers. However, if
// clients still get any readers that are in the ready set.
func (pp *pipePool) close() {
	pp.mutex.Lock()
	defer pp.mutex.Unlock()
	pp.closed = true
	pp.wrCond.Broadcast()
	pp.rdCond.Broadcast()
}

// Terminate the pool. Clients can no longer get any more writers or readers.
// Furthermore, all existing pipes are closed to cause early work termination.
func (pp *pipePool) terminate() {
	pp.mutex.Lock()
	defer pp.mutex.Unlock()
	pp.ready.close()
	pp.active.close()
	pp.ready = nil // Make sure getReader returns nil

	pp.closed = true
	pp.wrCond.Broadcast()
	pp.rdCond.Broadcast()
}

type workerPool struct {
	workers []chan bool
	closed  bool
	work    func(*sync.WaitGroup, chan bool) // Callback for a worker routine
	end     func()                           // Callback for termination cleanup
	group   sync.WaitGroup
	once    sync.Once
}

func newWorkerPool(work func(*sync.WaitGroup, chan bool), end func()) *workerPool {
	wp := new(workerPool)
	wp.work = work
	wp.end = end
	return wp
}

func (wp *workerPool) setCapacity(cnt int) {
	if wp.closed {
		return
	}

	for len(wp.workers) < cnt { // Spawn workers
		killChan := make(chan bool)
		wp.group.Add(1)
		go wp.work(&wp.group, killChan)
		wp.workers = append(wp.workers, killChan)
		wp.once.Do(wp.monitor)
	}

	for len(wp.workers) > cnt { // Kill workers
		killChan := wp.workers[len(wp.workers)-1]
		wp.workers = wp.workers[:len(wp.workers)-1]
		close(killChan)
	}
}

// A single monitor routine runs for the worker pool. It is started as soon as
// the first worker is spawned. Also, it blocks until all workers have died and
// then calls the provided end callback.
func (wp *workerPool) monitor() {
	go func() {
		wp.group.Wait()
		wp.end() // Call the termination callback
	}()
}

func (wp *workerPool) terminate() {
	wp.closed = true
	for _, killChan := range wp.workers {
		close(killChan)
	}
	wp.workers = nil
}
