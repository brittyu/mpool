package mpool

import "time"

type workerQueue interface {
	len() int
	isEmpty() bool
	insert(worker)
	detach() worker
	refresh(duration time.Duration) []worker
	reset()
}
