package kry

import "context"

type arity int

const (
	_ arity = iota // zero value means no callback
	arityWith
	arityVariadic
)

// TransitionHandler holds exactly one callback and the arity it was declared for.
// Construct via OnEnter or OnEnterVariadic — never directly.
type TransitionHandler[State comparable, Param any] struct {
	arity    arity
	with     func(ctx context.Context, instance InstanceFSM[State, Param], param Param) error
	variadic func(ctx context.Context, instance InstanceFSM[State, Param], param ...Param) error
}

// OnEnter declares a callback that fires when Apply is called with exactly one param.
func OnEnter[State comparable, Param any](
	fn func(ctx context.Context, instance InstanceFSM[State, Param], param Param) error,
) TransitionHandler[State, Param] {
	return TransitionHandler[State, Param]{
		arity: arityWith,
		with:  fn,
	}
}

// OnEnterVariadic declares a callback that fires when Apply is called with any number of params.
func OnEnterVariadic[State comparable, Param any](
	fn func(ctx context.Context, instance InstanceFSM[State, Param], param ...Param) error,
) TransitionHandler[State, Param] {
	return TransitionHandler[State, Param]{
		arity:    arityVariadic,
		variadic: fn,
	}
}

// NoOp declares a transition with no callback. Apply succeeds regardless of param count.
func NoOp[State comparable, Param any]() TransitionHandler[State, Param] {
	return TransitionHandler[State, Param]{}
}
