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
