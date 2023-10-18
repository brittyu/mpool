package mpool

import (
	"mpool/lib/slock"
	"sync"
	"sync/atomic"
)

type Pool struct {
	*CommonPool
}

func NewPool(size int, options ...Option) (*Pool, error) {
	if size <= 0 {
		size = -1
	}

	opts := loadOptions(options...)

	if !opts.DisablePurge {
		if expiry := opts.ExpiryDuration; expiry < 0 {
			return nil, ErrInvalidPoolExpiry
		} else if expiry == 0 {
			opts.ExpiryDuration = DefaultCleanIntervalTime
		}
	}

	if opts.Logger == nil {
		opts.Logger = defaultLogger
	}

	if opts.Dynamic && (opts.DynamicMin == 0 || opts.DynamicMax == 0) {
		opts.DynamicMin = defaultDynamicMin
		opts.DynamicMax = defaultDynamicMax
	}

	commonPool := &CommonPool{
		capacity: int32(size),
		lock:     slock.NewSpinLock(),
		options:  opts,
	}

	p := &Pool{
		commonPool,
	}

	if opts.Dynamic {
		p.dynamicMin = opts.DynamicMin
		p.dynamicMax = opts.DynamicMax
	}

	p.workerCache.New = func() interface{} {
		return &goWorker{
			pool: p,
			task: make(chan func(), workerChanCap),
		}
	}

	if size <= 0 {
		return nil, ErrInvalidSize
	} else {
		p.workers = newWorkerStack(size)
	}

	p.cond = sync.NewCond(p.lock)

	p.goPurge()
	p.goTicktock()

	return p, nil

}

func (p *Pool) retrieveWorker() (w worker, err error) {
	p.lock.Lock()
	defer p.lock.Unlock()

retry:
	if w = p.workers.detach(); w != nil {
		return
	}

	if capacity := p.Cap(); capacity > p.Running() || (p.options.Dynamic && capacity < p.dynamicMax) {
		w = p.workerCache.Get().(*goWorker)

		if capacity <= p.Running() && p.options.Dynamic && capacity < p.dynamicMax {
			atomic.StoreInt32(&p.capacity, int32(capacity)+1)
		}

		w.run()
		return
	}

	// 如果是堵塞 或者 等待队列大于等于设置的最大数值则判为过载
	if p.options.NonBlocking || (p.options.MaxBlockingTasks != 0 && p.Waiting() >= p.options.MaxBlockingTasks) {
		return nil, ErrPoolOverload
	}

	p.addWaiting(1)
	p.cond.Wait()
	p.addWaiting(-1)

	if p.IsClosed() {
		return nil, ErrPoolClosed
	}

	goto retry
}

// 把worker添加到池中
func (p *Pool) revertWorker(worker *goWorker) bool {
	if capacity := p.Cap(); (p.Running() < capacity && p.options.Dynamic && capacity > p.dynamicMin) || p.IsClosed() {
		p.cond.Broadcast()
		atomic.StoreInt32(&p.capacity, int32(capacity)-1)
		return false
	}

	worker.lastUsed = p.nowTime()

	p.lock.Lock()
	defer p.lock.Unlock()

	if p.IsClosed() {
		return false
	}

	if err := p.workers.insert(worker); err != nil {
		return false
	}

	p.cond.Signal()

	return true
}

func (p *Pool) Submit(task func()) error {
	if p.IsClosed() {
		return ErrPoolClosed
	}

	w, err := p.retrieveWorker()
	if w != nil {
		w.inputFunc(task)
	}

	return err
}
