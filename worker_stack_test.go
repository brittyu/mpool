package mpool

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewWorkerStack(t *testing.T) {
	size := 100
	q := newWorkerStack(size)
	assert.EqualValues(t, 0, q.len(), "len error")
	assert.Equal(t, true, q.isEmpty(), "empty error")
	assert.Nil(t, q.detach(), "detach error")
}

func TestWorkerStack(t *testing.T) {
	q := newWorkerQueue(0)

	for i := 0; i < 5; i++ {
		err := q.insert(&goWorker{lastUsed: time.Now()})
		if err != nil {
			break
		}
	}

	assert.EqualValues(t, 5, q.len(), "len error")

	expired := time.Now()

	err := q.insert(&goWorker{lastUsed: expired})
	if err != nil {
		t.Fatal("insert error")
	}

	time.Sleep(time.Second)

	for i := 0; i < 6; i++ {
		err := q.insert(&goWorker{lastUsed: time.Now()})

		if err != nil {
			t.Fatal("insert error")
		}
	}

	assert.EqualValues(t, 12, q.len(), "len error")

	q.refresh(time.Second)

	assert.EqualValues(t, 6, q.len(), "len error")
}

func TestSearch(t *testing.T) {
	q := newWorkerStack(0)

	expiry := time.Now()

	_ = q.insert(&goWorker{lastUsed: time.Now()})

	assert.EqualValues(t, 0, q.binarySearch(0, q.len()-1, time.Now()), "index 0")
	assert.EqualValues(t, -1, q.binarySearch(0, q.len()-1, expiry), "index -1")

	expiry2 := time.Now()
	_ = q.insert(&goWorker{lastUsed: time.Now()})

	assert.EqualValues(t, -1, q.binarySearch(0, q.len()-1, expiry), "index -1")
	assert.EqualValues(t, 0, q.binarySearch(0, q.len()-1, expiry2), "index 0")
	assert.EqualValues(t, 1, q.binarySearch(0, q.len()-1, time.Now()), "index 1")

	for i := 0; i < 5; i++ {
		_ = q.insert(&goWorker{lastUsed: time.Now()})
	}

	expiry3 := time.Now()
	_ = q.insert(&goWorker{lastUsed: expiry3})

	for i := 0; i < 10; i++ {
		_ = q.insert(&goWorker{lastUsed: time.Now()})
	}
	assert.EqualValues(t, 7, q.binarySearch(0, q.len()-1, expiry3), "index 7")
}
