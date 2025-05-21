package queue

import (
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
)

func TestNestNextPowerOfTwo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		n    int
		want uint64
	}{
		{"negative", -5, 0},
		{"zero", 0, 0},
		{"one", 1, 1},
		{"two", 2, 2},
		{"three", 3, 4},
		{"four", 4, 4},
		{"five", 5, 8},
		{"seven", 7, 8},
		{"eight", 8, 8},
		{"nine", 9, 16},
		{"large_number", 1023, 1024},
		{"exact_power", 1024, 1024},
		{"between_powers", 1025, 2048},
		{"very_large", 1<<30 - 10, 1 << 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := nextPowerOfTwo(tt.n); got != tt.want {
				t.Errorf("NextPowerOfTwo(%d) = %d, want %d", tt.n, got, tt.want)
			}
		})
	}
}

func BenchmarkNextPowerOfTwo(b *testing.B) {
	inputs := []int{3, 10, 100, 1000, 1 << 20}

	b.Run("Single", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = nextPowerOfTwo(100)
		}
	})

	for _, n := range inputs {
		b.Run("Size-"+strconv.FormatInt(int64(n), 10), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				_ = nextPowerOfTwo(n)
			}
		})
	}

	b.Run("Parallel", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = nextPowerOfTwo(100)
			}
		})
	})
}

func TestQueue(t *testing.T) {
	t.Parallel()

	n := 16
	q := NewFixedQueue[int](n)

	for range n {
		if q.PushAtomic(1) != PushSuccess {
			t.Fatal("queue is full")
		}
	}

	if q.PushAtomic(1) != PushFull {
		t.Fatal("queue is not full")
	}

	for range n {
		if val, status := q.PopAtomic(); status != PopSuccess && val != 1 {
			t.Fatal("queue is empty")
		}
	}

	if val, status := q.PopAtomic(); status != PopEmpty && val != 1 {
		t.Fatal("queue is not empty")
	}
}

func TestEnqueueConcurrent(t *testing.T) {
	t.Parallel()

	const (
		producers   = 32
		perProducer = 1000
		capacity    = producers * perProducer
	)

	q := NewFixedQueue[int](capacity)

	var wg sync.WaitGroup
	wg.Add(producers)

	for p := range producers {
		go func(offset int) {
			defer wg.Done()
			for i := range perProducer {
				v := offset*perProducer + i
				for !q.Push(v) {
					runtime.Gosched()
				}
			}
		}(p)
	}

	wg.Wait()

	// Verify that we can now dequeue exactly cap items.
	seen := make([]bool, capacity)
	for i := range capacity {
		v, ok := q.Pop()
		if !ok {
			t.Fatalf("queue exhausted after %d dequeues", i)
		}
		if v < 0 || v >= capacity {
			t.Fatalf("dequeued out‑of‑range value %d", v)
		}
		if seen[v] {
			t.Fatalf("duplicate value %d dequeued", v)
		}
		seen[v] = true
	}

	if _, status := q.Pop(); status {
		t.Fatalf("queue not empty after draining")
	}
}

func TestEnqueueDequeueRace(t *testing.T) {
	t.Parallel()

	const (
		producers = 32
		consumers = 32
		ops       = 10000
	)
	q := NewFixedQueue[int](1024)

	var produced, consumed atomic.Uint64
	var wg sync.WaitGroup

	// Producers
	wg.Add(producers)
	for range producers {
		go func() {
			defer wg.Done()
			for i := range ops {
				for !q.Push(i) {
					runtime.Gosched()
				}

				produced.Add(1)
			}
		}()
	}

	// Consumers
	wg.Add(consumers)
	for range consumers {
		go func() {
			defer wg.Done()
			for consumed.Load() < producers*ops {
				if _, status := q.Pop(); status {
					consumed.Add(1)
				} else {
					runtime.Gosched()
				}
			}
		}()
	}

	wg.Wait()
	if produced.Load() != consumed.Load() {
		t.Fatalf("produced %d, consumed %d", produced.Load(), consumed.Load())
	}
}
func TestFixedQueue_Len(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		queueSize   int
		pushItems   int
		popItems    int
		expectedLen uint64
	}{
		{
			name:        "empty queue",
			queueSize:   8,
			pushItems:   0,
			popItems:    0,
			expectedLen: 0,
		},
		{
			name:        "partially filled queue",
			queueSize:   8,
			pushItems:   3,
			popItems:    0,
			expectedLen: 3,
		},
		{
			name:        "fully filled queue",
			queueSize:   8,
			pushItems:   8,
			popItems:    0,
			expectedLen: 8,
		},
		{
			name:        "queue with pushes and pops",
			queueSize:   16,
			pushItems:   10,
			popItems:    4,
			expectedLen: 6,
		},
		{
			name:        "queue with all items popped",
			queueSize:   8,
			pushItems:   5,
			popItems:    5,
			expectedLen: 0,
		},
		{
			name:        "small queue size",
			queueSize:   2,
			pushItems:   2,
			popItems:    1,
			expectedLen: 1,
		},
		{
			name:        "default queue size",
			queueSize:   DefaultQueueSize,
			pushItems:   100,
			popItems:    50,
			expectedLen: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			q := NewFixedQueue[int](tt.queueSize)

			// Push items
			for i := 0; i < tt.pushItems; i++ {
				if !q.Push(i) {
					t.Fatalf("failed to push item %d, queue reported as full", i)
				}
			}

			// Pop items
			for i := 0; i < tt.popItems; i++ {
				if _, ok := q.Pop(); !ok {
					t.Fatalf("failed to pop item %d, queue reported as empty", i)
				}
			}

			// Test Len()
			if got := q.Len(); got != tt.expectedLen {
				t.Errorf("Len() = %d, want %d", got, tt.expectedLen)
			}
		})
	}
}

