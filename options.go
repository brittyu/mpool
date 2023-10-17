package mpool

import (
	"time"
)

type Options struct {
	ExpiryDuration time.Duration

	MaxBlockingTasks int

	NonBlocking bool

	PanicHandler func(interface{})

	Logger Logger

	DisablePurge bool

	Dynamic    bool
	DynamicMin int
	DynamicMax int
}

type Option func(opts *Options)

func loadOptions(options ...Option) *Options {
	opts := new(Options)

	for _, o := range options {
		o(opts)
	}
	return opts
}

func WithOptions(options Options) Option {
	return func(opts *Options) {
		*opts = options
	}
}

func WithExpiryDuraction(expiryDuration time.Duration) Option {
	return func(opts *Options) {
		opts.ExpiryDuration = expiryDuration
	}
}

func WithMaxBlockingTasks(maxBlockingTasks int) Option {
	return func(opts *Options) {
		opts.MaxBlockingTasks = maxBlockingTasks
	}
}

func WithNonBlocking(nonBlocking bool) Option {
	return func(opts *Options) {
		opts.NonBlocking = nonBlocking
	}
}

func WithPanicHandler(panicHandler func(interface{})) Option {
	return func(opts *Options) {
		opts.PanicHandler = panicHandler
	}
}

func WithLogger(logger Logger) Option {
	return func(opts *Options) {
		opts.Logger = logger
	}
}

func WithDisablePurge(disable bool) Option {
	return func(opts *Options) {
		opts.DisablePurge = disable
	}
}

func WithDynamic(dynamic bool) Option {
	return func(opts *Options) {
		opts.Dynamic = dynamic
	}
}

func WithDynamicMin(dynamicMin int) Option {
	return func(opts *Options) {
		opts.DynamicMin = dynamicMin
	}
}

func WithDynamicMax(dynamicMax int) Option {
	return func(opts *Options) {
		opts.DynamicMax = dynamicMax
	}
}
