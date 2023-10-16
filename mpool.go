package mpool

import "time"

type Logger interface {
	Printf(format string, args ...interface{})
}

func Submit(task func()) error {
	return defaultPool.Submit(task)
}

func Running() int {
	return defaultPool.Running()
}

func Cap() int {
	return defaultPool.Cap()
}

func Free() int {
	return defaultPool.Free()
}

func Release() {
	defaultPool.Release()
}

func ReleaseTimeout(timeout time.Duration) error {
	return defaultPool.ReleaseTimeout(timeout)
}

func Reboot() {
	defaultPool.Reboot()
}
