package kry

import (
	"context"
	"errors"
	"fmt"
)

func (fsk *FSM[State, Param]) apply(
	ctx context.Context,
	cbs callbacks[State, Param],
	from, to State,
	param ...Param,
) error {
	currentHistoryKeeper := fsk.historyKeeper

	currentState := fsk.currentState
	previousState := fsk.previousState

	fsk.currentName = cbs.Name
	fsk.currentState = to
	fsk.previousState = currentState
	fsk.runningApply = true

	expectFailed := fsk.checkCallbacksAgainstExpectHandlers(cbs)
	historyKeeper := newHistoryKeeper[State](
		fsk.historyKeeper.maxLength,
		fsk.stackTrace,
		fsk.cloneHandler,
	)
	fsk.historyKeeper = historyKeeper

	defer func() {
		currentHistoryKeeper.Append(historyKeeper)
		fsk.historyKeeper = currentHistoryKeeper
		fsk.runningApply = false

		if fsk.ignoreCurrent {
			fsk.ignoreCurrent = false

			fsk.currentState = currentState
			fsk.previousState = previousState
		}
	}()

	if err := fsk.applyTransitionByLengthParams(
		ctx, cbs, param...,
	); err != nil {
		ignored := fsk.ignoreCurrent
		fsk.ignoreCurrent = true

		if intermediateKeeper, errHistory := fsk.intermediateKeeper(
			historyKeeper,
			cbs.Name, from, to,
			errors.Unwrap(err), ignored, expectFailed, param...,
		); errHistory != nil {
			err = fmt.Errorf("%w: %w", err, errHistory)
		} else {
			historyKeeper = intermediateKeeper
		}

		return fmt.Errorf("failed to apply %q from '%v' to '%v': %w",
			cbs.Name, from, to, err)
	}

	if intermediateKeeper, errHistory := fsk.intermediateKeeper(
		historyKeeper,
		cbs.Name, from, to,
		nil, fsk.ignoreCurrent, expectFailed, param...,
	); errHistory != nil {
		return fmt.Errorf("failed to keep forced history: %w", errHistory)
	} else {
		historyKeeper = intermediateKeeper
	}

	return nil
}

func (fsk *FSM[State, Param]) applyByExact(ctx context.Context, newState State, param ...Param) (bool, error) {
	currentState := fsk.currentState

	foundDstState, ok := fsk.path[newState]
	if !ok {
		return false, nil
	}

	cbs, ok := foundDstState[currentState]
	if !ok {
		return false, nil
	}

	if err := fsk.apply(ctx, cbs, currentState, newState, param...); err != nil {
		return false, err
	}

	return true, nil
}

type matchType int

const (
	matchSrc matchType = iota + 1
	matchDst
)

func (fsk *FSM[State, Param]) applyByMatchSrcDst(ctx context.Context, mt matchType, newState State, param ...Param) (bool, error) {
	currentState := fsk.currentState
	var (
		foundStateByMatch []matchState[State, Param]
		ok                bool
	)

	switch mt {
	case matchSrc:
		foundStateByMatch, ok = fsk.pathByMatchSrc[newState]
		if !ok {
			return false, nil
		}

	case matchDst:
		foundStateByMatch, ok = fsk.pathByMatchDst[currentState]
		if !ok {
			return false, nil
		}
	}

	for _, ms := range foundStateByMatch {
		switch mt {
		case matchSrc:
			if ms.MatchSrc(currentState) {
				if err := fsk.apply(ctx, ms.Callbacks, currentState, newState, param...); err != nil {
					return false, err
				}

				return true, nil
			}

		case matchDst:
			if ms.MatchDst(newState) {
				if err := fsk.apply(ctx, ms.Callbacks, currentState, newState, param...); err != nil {
					return false, err
				}

				return true, nil
			}
		}
	}

	return false, nil
}

func (fsk *FSM[State, Param]) applyByMatch(ctx context.Context, newState State, param ...Param) (bool, error) {
	currentState := fsk.currentState

	for _, ms := range fsk.pathMatch {
		if ms.MatchSrc(currentState) && ms.MatchDst(newState) {
			if err := fsk.apply(ctx, ms.Callbacks, currentState, newState, param...); err != nil {
				return false, err
			}

			return true, nil
		}
	}

	return false, nil
}

func (fsk *FSM[State, Param]) applyTransitionByLengthParams(
	ctx context.Context, stateTransition callbacks[State, Param], param ...Param,
) error {
	switch {
	case stateTransition.Enter != nil:
		if len(param) != 1 {
			return fmt.Errorf("%w: expected 1 param, got %d", ErrNotAllowed, len(param))
		}
		if err := stateTransition.Enter(ctx, fsk, param[0]); err != nil {
			return fmt.Errorf("failed to execute enter (single-param) callback: %w", err)
		}

	case stateTransition.EnterVariadic != nil:
		if err := stateTransition.EnterVariadic(ctx, fsk, param...); err != nil {
			return fmt.Errorf("failed to execute enter (variadic) callback: %w", err)
		}
	}

	return nil
}

func (fsk *FSM[State, Param]) generateTransitionMsg(curr, next State) string {
	return fmt.Sprintf("transition from %v to %v", curr, next)
}

func (fsk *FSM[State, Param]) Apply(
	ctx context.Context, newState State, param ...Param,
) error {
	currentState := fsk.currentState

	defer func() {
		if errPanic := recover(); errPanic != nil {
			defer func() {
				fsk.currentState = currentState // rollback state
			}()

			err := fmt.Errorf("%v", errPanic)

			if errHistory := fsk.historyKeeper.Push(
				fsk.currentName, currentState, newState,
				err, defaultSkipStackTrace, fsk.ignoreCurrent, false,
				param...,
			); errHistory != nil {
				err = fmt.Errorf("%v:%w: failed to push history item: %w",
					fsk.generateTransitionMsg(currentState, newState), err, errHistory,
				)
			}

			if fsk.panicHandler != nil {
				fsk.panicHandler(ctx, errPanic)

				return
			}

			panic(err)
		}
	}()

	ctxWithLoop, err := fsk.checkLoop(ctx, currentState, newState)
	if err != nil {
		return fmt.Errorf("failed to apply: %w", err)
	}

	if applied, err := fsk.applyByExact(ctxWithLoop, newState, param...); err != nil {
		return fmt.Errorf("%v: %w", fsk.generateTransitionMsg(currentState, newState), err)
	} else if applied {
		return nil
	}

	if applied, err := fsk.applyByMatchSrcDst(ctxWithLoop, matchSrc, newState, param...); err != nil {
		return fmt.Errorf("%v: %w", fsk.generateTransitionMsg(currentState, newState), err)
	} else if applied {
		return nil
	}

	if applied, err := fsk.applyByMatchSrcDst(ctxWithLoop, matchDst, newState, param...); err != nil {
		return fmt.Errorf("%v: %w", fsk.generateTransitionMsg(currentState, newState), err)
	} else if applied {
		return nil
	}

	if applied, err := fsk.applyByMatch(ctxWithLoop, newState, param...); err != nil {
		return fmt.Errorf("%v: %w", fsk.generateTransitionMsg(currentState, newState), err)
	} else if applied {
		return nil
	}

	err = ErrNotFound
	if errHistory := fsk.historyKeeper.Push(
		"", currentState, newState,
		err, defaultSkipStackTrace, fsk.ignoreCurrent, false,
		param...,
	); errHistory != nil {
		err = fmt.Errorf("%w: failed to push history item: %w", err, errHistory)
	}

	return fmt.Errorf("%v: %w", fsk.generateTransitionMsg(currentState, newState), err)
}
