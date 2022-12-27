package pipe

import (
	"sync"
)

type Result[T any] struct {
	Error error
	Value T
}

type Handler[T, K any] func(T) Result[K]

func internalMapper[A, B any](mapper func(A) (B, error)) func(A) Result[B] {
	if mapper == nil {
		// nop function
		return func(A) Result[B] {
			return Result[B]{}
		}
	}

	return func(a A) Result[B] {
		b, err := mapper(a)
		return Result[B]{
			Value: b,
			Error: err,
		}
	}
}

// Multiplexes multiple channels into a single one
func FanIn[T any](done <-chan struct{}, channels ...<-chan T) <-chan T {
	var wg sync.WaitGroup
	multiplexedStream := make(chan T)

	multiplex := func(c <-chan T) {
		defer wg.Done()

		for i := range c {
			select {
			case <-done:
				return
			case multiplexedStream <- i:
			}
		}
	}

	wg.Add(len(channels))
	for _, c := range channels {
		go multiplex(c)
	}

	go func() {
		defer close(multiplexedStream)
		wg.Wait()
	}()

	return multiplexedStream
}

// Ensures that the goroutine is finished on done being closed
func OrDone[T any](done <-chan struct{}, c <-chan T) <-chan T {
	stream := make(chan T)

	go func() {
		defer close(stream)

		for {
			select {
			case <-done:
				return
			case v, ok := <-c:
				if !ok {
					return
				}
				select {
				case stream <- v:
				case <-done:
				}
			}
		}
	}()

	return stream
}

// Creates two different streams from one origin stream
func Tee[T any](done <-chan struct{}, in <-chan T) (_, _ <-chan T) {
	out1 := make(chan T)
	out2 := make(chan T)

	go func() {
		defer close(out1)
		defer close(out2)

		for val := range OrDone(done, in) {
			// intentional shadowing
			var out1, out2 = out1, out2

			for i := 0; i < 2; i++ {
				select {
				case out1 <- val:
					out1 = nil
				case out2 <- val:
					out2 = nil
				}
			}
		}
	}()

	return out1, out2
}

// Handles errors and only streams errorless ones
func OnError[T any](done <-chan struct{}, in <-chan Result[T], handler func(T, error)) <-chan T {
	out := make(chan T, 1)

	go func() {
		defer close(out)

		for val := range OrDone(done, in) {
			if val.Error != nil {
				go func(val Result[T]) {
					handler(val.Value, val.Error)
				}(val)
			} else {
				select {
				case <-done:
					return
				case out <- val.Value:
				}
			}
		}
	}()

	return out
}

// Only output results that are not with error
func IgnoreOnError[T any](done <-chan struct{}, in <-chan Result[T]) <-chan T {
	return OnError(done, in, func(T, error) {
		// nop operation
	})
}

// Only output results that are with error
func OnlyOnError[T any](done <-chan struct{}, in <-chan Result[T]) <-chan Result[T] {
	out := make(chan Result[T], 1)

	go func() {
		defer close(out)

		for val := range OrDone(done, in) {
			if val.Error != nil {
				select {
				case <-done:
					return
				case out <- val:
				}
			}
		}
	}()

	return out
}

// Maps from channel of type A to a channel of type B
func Map[A, B any](done <-chan struct{}, in <-chan A, mapper func(A) (B, error)) <-chan Result[B] {
	out := make(chan Result[B])

	inmap := internalMapper(mapper)
	go func() {
		defer close(out)

		for val := range OrDone(done, in) {
			select {
			case <-done:
				return
			case out <- inmap(val):
			}
		}
	}()

	return out
}

// Maps from channel of type A to a channel of type B concurrently
func ConcurrentMap[A, B any](done <-chan struct{}, coroutines uint, in <-chan A, mapper func(A) (B, error)) <-chan Result[B] {
	if coroutines == 0 {
		coroutines = 1
	}

	out := make(chan Result[B], coroutines)

	var wg sync.WaitGroup
	wg.Add(int(coroutines))
	inmap := internalMapper(mapper)
	for i := 0; i < int(coroutines); i++ {
		go func() {
			defer wg.Done()

			for val := range OrDone(done, in) {
				select {
				case <-done:
					return
				case out <- inmap(val):
				}
			}
		}()
	}

	go func() {
		defer close(out)
		wg.Wait()
	}()

	return out
}

// Consumes from channel until it is closed or done channel is closed
func WaitClosed[T any](done <-chan struct{}, in <-chan T) {
	if in == nil {
		return
	}

	for {
		select {
		case <-done:
			return
		case _, ok := <-in:
			if !ok {
				return
			}
		}
	}
}

func AsyncResult[T any](done <-chan struct{}, callback func() (T, error)) <-chan Result[T] {
	if callback == nil {
		return nil
	}

	out := make(chan Result[T], 1)
	proxy := make(chan Result[T], 1)

	go func(proxy chan<- Result[T]) {
		defer close(proxy)

		t, err := callback()
		proxy <- Result[T]{
			Value: t,
			Error: err,
		}
	}(proxy)

	go func(c chan<- Result[T], proxy <-chan Result[T]) {
		defer close(c)

		select {
		case <-done:
			return
		case r, ok := <-proxy:
			if ok {
				out <- r
			}
		}
	}(out, proxy)

	return out
}

func NopConsumer[T any](done <-chan struct{}, in <-chan T) {
	go func() {
		for {
			select {
			case <-done:
				return
			case _, ok := <-in:
				if !ok {
					return
				}
			}
		}
	}()
}

func Gather[A, B any](done <-chan struct{}, in <-chan A, mapper func(A) (B, error)) (res []Result[B]) {
	for {
		select {
		case <-done:
			return
		case v, ok := <-in:
			if ok {
				m, err := mapper(v)
				r := Result[B]{
					Error: err,
					Value: m,
				}
				res = append(res, r)
			} else {
				return
			}
		}
	}
}
