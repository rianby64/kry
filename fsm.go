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
	Previous() State

	Event(ctx context.Context, action Action, param ...Param) error
	Apply(ctx context.Context, action Action, newState State, param ...Param) error

	ForceState(newState State) error
	IgnoreCurrentTransition()
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
	Name  Action
	Src   []State
	SrcFn func(state State) bool // optional custom matching function for source states
	Dst   State
	DstFn func(state State) bool // optional custom matching function for destination states

	EnterNoParams handlerNoParams[Action, State, Param]
	Enter         handler[Action, State, Param]
	EnterVariadic handlerVariadic[Action, State, Param]
}

type matchState[Action, State comparable, Param any] struct {
	MatchSrc  func(state State) bool // function to determine if transition is valid from the given state
	MatchDst  func(state State) bool // function to determine if transition is valid to the given state
	Callbacks callbacks[Action, State, Param]
}

type FSM[Action, State comparable, Param any] struct {
	id            uint64
	currentAction Action
	currentState  State
	previousState State
	ignoreCurrent bool
	runningApply  bool

	states         map[State]struct{}
	path           map[Action]map[State]map[State]callbacks[Action, State, Param] // action -> dst state -> src state -> callbacks
	pathByMatchSrc map[Action]map[State][]matchState[Action, State, Param]        // action -> dst state -> list of match conditions for dst states
	pathByMatchDst map[Action]map[State][]matchState[Action, State, Param]        // action -> src state -> list of match conditions for src states
	pathMatch      map[Action][]matchState[Action, State, Param]                  // action -> list of match conditions for both src and dst states
	events         map[Action]Transition[Action, State, Param]                    // action -> transition

	canTriggerEvents bool
	graphic          string
	historyKeeper    *historyKeeper[Action, State, Param]
	stackTrace       bool
	panicHandler     PanicHandler
	cloneHandler     CloneHandler[Param]
}

// New creates a new FSM instance with the given initial state, transitions, and options.
//
// The initial state and transitions are required and also these parameters are immutable after creation.
//
// The transitions define the allowed state changes.
func New[Action, State comparable, Param any](
	initialState State,
	transitions []Transition[Action, State, Param],
	options ...func(o *Options[Param]) *Options[Param],
) (*FSM[Action, State, Param], error) {
	finalOptions := &Options[Param]{}
	for _, opt := range options {
		finalOptions = opt(finalOptions)
	}

	if finalOptions.cloneHandler == nil {
		finalOptions.cloneHandler = cloneHandler[Param]
	}

	path, pathByMatchSrc, pathByMatchDst, pathMatch, states, events,
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
		id:             idMachine,
		currentState:   initialState,
		path:           path,
		pathByMatchSrc: pathByMatchSrc,
		pathByMatchDst: pathByMatchDst,
		pathMatch:      pathMatch,
		states:         states,

		events:           events,
		canTriggerEvents: canTriggerEvents,
		graphic:          graphic,
		historyKeeper: newHistoryKeeper[Action, State](
			finalOptions.historySize,
			finalOptions.stackTrace,
			finalOptions.cloneHandler,
		),
		stackTrace:   finalOptions.stackTrace,
		panicHandler: finalOptions.panicHandler,
		cloneHandler: finalOptions.cloneHandler,
	}, nil
}

func (fsk *FSM[Action, State, Param]) String() string {
	return fsk.graphic
}

func (fsk *FSM[Action, State, Param]) Current() State {
	return fsk.currentState
}

func (fsk *FSM[Action, State, Param]) Previous() State {
	return fsk.previousState
}

func (fsk *FSM[Action, State, Param]) ForceState(newState State) error {
	_, ok := fsk.states[newState]
	if !ok {
		return fmt.Errorf("state %w: %v", ErrUnknown, newState)
	}

	fsk.previousState = fsk.currentState
	fsk.currentState = newState

	return nil
}

func (fsk *FSM[Action, State, Param]) IgnoreCurrentTransition() {
	if !fsk.runningApply {
		return
	}

	fsk.ignoreCurrent = true
}
