package mpool

import (
	"context"
	"mpool/lib/slock"
	"sync"
	"sync/atomic"
	"time"
)

type PoolWithFunc struct {
	capacity int32

	// 只在开启动态调整时候有效
	minCapacity int32
	maxCapacity int32

	running int32

	lock sync.Locker

	workers workerQueue

	state int32

	cond *sync.Cond

	poolFunc func(interface{})

	workerCache sync.Pool

	waiting int32

	purgeDone int32
	stopPurge context.CancelFunc

	ticktockDone int32
	stopTicktock context.CancelFunc

	now atomic.Value

	options *Options
}

func NewPoolWithFunc(size int, pf func(interface{}), options ...Option) (*PoolWithFunc, error) {
	if size <= 0 {
		size = -1
	}

	if pf == nil {
		return nil, ErrLackPoolFunc
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

	p := &PoolWithFunc{
		capacity: int32(size),
		poolFunc: pf,
		lock:     slock.NewSpinLock(),
		options:  opts,
	}
	p.workerCache.New = func() interface{} {
		return &goWorkerWithFunc{
			pool: p,
			args: make(chan interface{}, workerChanCap),
		}
	}
	if p.options.PreAlloc && size <= 0 {
		return nil, ErrInvalidPreAllocSize
	} else {
		p.workers = newWorkerQueue(queueTypeStack, size)
	}

	p.cond = sync.NewCond(p.lock)

	p.goPurge()
	p.goTicktock()

	return p, nil
}

func (p *PoolWithFunc) purgeStaleWorkers(ctx context.Context) {
	ticker := time.NewTicker(p.options.ExpiryDuration)

	defer func() {
		ticker.Stop()
		atomic.StoreInt32(&p.purgeDone, 1)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		if p.IsClosed() {
			break
		}

		var isDormant bool
		p.lock.Lock()
		staleWorkers := p.workers.refresh(p.options.ExpiryDuration)
		n := p.Running()
		isDormant = n == 0 || n == len(staleWorkers)
		p.lock.Unlock()

		for i := range staleWorkers {
			staleWorkers[i].finish()
			staleWorkers[i] = nil
		}

		if isDormant && p.Waiting() > 0 {
			p.cond.Broadcast()
		}
	}
}

func (p *PoolWithFunc) ticktock(ctx context.Context) {
	ticker := time.NewTicker(nowTimeUpdateInterval)

	defer func() {
		ticker.Stop()
		atomic.StoreInt32(&p.ticktockDone, 1)
	}()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			break
		}

		p.now.Store(time.Now())
	}
}

func (p *PoolWithFunc) goPurge() {
	if p.options.DisablePurge {
		return
	}

	var ctx context.Context
	ctx, p.stopPurge = context.WithCancel(context.Background())
	go p.purgeStaleWorkers(ctx)
}

func (p *PoolWithFunc) goTicktock() {
	p.now.Store(time.Now())
	var ctx context.Context
	ctx, p.stopTicktock = context.WithCancel(context.Background())
	go p.ticktock(ctx)
}

func (p *PoolWithFunc) nowTime() time.Time {
	return p.now.Load().(time.Time)
}

func (p *PoolWithFunc) Invoke(args interface{}) error {
	if p.IsClosed() {
		return ErrPoolClosed
	}

	w, err := p.retrieveWorker()
	if w != nil {
		w.inputParam(args)
	}

	return err
}

func (p *PoolWithFunc) Running() int {
	return int(atomic.LoadInt32(&p.running))
}

func (p *PoolWithFunc) Free() int {
	c := p.Cap()
	if c < 0 {
		return -1
	}

	return c - Running()
}

func (p *PoolWithFunc) Waiting() int {
	return int(atomic.LoadInt32(&p.waiting))
}

func (p *PoolWithFunc) Cap() int {
	return int(atomic.LoadInt32(&p.capacity))
}

func (p *PoolWithFunc) Tune(size int) {
	capacity := p.Cap()
	if capacity == -1 || size <= 0 || size == capacity || p.options.PreAlloc {
		return
	}

	atomic.StoreInt32(&p.capacity, int32(size))

	if size > capacity {
		if size-capacity == 1 {
			p.cond.Signal()
			return
		}

		p.cond.Broadcast()
	}
}

func (p *PoolWithFunc) IsClosed() bool {
	return atomic.LoadInt32(&p.state) == CLOSED
}

func (p *PoolWithFunc) Release() {
	if !atomic.CompareAndSwapInt32(&p.state, OPENED, CLOSED) {
		return
	}

	if p.stopPurge != nil {
		p.stopPurge()
		p.stopPurge = nil
	}

	p.stopTicktock()
	p.stopTicktock = nil

	p.lock.Lock()
	p.workers.reset()
	p.lock.Unlock()

	p.cond.Broadcast()
}

func (p *PoolWithFunc) ReleaseTimeout(timeout time.Duration) error {
	if p.IsClosed() || (!p.options.DisablePurge && p.stopPurge == nil) || p.stopTicktock == nil {
		return ErrPoolClosed
	}

	p.Release()

	endTime := time.Now().Add(timeout)
	for time.Now().Before(endTime) {
		if p.Running() == 0 &&
			(p.options.DisablePurge || atomic.LoadInt32(&p.purgeDone) == 1) &&
			atomic.LoadInt32(&p.ticktockDone) == 1 {
			return nil
		}
	}

	return ErrTimeout
}

func (p *PoolWithFunc) Reboot() {
	if atomic.CompareAndSwapInt32(&p.state, CLOSED, OPENED) {
		atomic.StoreInt32(&p.purgeDone, 0)
		p.goPurge()
		atomic.StoreInt32(&p.ticktockDone, 0)
		p.goTicktock()
	}
}

func (p *PoolWithFunc) addRunning(delta int) {
	atomic.AddInt32(&p.running, int32(delta))
}

func (p *PoolWithFunc) addWaiting(delta int) {
	atomic.AddInt32(&p.waiting, int32(delta))
}

func (p *PoolWithFunc) retrieveWorker() (w worker, err error) {
	p.lock.Lock()
retry:
	if w = p.workers.detach(); w != nil {
		p.lock.Unlock()
		return
	}

	if capacity := p.Cap(); capacity == -1 || capacity > p.Running() {
		p.lock.Unlock()
		w = p.workerCache.Get().(*goWorkerWithFunc)
		w.run()
		return
	}

	if p.options.NonBlocking || (p.options.MaxBlockingTasks != 0 && p.Waiting() >= p.options.MaxBlockingTasks) {
		p.lock.Unlock()
		return nil, ErrPoolOverload
	}

	p.addWaiting(1)
	p.cond.Wait()
	p.addWaiting(-1)

	if p.IsClosed() {
		p.lock.Unlock()
		return nil, ErrPoolClosed
	}

	goto retry
}

func (p *PoolWithFunc) revertWorker(worker *goWorkerWithFunc) bool {
	if capacity := p.Cap(); (capacity > 0 && p.Running() > capacity) || p.IsClosed() {
		p.cond.Broadcast()
		return false
	}

	worker.lastUsed = p.nowTime()

	p.lock.Lock()

	if p.IsClosed() {
		p.lock.Unlock()
		return false
	}

	if err := p.workers.insert(worker); err != nil {
		p.lock.Unlock()
		return false
	}

	p.cond.Signal()
	p.lock.Unlock()

	return true
}
