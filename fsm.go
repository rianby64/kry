package kry

import (
	"context"
	"fmt"
	"slices"
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

	Enter func(ctx context.Context, instance InstanceFSM[E, S, P], param ...P) error
}

type Event[E, S, P comparable] struct {
	Name E
	Src  []S
	Dst  S

	Enter func(ctx context.Context, instance InstanceFSM[E, S, P], param ...P) error
}

type FSM[E, S, P comparable] struct {
	currentState S

	mapEvents map[E]eventTransitionState[E, S, P]
	states    map[S]struct{}
}

func New[E, S, P comparable](initialState S, events []Event[E, S, P]) *FSM[E, S, P] {
	states := map[S]struct{}{
		initialState: {},
	}

	mapEvents := make(map[E]eventTransitionState[E, S, P])
	for _, event := range events {
		mapEvents[event.Name] = eventTransitionState[E, S, P]{
			Src:   event.Src,
			Dst:   event.Dst,
			Enter: event.Enter,
		}

		for _, state := range event.Src {
			states[state] = struct{}{}
		}

		states[event.Dst] = struct{}{}
	}

	f := &FSM[E, S, P]{
		currentState: initialState,
		mapEvents:    mapEvents,
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

func (fsm *FSM[E, S, P]) Event(ctx context.Context, event E, param ...P) error {
	stateTransition, ok := fsm.mapEvents[event]
	if !ok {
		return fmt.Errorf("event %w: %v", ErrUnknown, event)
	}

	currentState := fsm.currentState

	if slices.Contains(stateTransition.Src, currentState) {
		newState := stateTransition.Dst
		fsm.currentState = newState

		if stateTransition.Enter == nil {
			return nil
		}

		err := stateTransition.Enter(ctx, fsm, param...)
		if err != nil {
			fsm.currentState = currentState

			return fmt.Errorf("event (%v) from state(%v) enter state(%v): %w",
				event, currentState, newState, err)
		}

		return nil
	}

	return fmt.Errorf("event (%v) state transition %w: %v", event, ErrNotFound, currentState)
}
