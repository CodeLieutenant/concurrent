package queue

type (
	PushResult int
	PopResult  int
)

const (
	PushFailed PushResult = iota - 1
	PushSuccess
	PushFull
)

const (
	PopFailed PopResult = iota - 1
	PopSuccess
	PopEmpty
)

func (r PushResult) String() string {
	switch r {
	case PushFailed:
		return "PushFailed"
	case PushSuccess:
		return "PushSuccess"
	case PushFull:
		return "PushFull"
	default:
		panic("unknown PushResult")
	}
}

func (r PopResult) String() string {
	switch r {
	case PopFailed:
		return "PopFailed"
	case PopSuccess:
		return "PopSuccess"
	case PopEmpty:
		return "PopEmpty"
	default:
		panic("unknown PopResult")
	}
}

func (r PushResult) IsSuccess() bool {
	return r == PushSuccess
}

func (r PopResult) IsSuccess() bool {
	return r == PopSuccess
}
