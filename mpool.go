package mpool

type Logger interface {
	Printf(format string, args ...interface{})
}
