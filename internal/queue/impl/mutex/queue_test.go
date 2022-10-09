package mutex

import (
	"sync"
	"testing"
	"time"

	"github.com/Philanthropists/toshl-email-autosync/internal/queue"
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

func Test_IfPopElementSizeShouldDecrease(t *testing.T) {
	const Insertions = 10
	const Deletions = 7
	q := CreateQueue[someType](0)

	assert.True(t, q.IsEmpty())

	for i := 0; i < Insertions; i++ {
		e := someType{}
		assert.True(t, q.PushBack(&e))

		assert.False(t, q.IsFull())
		assert.False(t, q.IsEmpty())
	}

	for i := 0; i < Deletions; i++ {
		e, err := q.Pop()
		assert.NoError(t, err)
		assert.NotNil(t, e)

		assert.False(t, q.IsFull())
		assert.False(t, q.IsEmpty())
	}

	assert.Equal(t, Insertions-Deletions, q.Size())
	assert.False(t, q.IsFull())
	assert.False(t, q.IsEmpty())
}

func Test_IfPopOnEmptyShouldGiveError(t *testing.T) {
	q := CreateQueue[someType](0)

	assert.True(t, q.IsEmpty())

	e, err := q.Pop()

	assert.NotNil(t, err)
	assert.Nil(t, e)
}

func Test_ConcurrentInsertionsAndDeletionsShouldBeConsistent(t *testing.T) {
	const Concurrency = 7
	const Insertions = 100
	const Deletions = 99

	q := CreateQueue[someType](0)

	assert.True(t, q.IsEmpty())

	var wg sync.WaitGroup
	wg.Add(2 * Concurrency)

	for it := 0; it < Concurrency; it++ {
		go func(q queue.FIFOQueue[someType]) {
			defer wg.Done()

			for i := 0; i < Insertions; i++ {
				e := someType{}
				assert.True(t, q.PushBack(&e))
				assert.False(t, q.IsFull())
			}
		}(q)
	}

	<-time.After(1 * time.Millisecond)

	for it := 0; it < Concurrency; it++ {
		go func(q queue.FIFOQueue[someType]) {
			defer wg.Done()

			for i := 0; i < Deletions; i++ {
				e, err := q.Pop()

				assert.Nil(t, err)
				assert.NotNil(t, e)
			}
		}(q)
	}

	wg.Wait()
	assert.Equal(t, Concurrency*(Insertions-Deletions), q.Size())
}
