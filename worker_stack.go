package mpool

type workerStack struct {
	items  []worker
	expiry []worker
}
