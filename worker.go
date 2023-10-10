package mpool

import "time"

type worker interface {
	run()
	finish()
	lastUsedTime() time.Time
	inputFunc(func())
	intputParam(interface{})
}
