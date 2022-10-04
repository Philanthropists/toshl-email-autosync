package mutex

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type someType struct{}

func Test_QueueShouldReachMaxSizeIfSpecified(t *testing.T) {
	const Size = 10
	q := CreateQueue[someType](Size)

	for i := 0; i < Size; i++ {
		e := someType{}
		assert.False(t, q.IsFull())
		assert.True(t, q.PushBack(&e))
		assert.Equal(t, i+1, q.Size())
	}

	e := someType{}
	assert.True(t, q.IsFull())
	assert.False(t, q.PushBack(&e))
	assert.Equal(t, Size, q.Size())
}

func Test_PushBackWithoutMaxLimit(t *testing.T) {
	const Size = 10000
	q := CreateQueue[someType](0)

	for i := 0; i < Size; i++ {
		e := someType{}
		assert.True(t, q.PushBack(&e))
		assert.Equal(t, i+1, q.Size())
		assert.False(t, q.IsFull())
		assert.False(t, q.IsEmpty())
	}

	assert.Equal(t, Size, q.Size())
}

func Test_IsConsistentWithMultipleCorroutines(t *testing.T) {
	const Concurrency = 16
	const Size = 1000
	q := CreateQueue[someType](0)

	assert.True(t, q.IsEmpty())

	var wg sync.WaitGroup
	wg.Add(Concurrency)
	for it := 0; it < Concurrency; it++ {
		go func() {
			defer wg.Done()

			for i := 0; i < Size; i++ {
				e := someType{}
				assert.True(t, q.PushBack(&e))
				assert.False(t, q.IsFull())
				assert.False(t, q.IsEmpty())
			}
		}()
	}

	wg.Wait()
	assert.Equal(t, Concurrency*Size, q.Size())
}
