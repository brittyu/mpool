package mpool

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type CommonPool struct {
	capacity int32

	// 只在开启动态调整时候有效
	dynamicMin int
	dynamicMax int

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

func (p *CommonPool) purgeStaleWorkers(ctx context.Context) {
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

func (p *CommonPool) IsClosed() bool {
	return atomic.LoadInt32(&p.state) == CLOSED
}

func (p *CommonPool) addRunning(delta int) {
	atomic.AddInt32(&p.running, int32(delta))
}

func (p *CommonPool) addWaiting(delta int) {
	atomic.AddInt32(&p.waiting, int32(delta))
}

func (p *CommonPool) Waiting() int {
	return int(atomic.LoadInt32(&p.waiting))
}

func (p *CommonPool) Running() int {
	return int(atomic.LoadInt32(&p.running))
}

func (p *CommonPool) Cap() int {
	return int(atomic.LoadInt32(&p.capacity))
}

func (p *CommonPool) Reboot() {
	if atomic.CompareAndSwapInt32(&p.state, CLOSED, OPENED) {
		atomic.StoreInt32(&p.purgeDone, 0)
		p.goPurge()
		atomic.StoreInt32(&p.ticktockDone, 0)
		p.goTicktock()
	}
}

func (p *CommonPool) ticktock(ctx context.Context) {
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
		}

		if p.IsClosed() {
			break
		}

		p.now.Store(time.Now())
	}
}

func (p *CommonPool) goPurge() {
	if p.options.DisablePurge {
		return
	}

	var ctx context.Context
	ctx, p.stopPurge = context.WithCancel(context.Background())
	go p.purgeStaleWorkers(ctx)
}

func (p *CommonPool) goTicktock() {
	p.now.Store(time.Now())
	var ctx context.Context
	ctx, p.stopTicktock = context.WithCancel(context.Background())
	go p.ticktock(ctx)
}

func (p *CommonPool) nowTime() time.Time {
	return p.now.Load().(time.Time)
}

func (p *CommonPool) Release() {
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

func (p *CommonPool) ReleaseTimeout(timeout time.Duration) error {
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
		time.Sleep(10 * time.Millisecond)
	}

	return ErrTimeout
}

func (p *CommonPool) Free() int {
	c := p.Cap()
	if c < 0 {
		return -1
	}

	return c - Running()
}
