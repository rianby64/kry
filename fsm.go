package kry

import (
	"context"
	"fmt"
)

type ctxKeyLoop int

// loopDetection keeps track of state transitions to detect loops.
// It maps from a source state to a map of destination states and their transition counts.
type loopDetection[State comparable] map[uint64]map[State]map[State]int

func newLoopDetection[State comparable](id uint64) loopDetection[State] {
	ld := loopDetection[State]{}
	ld[id] = make(map[State]map[State]int)

	return ld
}

func (ld loopDetection[State]) Inc(id uint64, stateFrom, stateTo State) {
	if _, ok := ld[id]; !ok {
		ld[id] = make(map[State]map[State]int)
	}

	if _, ok := ld[id][stateFrom]; !ok {
		ld[id][stateFrom] = make(map[State]int)
	}

	ld[id][stateFrom][stateTo]++
}

func (ld loopDetection[State]) Get(id uint64, stateFrom, stateTo State) int {
	if _, ok := ld[id]; !ok {
		return 0
	}

	if _, ok := ld[id][stateFrom]; !ok {
		return 0
	}

	return ld[id][stateFrom][stateTo]
}

type errString string

func (e errString) Error() string {
	return string(e)
}

const (
	loopKey ctxKeyLoop = 482 // just a random number

	ErrUnknown  errString = "unknown"
	ErrNotFound errString = "not found"
	ErrRepeated errString = "already exists"

	ErrLoopFound  errString = "loop found"
	ErrNotAllowed errString = "not allowed"
)

type InstanceFSM[Action, State comparable, Param any] interface {
	Current() State

	Event(ctx context.Context, action Action, param ...Param) error
	Apply(ctx context.Context, action Action, newState State, param ...Param) error

	ForceState(state State) error
}

type handlerNoParams[Action, State comparable, Param any] = func(ctx context.Context, instance InstanceFSM[Action, State, Param]) error
type handler[Action, State comparable, Param any] = func(ctx context.Context, instance InstanceFSM[Action, State, Param], param Param) error
type handlerVariadic[Action, State comparable, Param any] = func(ctx context.Context, instance InstanceFSM[Action, State, Param], param ...Param) error
type callbacks[Action, State comparable, Param any] struct {
	EnterNoParams handlerNoParams[Action, State, Param]
	Enter         handler[Action, State, Param]
	EnterVariadic handlerVariadic[Action, State, Param]
}

// Transition contains the name of the action, the source states, the destination state,
// and optional callbacks that are executed when the action is triggered.
type Transition[Action, State comparable, Param any] struct {
	Name Action
	Src  []State
	Dst  State

	Match func(state State) bool // optional custom matching function for source states

	EnterNoParams handlerNoParams[Action, State, Param]
	Enter         handler[Action, State, Param]
	EnterVariadic handlerVariadic[Action, State, Param]
}

type matchState[Action, State comparable, Param any] struct {
	Match     func(state State) bool // function to determine if transition is valid from the given state
	Callbacks callbacks[Action, State, Param]
}

type FSM[Action, State comparable, Param any] struct {
	id           uint64
	currentState State
	states       map[State]struct{}
	path         map[Action]map[State]map[State]callbacks[Action, State, Param] // action -> dst state -> src state -> callbacks
	pathByMatch  map[Action]map[State][]matchState[Action, State, Param]        // action -> dst state -> list of match conditions for dst states

	events           map[Action]Transition[Action, State, Param]
	canTriggerEvents bool

	gaphic string
}

var (
	idMachine uint64
)

