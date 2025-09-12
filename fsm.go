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
	ErrRepeated errString = "already exists"
)

type InstanceFSM[Action, State, Param comparable] interface {
	Current() State

	// Event(ctx context.Context, event E, param ...P) error // TODO: ask why this method should be here. If YES, then I've to deal with infinity loops

	ForceState(state State) error
}

type callbacks[Action, State, Param comparable] struct {
	EnterNoParams func(ctx context.Context, instance InstanceFSM[Action, State, Param]) error
	Enter         func(ctx context.Context, instance InstanceFSM[Action, State, Param], param Param) error
	EnterVariadic func(ctx context.Context, instance InstanceFSM[Action, State, Param], param ...Param) error
}

// Transition contains the name of the action, the source states, the destination state,
// and optional callbacks that are executed when the action is triggered.
type Transition[Action, State, Param comparable] struct {
	Name Action
	Src  []State
	Dst  State

	EnterNoParams func(ctx context.Context, instance InstanceFSM[Action, State, Param]) error
	Enter         func(ctx context.Context, instance InstanceFSM[Action, State, Param], param Param) error
	EnterVariadic func(ctx context.Context, instance InstanceFSM[Action, State, Param], param ...Param) error
}

type FSK[Action, State, Param comparable] struct {
	currentState State
	states       map[State]struct{}
	path         map[Action]map[State]map[State]callbacks[Action, State, Param]
}

func New[Action, State, Param comparable](
	initialState State,
	transitions []Transition[Action, State, Param],
) (*FSK[Action, State, Param], error) {
	path := make(map[Action]map[State]map[State]callbacks[Action, State, Param])
	states := map[State]struct{}{
		initialState: {},
	}

	for _, transition := range transitions {
		action := transition.Name
		if _, ok := path[action]; !ok {
			path[action] = make(map[State]map[State]callbacks[Action, State, Param])
		}

		dst := transition.Dst

		for _, src := range transition.Src {
			if _, ok := path[action][src]; !ok {
				path[action][src] = make(map[State]callbacks[Action, State, Param])
			}

			if _, ok := path[action][src][dst]; ok {
				return nil, fmt.Errorf(
					"action %v from state %v to state %v: %w",
					action, src, dst, ErrRepeated,
				)
			}

			path[action][src][dst] = callbacks[Action, State, Param]{
				EnterVariadic: transition.EnterVariadic,
				Enter:         transition.Enter,
				EnterNoParams: transition.EnterNoParams,
			}
		}

		for _, state := range transition.Src {
			states[state] = struct{}{}
		}

		states[transition.Dst] = struct{}{}
	}

	return &FSK[Action, State, Param]{
		currentState: initialState,
		path:         path,
		states:       states,
	}, nil
}

func (fsk *FSK[Action, State, Param]) Current() State {
	return fsk.currentState
}

func (fsk *FSK[Action, State, Param]) ForceState(state State) error {
	_, ok := fsk.states[state]
	if !ok {
		return fmt.Errorf("state %w: %v", ErrUnknown, state)
	}

	fsk.currentState = state

	return nil
}

func (fsk *FSK[Action, State, Param]) Apply(ctx context.Context, action Action, newState State, param ...Param) error {
	currentState := fsk.currentState
	foundAction, ok := fsk.path[action]
	if !ok {
		return fmt.Errorf("action %w: %v", ErrUnknown, action)
	}

	foundSrcState, ok := foundAction[currentState]
	if ok {
		callbacks, ok := foundSrcState[newState]
		if ok {
			fsk.currentState = newState

			if err := fsk.switchEventByLengthParams(ctx, callbacks, param...); err != nil {
				fsk.currentState = currentState

				return fmt.Errorf("failed to apply (%v) from %v to %v: %w",
					action, currentState, newState, err)
			}

			return nil
		}
	}

	return fmt.Errorf("transition (%v) from state %w: %v", action, ErrNotFound, currentState)
}

func (fsk *FSK[Action, State, Param]) switchEventByLengthParams(ctx context.Context, stateTransition callbacks[Action, State, Param], param ...Param) error {
	switch len(param) {
	case 0:
		if stateTransition.EnterNoParams != nil {
			return stateTransition.EnterNoParams(ctx, fsk)
		}

	case 1:
		if stateTransition.Enter != nil {
			return stateTransition.Enter(ctx, fsk, param[0])
		}
	}

	if stateTransition.EnterVariadic != nil {
		return stateTransition.EnterVariadic(ctx, fsk, param...)
	}

	return nil
}
