package mpool

import (
	"fmt"
	"runtime/debug"
	"time"
)

type goWorkerWithFunc struct {
	pool *PoolWithFunc

	args chan interface{}

	lastUsed time.Time
}

func (w *goWorkerWithFunc) run() {
	w.pool.addRunning(1)

	go func() {
		defer func() {
			fmt.Println("log")
			w.pool.addRunning(-1)
			w.pool.workerCache.Put(w)

			if p := recover(); p != nil {
				if ph := w.pool.options.PanicHandler; ph != nil {
					ph(p)
				} else {
					w.pool.options.Logger.Printf("worker exit from panic: %v\n%s\n", p, debug.Stack())
				}
			}
			fmt.Println("signal")
			w.pool.cond.Signal()
		}()

		for args := range w.args {
			if args == nil {
				return
			}

			w.pool.poolFunc(args)

			if ok := w.pool.revertWorker(w); !ok {
				return
			}
		}
	}()
}

func (w *goWorkerWithFunc) finish() {
	w.args <- nil
}

func (w *goWorkerWithFunc) lastUsedTime() time.Time {
	return w.lastUsed
}

func (w *goWorkerWithFunc) inputFunc(func()) {
	panic("unreachable")
}

func (w *goWorkerWithFunc) inputParam(arg interface{}) {
	w.args <- arg
}
