package kry

const (
	fullHistorySize = -1
)

type Options struct {
	historySize int
}

// WithHistory enables history tracking for the FSM with a specified size.
func WithHistory(size int) func(o *Options) *Options {
	return func(o *Options) *Options {
		o.historySize = size

		return o
	}
}

// WithFullHistory enables full history tracking for the FSM, so no size limit.
func WithFullHistory() func(o *Options) *Options {
	return func(o *Options) *Options {
		o.historySize = fullHistorySize

		return o
	}
}
