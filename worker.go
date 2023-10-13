package mpool

import "time"

type worker interface {
	run()
	finish()
	lastUsedTime() time.Time
	inputFunc(func())
	inputParam(interface{})
}

type goWorker struct {
	pool     *Pool
	task     chan func()
	lastUsed time.Time
}

func (w *goWorker) run() {

}

func (w *goWorker) finish() {
	w.task <- nil
}

func (w *goWorker) lastUsedTime() time.Time {
	return w.lastUsed
}

func (w *goWorker) inputFunc(fn func()) {
	w.task <- fn
}

func (w *goWorker) inputParam(interface{}) {
	panic("xxxx")
}