func New[Action, State comparable, Param any](
	initialState State,
	transitions []Transition[Action, State, Param],
) (*FSM[Action, State, Param], error) {
	path := make(map[Action]map[State]map[State]callbacks[Action, State, Param])
	pathByMatch := make(map[Action]map[State][]matchState[Action, State, Param])
	states := map[State]struct{}{
		initialState: {},
	}
	canTriggerEvents := true
	events := make(map[Action]Transition[Action, State, Param])

	for _, transition := range transitions {
		action := transition.Name
		if _, ok := path[action]; !ok {
			path[action] = make(map[State]map[State]callbacks[Action, State, Param])
		}

		if _, ok := events[action]; ok {
			canTriggerEvents = false
		}

		if len(transition.Src) == 0 && transition.Match == nil {
			return nil, fmt.Errorf("for action %v neither src states nor matching function found: %w", action, ErrNotFound)
		}

		dst := transition.Dst
		if _, ok := path[action][dst]; !ok {
			path[action][dst] = make(map[State]callbacks[Action, State, Param])
		}

		if transition.Match != nil {
			if _, ok := pathByMatch[action]; !ok {
				pathByMatch[action] = make(map[State][]matchState[Action, State, Param])
			}

			if _, ok := pathByMatch[action][dst]; !ok {
				pathByMatch[action][dst] = make([]matchState[Action, State, Param], 0)
			}

			pathByMatch[action][dst] = append(pathByMatch[action][dst], matchState[Action, State, Param]{
				Match: transition.Match,
				Callbacks: callbacks[Action, State, Param]{
					EnterVariadic: transition.EnterVariadic,
					Enter:         transition.Enter,
					EnterNoParams: transition.EnterNoParams,
				},
			})
		}

		for _, src := range transition.Src {
			if _, ok := path[action][dst][src]; ok {
				return nil, fmt.Errorf(
					"action %v from state %v to state %v: %w",
					action, src, dst, ErrRepeated,
				)
			}

			path[action][dst][src] = callbacks[Action, State, Param]{
				EnterVariadic: transition.EnterVariadic,
				Enter:         transition.Enter,
				EnterNoParams: transition.EnterNoParams,
			}
		}

		for _, state := range transition.Src {
			states[state] = struct{}{}
		}

		events[action] = transition
		states[transition.Dst] = struct{}{}
	}

	if !canTriggerEvents {
		events = nil
	}

	idMachine++

	graphic := fmt.Sprintf("digraph fsm_%d {\n%s\n}", idMachine, VisualizeActions(transitions))

	return &FSM[Action, State, Param]{
		id:           idMachine,
		currentState: initialState,
		path:         path,
		pathByMatch:  pathByMatch,
		states:       states,

		events:           events,
		canTriggerEvents: canTriggerEvents,
		gaphic:           graphic,
	}, nil
}

func (fsk *FSM[Action, State, Param]) String() string {
	return fsk.gaphic
}

func (fsk *FSM[Action, State, Param]) Current() State {
	return fsk.currentState
}

func (fsk *FSM[Action, State, Param]) ForceState(state State) error {
	_, ok := fsk.states[state]
	if !ok {
		return fmt.Errorf("state %w: %v", ErrUnknown, state)
	}

	fsk.currentState = state

	return nil
}

func (fsk *FSM[Action, State, Param]) Event(ctx context.Context, action Action, param ...Param) error {
	if !fsk.canTriggerEvents {
		return fmt.Errorf("event %v: %w", action, ErrNotAllowed)
	}

	foundEvent, ok := fsk.events[action]
	if !ok {
		return fmt.Errorf("event %w: %v", ErrUnknown, action)
	}

	newState := foundEvent.Dst

	if err := fsk.Apply(ctx, action, newState, param...); err != nil {
		return fmt.Errorf("failed to apply event %v: %w", action, err)
	}

	return nil
}

func (fsk *FSM[Action, State, Param]) checkLoop(
	ctx context.Context,
	currentState,
	newState State,
) (context.Context, error) {
	var (
		loopEx      loopDetection[State]
		ctxWithLoop context.Context
		ok          bool
	)

	loopFromCtx := ctx.Value(loopKey)
	if loopFromCtx == nil {
		loopEx = newLoopDetection[State](fsk.id)
		ctxWithLoop = context.WithValue(ctx, loopKey, loopEx)
	} else {
		loopEx, ok = loopFromCtx.(loopDetection[State])
		if !ok {
			return nil, fmt.Errorf("type assertion for loop detection failed: %w", ErrUnknown)
		}
		ctxWithLoop = ctx
	}

	if loopEx.Get(fsk.id, currentState, newState) > 0 {
		return nil, fmt.Errorf("from '%v' to '%v': %w",
			currentState, newState, ErrLoopFound)
	}

	loopEx.Inc(fsk.id, currentState, newState)

	return ctxWithLoop, nil
}

