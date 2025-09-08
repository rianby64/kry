package kry

import (
	"context"
	"fmt"
)

type errString string

func (e errString) Error() string {
	return string(e)
}

const (
	ErrUnknown  errString = "unknown"
	ErrNotFound errString = "not found"
)

type InstanceFSM[E, S, P comparable] interface {
	Current() S

	// Event(ctx context.Context, event E, param ...P) error // TODO: ask why this method should be here. If YES, then I've to deal with infinity loops

	ForceState(state S) error
}

type eventTransitionState[E, S, P comparable] struct {
	Src []S
	Dst S

	EnterNoParams func(ctx context.Context, instance InstanceFSM[E, S, P]) error
	Enter         func(ctx context.Context, instance InstanceFSM[E, S, P], param P) error
	EnterVariadic func(ctx context.Context, instance InstanceFSM[E, S, P], param ...P) error
}

type callbacks[E, S, P comparable] struct {
	EnterNoParams func(ctx context.Context, instance InstanceFSM[E, S, P]) error
	Enter         func(ctx context.Context, instance InstanceFSM[E, S, P], param P) error
	EnterVariadic func(ctx context.Context, instance InstanceFSM[E, S, P], param ...P) error
}

// Action describes a transition. It contains the name of the action, the source states,
// the destination state, and optional callbacks that are executed when the action is triggered.
//
//	// Implementation notes:
//	EnterNoParams(...)             // if you want to execute a callback without parameters.
//	Enter(..., param P)            // if you want to execute a callback with one parameter.
//	EnterVariadic(..., param ...P) // if you want to execute a callback with multiple parameters.
type Action[E, S, P comparable] struct {
	Name E
	Src  []S
	Dst  S

	EnterNoParams func(ctx context.Context, instance InstanceFSM[E, S, P]) error
	Enter         func(ctx context.Context, instance InstanceFSM[E, S, P], param P) error
	EnterVariadic func(ctx context.Context, instance InstanceFSM[E, S, P], param ...P) error
}

type FSM[E, S, P comparable] struct {
	currentState S
	states       map[S]struct{}
	path         map[E]map[S]map[S]callbacks[E, S, P]
}

func New[E, S, P comparable](initialState S, events []Action[E, S, P]) *FSM[E, S, P] {
	path := make(map[E]map[S]map[S]callbacks[E, S, P])
	states := map[S]struct{}{
		initialState: {},
	}

	for _, event := range events {
		if _, ok := path[event.Name]; !ok {
			path[event.Name] = make(map[S]map[S]callbacks[E, S, P])
		}

		for _, srtState := range event.Src {
			if _, ok := path[event.Name][srtState]; !ok {
				path[event.Name][srtState] = make(map[S]callbacks[E, S, P])
			}

			if _, ok := path[event.Name][srtState][event.Dst]; ok {
				panic(fmt.Sprintf("event %v from state %v to state %v already exists", event.Name, srtState, event.Dst))
			}

			path[event.Name][srtState][event.Dst] = callbacks[E, S, P]{
				EnterVariadic: event.EnterVariadic,
				Enter:         event.Enter,
				EnterNoParams: event.EnterNoParams,
			}
		}

		for _, state := range event.Src {
			states[state] = struct{}{}
		}

		states[event.Dst] = struct{}{}
	}

	f := &FSM[E, S, P]{
		currentState: initialState,
		path:         path,
		states:       states,
	}

	return f
}

func (fsm *FSM[E, S, P]) Current() S {
	return fsm.currentState
}

func (fsm *FSM[E, S, P]) ForceState(state S) error {
	if _, ok := fsm.states[state]; !ok {
		return fmt.Errorf("state %w: %v", ErrUnknown, state)
	}

	fsm.currentState = state

	return nil
}

func (fsm *FSM[E, S, P]) Apply(ctx context.Context, action E, newState S, param ...P) error {
	currentState := fsm.currentState
	foundEvent, ok := fsm.path[action]
	if !ok {
		return fmt.Errorf("event %w: %v", ErrUnknown, action)
	}

	foundSrcState, ok := foundEvent[currentState]
	if ok {
		callbacks, ok := foundSrcState[newState]
		if ok {
			fsm.currentState = newState

			if err := fsm.switchEventByLengthParams(ctx, callbacks, param...); err != nil {
				fsm.currentState = currentState

				return fmt.Errorf("event (%v) from state(%v) enter state(%v): %w",
					action, currentState, newState, err)
			}

			return nil
		}
	}

	return fmt.Errorf("event (%v) state transition %w: %v", action, ErrNotFound, currentState)
}

func (fsm *FSM[E, S, P]) switchEventByLengthParams(ctx context.Context, stateTransition callbacks[E, S, P], param ...P) error {
	switch len(param) {
	case 0:
		if stateTransition.EnterNoParams != nil {
			return stateTransition.EnterNoParams(ctx, fsm)
		}

	case 1:
		if stateTransition.Enter != nil {
			return stateTransition.Enter(ctx, fsm, param[0])
		}
	}

	if stateTransition.EnterVariadic != nil {
		return stateTransition.EnterVariadic(ctx, fsm, param...)
	}

	return nil
}
