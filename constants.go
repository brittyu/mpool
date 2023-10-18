package mpool

import (
	"errors"
	"log"
	"math"
	"os"
	"runtime"
	"time"
)

const (
	DefaultPoolSize          = math.MaxInt32
	DefaultCleanIntervalTime = time.Second
)

const (
	OPENED = iota
	CLOSED
)

const nowTimeUpdateInterval = 500 * time.Millisecond

var (
	ErrLackPoolFunc      = errors.New("must provide function for pool")
	ErrInvalidPoolExpiry = errors.New("invalid expiry for pool")
	ErrPoolClosed        = errors.New("pool has been closed")
	ErrPoolOverload      = errors.New("too many goroutines blocked")
	ErrInvalidSize       = errors.New("can not set up a negative capacity")
	ErrTimeout           = errors.New("operation timeout")

	workerChanCap = func() int {
		if runtime.GOMAXPROCS(0) == 1 {
			return 0
		}

		return 1
	}()

	logLmsgprefix = 64
	defaultLogger = Logger(log.New(os.Stderr, "[mpool]: ", log.LstdFlags|logLmsgprefix|log.Lmicroseconds))

	defaultPool, _ = NewPool(DefaultPoolSize)

	defaultDynamicMin = 10
	defaultDynamicMax = 100
)
