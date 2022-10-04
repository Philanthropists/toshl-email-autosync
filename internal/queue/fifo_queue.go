package queue

type FIFOQueue[T any] interface {
	PushBack(*T) bool
	Pop() (*T, error)
	Size() int
	IsEmpty() bool
	IsFull() bool
}
