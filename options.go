package kry

import "context"

const (
	fullHistorySize = -1
)

type Options[Param any] struct {
	historySize  int
	stackTrace   bool
	panicHandler PanicHandler
	cloneHandler CloneHandler[Param]
}

// WithHistory enables history tracking for the FSM with a specified size.
func WithHistory[Param any](size int) func(o *Options[Param]) *Options[Param] {
	return func(o *Options[Param]) *Options[Param] {
		o.historySize = size

		return o
	}
}

// WithFullHistory enables full history tracking for the FSM, so no size limit.
func WithFullHistory[Param any]() func(o *Options[Param]) *Options[Param] {
	return func(o *Options[Param]) *Options[Param] {
		o.historySize = fullHistorySize

		return o
	}
}

// WithEnabledStackTrace enables stack trace capturing for each history item.
func WithEnabledStackTrace[Param any]() func(o *Options[Param]) *Options[Param] {
	return func(o *Options[Param]) *Options[Param] {
		o.stackTrace = true

		return o
	}
}

type PanicHandler = func(ctx context.Context, panicReason any)

// WithPanicHandler sets a custom panic handler for the FSM.
func WithPanicHandler[Param any](handler PanicHandler) func(o *Options[Param]) *Options[Param] {
	return func(o *Options[Param]) *Options[Param] {
		o.panicHandler = handler

		return o
	}
}

type CloneHandler[Param any] = func(params ...Param) ([]Param, error)

// WithCloneHandler sets a custom clone handler for the FSM.
//
// As example, could be used to deep copy parameters if they are complex types.
//
//	import "github.com/fxamacker/cbor/v2"
//	func cloneHandler[Param any](params ...Param) ([]Param, error) {
//		if len(params) == 0 {
//			return params, nil
//		}
//
//		data, err := cbor.Marshal(params)
//		if err != nil {
//			return nil, fmt.Errorf("failed to marshal params: %w", err)
//		}
//
//		var cloned []Param
//
//		if err := cbor.Unmarshal(data, &cloned); err != nil {
//			return nil, fmt.Errorf("failed to unmarshal params: %w", err)
//		}
//
//		return cloned, nil
//	}
func WithCloneHandler[Param any](handler CloneHandler[Param]) func(o *Options[Param]) *Options[Param] {
	return func(o *Options[Param]) *Options[Param] {
		o.cloneHandler = handler

		return o
	}
}

// ExpectEnter accepts the handler that is expected to be called when applying the transition.
//
// During apply, in the history, the transition will reflect if the expected handler was called or not.
func ExpectEnter[Action comparable, State comparable, Param any](
	handler handler[Action, State, Param],
) func(fsk InstanceFSM[Action, State, Param]) InstanceFSM[Action, State, Param] {
	return func(fsk InstanceFSM[Action, State, Param]) InstanceFSM[Action, State, Param] {
		fsm, ok := fsk.(*FSM[Action, State, Param])
		if !ok {
			panic("unable to cast FSM instance in ExpectEnter")
		}

		if fsm.decoratorApply == nil {
			fsm.decoratorApply = &decoratorApply[Action, State, Param]{}
		}

		fsm.decoratorApply.expectToCallEnter = append(fsm.decoratorApply.expectToCallEnter, handler)

		return fsk
	}
}

// ExpectEnterNoParams accepts the handler that is expected to be called when applying the transition.
//
// During apply, in the history, the transition will reflect if the expected handler was called or not.
func ExpectEnterNoParams[Action comparable, State comparable, Param any](
	handler handlerNoParams[Action, State, Param],
) func(fsk InstanceFSM[Action, State, Param]) InstanceFSM[Action, State, Param] {
	return func(fsk InstanceFSM[Action, State, Param]) InstanceFSM[Action, State, Param] {
		fsm, ok := fsk.(*FSM[Action, State, Param])
		if !ok {
			panic("unable to cast FSM instance in ExpectEnterNoParams")
		}

		if fsm.decoratorApply == nil {
			fsm.decoratorApply = &decoratorApply[Action, State, Param]{}
		}

		fsm.decoratorApply.expectToCallEnterNoParams = append(fsm.decoratorApply.expectToCallEnterNoParams, handler)

		return fsk
	}
}

// ExpectEnterVariadic accepts the handler that is expected to be called when applying the transition.
//
// During apply, in the history, the transition will reflect if the expected handler was called or not.
func ExpectEnterVariadic[Action comparable, State comparable, Param any](
	handler handlerVariadic[Action, State, Param],
) func(fsk InstanceFSM[Action, State, Param]) InstanceFSM[Action, State, Param] {
	return func(fsk InstanceFSM[Action, State, Param]) InstanceFSM[Action, State, Param] {
		fsm, ok := fsk.(*FSM[Action, State, Param])
		if !ok {
			panic("unable to cast FSM instance in ExpectEnterVariadic")
		}

		if fsm.decoratorApply == nil {
			fsm.decoratorApply = &decoratorApply[Action, State, Param]{}
		}

		fsm.decoratorApply.expectToCallEnterVariadic = append(fsm.decoratorApply.expectToCallEnterVariadic, handler)

		return fsk
	}
}

func (fsk *FSM[Action, State, Param]) With(opts ...func(fsk InstanceFSM[Action, State, Param]) InstanceFSM[Action, State, Param]) InstanceFSM[Action, State, Param] {
	for _, o := range opts {
		fsm, ok := o(fsk).(*FSM[Action, State, Param])
		if !ok {
			return nil
		}

		fsk = fsm
	}

	return fsk
}
