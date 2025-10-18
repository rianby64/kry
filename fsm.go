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
	id            uint64
	currentState  State
	currentAction Action
	states        map[State]struct{}
	path          map[Action]map[State]map[State]callbacks[Action, State, Param] // action -> dst state -> src state -> callbacks
	pathByMatch   map[Action]map[State][]matchState[Action, State, Param]        // action -> dst state -> list of match conditions for dst states

	events           map[Action]Transition[Action, State, Param]
	canTriggerEvents bool
	graphic          string
	historyKeeper    *historyKeeper[Action, State, Param]
}

func New[Action, State comparable, Param any](
	initialState State, // I don't want to allow this value to change after creation
	transitions []Transition[Action, State, Param], // also immutable after creation
	options ...func(o *Options) *Options,
) (*FSM[Action, State, Param], error) {
	finalOptions := &Options{}
	for _, opt := range options {
		finalOptions = opt(finalOptions)
	}

	path, pathByMatch, states, events,
		canTriggerEvents, err := constructFromTransitions(initialState, transitions)
	if err != nil {
		return nil, err
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
		graphic:          graphic,
		historyKeeper:    newHistoryKeeper[Action, State, Param](finalOptions.historySize),
	}, nil
}

func (fsk *FSM[Action, State, Param]) String() string {
	return fsk.graphic
}

func (fsk *FSM[Action, State, Param]) Current() State {
	return fsk.currentState
}

func (fsk *FSM[Action, State, Param]) ForceState(state State) error {
	_, ok := fsk.states[state]
	if !ok {
		return fmt.Errorf("state %w: %v", ErrUnknown, state)
	}

	currentState := fsk.currentState
	currentAction := fsk.currentAction
	fsk.currentState = state

	if errHistory := fsk.historyKeeper.
		PushForced(currentAction, currentState, state, nil); errHistory != nil {
		return fmt.Errorf("failed to push history item: %w", errHistory)
	}

	return nil
}
