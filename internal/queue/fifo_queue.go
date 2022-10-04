package queue

type FIFOQueue[T any] interface {
	PushBack(*T) bool
	Pop() (*T, error)
	Size() uint
	IsEmpty() bool
	IsFull() bool
}
