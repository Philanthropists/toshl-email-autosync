package result

type Result[T any] interface {
	Value() T
	Err() error
}

type ConcreteResult[T any] struct {
	Val   T
	Error error
}

func (r ConcreteResult[T]) Value() T {
	return r.Val
}

func (r ConcreteResult[T]) Err() error {
	return r.Error
}
