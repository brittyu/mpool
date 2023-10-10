package mpool

import (
	"time"
)

type loopQueue struct {
	items  []worker
	expiry []worker
	head   int
	tail   int
	size   int
	isFull bool
}

func newWorkerLoopQueue(size int) *loopQueue {
	return &loopQueue{
		items: make([]worker, 0, size),
		size:  size,
	}
}

func (lq *loopQueue) len() int {
	if lq.size == 0 {
		return 0
	}
	return 0
}

func (lq *loopQueue) isEmpty() bool {
	return lq.head == lq.tail
}

func (lq *loopQueue) insert(w worker) error {
	if lq.size == 0 {
		return nil
	}

	if lq.isFull {
		return nil
	}

	lq.items[lq.tail] = w
	lq.tail++

	if lq.tail == lq.size {
		lq.tail = 0
	}

	if lq.tail == lq.head {
		lq.isFull = true
	}

	return nil
}

func (lq *loopQueue) detach() worker {
	if lq.isEmpty() {
		return nil
	}

	w := lq.items[lq.head]
	lq.items[lq.head] = nil
	lq.head++

	if lq.head == lq.size {
		lq.head = 0
	}

	lq.isFull = false
	return w
}

func (lq *loopQueue) refresh(duration time.Duration) []worker {
	expiryTime := time.Now().Add(-duration)
	index := lq.binarySearch(expiryTime)
	if index == -1 {
		return nil
	}

	lq.expiry = lq.expiry[:0]

	if lq.head <= index {
		lq.expiry = append(lq.expiry, lq.items[lq.head:index+1]...)
		for i := lq.head; i < index+1; i++ {
			lq.items[i] = nil
		}
	} else {
		lq.expiry = append(lq.expiry, lq.items[0:index+1]...)
		lq.expiry = append(lq.expiry, lq.items[lq.head:]...)
		for i := 0; i < index+1; i++ {
			lq.items[i] = nil
		}
		for i := lq.head; i < lq.size; i++ {
			lq.items[i] = nil
		}
	}
	head := (index + 1) % lq.size
	lq.head = head
	if len(lq.expiry) > 0 {
		lq.isFull = false
	}

	return lq.expiry
}

func (lq *loopQueue) binarySearch(expiryTime time.Time) int {
	var mid, nlen, basel, tmid int
	nlen = len(lq.items)

	if lq.isEmpty() || expiryTime.Before(lq.items[lq.head].lastUsedTime()) {
		return -1
	}

	r := (lq.tail - 1 - lq.head + nlen) % nlen
	basel = lq.head
	l := 0
	for l <= r {
		mid = l + ((r - l) >> 1)
		tmid = (mid + basel + nlen) % nlen
		if expiryTime.Before(lq.items[tmid].lastUsedTime()) {
			r = mid - 1
		} else {
			l = mid + 1
		}
	}

	return (r + basel + nlen) % nlen
}

func (lq *loopQueue) reset() {
	if lq.isEmpty() {
		return
	}

retry:
	if w := lq.detach(); w != nil {
		w.finish()
		goto retry
	}
	lq.items = lq.items[:0]
	lq.size = 0
	lq.head = 0
	lq.tail = 0
}
