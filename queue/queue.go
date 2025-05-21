package queue

type (
	MemorySize interface {
		MemoryUsage() uint64
	}

	Queue[T any] interface {
		Len() uint64
		Cap() uint64
		PushAtomic(T) PushResult
		Push(v T) bool
		PopAtomic() (T, PopResult)
		Pop() (T, bool)
	}
)