func (fsk *FSM[Action, State, Param]) apply(
	ctx context.Context,
	callbacks callbacks[Action, State, Param],
	action Action,
	currentState, newState State,
	param ...Param,
) error {
	fsk.currentState = newState

	if err := fsk.switchEventByLengthParams(ctx, callbacks, param...); err != nil {
		fsk.currentState = currentState

		return fmt.Errorf("failed to apply (%v) from '%v' to '%v': %w",
			action, currentState, newState, err)
	}

	return nil
}

func (fsk *FSM[Action, State, Param]) applyByExact(ctx context.Context, action Action, newState State, param ...Param) (bool, error) {
	foundAction := fsk.path[action]
	currentState := fsk.currentState

	foundDstState, ok := foundAction[newState]
	if !ok {
		return false, nil
	}

	callbacks, ok := foundDstState[currentState]
	if !ok {
		return false, nil
	}

	if err := fsk.apply(ctx, callbacks, action, currentState, newState, param...); err != nil {
		return false, err
	}

	return true, nil
}

func (fsk *FSM[Action, State, Param]) applyByMatch(ctx context.Context, action Action, newState State, param ...Param) (bool, error) {
	currentState := fsk.currentState
	foundActionByMatch, ok := fsk.pathByMatch[action]
	if !ok {
		return false, nil
	}

	foundDstByMatch, ok := foundActionByMatch[newState]
	if !ok {
		return false, nil
	}

	for _, matchState := range foundDstByMatch {
		if matchState.Match(currentState) {
			if err := fsk.apply(ctx, matchState.Callbacks, action, currentState, newState, param...); err != nil {
				return false, err
			}

			return true, nil
		}
	}

	return false, nil
}

func (fsk *FSM[Action, State, Param]) Apply(ctx context.Context, action Action, newState State, param ...Param) error {
	currentState := fsk.currentState

	ctxWithLoop, err := fsk.checkLoop(ctx, currentState, newState)
	if err != nil {
		return fmt.Errorf("failed to apply (%v): %w", action, err)
	}

	if _, ok := fsk.path[action]; !ok {
		return fmt.Errorf("action %w: %v", ErrUnknown, action)
	}

	if applied, err := fsk.applyByExact(ctxWithLoop, action, newState, param...); err != nil {
		return err
	} else if applied {
		return nil
	}

	if applied, err := fsk.applyByMatch(ctxWithLoop, action, newState, param...); err != nil {
		return err
	} else if applied {
		return nil
	}

	return fmt.Errorf("transition (%v) from state %w: %v", action, ErrNotFound, currentState)
}

func (fsk *FSM[Action, State, Param]) switchEventByLengthParams(ctx context.Context, stateTransition callbacks[Action, State, Param], param ...Param) error {
	switch len(param) {
	case 0:
		if stateTransition.EnterNoParams != nil {
			if err := stateTransition.EnterNoParams(ctx, fsk); err != nil {
				return fmt.Errorf("failed to execute enter (no-params) callback: %w", err)
			}

			return nil
		}

	case 1:
		if stateTransition.Enter != nil {
			if err := stateTransition.Enter(ctx, fsk, param[0]); err != nil {
				return fmt.Errorf("failed to execute enter (single-param) callback: %w", err)
			}

			return nil
		}
	}

	if stateTransition.EnterVariadic != nil {
		if err := stateTransition.EnterVariadic(ctx, fsk, param...); err != nil {
			return fmt.Errorf("failed to execute enter (variadic) callback: %w", err)
		}

		return nil
	}

	return nil
}
