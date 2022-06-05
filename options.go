package fastnet

import "time"

/**
* golang-sample源代码，版权归锦翰科技（深圳）有限公司所有。
* <p>
* 文件名称 : options
* 文件路径 :
* 作者 : DavidLiu
× Email: david.liu@ginghan.com
*
* 创建日期 : 2022/6/4 20:48
* 修改历史 : 1. [2022/6/4 20:48] 创建文件 by LongYong
*/

// Option represents the optional function.
type Option func(opts *Options)

// Options contains all options which will be applied when instantiating an ants pool.
type Options struct {
	// MaxIdleWorkerDuration is a period for the scavenger goroutine to clean up those expired workers,
	// the scavenger scans all workers every `MaxIdleWorkerDuration` and clean up those workers that haven't been
	// used for more than `ExpiryDuration`.
	MaxIdleWorkerDuration time.Duration

	// Max number of goroutine blocking on pool.Submit.
	// 0 (default value) means no such limit.
	MaxWorkersCount uint32

	MaxPackageFrameSize uint

	LogAllErrors bool
}

func LoadOptions(options ...Option) *Options {
	opts := new(Options)
	for _, option := range options {
		option(opts)
	}
	return opts
}

// WithOptions accepts the whole Options config.
func WithOptions(options Options) Option {
	return func(opts *Options) {
		*opts = options
	}
}

// WithMaxPackageFrameSize
func WithMaxPackageFrameSize(maxPackageFrameSize uint) Option {
	return func(opts *Options) {
		opts.MaxPackageFrameSize = maxPackageFrameSize
	}
}

// WithMaxIdleWorkerDuration sets up the interval time of cleaning up goroutines.
func WithMaxIdleWorkerDuration(maxIdleWorkerDuration time.Duration) Option {
	return func(opts *Options) {
		opts.MaxIdleWorkerDuration = maxIdleWorkerDuration
	}
}

// WithLogAllErrors indicates whether it should malloc for workers.
func WithLogAllErrors(logAllErrors bool) Option {
	return func(opts *Options) {
		opts.LogAllErrors = logAllErrors
	}
}

// WithMaxWorkersCount sets up the maximum number of goroutines that are blocked when it reaches the capacity of pool.
func WithMaxWorkersCount(maxWorkersCount uint32) Option {
	return func(opts *Options) {
		opts.MaxWorkersCount = maxWorkersCount
	}
}
