package mutex

import (
	"fmt"
	"sync"
)

type mutexFifoQueue[T any] struct {
	MaxSize int
	Store   []*T
	Mutex   *sync.Mutex
}

func CreateQueue[T any](maxsize int) *mutexFifoQueue[T] {
	return &mutexFifoQueue[T]{
		MaxSize: maxsize,
		Mutex:   &sync.Mutex{},
	}
}

func (q *mutexFifoQueue[T]) PushBack(e *T) bool {
	q.Mutex.Lock()
	defer q.Mutex.Unlock()

	if q.MaxSize > 0 && len(q.Store) == q.MaxSize {
		return false
	}

	q.Store = append(q.Store, e)

	return true
}

func (q *mutexFifoQueue[T]) Pop() (*T, error) {
	q.Mutex.Lock()
	defer q.Mutex.Unlock()

	if len(q.Store) == 0 {
		return nil, fmt.Errorf("queue is empty")
	}

	topElement := q.Store[0]
	q.Store = q.Store[1:]

	return topElement, nil
}

func (q *mutexFifoQueue[T]) Size() int {
	q.Mutex.Lock()
	defer q.Mutex.Unlock()
	return len(q.Store)
}

func (q *mutexFifoQueue[T]) IsEmpty() bool {
	return q.Size() == 0
}

func (q *mutexFifoQueue[T]) IsFull() bool {
	length := q.Size()
	return (q.MaxSize > 0 && length == q.MaxSize)
}
