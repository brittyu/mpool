package mpool

import "time"

type Options struct {
	ExpiryDuration time.Duration

	PreAlloc bool

	MaxBlockingTasks int

	NonBlocking bool

	PanicHandler func(interface{})

	Logger Logger

	DisablePurge bool

	Dynamic bool
}
