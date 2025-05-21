package queue

import (
	"runtime"
	"sync/atomic"
)

const (
	spinTries = 128
)

const DefaultQueueSize = 256

type (
	slot[T any] struct {
		value T
		seq   atomic.Uint64
	}

	FixedQueue[T any] struct {
		buf  []slot[T]
		tail atomic.Uint64
		head atomic.Uint64
		mask uint64
	}
)

// NewFixedQueue returns a pointer to a queue whose capacity is the next power of two
// equal to or greater than n (minimum 2).
//
//go:nosplit
//go:registerparams
//go:norace
func NewFixedQueue[T any](n int) *FixedQueue[T] {
	switch {
	case n < 0:
		panic("queue size cannot be negative")
	case n == 0:
		n = DefaultQueueSize
	case n < 2:
		n = 2
	}

	capPow := nextPowerOfTwo(n)
	q := &FixedQueue[T]{
		mask: capPow - 1,
		buf:  make([]slot[T], capPow),
	}

	for i := range capPow {
		q.buf[i].seq.Store(i)
	}

	return q
}

//go:nosplit
//go:registerparams
//go:norace
func (q *FixedQueue[T]) Len() uint64 {
	return q.tail.Load() - q.head.Load()
}

//go:nosplit
//go:registerparams
//go:norace
func (q *FixedQueue[T]) Cap() uint64 {
	return q.mask + 1
}

//go:nosplit
//go:registerparams
//go:norace
func (q *FixedQueue[T]) PushAtomic(v T) PushResult {
	pos := q.tail.Load()
	cell := &q.buf[pos&q.mask]

	if int64(cell.seq.Load())-int64(pos) < 0 {
		return PushFull
	}

	if q.tail.CompareAndSwap(pos, pos+1) {
		cell.value = v
		cell.seq.Store(pos + 1)

		return PushSuccess
	}

	return PushFailed
}

// Push attempts to push v into the queue.
// Returns false if the queue is full.
//
//go:inline
//go:nosplit
//go:registerparams
//go:norace
func (q *FixedQueue[T]) Push(v T) bool {
	spin := 0
	for {
		switch q.PushAtomic(v) {
		case PushSuccess:
			return true
		case PushFull:
			return false
		case PushFailed:
			if spin < spinTries {
				spin++
				_ = spin // Hint to CPU: a tiny empty loop gives the core a branch‑miss‑free pause.
			} else {
				runtime.Gosched()
				spin = 0
			}
		}
	}
}

//go:nosplit
//go:registerparams
//go:norace
func (q *FixedQueue[T]) PopAtomic() (T, PopResult) {
	var zero T
	pos := q.head.Load()
	cell := &q.buf[pos&q.mask]
	seq := cell.seq.Load()
	diff := int64(seq) - int64(pos+1)

	if diff < 0 {
		return zero, PopEmpty
	}

	if diff == 0 && q.head.CompareAndSwap(pos, pos+1) {
		val := cell.value // Help GC by clearing the slot.

		cell.value = zero
		cell.seq.Store(pos + q.mask + 1)

		return val, PopSuccess
	}

	return zero, PopFailed
}

//go:nosplit
//go:registerparams
//go:norace
func (q *FixedQueue[T]) Pop() (T, bool) {
	spin := 0
	for {
		switch v, status := q.PopAtomic(); status {
		case PopSuccess:
			return v, true
		case PopEmpty:
			return v, false
		default:
			if spin < spinTries {
				spin++
				_ = spin // Hint to CPU: a tiny empty loop gives the core a branch‑miss‑free pause.
			} else {
				runtime.Gosched()
				spin = 0
			}
		}
	}
}

//go:nosplit
//go:registerparams
//go:norace
func nextPowerOfTwo(n int) uint64 {
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	return uint64(n + 1)
}
