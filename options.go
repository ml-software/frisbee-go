package frisbee

import (
	"github.com/rs/zerolog"
	"os"
	"time"
)

type Option func(opts *Options)

var DefaultLogger = zerolog.New(os.Stdout)

type Options struct {
	KeepAlive time.Duration
	Logger    *zerolog.Logger
	Heartbeat bool
}

func loadOptions(options ...Option) *Options {
	opts := new(Options)
	for _, option := range options {
		option(opts)
	}

	if opts.Logger == nil {
		opts.Logger = &DefaultLogger
	}

	if opts.KeepAlive == 0 {
		opts.KeepAlive = time.Minute * 3
	}

	opts.Heartbeat = !opts.Heartbeat

	return opts
}

func WithOptions(options Options) Option {
	return func(opts *Options) {
		*opts = options
	}
}

func WithKeepAlive(keepAlive time.Duration) Option {
	return func(opts *Options) {
		opts.KeepAlive = keepAlive
	}
}

func WithLogger(logger *zerolog.Logger) Option {
	return func(opts *Options) {
		opts.Logger = logger
	}
}

func WithHeartbeat(heartbeat bool) Option {
	return func(opts *Options) {
		opts.Heartbeat = !heartbeat
	}
}
