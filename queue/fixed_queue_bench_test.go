package queue

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type (
	QueueLike[T any] interface {
		Push(T) bool
		Pop() (T, bool)
		Len() int64
	}
	ChannelQueue[T any] struct {
		ch chan T
	}

	RWQueue[T any] struct {
		buf      []T
		head     int
		tail     int
		size     int
		capacity int
		mu       sync.RWMutex
	}

	MutexQueue[T any] struct {
		buf      []T
		head     int
		tail     int
		size     int
		capacity int
		mu       sync.Mutex
	}
)

func NewChannelQueue[T any](capacity int) *ChannelQueue[T] {
	return &ChannelQueue[T]{ch: make(chan T, capacity)}
}

func (q *ChannelQueue[T]) Push(v T) bool {
	select {
	case q.ch <- v:
		return true
	default:
		return false // buffer full
	}
}

func (q *ChannelQueue[T]) Pop() (out T, ok bool) {
	select {
	case out = <-q.ch:
		return out, true
	default:
		return out, false // buffer empty
	}
}

func (q *ChannelQueue[T]) Len() int64 { return int64(len(q.ch)) }

func NewRWQueue[T any](capacity int) *RWQueue[T] {
	return &RWQueue[T]{
		buf:      make([]T, capacity),
		capacity: capacity,
	}
}

func (q *RWQueue[T]) Push(v T) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.size == q.capacity { // full
		return false
	}
	q.buf[q.tail] = v
	q.tail = (q.tail + 1) % q.capacity
	q.size++
	return true
}

func (q *RWQueue[T]) Pop() (zero T, ok bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.size == 0 {
		return zero, false
	}
	val := q.buf[q.head]
	q.head = (q.head + 1) % q.capacity
	q.size--
	return val, true
}

func (q *RWQueue[T]) Len() int64 {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return int64(q.size)
}

func NewMutexQueue[T any](capacity int) *MutexQueue[T] {
	return &MutexQueue[T]{
		buf:      make([]T, capacity),
		capacity: capacity,
	}
}

func (q *MutexQueue[T]) Push(v T) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.size == q.capacity {
		return false
	}
	q.buf[q.tail] = v
	q.tail = (q.tail + 1) % q.capacity
	q.size++
	return true
}

func (q *MutexQueue[T]) Pop() (zero T, ok bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.size == 0 {
		return zero, false
	}
	val := q.buf[q.head]
	q.head = (q.head + 1) % q.capacity
	q.size--
	return val, true
}

func (q *MutexQueue[T]) Len() int64 {
	q.mu.Lock()
	defer q.mu.Unlock()
	return int64(q.size)
}

const payload = 12345

func runBench(b *testing.B, makeQueue func() QueueLike[int]) {
	b.Helper()
	q := makeQueue()

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			q.Pop()
		}
	})

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			q.Push(payload)
		}
	})
}

func BenchmarkChannelQueue(b *testing.B) {
	runBench(b, func() QueueLike[int] { return NewChannelQueue[int](DefaultQueueSize) })
}

func BenchmarkRWQueue(b *testing.B) {
	runBench(b, func() QueueLike[int] { return NewRWQueue[int](DefaultQueueSize) })
}

func BenchmarkMutexQueue(b *testing.B) {
	runBench(b, func() QueueLike[int] { return NewMutexQueue[int](DefaultQueueSize) })
}

func BenchmarkLockFreeQueue(b *testing.B) {
	runBench(b, func() QueueLike[int] { return NewFixedQueue[int](DefaultQueueSize) })
}

const (
	producerRatio = 0.50            // 50 % producers, 50 % consumers
	testDuration  = 2 * time.Second // total wall-clock time per benchmark
)

func runContended(b *testing.B, makeQ func() QueueLike[int]) {
	b.Helper()

	var (
		procs     = runtime.GOMAXPROCS(0)
		producers = int(float64(procs)*producerRatio + .5)
		consumers = procs - producers
		pushes    int64
		pops      int64
		stop      int32
		wg        sync.WaitGroup
		q         = makeQ()
	)

	startBenchmark := func() {
		wg.Add(producers + consumers)

		// producers
		for p := 0; p < producers; p++ {
			go func() {
				defer wg.Done()
				for atomic.LoadInt32(&stop) == 0 {
					for !q.Push(payload) && atomic.LoadInt32(&stop) == 0 {
						runtime.Gosched() // back off if full
					}
					atomic.AddInt64(&pushes, 1)
				}
			}()
		}

		// consumers
		for c := 0; c < consumers; c++ {
			go func() {
				defer wg.Done()
				for atomic.LoadInt32(&stop) == 0 {
					if _, ok := q.Pop(); ok {
						atomic.AddInt64(&pops, 1)
					} else {
						runtime.Gosched() // back off if empty
					}
				}
			}()
		}
	}

	b.ReportAllocs()
	b.ResetTimer()

	startBenchmark()
	time.Sleep(testDuration)
	atomic.StoreInt32(&stop, 1)
	wg.Wait()

	b.StopTimer()

	// treat every completed operation as one unit of work
	totalOps := pushes + pops
	b.SetBytes(0) // unrelated to bytes processed
	b.ReportMetric(float64(totalOps)/testDuration.Seconds(), "ops/s")
}

func BenchmarkContended_ChannelQueue(b *testing.B) {
	runContended(b, func() QueueLike[int] { return NewChannelQueue[int](DefaultQueueSize) })
}

func BenchmarkContended_LockFreeQueue(b *testing.B) {
	runContended(b, func() QueueLike[int] { return NewFixedQueue[int](DefaultQueueSize) })
}
