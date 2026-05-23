package kry

// Builder constructs an FSM using a fluent API. Use Build to create one,
// chain GoingTo / FnGoingTo calls to declare transitions, attach options,
// then call Create to produce the FSM.
type Builder[State comparable, Param any] struct {
	initialState State
	transitions  []Transition[State, Param]
	options      []func(o *Options[Param]) *Options[Param]
}

// Build starts a new Builder for an FSM with the given initial state.
func Build[State comparable, Param any](initialState State) *Builder[State, Param] {
	return &Builder[State, Param]{initialState: initialState}
}

// GoingTo declares one or more transitions to an exact destination state.
// Each SourceMatcher in sources describes a set of source states and the
// callback that fires when the transition is taken from one of those sources.
func (b *Builder[State, Param]) GoingTo(dst State, sources ...SourceMatcher[State, Param]) *Builder[State, Param] {
	for _, src := range sources {
		for _, entry := range src.entries {
			b.transitions = append(b.transitions, Transition[State, Param]{
				Name:  entry.name,
				Src:   src.states,
				SrcFn: src.matchFn,
				Dst:   dst,
				Enter: entry.handler,
			})
		}
	}
	return b
}

// FnGoingTo declares one or more transitions to a function-matched destination.
// dstFn receives the requested destination state and returns true when it matches.
func (b *Builder[State, Param]) FnGoingTo(dstFn func(State) bool, sources ...SourceMatcher[State, Param]) *Builder[State, Param] {
	for _, src := range sources {
		for _, entry := range src.entries {
			b.transitions = append(b.transitions, Transition[State, Param]{
				Name:  entry.name,
				Src:   src.states,
				SrcFn: src.matchFn,
				DstFn: dstFn,
				Enter: entry.handler,
			})
		}
	}
	return b
}

// WithFullHistory enables full (unbounded) history tracking on the FSM.
func (b *Builder[State, Param]) WithFullHistory() *Builder[State, Param] {
	b.options = append(b.options, WithFullHistory[Param]())
	return b
}

// WithHistory enables history tracking with the given maximum number of entries.
func (b *Builder[State, Param]) WithHistory(size int) *Builder[State, Param] {
	b.options = append(b.options, WithHistory[Param](size))
	return b
}

// WithEnabledStackTrace enables stack trace capture in each history item.
func (b *Builder[State, Param]) WithEnabledStackTrace() *Builder[State, Param] {
	b.options = append(b.options, WithEnabledStackTrace[Param]())
	return b
}

// WithPanicHandler sets a custom panic handler for the FSM.
func (b *Builder[State, Param]) WithPanicHandler(handler PanicHandler) *Builder[State, Param] {
	b.options = append(b.options, WithPanicHandler[Param](handler))
	return b
}

// WithCloneHandler sets a custom parameter clone handler for the FSM.
func (b *Builder[State, Param]) WithCloneHandler(handler CloneHandler[Param]) *Builder[State, Param] {
	b.options = append(b.options, WithCloneHandler(handler))
	return b
}

// Create builds and returns the FSM, returning any construction error.
func (b *Builder[State, Param]) Create() (*FSM[State, Param], error) {
	return New(b.initialState, b.transitions, b.options...)
}
