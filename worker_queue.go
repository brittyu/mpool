package mpool

import (
	"time"
)

type queueType int

type workerQueue interface {
	len() int
	isEmpty() bool
	insert(worker) error
	detach() worker
	refresh(duration time.Duration) []worker
	reset()
}

func newWorkerQueue(size int) workerQueue {
	return newWorkerStack(size)
}
