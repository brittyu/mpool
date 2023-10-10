package mpool

import (
	"context"
	"sync"
	"sync/atomic"
)

type Pool struct {
	capacity int32

	// 只在开启动态调整时候有效
	minCapacity int32
	maxCapacity int32

	running int32

	lock sync.Locker

	workers workerQueue

	state int32

	cond *sync.Cond

	workerCache sync.Pool

	waiting int32

	purgeDone int32
	stopPurge context.CancelFunc

	ticktockDone int32
	stopTickTock context.CancelFunc

	now atomic.Value

	options *Options
}
