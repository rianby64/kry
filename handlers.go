package kry

import "context"

const (
	arityNoParams = 0
	arityWith     = 1
	arityVariadic = 2
)

// TransitionHandler holds exactly one callback and the arity it was declared for.
// Construct via OnEnter, OnEnterWith, or OnEnterVariadic — never directly.
type TransitionHandler[State comparable, Param any] struct {
	arity     int
	noParams  func(ctx context.Context, instance InstanceFSM[State, Param]) error
	with      func(ctx context.Context, instance InstanceFSM[State, Param], param Param) error
	variadic  func(ctx context.Context, instance InstanceFSM[State, Param], param ...Param) error
}

// OnEnter declares a callback that fires when Apply is called with zero params.
func OnEnter[State comparable, Param any](
	fn func(ctx context.Context, instance InstanceFSM[State, Param]) error,
) TransitionHandler[State, Param] {
	return TransitionHandler[State, Param]{
		arity:    arityNoParams,
		noParams: fn,
	}
}

// OnEnterWith declares a callback that fires when Apply is called with exactly one param.
func OnEnterWith[State comparable, Param any](
	fn func(ctx context.Context, instance InstanceFSM[State, Param], param Param) error,
) TransitionHandler[State, Param] {
	return TransitionHandler[State, Param]{
		arity: arityWith,
		with:  fn,
	}
}

// OnEnterVariadic declares a callback that fires when Apply is called with two or more params.
func OnEnterVariadic[State comparable, Param any](
	fn func(ctx context.Context, instance InstanceFSM[State, Param], param ...Param) error,
) TransitionHandler[State, Param] {
	return TransitionHandler[State, Param]{
		arity:    arityVariadic,
		variadic: fn,
	}
}