func TestFixedQueue_Cap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		queueSize    int
		expectedCap  uint64
		expectedPow2 bool
	}{
		{
			name:         "zero size becomes default",
			queueSize:    0,
			expectedCap:  DefaultQueueSize,
			expectedPow2: true,
		},
		{
			name:         "negative size panics",
			queueSize:    -1,
			expectedCap:  0, // Not used due to panic
			expectedPow2: true,
		},
		{
			name:         "size 1 becomes 2",
			queueSize:    1,
			expectedCap:  2,
			expectedPow2: true,
		},
		{
			name:         "exact power of 2",
			queueSize:    8,
			expectedCap:  8,
			expectedPow2: true,
		},
		{
			name:         "non-power of 2 rounds up",
			queueSize:    10,
			expectedCap:  16,
			expectedPow2: true,
		},
		{
			name:         "large size",
			queueSize:    1000,
			expectedCap:  1024,
			expectedPow2: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.queueSize < 0 {
				defer func() {
					if r := recover(); r == nil {
						t.Error("Expected panic for negative queue size, but no panic occurred")
					}
				}()
			}

			q := NewFixedQueue[int](tt.queueSize)

			got := q.Cap()
			if got != tt.expectedCap {
				t.Errorf("Cap() = %d, want %d", got, tt.expectedCap)
			}

			// Verify capacity is a power of 2
			if tt.expectedPow2 && (got&(got-1)) != 0 {
				t.Errorf("Cap() = %d is not a power of 2", got)
			}

			// Verify that the capacity matches mask+1
			if got != q.mask+1 {
				t.Errorf("Cap() = %d, but q.mask+1 = %d", got, q.mask+1)
			}
		})
	}
}

func TestFixedQueue_LenAfterWrap(t *testing.T) {
	t.Parallel()

	// Test that Len() works correctly after queue indices wrap around
	q := NewFixedQueue[int](4) // Capacity = 4

	// Fill and drain the queue several times to ensure indices wrap around
	for cycle := 0; cycle < 10; cycle++ {
		// Fill queue
		for i := 0; i < 4; i++ {
			if !q.Push(i) {
				t.Fatalf("cycle %d: failed to push item %d", cycle, i)
			}

			// Check length increases with each push
			expectedLen := uint64(i + 1)
			if got := q.Len(); got != expectedLen {
				t.Errorf("cycle %d after push %d: Len() = %d, want %d", cycle, i, got, expectedLen)
			}
		}

		// Empty queue
		for i := 0; i < 4; i++ {
			if _, ok := q.Pop(); !ok {
				t.Fatalf("cycle %d: failed to pop item %d", cycle, i)
			}

			// Check length decreases with each pop
			expectedLen := uint64(3 - i)
			if got := q.Len(); got != expectedLen {
				t.Errorf("cycle %d after pop %d: Len() = %d, want %d", cycle, i, got, expectedLen)
			}
		}

		// Queue should be empty now
		if got := q.Len(); got != 0 {
			t.Errorf("cycle %d after emptying: Len() = %d, want 0", cycle, got)
		}
	}
}

func TestFixedQueue_CapAndLenRelationship(t *testing.T) {
	t.Parallel()

	sizes := []int{2, 4, 7, 8, 15, 16, 100, 256}

	for _, size := range sizes {
		t.Run("size_"+string(rune(size)), func(t *testing.T) {
			t.Parallel()
			q := NewFixedQueue[int](size)

			cap := q.Cap()
			expectedCap := nextPowerOfTwo(size)
			if cap != expectedCap {
				t.Errorf("Cap() = %d, want %d", cap, expectedCap)
			}

			// Initially empty
			if got := q.Len(); got != 0 {
				t.Errorf("Initial Len() = %d, want 0", got)
			}

			// Fill to capacity
			for i := uint64(0); i < cap; i++ {
				if !q.Push(int(i)) {
					t.Fatalf("failed to push item %d", i)
				}

				if got := q.Len(); got != i+1 {
					t.Errorf("After pushing %d items: Len() = %d, want %d", i+1, got, i+1)
				}
			}

			// Should be full now
			if got := q.Len(); got != cap {
				t.Errorf("When full: Len() = %d, want Cap() = %d", got, cap)
			}

			// Cannot push more
			if q.Push(0) {
				t.Errorf("Push succeeded on full queue")
			}
		})
	}
}
