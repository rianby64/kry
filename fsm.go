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

	ErrLoopFound  errString = "loop found"
	ErrNotAllowed errString = "not allowed"
)

type InstanceFSM[State comparable, Param any] interface {
	Current() State
	Previous() State

	With(opts ...func(fsk InstanceFSM[State, Param]) InstanceFSM[State, Param]) InstanceFSM[State, Param]
	Apply(ctx context.Context, newState State, param ...Param) error

	ForceState(newState State) error
	IgnoreCurrentTransition()
}

type handlerNoParams[State comparable, Param any] = func(ctx context.Context, instance InstanceFSM[State, Param]) error
type handler[State comparable, Param any] = func(ctx context.Context, instance InstanceFSM[State, Param], param Param) error
type handlerVariadic[State comparable, Param any] = func(ctx context.Context, instance InstanceFSM[State, Param], param ...Param) error
type callbacks[State comparable, Param any] struct {
	Name          string
	EnterNoParams handlerNoParams[State, Param]
	Enter         handler[State, Param]
	EnterVariadic handlerVariadic[State, Param]
}

// Transition contains the name label, the source states, the destination state,
// and an optional callback that is executed when the transition is triggered.
// Declare the callback via OnEnter, OnEnterWith, or OnEnterVariadic.
type Transition[State comparable, Param any] struct {
	Name  string
	Src   []State
	SrcFn func(state State) bool // optional custom matching function for source states
	Dst   State
	DstFn func(state State) bool // optional custom matching function for destination states

	Enter TransitionHandler[State, Param]
}

type matchState[State comparable, Param any] struct {
	MatchSrc  func(state State) bool
	MatchDst  func(state State) bool
	Callbacks callbacks[State, Param]
}

type decoratorApply[State comparable, Param any] struct {
	expectToCallEnterNoParams []handlerNoParams[State, Param]
	expectToCallEnter         []handler[State, Param]
	expectToCallEnterVariadic []handlerVariadic[State, Param]
}

type FSM[State comparable, Param any] struct {
	id            uint64
	currentName   string // name of the currently executing transition, used by panic recovery
	currentState  State
	previousState State
	ignoreCurrent bool
	runningApply  bool

	states         map[State]struct{}
	path           map[State]map[State]callbacks[State, Param] // dst state -> src state -> callbacks
	pathByMatchSrc map[State][]matchState[State, Param]        // dst state -> list of match conditions for src states
	pathByMatchDst map[State][]matchState[State, Param]        // src state -> list of match conditions for dst states
	pathMatch      []matchState[State, Param]                  // list of match conditions for both src and dst states

	graphic        string
	historyKeeper  *historyKeeper[State, Param]
	decoratorApply *decoratorApply[State, Param]
	stackTrace     bool
	panicHandler   PanicHandler
	cloneHandler   CloneHandler[Param]
}

// New creates a new FSM instance with the given initial state, transitions, and options.
//
// The initial state and transitions are required and also these parameters are immutable after creation.
//
// The transitions define the allowed state changes.
func New[State comparable, Param any](
	initialState State,
	transitions []Transition[State, Param],
	options ...func(o *Options[Param]) *Options[Param],
) (*FSM[State, Param], error) {
	finalOptions := &Options[Param]{}
	for _, opt := range options {
		finalOptions = opt(finalOptions)
	}

	if finalOptions.cloneHandler == nil {
		finalOptions.cloneHandler = cloneHandler[Param]
	}

	path, pathByMatchSrc, pathByMatchDst, pathMatch, states, err := constructFromTransitions(initialState, transitions)
	if err != nil {
		return nil, err
	}

	idMachine++

	graphic := fmt.Sprintf("digraph fsm_%d {\n%s\n}", idMachine, VisualizeActions(transitions))

	return &FSM[State, Param]{
		id:             idMachine,
		currentState:   initialState,
		previousState:  initialState,
		path:           path,
		pathByMatchSrc: pathByMatchSrc,
		pathByMatchDst: pathByMatchDst,
		pathMatch:      pathMatch,
		states:         states,

		graphic: graphic,
		historyKeeper: newHistoryKeeper[State](
			finalOptions.historySize,
			finalOptions.stackTrace,
			finalOptions.cloneHandler,
		),
		stackTrace:   finalOptions.stackTrace,
		panicHandler: finalOptions.panicHandler,
		cloneHandler: finalOptions.cloneHandler,
	}, nil
}

func (fsk *FSM[State, Param]) String() string {
	return fsk.graphic
}

func (fsk *FSM[State, Param]) Current() State {
	return fsk.currentState
}

func (fsk *FSM[State, Param]) Previous() State {
	return fsk.previousState
}

func (fsk *FSM[State, Param]) ForceState(newState State) error {
	_, ok := fsk.states[newState]
	if !ok {
		return fmt.Errorf("state %w: %v", ErrUnknown, newState)
	}

	fsk.previousState = fsk.currentState
	fsk.currentState = newState

	return nil
}

func (fsk *FSM[State, Param]) IgnoreCurrentTransition() {
	if !fsk.runningApply {
		return
	}

	fsk.ignoreCurrent = true
}
