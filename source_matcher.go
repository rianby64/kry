package kry

type namedEntry[State comparable, Param any] struct {
	name    string
	handler TransitionHandler[State, Param]
}

// SourceMatcher describes a set of source states (exact or function-based) and
// the callbacks that fire when a transition originates from one of them.
// Construct one with From or FnFrom, then attach callbacks with .On.
type SourceMatcher[State comparable, Param any] struct {
	states  []State
	matchFn func(State) bool
	entries []namedEntry[State, Param]
}

// From returns a SourceMatcher that matches any of the listed exact states.
func From[State comparable, Param any](states ...State) SourceMatcher[State, Param] {
	return SourceMatcher[State, Param]{states: states}
}

// FnFrom returns a SourceMatcher that matches sources for which fn returns true.
func FnFrom[State comparable, Param any](fn func(State) bool) SourceMatcher[State, Param] {
	return SourceMatcher[State, Param]{matchFn: fn}
}

// On attaches a named callback to the source matcher and returns the updated matcher.
// Multiple .On calls on the same SourceMatcher register multiple handlers (only
// meaningful on Fn* paths; exact paths enforce at most one handler per (src, dst)).
func (sm SourceMatcher[State, Param]) On(name string, handler TransitionHandler[State, Param]) SourceMatcher[State, Param] {
	sm.entries = append(sm.entries, namedEntry[State, Param]{name: name, handler: handler})

	return sm
}
